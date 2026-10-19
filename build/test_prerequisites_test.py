import copy
import hashlib
from http.server import BaseHTTPRequestHandler, HTTPServer
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import threading
import time
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
                self.send_response(200)
                self.end_headers()
                self.wfile.write(payloads[self.path])

            def log_message(self, *arguments):
                pass

        server = HTTPServer(("127.0.0.1", 0), Handler)
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
                with mock.patch.object(TOOLS.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, version.encode(), b"")):
                    self.assertEqual(expected, TOOLS.usable(name, os.environ["PATH"]))
        with self.assertRaisesRegex(RuntimeError, "requires HTTPS"):
            TOOLS.HTTPSRedirect().redirect_request(None, None, 302, "", {}, "http://outside.invalid")

    def test_cache_symlink_and_insecure_mode_fail(self):
        outside = self.root / "outside"
        outside.mkdir(mode=0o700)
        (self.root / "cache").symlink_to(outside)
        with self.assertRaises(RuntimeError):
            self.prepare()
        (self.root / "cache").unlink()
        (self.root / "cache").mkdir(mode=0o755)
        with self.assertRaises(RuntimeError):
            self.prepare()

    def test_default_make_consumer_preserves_command_and_environment(self):
        repository = Path(__file__).resolve().parent.parent
        binary = self.root / "bin"
        binary.mkdir()
        marker = self.root / "invocation"
        (binary / "go").write_text("#!/bin/sh\nprintf '%s\\n' \"$GOFLAGS\" \"$XDG_DATA_HOME\" \"$@\" > \"$OWNED_MARKER\"\nexit 17\n")
        (binary / "go").chmod(0o700)
        environment = dict(os.environ, PATH=str(binary) + os.pathsep + os.environ["PATH"], GOFLAGS="-mod=readonly -p=1", XDG_DATA_HOME="/tmp/owned-data", OWNED_MARKER=str(marker), KUBECONFIG="/dev/null")
        result = subprocess.run(["make", "test"], cwd=repository, env=environment, capture_output=True, timeout=30)
        self.assertEqual(2, result.returncode, result.stderr)
        self.assertEqual(["-mod=readonly -p=1", "/tmp/owned-data", "test", "-race", "-count=1", "-v", "./..."], marker.read_text().splitlines())


if __name__ == "__main__":
    unittest.main()
