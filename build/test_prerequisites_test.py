import copy
from concurrent.futures import ThreadPoolExecutor
import hashlib
from http.server import BaseHTTPRequestHandler, HTTPServer
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import threading
import time
from types import SimpleNamespace
import unittest
from unittest import mock
import urllib.error
import urllib.request


SPEC = importlib.util.spec_from_file_location("prerequisites", Path(__file__).with_name("test-prerequisites.py"))
TOOLS = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(TOOLS)


class TestPrerequisites(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.requests = []
        self.payloads = {}
        self.manifest = {}
        for name in ("node", "jq", "shellcheck"):
            payload = b"#!/bin/sh\nexit 0\n" if name != "shellcheck" else b"#!/bin/sh\nprintf 'version: 0.10.0\\n'\n"
            if name == "jq":
                payload = b"#!/bin/sh\nprintf 'jq-1.7.1\\n'\n"
            digest = hashlib.sha256(payload).hexdigest()
            self.payloads["/" + name] = payload
            self.manifest[name] = {"version": "fixture", "url": "https://official.invalid/" + name, "sha256": digest, "member": None, "binary_sha256": digest}
        requests, payloads = self.requests, self.payloads

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                requests.append(self.path)
                payload = payloads[self.path]
                self.send_response(200)
                self.send_header("Content-Type", "application/octet-stream")
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)

            def log_message(self, *arguments):
                pass

        server = HTTPServer(("127.0.0.1", 0), Handler)
        self.server_port = server.server_port
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        self.addCleanup(thread.join)
        self.addCleanup(server.server_close)
        self.addCleanup(server.shutdown)
        opener = urllib.request.urlopen

        def local_open(url, timeout):
            self.assertEqual(10, timeout)
            response = opener("http://127.0.0.1:%d/%s" % (server.server_port, url.rsplit("/", 1)[1]), timeout=timeout)
            response.geturl = lambda: url
            return response

        patcher = mock.patch.object(TOOLS, "open_release", side_effect=local_open)
        patcher.start()
        self.addCleanup(patcher.stop)
        original = TOOLS.usable
        patcher = mock.patch.object(TOOLS, "usable", side_effect=lambda name, search: False if search == "missing" else original(name, search))
        patcher.start()
        self.addCleanup(patcher.stop)

    def prepare(self, manifest=None, system=None):
        return TOOLS.prepare(manifest or self.manifest, self.root / "cache", "missing", system or ("Linux", "x86_64"))

    def test_fresh_and_verified_cache(self):
        first = self.prepare()
        self.assertEqual(["/node", "/jq", "/shellcheck"], self.requests)
        self.assertEqual(first, self.prepare())
        self.assertEqual(3, len(self.requests))
        self.assertEqual(0o700, (self.root / "cache").stat().st_mode & 0o777)

    def test_only_missing_tools_and_existing_path(self):
        with mock.patch.object(TOOLS, "usable", side_effect=lambda name, search: name != "jq" or search != "missing"):
            result = self.prepare()
        self.assertEqual(["/jq"], self.requests)
        self.assertTrue(result.endswith(os.pathsep + "missing"))
        with mock.patch.object(TOOLS, "usable", return_value=True):
            self.assertEqual("original", TOOLS.prepare(self.manifest, self.root / "unused", "original", ("Darwin", "arm64")))
        self.assertFalse((self.root / "unused").exists())

    def test_tampered_cache_is_not_executed_or_redownloaded(self):
        search = self.prepare()
        target = Path(search.split(os.pathsep)[0]) / "node"
        target.write_bytes(b"#!/bin/sh\nexit 99\n")
        with self.assertRaisesRegex(RuntimeError, "checksum mismatch"):
            self.prepare()
        self.assertEqual(3, len(self.requests))

    def test_checksum_rejected_before_archive_parsing(self):
        manifest = copy.deepcopy(self.manifest)
        manifest["node"]["sha256"] = "0" * 64
        manifest["node"]["member"] = "some/member"
        with mock.patch.object(TOOLS.tarfile, "open", side_effect=AssertionError("must not parse")):
            with self.assertRaisesRegex(RuntimeError, "archive checksum mismatch"):
                self.prepare(manifest)
        self.assertEqual(["/node"], self.requests)
        self.assertFalse(list((self.root / "cache").glob("*/node")))

    def test_exact_regular_archive_member_and_binary_hash(self):
        payload = self.payloads["/node"]
        for scenario in ("good", "symlink", "missing", "bad-binary"):
            with self.subTest(scenario=scenario):
                archive = io.BytesIO()
                with tarfile.open(fileobj=archive, mode="w:xz") as writer:
                    member = tarfile.TarInfo("node/bin/node" if scenario != "missing" else "other")
                    member.size = len(payload)
                    if scenario == "symlink":
                        member.type = tarfile.SYMTYPE
                        member.linkname = "/outside"
                        member.size = 0
                    writer.addfile(member, io.BytesIO(payload))
                self.payloads["/node"] = archive.getvalue()
                entry = dict(self.manifest["node"], sha256=hashlib.sha256(archive.getvalue()).hexdigest(), member="node/bin/node")
                if scenario == "bad-binary":
                    entry["binary_sha256"] = "0" * 64
                if scenario == "good":
                    self.assertEqual(payload, TOOLS.download(entry))
                else:
                    with self.assertRaises(RuntimeError):
                        TOOLS.download(entry)

    def test_partial_failure_never_returns_path_and_is_retryable(self):
        manifest = copy.deepcopy(self.manifest)
        manifest["jq"]["sha256"] = "0" * 64
        with self.assertRaises(RuntimeError):
            self.prepare(manifest)
        self.assertEqual(["/node", "/jq"], self.requests)
        self.prepare()
        self.assertEqual(["/node", "/jq", "/jq", "/shellcheck"], self.requests)
        self.assertFalse(list((self.root / "cache").glob("*/.download-*")))

    def test_network_failure_is_bounded_and_platform_is_clear(self):
        started = time.monotonic()
        with mock.patch.object(TOOLS, "open_release", side_effect=urllib.error.URLError("offline")):
            with self.assertRaises(urllib.error.URLError):
                self.prepare()
        self.assertLess(time.monotonic() - started, 2)
        with self.assertRaisesRegex(RuntimeError, "Linux x86_64 only"):
            self.prepare(system=("Darwin", "arm64"))

    def test_requirement_versions_and_redirect_rejection(self):
        for name, version, expected in (("jq", "jq-1.5", False), ("jq", "jq-1.6", True), ("shellcheck", "version: 0.8.0", False), ("shellcheck", "version: 0.9.0", True)):
            with self.subTest(name=name, version=version):
                with mock.patch.object(TOOLS.shutil, "which", return_value=str(self.root / name)), mock.patch.object(TOOLS.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, version.encode(), b"")):
                    self.assertEqual(expected, TOOLS.usable(name, "version-only-fixture"))
        for returncode in (0, 1):
            with mock.patch.object(TOOLS.shutil, "which", return_value=str(self.root / "node")), mock.patch.object(TOOLS.subprocess, "run", return_value=subprocess.CompletedProcess([], returncode, b"", b"")):
                self.assertEqual(returncode == 0, TOOLS.usable("node", "version-only-fixture"))
        with self.assertRaisesRegex(RuntimeError, "requires HTTPS"):
            TOOLS.HTTPSRedirect().redirect_request(None, None, 302, "", {}, "http://outside.invalid")

    def test_cache_symlink_and_insecure_mode_fail(self):
        outside = self.root / "outside"
        outside.mkdir(mode=0o700)
        (self.root / "cache").symlink_to(outside)
        with self.assertRaises((OSError, RuntimeError)):
            self.prepare()
        (self.root / "cache").unlink()
        (self.root / "cache").mkdir(mode=0o755)
        with self.assertRaises(RuntimeError):
            self.prepare()

    def test_default_make_consumer_preserves_command_and_environment(self):
        repository = Path(__file__).resolve().parent.parent
        fixture = self.root / "repository"
        (fixture / "build").mkdir(parents=True)
        shutil.copyfile(repository / "Makefile", fixture / "Makefile")
        shutil.copyfile(repository / "build/test-prerequisites.py", fixture / "build/test-prerequisites.py")
        (fixture / "build/test-prerequisites.json").write_text(json.dumps(self.manifest))
        binary = self.root / "bin"
        binary.mkdir()
        for name in ("bash", "make", "tar", "sha256sum", "unzip"):
            executable = shutil.which(name)
            self.assertIsNotNone(executable, "missing baseline prerequisite: " + name)
            (binary / name).symlink_to(executable)
        (binary / "python3").write_text(
            "#!" + sys.executable + "\n"
            "import os,runpy,sys,urllib.request\nfrom unittest import mock\n"
            "original=urllib.request.build_opener()\n"
            "class LocalRelease:\n"
            " def open(self,url,timeout):\n"
            "  if not url.startswith('https://official.invalid/'): raise RuntimeError('nonfixture network forbidden')\n"
            "  response=original.open('http://127.0.0.1:' + os.environ['FIXTURE_PORT'] + '/' + url.rsplit('/',1)[1],timeout=timeout)\n"
            "  response.geturl=lambda:url\n  return response\n"
            "sys.argv=sys.argv[1:]\n"
            "with mock.patch.object(urllib.request,'build_opener',return_value=LocalRelease()):\n"
            " runpy.run_path(sys.argv[0],run_name='__main__')\n"
        )
        (binary / "python3").chmod(0o700)
        marker = self.root / "invocation"
        (binary / "go").write_text(
            "#!" + sys.executable + "\nimport json,os,pathlib,stat,sys\n"
            "paths=os.environ['PATH'].split(os.pathsep)[:3]\n"
            "cache=pathlib.Path(paths[0]).parent\n"
            "record={'argv':sys.argv[1:],'flags':os.environ['GOFLAGS'],'data':os.environ['XDG_DATA_HOME'],"
            "'sentinel':os.environ['OWNED_SENTINEL'],'cache':str(cache),'paths':paths,"
            "'private_mode':stat.S_IMODE(cache.parent.stat().st_mode),'cache_mode':stat.S_IMODE(cache.stat().st_mode),"
            "'private_uid':cache.parent.stat().st_uid,'cache_uid':cache.stat().st_uid,'uid':os.getuid(),"
            "'private':all(pathlib.Path(path).stat().st_uid==os.getuid() and stat.S_IMODE(pathlib.Path(path).stat().st_mode)==0o700 for path in [*paths,str(cache),str(cache.parent)])}\n"
            "pathlib.Path(os.environ['OWNED_MARKER']).write_text(json.dumps(record))\n"
            "sys.exit(int(os.environ['FIXTURE_EXIT']))\n"
        )
        (binary / "go").chmod(0o700)
        temporary = self.root / "temporary"
        temporary.mkdir(mode=0o700)
        unwritable = self.root / "unwritable-home"
        unwritable.mkdir(mode=0o500)
        self.addCleanup(unwritable.chmod, 0o700)
        caches = []
        cases = [(mode, home, child_exit, False) for mode in (0o700, 0o2700) for home, child_exit in (("/", 17), (str(unwritable), 17), ("/", 0))]
        cases.extend((0o2700, "/", child_exit, True) for child_exit in (0, 17))
        for mode, home, child_exit, direct in cases:
            with self.subTest(mode=oct(mode), home=home, child_exit=child_exit, direct=direct):
                temporary.chmod(mode)
                base_info = temporary.stat()
                self.assertEqual(os.getuid(), base_info.st_uid)
                self.assertEqual(mode, stat.S_IMODE(base_info.st_mode))
                with tempfile.TemporaryDirectory(dir=temporary) as raw:
                    self.assertEqual(mode, stat.S_IMODE(Path(raw).stat().st_mode))
                if marker.exists():
                    marker.unlink()
                environment = dict(os.environ, PATH=str(binary), HOME=home, TMPDIR=str(temporary), GOFLAGS="-mod=readonly -p=1", XDG_DATA_HOME="/tmp/owned-data", OWNED_MARKER=str(marker), OWNED_SENTINEL="retained", FIXTURE_EXIT=str(child_exit), FIXTURE_PORT=str(self.server_port), KUBECONFIG="/dev/null", PYTHONDONTWRITEBYTECODE="1")
                environment.pop("XDG_CACHE_HOME", None)
                command = [str(binary / "python3"), "build/test-prerequisites.py", "--", "go", "test", "-race", "-count=1", "-v", "./..."] if direct else [str(binary / "make"), "test"]
                result = subprocess.run(command, cwd=fixture, env=environment, capture_output=True, timeout=30)
                self.assertEqual(child_exit if direct else (2 if child_exit else 0), result.returncode, result.stderr)
                record = json.loads(marker.read_text())
                self.assertEqual(["test", "-race", "-count=1", "-v", "./..."], record["argv"])
                self.assertEqual("-mod=readonly -p=1", record["flags"])
                self.assertEqual("/tmp/owned-data", record["data"])
                self.assertEqual("retained", record["sentinel"])
                self.assertTrue(record["private"])
                self.assertEqual(0o700, record["private_mode"])
                self.assertEqual(0o700, record["cache_mode"])
                self.assertEqual(os.getuid(), record["uid"])
                self.assertEqual(record["uid"], record["private_uid"])
                self.assertEqual(record["uid"], record["cache_uid"])
                caches.append(record["cache"])
                self.assertFalse(Path(record["cache"]).parent.exists())
                self.assertEqual([], list(temporary.iterdir()))
                after = temporary.stat()
                self.assertEqual((base_info.st_dev, base_info.st_ino, base_info.st_uid, mode), (after.st_dev, after.st_ino, after.st_uid, stat.S_IMODE(after.st_mode)))
        self.assertEqual(len(cases), len(set(caches)))
        self.assertFalse((unwritable / ".cache").exists())
        self.assertEqual(["/node", "/jq", "/shellcheck"] * len(cases), self.requests)

    def test_real_missing_tools_and_fixed_dispatch(self):
        empty = self.root / "empty-path"
        empty.mkdir()
        with mock.patch.object(TOOLS.subprocess, "run", side_effect=AssertionError("missing executable must not run")):
            for name in ("node", "jq", "shellcheck"):
                self.assertFalse(TOOLS.usable(name, str(empty)))
            with self.assertRaisesRegex(ValueError, "unsupported test executable"):
                TOOLS.usable("unexpected", str(empty))

    def test_explicit_cache_does_not_resolve_home(self):
        explicit = self.root / "explicit"
        with mock.patch.dict(os.environ, {"XDG_CACHE_HOME": str(explicit)}), mock.patch.object(TOOLS.Path, "home", side_effect=RuntimeError("no home")):
            with TOOLS.cache_directory() as cache:
                self.assertEqual(explicit / "openshift-mcp-test-tools", cache)
        self.assertEqual(0o700, stat.S_IMODE(explicit.stat().st_mode))

    def test_invalid_explicit_cache_has_no_outside_effects(self):
        outside = self.root / "outside"
        outside.mkdir(mode=0o700)
        alias = self.root / "alias"
        alias.symlink_to(outside, target_is_directory=True)
        denied = self.root / "denied"
        denied.mkdir(mode=0o500)
        self.addCleanup(denied.chmod, 0o700)
        shared = self.root / "shared"
        shared.mkdir(mode=0o700)
        shared.chmod(0o777)
        for value in ("", "relative", str(alias), str(alias / "created"), str(denied), str(shared / "created"), str(self.root) + "/../traversal"):
            with self.subTest(value=value), mock.patch.dict(os.environ, {"XDG_CACHE_HOME": value}), mock.patch.object(TOOLS.tempfile, "TemporaryDirectory", side_effect=AssertionError("explicit setting must not fall back")):
                with self.assertRaisesRegex(RuntimeError, "XDG_CACHE_HOME"):
                    with TOOLS.cache_directory():
                        self.fail("invalid explicit cache accepted")
        self.assertEqual([], list(outside.iterdir()))
        self.assertEqual([], list(shared.iterdir()))
        self.assertEqual(0o700, stat.S_IMODE(outside.stat().st_mode))

    def test_default_home_and_passwd_resolution(self):
        home = self.root / "home"
        home.mkdir(mode=0o700)
        environment = dict(os.environ, HOME=str(home))
        environment.pop("XDG_CACHE_HOME", None)
        with mock.patch.dict(os.environ, environment, clear=True):
            with TOOLS.cache_directory() as cache:
                self.assertEqual(home / ".cache/openshift-mcp-test-tools", cache)
        import pwd
        environment.pop("HOME")
        with mock.patch.dict(os.environ, environment, clear=True), mock.patch.object(pwd, "getpwuid", return_value=SimpleNamespace(pw_dir=str(home))):
            with TOOLS.cache_directory() as cache:
                self.assertEqual(home / ".cache/openshift-mcp-test-tools", cache)

    def test_missing_empty_and_unresolvable_home_fall_back(self):
        import pwd
        for home in (None, ""):
            environment = dict(os.environ, TMPDIR=str(self.root))
            environment.pop("XDG_CACHE_HOME", None)
            environment.pop("HOME", None)
            if home is not None:
                environment["HOME"] = home
            with self.subTest(home=home), mock.patch.dict(os.environ, environment, clear=True), mock.patch.object(pwd, "getpwuid", side_effect=KeyError("no account")):
                with TOOLS.cache_directory() as cache:
                    parent = cache.parent
                    self.assertEqual(self.root, parent.parent)
                    self.assertEqual(0o700, stat.S_IMODE(parent.stat().st_mode))
                    self.assertEqual(os.getuid(), parent.stat().st_uid)
                self.assertFalse(parent.exists())

    def test_fallback_is_unique_and_cleans_up_on_failure(self):
        environment = dict(os.environ, HOME="/", TMPDIR=str(self.root))
        environment.pop("XDG_CACHE_HOME", None)
        with mock.patch.dict(os.environ, environment, clear=True):
            with TOOLS.cache_directory() as first:
                with self.assertRaisesRegex(RuntimeError, "child failed"):
                    with TOOLS.cache_directory() as second:
                        self.assertNotEqual(first, second)
                        self.assertTrue(first.parent.exists())
                        raise RuntimeError("child failed")
                self.assertFalse(second.parent.exists())
                self.assertTrue(first.parent.exists())
            self.assertFalse(first.parent.exists())

    def test_system_temporary_default_is_private(self):
        environment = dict(os.environ, HOME="/")
        environment.pop("XDG_CACHE_HOME", None)
        environment.pop("TMPDIR", None)
        with mock.patch.dict(os.environ, environment, clear=True):
            with TOOLS.cache_directory() as cache:
                self.assertEqual(Path("/tmp"), cache.parent.parent)
                self.assertEqual(0o700, stat.S_IMODE(cache.parent.stat().st_mode))
                self.assertEqual(os.getuid(), cache.parent.stat().st_uid)
            self.assertFalse(cache.parent.exists())

    def test_unsafe_temporary_settings_rejected(self):
        shared = self.root / "shared"
        shared.mkdir(mode=0o700)
        shared.chmod(0o777)
        alias = self.root / "alias"
        alias.symlink_to(self.root, target_is_directory=True)
        denied = self.root / "denied"
        denied.mkdir(mode=0o500)
        self.addCleanup(denied.chmod, 0o700)
        for value in ("", "relative", str(shared), str(alias), str(denied)):
            environment = dict(os.environ, HOME="/", TMPDIR=value)
            environment.pop("XDG_CACHE_HOME", None)
            with self.subTest(value=value), mock.patch.dict(os.environ, environment, clear=True):
                with self.assertRaises((OSError, RuntimeError)):
                    with TOOLS.cache_directory():
                        self.fail("unsafe temporary base accepted")
        self.assertEqual([], list(shared.iterdir()))

    def test_parent_symlink_rejected_before_creation(self):
        outside = self.root / "outside"
        outside.mkdir(mode=0o755)
        alias = self.root / "alias"
        alias.symlink_to(outside, target_is_directory=True)
        with self.assertRaises((OSError, RuntimeError)):
            TOOLS.private_directory(alias / "must-not-create")
        self.assertEqual([], list(outside.iterdir()))
        self.assertEqual(0o755, stat.S_IMODE(outside.stat().st_mode))

    def test_new_directory_swap_cannot_chmod_victim(self):
        victim = self.root / "victim"
        victim.mkdir(mode=0o755)
        target = self.root / "new-cache"
        original = os.mkdir
        def swap(component, *arguments, **keywords):
            result = original(component, *arguments, **keywords)
            if component == target.name:
                target.rename(self.root / "moved-cache")
                target.symlink_to(victim, target_is_directory=True)
            return result
        with mock.patch.object(TOOLS.os, "mkdir", side_effect=swap):
            with self.assertRaises((OSError, RuntimeError)):
                TOOLS.private_directory(target)
        self.assertEqual(0o755, stat.S_IMODE(victim.stat().st_mode))

    def test_untrusted_ancestor_rejected_before_cached_execution(self):
        shared = self.root / "shared"
        shared.mkdir(mode=0o700)
        cache = shared / "cache"
        TOOLS.prepare(self.manifest, cache, "missing", ("Linux", "x86_64"))
        shared.chmod(0o777)
        with mock.patch.object(TOOLS, "verified_file", side_effect=AssertionError("must reject before reading or executing")):
            with self.assertRaisesRegex(RuntimeError, "untrusted"):
                TOOLS.prepare(self.manifest, cache, "missing", ("Linux", "x86_64"))
        self.assertEqual(3, len(self.requests))

    def test_cache_swap_after_verification_does_not_execute(self):
        search = self.prepare()
        cache = self.root / "cache"
        target = Path(search.split(os.pathsep)[0]) / "node"
        replacement = self.root / "replacement"
        replacement.mkdir(mode=0o700)
        tool = replacement / target.parent.name
        tool.mkdir(mode=0o700)
        marker = self.root / "must-not-execute"
        (tool / "node").write_text("#!/bin/sh\nprintf executed > '" + str(marker) + "'\n")
        (tool / "node").chmod(0o700)
        original = TOOLS.verified_file
        def swap(filename, digest):
            original(filename, digest)
            if Path(filename) == target:
                cache.rename(self.root / "verified-cache")
                cache.symlink_to(replacement, target_is_directory=True)
        with mock.patch.object(TOOLS, "verified_file", side_effect=swap):
            with self.assertRaises((OSError, RuntimeError)):
                self.prepare()
        self.assertFalse(marker.exists())
        self.assertEqual(3, len(self.requests))

    def test_cached_file_metadata_and_links_fail_closed(self):
        target = Path(self.prepare().split(os.pathsep)[0]) / "node"
        digest = self.manifest["node"]["binary_sha256"]
        target.chmod(0o755)
        with self.assertRaisesRegex(RuntimeError, "invalid cached"):
            self.prepare()
        target.chmod(0o700)
        linked = self.root / "hardlink"
        os.link(target, linked)
        with self.assertRaisesRegex(RuntimeError, "invalid cached"):
            self.prepare()
        linked.unlink()
        info = target.stat()
        wrong_owner = SimpleNamespace(st_mode=info.st_mode, st_uid=os.getuid() + 1, st_nlink=1, st_size=info.st_size)
        with mock.patch.object(TOOLS.os, "fstat", return_value=wrong_owner):
            with self.assertRaisesRegex(RuntimeError, "invalid cached"):
                TOOLS.verified_file(target, digest)
        target.rename(self.root / "original-node")
        target.symlink_to(self.root / "original-node")
        with self.assertRaises(OSError):
            self.prepare()
        self.assertEqual(3, len(self.requests))

    def test_cache_lock_metadata_and_timeout(self):
        import fcntl
        self.prepare()
        lock = self.root / "cache/lock"
        lock.chmod(0o644)
        with self.assertRaisesRegex(RuntimeError, "invalid test-tool cache lock"):
            self.prepare()
        lock.chmod(0o600)
        linked = self.root / "lock-link"
        os.link(lock, linked)
        with self.assertRaisesRegex(RuntimeError, "invalid test-tool cache lock"):
            self.prepare()
        linked.unlink()
        with mock.patch.object(fcntl, "flock", side_effect=BlockingIOError), mock.patch.object(TOOLS.time, "monotonic", side_effect=[0, 301]):
            with self.assertRaisesRegex(RuntimeError, "lock timeout"):
                self.prepare()
        lock.unlink()
        os.mkfifo(lock, 0o600)
        with self.assertRaisesRegex(RuntimeError, "invalid test-tool cache lock"):
            self.prepare()

    def test_concurrent_preparation_uses_verified_cache(self):
        with ThreadPoolExecutor(max_workers=2) as executor:
            results = list(executor.map(lambda unused: self.prepare(), range(2)))
        self.assertEqual(results[0], results[1])
        self.assertEqual(["/node", "/jq", "/shellcheck"], self.requests)

    def test_download_size_and_elapsed_limits(self):
        with mock.patch.object(TOOLS, "MAXIMUM", 1):
            with self.assertRaisesRegex(RuntimeError, "size/time limit"):
                TOOLS.download(self.manifest["node"])
        with mock.patch.object(TOOLS.time, "monotonic", side_effect=[0, 121]):
            with self.assertRaisesRegex(RuntimeError, "size/time limit"):
                TOOLS.download(self.manifest["node"])

    def test_installed_tools_do_not_select_cache_or_download(self):
        with mock.patch.object(TOOLS, "usable", return_value=True), mock.patch.object(TOOLS, "cache_directory", side_effect=AssertionError("cache is not needed")), mock.patch.object(TOOLS.subprocess, "call", return_value=17) as child, mock.patch.object(sys, "argv", ["test-prerequisites.py", "--", "bash", "-c", "exit 17"]):
            self.assertEqual(17, TOOLS.main())
        self.assertEqual(["bash", "-c", "exit 17"], child.call_args.args[0])
        self.assertEqual(dict(os.environ), child.call_args.kwargs["env"])

    def test_persistent_corruption_never_falls_back(self):
        cache = self.root / "openshift-mcp-test-tools"
        search = TOOLS.prepare(self.manifest, cache, "missing", ("Linux", "x86_64"))
        (Path(search.split(os.pathsep)[0]) / "node").write_bytes(b"#!/bin/sh\nexit 99\n")
        original = shutil.which
        def lookup(name, path=None):
            if name in ("bash", "make", "tar", "sha256sum", "unzip"):
                return "/baseline/" + name
            return original(name, path=path)
        with mock.patch.dict(os.environ, {"XDG_CACHE_HOME": str(self.root), "PATH": "missing"}), mock.patch.object(TOOLS.json, "load", return_value=self.manifest), mock.patch.object(TOOLS.shutil, "which", side_effect=lookup), mock.patch.object(TOOLS.tempfile, "TemporaryDirectory", side_effect=AssertionError("corruption must not fall back")), mock.patch.object(TOOLS.subprocess, "call", side_effect=AssertionError("corruption must not start Go")), mock.patch.object(sys, "argv", ["test-prerequisites.py", "--", "go", "test", "-race", "-count=1", "-v", "./..."]):
            with self.assertRaisesRegex(RuntimeError, "checksum mismatch"):
                TOOLS.main()
        self.assertEqual(3, len(self.requests))

    def test_main_fallback_preserves_raw_child_exit_and_environment(self):
        environment = dict(os.environ, HOME="/", TMPDIR=str(self.root), GOFLAGS="-mod=readonly -p=1", XDG_DATA_HOME="/tmp/owned-data")
        environment.pop("XDG_CACHE_HOME", None)
        selected = []
        def prepare(manifest, cache, search):
            selected.append(cache)
            self.assertTrue(cache.parent.is_dir())
            return "prepared-path"
        def child(arguments, env):
            self.assertEqual(["go", "test", "-race", "-count=1", "-v", "./..."], arguments)
            self.assertTrue(selected[0].parent.is_dir())
            self.assertEqual("prepared-path", env["PATH"])
            self.assertEqual(environment["GOFLAGS"], env["GOFLAGS"])
            self.assertEqual(environment["XDG_DATA_HOME"], env["XDG_DATA_HOME"])
            return 17
        with mock.patch.dict(os.environ, environment, clear=True), mock.patch.object(TOOLS, "usable", return_value=False), mock.patch.object(TOOLS, "prepare", side_effect=prepare), mock.patch.object(TOOLS.subprocess, "call", side_effect=child), mock.patch.object(sys, "argv", ["test-prerequisites.py", "--", "go", "test", "-race", "-count=1", "-v", "./..."]):
            self.assertEqual(17, TOOLS.main())
        self.assertFalse(selected[0].parent.exists())


if __name__ == "__main__":
    unittest.main()
