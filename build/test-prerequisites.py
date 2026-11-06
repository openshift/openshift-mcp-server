import hashlib
from contextlib import contextmanager
import io
import json
import os
from pathlib import Path
import platform
import re
import shutil
import stat
import subprocess
import sys
import tarfile
import tempfile
import time
import urllib.request


MAXIMUM = 80 * 1024 * 1024
MAXIMUM_BINARY = 256 * 1024 * 1024


class HTTPSRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, request, response, code, message, headers, url):
        if not url.startswith("https://"):
            raise RuntimeError("test-tool redirect requires HTTPS")
        return super().redirect_request(request, response, code, message, headers, url)


def open_release(url, timeout):
    return urllib.request.build_opener(HTTPSRedirect()).open(url, timeout=timeout)


@contextmanager
def checked_directory(directory, create=False, private=False):
    directory = Path(directory)
    if not directory.is_absolute() or ".." in directory.parts:
        raise RuntimeError("test-tool directory must be an absolute physical path")
    descriptor = os.open(directory.anchor, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        for component in (*directory.parts[1:], None):
            info = os.fstat(descriptor)
            shared_temporary = info.st_uid == 0 and bool(info.st_mode & stat.S_ISVTX)
            if info.st_uid not in (0, os.getuid()) or (info.st_mode & 0o022 and not shared_temporary):
                raise RuntimeError("test-tool directory has untrusted ownership or writable ancestors")
            if component is None:
                break
            created = False
            try:
                child = os.open(component, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=descriptor)
            except FileNotFoundError:
                if not create:
                    raise
                if info.st_uid != os.getuid() or info.st_mode & 0o022:
                    raise RuntimeError("test-tool directories require an owned non-shared parent")
                try:
                    os.mkdir(component, mode=0o700, dir_fd=descriptor)
                    created = True
                except FileExistsError:
                    pass
                child = os.open(component, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=descriptor)
            os.close(descriptor)
            descriptor = child
            if created:
                if os.fstat(descriptor).st_uid != os.getuid():
                    raise RuntimeError("new test-tool directory is not owned")
                os.fchmod(descriptor, 0o700)
        if private and (info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700):
            raise RuntimeError("test-tool cache must be an owned physical 0700 directory")
        yield descriptor
    finally:
        os.close(descriptor)


def private_directory(directory):
    with checked_directory(directory, create=True, private=True):
        pass


def writable_cache_base(base):
    with checked_directory(base, create=True) as descriptor:
        info = os.fstat(descriptor)
        if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) & 0o300 != 0o300 or not os.access(base, os.W_OK | os.X_OK):
            raise RuntimeError("test-tool cache base must be owned and writable")


@contextmanager
def cache_directory():
    if "XDG_CACHE_HOME" in os.environ:
        explicit = os.environ["XDG_CACHE_HOME"]
        if not explicit:
            raise RuntimeError("XDG_CACHE_HOME must be a nonempty absolute physical writable directory")
        base = Path(explicit)
        try:
            writable_cache_base(base)
        except (OSError, RuntimeError) as error:
            raise RuntimeError("invalid XDG_CACHE_HOME: " + str(error)) from error
        yield base / "openshift-mcp-test-tools"
        return
    try:
        home = Path.home()
        if home == Path(home.anchor):
            raise RuntimeError("root is not a private cache home")
        with checked_directory(home) as descriptor:
            info = os.fstat(descriptor)
            if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) & 0o300 != 0o300 or not os.access(home, os.W_OK | os.X_OK):
                raise RuntimeError("test-tool home must be owned and writable")
        base = home / ".cache"
        writable_cache_base(base)
    except (KeyError, OSError, RuntimeError):
        base = None
    if base is not None:
        yield base / "openshift-mcp-test-tools"
        return
    temporary_base = os.environ.get("TMPDIR", "/tmp")
    if not temporary_base:
        raise RuntimeError("TMPDIR must be a nonempty absolute physical writable directory")
    with checked_directory(temporary_base) as descriptor:
        info = os.fstat(descriptor)
        if not os.access(temporary_base, os.W_OK | os.X_OK) or not (info.st_uid == os.getuid() or info.st_uid == 0 and info.st_mode & stat.S_ISVTX):
            raise RuntimeError("unsafe or unwritable test-tool temporary directory")
        with tempfile.TemporaryDirectory(prefix="openshift-mcp-test-", dir=temporary_base) as temporary:
            with checked_directory(temporary) as child:
                info = os.fstat(child)
                current = os.stat(temporary, follow_symlinks=False)
                if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) not in (0o700, 0o2700) or (info.st_dev, info.st_ino) != (current.st_dev, current.st_ino):
                    raise RuntimeError("new test-tool temporary directory is not private and owned")
                os.fchmod(child, 0o700)
            private_directory(temporary)
            yield Path(temporary) / "openshift-mcp-test-tools"


def verified_file(filename, digest):
    descriptor = os.open(filename, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, "rb") as source:
        info = os.fstat(source.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o700 or info.st_size > MAXIMUM_BINARY:
            raise RuntimeError("invalid cached test executable")
        if hashlib.sha256(source.read()).hexdigest() != digest:
            raise RuntimeError("cached test executable checksum mismatch")


def download(entry):
    if not entry["url"].startswith("https://"):
        raise RuntimeError("test-tool downloads require HTTPS")
    started = time.monotonic()
    chunks = []
    total = 0
    with open_release(entry["url"], timeout=10) as response:
        if not response.geturl().startswith("https://"):
            raise RuntimeError("test-tool redirect requires HTTPS")
        while True:
            chunk = response.read1(65536)
            if not chunk:
                break
            total += len(chunk)
            if total > MAXIMUM or time.monotonic() - started > 120:
                raise RuntimeError("test-tool download exceeded size/time limit")
            chunks.append(chunk)
    payload = b"".join(chunks)
    if hashlib.sha256(payload).hexdigest() != entry["sha256"]:
        raise RuntimeError("test-tool archive checksum mismatch")
    if entry["member"]:
        with tarfile.open(fileobj=io.BytesIO(payload), mode="r:xz") as archive:
            members = [member for member in archive if member.name == entry["member"]]
            if len(members) != 1 or not members[0].isfile() or members[0].size > MAXIMUM_BINARY:
                raise RuntimeError("invalid test-tool archive member")
            with archive.extractfile(members[0]) as source:
                payload = source.read(MAXIMUM_BINARY + 1)
    if len(payload) > MAXIMUM_BINARY or hashlib.sha256(payload).hexdigest() != entry["binary_sha256"]:
        raise RuntimeError("test-tool executable checksum mismatch")
    return payload


def prepare(manifest, cache, search_path, system=None):
    if (system or (platform.system(), platform.machine())) != ("Linux", "x86_64"):
        raise RuntimeError("pinned bootstrap supports Linux x86_64 only; other platforms require reviewed release pins")
    import fcntl
    cache = Path(cache)
    private_directory(cache)
    lock = os.open(cache / "lock", os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW | os.O_NONBLOCK, 0o600)
    paths = []
    with os.fdopen(lock, "w") as lockfile:
        info = os.fstat(lockfile.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid() or info.st_nlink != 1 or stat.S_IMODE(info.st_mode) != 0o600:
            raise RuntimeError("invalid test-tool cache lock")
        deadline = time.monotonic() + 300
        while True:
            try:
                fcntl.flock(lockfile, fcntl.LOCK_EX | fcntl.LOCK_NB)
                break
            except BlockingIOError:
                if time.monotonic() >= deadline:
                    raise RuntimeError("test-tool cache lock timeout")
                time.sleep(0.1)
        for name in ("node", "jq", "shellcheck"):
            entry = manifest[name]
            if not re.fullmatch(r"[a-f0-9]{64}", entry["binary_sha256"]):
                raise RuntimeError("invalid test-tool manifest")
            directory = cache / (name + "-" + entry["binary_sha256"])
            private_directory(directory)
            target = directory / name
            if not os.path.lexists(target):
                payload = download(entry)
                descriptor, temporary = tempfile.mkstemp(prefix=".download-", dir=directory)
                try:
                    with os.fdopen(descriptor, "wb") as output:
                        output.write(payload)
                        os.fchmod(output.fileno(), 0o700)
                    verified_file(temporary, entry["binary_sha256"])
                    private_directory(directory)
                    os.replace(temporary, target)
                finally:
                    private_directory(directory)
                    if os.path.lexists(temporary):
                        os.unlink(temporary)
            verified_file(target, entry["binary_sha256"])
            private_directory(directory)
            paths.append(str(directory))
    return os.pathsep.join([*paths, search_path])


def main():
    if sys.version_info < (3, 9):
        raise RuntimeError("make test requires Python >=3.9")
    if len(sys.argv) != 1:
        raise RuntimeError("test-prerequisites.py accepts no arguments; only the fixed Go test recipe is supported")
    search_path = os.environ.get("PATH", "")
    required = ("bash", "make", "tar", "sha256sum", "unzip")
    missing = [name for name in required if not shutil.which(name, path=search_path)]
    if missing:
        raise RuntimeError("missing baseline test prerequisites: " + ", ".join(missing))
    with Path(__file__).with_suffix(".json").open() as source:
        manifest = json.load(source)
    environment = os.environ.copy()
    with cache_directory() as cache:
        environment["PATH"] = prepare(manifest, cache, search_path)
        return subprocess.call(["go", "test", "-race", "-count=1", "-v", "./..."], env=environment)


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, RuntimeError, ValueError, tarfile.TarError) as error:
        print("test prerequisite preparation failed: " + str(error), file=sys.stderr)
        sys.exit(1)
