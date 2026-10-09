package build_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

const (
	fakeLinuxBinary   = "#!/bin/sh\necho 'mcpchecker linux v0.0.21'\n"
	fakeDarwinBinary  = "#!/bin/sh\necho 'mcpchecker darwin v0.0.21'\n"
	fakeWindowsBinary = "mcpchecker windows v0.0.21\r\n"
)

func installerPath(t *testing.T) string {
	t.Helper()
	if overridden := os.Getenv("MCPCHECKER_INSTALLER_UNDER_TEST"); overridden != "" {
		return overridden
	}
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate installer test source")
	}
	return filepath.Join(filepath.Dir(source), "install-mcpchecker.sh")
}

func createArchive(t *testing.T, directory, name, memberName, contents string) (string, string) {
	t.Helper()
	archivePath := filepath.Join(directory, name)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	zipWriter := zip.NewWriter(archiveFile)
	header := &zip.FileHeader{Name: memberName, Method: zip.Deflate}
	header.SetMode(0o755)
	entry, err := zipWriter.CreateHeader(header)
	if err != nil {
		t.Fatalf("create archive entry: %v", err)
	}
	if _, err := entry.Write([]byte(contents)); err != nil {
		t.Fatalf("write archive entry: %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	return archivePath, fmt.Sprintf("%x", sha256.Sum256(data))
}

func writeManifest(t *testing.T, path string, entries ...string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.Join(entries, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func createFakeCurl(t *testing.T, directory string) string {
	t.Helper()
	binDir := filepath.Join(directory, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin directory: %v", err)
	}
	curlPath := filepath.Join(binDir, "curl")
	script := `#!/bin/sh
set -eu
if [ "${FAKE_CURL_FAIL:-}" = "1" ]; then
    echo "fake curl must not be called" >&2
    exit 97
fi
output=
url=
proto=
tlsv12=false
while [ "$#" -gt 0 ]; do
    case "$1" in
        --proto)
            proto=$2
            shift 2
            ;;
        --tlsv1.2)
            tlsv12=true
            shift
            ;;
        -fsSL)
            shift
            ;;
        -o)
            output=$2
            shift 2
            ;;
        https://*)
            url=$1
            shift
            ;;
        *)
            echo "unexpected curl argument: $1" >&2
            exit 96
            ;;
    esac
done
[ "$proto" = "=https" ] || { echo "missing curl --proto '=https'" >&2; exit 95; }
[ "$tlsv12" = "true" ] || { echo "missing curl --tlsv1.2" >&2; exit 94; }
[ -n "$url" ] && [ -n "$output" ] || { echo "missing curl URL or output" >&2; exit 93; }
printf 'proto=%s tlsv1.2=%s url=%s\n' "$proto" "$tlsv12" "$url" >> "$FAKE_CURL_LOG"
if [ -n "${FAKE_CURL_READY:-}" ]; then
    active_file=
    if [ -n "${FAKE_FIXTURE_ACTIVE_DIR:-}" ]; then
        active_file="$FAKE_FIXTURE_ACTIVE_DIR/curl.$$"
        : > "$active_file"
        trap 'rm -f "$active_file"' EXIT
    fi
    : > "$FAKE_CURL_READY"
    while [ ! -f "$FAKE_CURL_RELEASE" ]; do
        if [ -n "${FAKE_FIXTURE_SHUTDOWN:-}" ] && [ -f "$FAKE_FIXTURE_SHUTDOWN" ]; then
            exit 98
        fi
        sleep 0.01
    done
fi
cp "$FAKE_ARCHIVE" "$output"
`
	if err := os.WriteFile(curlPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake curl: %v", err)
	}
	return binDir
}

func createFakeInstallerCommands(t *testing.T, binDir string) []string {
	t.Helper()
	realCommands := make(map[string]string, 4)
	for _, name := range []string{"awk", "mkdir", "mv", "sleep"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("locate real %s: %v", name, err)
		}
		realCommands[name] = path
	}

	scripts := map[string]string{
		"mkdir": `#!/bin/sh
set -eu
if [ -n "${FAKE_LOCK_PATH:-}" ] && [ "$#" -eq 1 ] && [ "$1" = "$FAKE_LOCK_PATH" ]; then
    printf 'attempt\n' >> "$FAKE_LOCK_ATTEMPTS"
    if "$FAKE_REAL_MKDIR" "$@"; then
        printf 'acquired\n' >> "$FAKE_LOCK_ACQUIRED"
        exit 0
    else
        status=$?
        printf 'blocked\n' >> "$FAKE_LOCK_BLOCKED"
        exit "$status"
    fi
fi
exec "$FAKE_REAL_MKDIR" "$@"
`,
		"sleep": `#!/bin/sh
set -eu
if [ "${1:-}" = "1" ] && [ -n "${FAKE_LOCK_BLOCKED:-}" ]; then
    attempt=$(wc -l < "$FAKE_LOCK_BLOCKED" | tr -d ' ')
    ready="${FAKE_SLEEP_PREFIX}.${attempt}.ready"
    release="${FAKE_SLEEP_PREFIX}.${attempt}.release"
    active_file=
    if [ -n "${FAKE_FIXTURE_ACTIVE_DIR:-}" ]; then
        active_file="$FAKE_FIXTURE_ACTIVE_DIR/sleep.$$"
        : > "$active_file"
        trap 'rm -f "$active_file"' EXIT
    fi
    : > "$ready"
    while [ ! -f "$release" ]; do
        if [ -n "${FAKE_FIXTURE_SHUTDOWN:-}" ] && [ -f "$FAKE_FIXTURE_SHUTDOWN" ]; then
            exit 98
        fi
        "$FAKE_REAL_SLEEP" 0.01
    done
    exit 0
fi
exec "$FAKE_REAL_SLEEP" "$@"
`,
		"awk": `#!/bin/sh
set -eu
if [ -n "${FAKE_CACHE_PATH:-}" ]; then
    for argument in "$@"; do
        if [ "$argument" = "$FAKE_CACHE_PATH" ]; then
            : > "$FAKE_CACHE_REACHED"
            break
        fi
    done
fi
exec "$FAKE_REAL_AWK" "$@"
`,
		"mv": `#!/bin/sh
set -eu
if [ "$#" -eq 3 ] && [ "$1" = "-f" ]; then
    source=$2
    destination=$3
    if [ -n "${FAKE_PUBLICATION_REACHED:-}" ] && { [ "$destination" = "$FAKE_MV_DESTINATION" ] || [ "$destination" = "${FAKE_MV_DESTINATION}.metadata" ]; }; then
        : > "$FAKE_PUBLICATION_REACHED"
    fi
    if [ -n "${FAKE_MV_FIRST_BEFORE:-}" ] && [ "$destination" = "$FAKE_MV_DESTINATION" ]; then
        active_file=
        if [ -n "${FAKE_FIXTURE_ACTIVE_DIR:-}" ]; then
            active_file="$FAKE_FIXTURE_ACTIVE_DIR/mv-first.$$"
            : > "$active_file"
            trap 'rm -f "$active_file"' EXIT
        fi
        : > "$FAKE_MV_FIRST_BEFORE"
        while [ ! -f "$FAKE_MV_FIRST_RELEASE" ]; do
            if [ -n "${FAKE_FIXTURE_SHUTDOWN:-}" ] && [ -f "$FAKE_FIXTURE_SHUTDOWN" ]; then
                exit 98
            fi
            "$FAKE_REAL_SLEEP" 0.01
        done
        "$FAKE_REAL_MV" "$@"
        : > "$FAKE_MV_FIRST_DONE"
        exit 0
    fi
    if [ -n "${FAKE_MV_SECOND_BEFORE:-}" ] && [ "$destination" = "${FAKE_MV_DESTINATION}.metadata" ]; then
        active_file=
        if [ -n "${FAKE_FIXTURE_ACTIVE_DIR:-}" ]; then
            active_file="$FAKE_FIXTURE_ACTIVE_DIR/mv-second.$$"
            : > "$active_file"
            trap 'rm -f "$active_file"' EXIT
        fi
        : > "$FAKE_MV_SECOND_BEFORE"
        while [ ! -f "$FAKE_MV_SECOND_RELEASE" ]; do
            if [ -n "${FAKE_FIXTURE_SHUTDOWN:-}" ] && [ -f "$FAKE_FIXTURE_SHUTDOWN" ]; then
                exit 98
            fi
            "$FAKE_REAL_SLEEP" 0.01
        done
        "$FAKE_REAL_MV" "$@"
        : > "$FAKE_MV_SECOND_DONE"
        while [ ! -f "$FAKE_MV_SECOND_RETURN_RELEASE" ]; do
            if [ -n "${FAKE_FIXTURE_SHUTDOWN:-}" ] && [ -f "$FAKE_FIXTURE_SHUTDOWN" ]; then
                exit 98
            fi
            "$FAKE_REAL_SLEEP" 0.01
        done
        exit 0
    fi
fi
exec "$FAKE_REAL_MV" "$@"
`,
	}
	for name, script := range scripts {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(script), 0o755); err != nil {
			t.Fatalf("write fake %s: %v", name, err)
		}
	}

	return []string{
		"FAKE_REAL_AWK=" + realCommands["awk"],
		"FAKE_REAL_MKDIR=" + realCommands["mkdir"],
		"FAKE_REAL_MV=" + realCommands["mv"],
		"FAKE_REAL_SLEEP=" + realCommands["sleep"],
	}
}

func fakeCurlEnvironment(fakeBin, archive, log, fail, ready, release string) []string {
	return []string{
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
		"FAKE_ARCHIVE=" + archive,
		"FAKE_CURL_LOG=" + log,
		"FAKE_CURL_FAIL=" + fail,
		"FAKE_CURL_READY=" + ready,
		"FAKE_CURL_RELEASE=" + release,
	}
}

func installerCommand(t *testing.T, environment []string, args ...string) *exec.Cmd {
	t.Helper()
	command := exec.Command("bash", append([]string{installerPath(t)}, args...)...)
	baseEnvironment := make([]string, 0, len(os.Environ())+len(environment))
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "FAKE_ARCHIVE=") ||
			strings.HasPrefix(value, "FAKE_CURL_") ||
			strings.HasPrefix(value, "FAKE_LOCK_") ||
			strings.HasPrefix(value, "FAKE_SLEEP_") ||
			strings.HasPrefix(value, "FAKE_CACHE_") ||
			strings.HasPrefix(value, "FAKE_MV_") ||
			strings.HasPrefix(value, "FAKE_PUBLICATION_") ||
			strings.HasPrefix(value, "FAKE_FIXTURE_") ||
			strings.HasPrefix(value, "FAKE_REAL_") ||
			strings.HasPrefix(value, "MCPCHECKER_LOCK_TIMEOUT_SECONDS=") {
			continue
		}
		baseEnvironment = append(baseEnvironment, value)
	}
	command.Env = append(baseEnvironment, environment...)
	return command
}

func runInstaller(t *testing.T, environment []string, args ...string) (string, error) {
	t.Helper()
	returnOutput, err := installerCommand(t, environment, args...).CombinedOutput()
	return string(returnOutput), err
}

type runningInstaller struct {
	command          *exec.Cmd
	output           bytes.Buffer
	done             chan struct{}
	waitErr          error
	fixtureShutdown  string
	fixtureActiveDir string
}

func startInstaller(t *testing.T, environment []string, args ...string) *runningInstaller {
	t.Helper()
	fixtureDir := t.TempDir()
	activeDir := filepath.Join(fixtureDir, "active")
	if err := os.Mkdir(activeDir, 0o755); err != nil {
		t.Fatalf("create active fixture-command directory: %v", err)
	}
	running := &runningInstaller{
		command:          installerCommand(t, environment, args...),
		done:             make(chan struct{}),
		fixtureShutdown:  filepath.Join(fixtureDir, "shutdown"),
		fixtureActiveDir: activeDir,
	}
	running.command.Env = append(running.command.Env,
		"FAKE_FIXTURE_SHUTDOWN="+running.fixtureShutdown,
		"FAKE_FIXTURE_ACTIVE_DIR="+running.fixtureActiveDir,
	)
	running.command.Stdout = &running.output
	running.command.Stderr = &running.output
	if err := running.command.Start(); err != nil {
		t.Fatalf("start installer: %v", err)
	}
	go func() {
		running.waitErr = running.command.Wait()
		close(running.done)
	}()
	t.Cleanup(func() {
		if err := running.shutdown(2 * time.Second); err != nil {
			t.Errorf("clean up installer fixture: %v", err)
		}
	})
	return running
}

func waitForChannel(done <-chan struct{}, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

func (running *runningInstaller) waitForFixtureCommands(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		entries, err := os.ReadDir(running.fixtureActiveDir)
		if err != nil {
			return fmt.Errorf("read active fixture-command directory: %w", err)
		}
		if len(entries) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("fixture commands did not exit: %v", entries)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (running *runningInstaller) shutdown(timeout time.Duration) error {
	select {
	case <-running.done:
		return running.waitForFixtureCommands(timeout)
	default:
	}

	if err := os.WriteFile(running.fixtureShutdown, nil, 0o644); err != nil {
		return fmt.Errorf("signal fixture shutdown: %w", err)
	}
	if !waitForChannel(running.done, timeout) {
		if err := running.command.Process.Kill(); err != nil {
			select {
			case <-running.done:
			default:
				return fmt.Errorf("kill installer after fixture shutdown timeout: %w", err)
			}
		}
		if !waitForChannel(running.done, timeout) {
			return fmt.Errorf("installer did not exit after fixture shutdown and kill")
		}
	}
	return running.waitForFixtureCommands(timeout)
}

func waitForInstallerResultWithin(running *runningInstaller, timeout time.Duration) (string, error, bool, error) {
	if waitForChannel(running.done, timeout) {
		return running.output.String(), running.waitErr, false, nil
	}
	if err := running.shutdown(2 * time.Second); err != nil {
		return "", nil, true, err
	}
	return running.output.String(), running.waitErr, true, nil
}

func waitForInstallerResult(t *testing.T, running *runningInstaller) (string, error) {
	t.Helper()
	output, err, timedOut, shutdownErr := waitForInstallerResultWithin(running, 10*time.Second)
	if timedOut {
		if shutdownErr != nil {
			t.Fatalf("installer timed out and fixture cleanup failed: %v", shutdownErr)
		}
		t.Fatalf("installer timed out\n%s", output)
	}
	return output, err
}

func waitForInstaller(t *testing.T, running *runningInstaller) string {
	t.Helper()
	output, err := waitForInstallerResult(t, running)
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	return output
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func waitForLineCount(t *testing.T, path string, expected int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		contents, err := os.ReadFile(path)
		if err == nil {
			if count := len(strings.Fields(string(contents))); count >= expected {
				return
			}
		} else if !os.IsNotExist(err) {
			t.Fatalf("read event log %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d events in %s", expected, path)
}

func assertFilesMissing(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unexpected event before lock release at %s: %v", path, err)
		}
	}
}

func signalFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatalf("write signal %s: %v", path, err)
	}
}

func expectedCurlLog(version, asset string) string {
	return fmt.Sprintf(
		"proto==https tlsv1.2=true url=https://github.com/mcpchecker/mcpchecker/releases/download/%s/%s\n",
		version,
		asset,
	)
}

func assertInstalledPair(t *testing.T, destination, contents, version, platform, archiveDigest string) {
	t.Helper()
	binary, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if string(binary) != contents {
		t.Fatalf("installed binary differs from the requested archive: %q", binary)
	}
	metadata, err := os.ReadFile(destination + ".metadata")
	if err != nil {
		t.Fatalf("read install metadata: %v", err)
	}
	binaryDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(contents)))
	expected := fmt.Sprintf(
		"cache_key=%s|%s|%s\nbinary_sha256=%s\n",
		version,
		platform,
		archiveDigest,
		binaryDigest,
	)
	if string(metadata) != expected {
		t.Fatalf("binary and metadata are not a coherent pair\nexpected: %q\nactual:   %q", expected, metadata)
	}
}

func writeInstalledPair(t *testing.T, destination, contents, version, platform, archiveDigest string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("create installed pair directory: %v", err)
	}
	if err := os.WriteFile(destination, []byte(contents), 0o755); err != nil {
		t.Fatalf("write installed binary: %v", err)
	}
	binaryDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(contents)))
	metadata := fmt.Sprintf(
		"cache_key=%s|%s|%s\nbinary_sha256=%s\n",
		version,
		platform,
		archiveDigest,
		binaryDigest,
	)
	if err := os.WriteFile(destination+".metadata", []byte(metadata), 0o644); err != nil {
		t.Fatalf("write installed metadata: %v", err)
	}
}

func assertNoInstallerArtifacts(t *testing.T, destination string) {
	t.Helper()
	patterns := []string{
		filepath.Join(filepath.Dir(destination), ".mcpchecker-install.*"),
		destination + ".tmp.*",
		destination + ".metadata.tmp.*",
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob installer artifacts: %v", err)
		}
		if len(matches) != 0 {
			t.Fatalf("installer artifacts were not cleaned up: %v", matches)
		}
	}
	if _, err := os.Stat(destination + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("installer lock was not cleaned up: %v", err)
	}
}

type InstallMCPCheckerSuite struct {
	suite.Suite
}

func (s *InstallMCPCheckerSuite) TestInstaller() {
	s.Run("installs, reuses, and repairs a checksum-verified binary", func() {
		t := s.T()
		tempDir := t.TempDir()
		asset := "mcpchecker-linux-amd64.zip"
		archive, digest := createArchive(t, tempDir, asset, "mcpchecker", fakeLinuxBinary)
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, digest+"  "+asset)
		curlLog := filepath.Join(tempDir, "curl.log")
		fakeBin := createFakeCurl(t, tempDir)
		destination := filepath.Join(tempDir, "tools", "mcpchecker")
		environment := fakeCurlEnvironment(fakeBin, archive, curlLog, "", "", "")

		output, err := runInstaller(t, environment, "v0.0.21", "linux", "amd64", destination, manifest)
		if err != nil {
			t.Fatalf("first install failed: %v\n%s", err, output)
		}
		assertInstalledPair(t, destination, fakeLinuxBinary, "v0.0.21", "linux/amd64", digest)
		curlCalls, err := os.ReadFile(curlLog)
		if err != nil {
			t.Fatalf("read curl log: %v", err)
		}
		expectedCall := expectedCurlLog("v0.0.21", asset)
		if string(curlCalls) != expectedCall {
			t.Fatalf("installer curl arguments differ from the required hardened request: %q", curlCalls)
		}

		cachedEnvironment := fakeCurlEnvironment(fakeBin, archive, curlLog, "1", "", "")
		output, err = runInstaller(t, cachedEnvironment, "v0.0.21", "linux", "amd64", destination, manifest)
		if err != nil {
			t.Fatalf("matching cache was not reused: %v\n%s", err, output)
		}
		if !strings.Contains(output, "matches its cached metadata and binary digest") {
			t.Fatalf("cache output did not describe the bounded integrity check: %s", output)
		}
		curlCalls, err = os.ReadFile(curlLog)
		if err != nil {
			t.Fatalf("read cache curl log: %v", err)
		}
		if string(curlCalls) != expectedCall {
			t.Fatalf("matching cache unexpectedly downloaded an archive")
		}

		metadataPath := destination + ".metadata"
		metadata, err := os.ReadFile(metadataPath)
		if err != nil {
			t.Fatalf("read install metadata: %v", err)
		}
		wrongVersionMetadata := strings.Replace(string(metadata), "v0.0.21", "v0.0.20", 1)
		if err := os.WriteFile(metadataPath, []byte(wrongVersionMetadata), 0o644); err != nil {
			t.Fatalf("write wrong-version metadata: %v", err)
		}
		output, err = runInstaller(t, environment, "v0.0.21", "linux", "amd64", destination, manifest)
		if err != nil {
			t.Fatalf("wrong-version cache was not repaired: %v\n%s", err, output)
		}
		if !strings.Contains(output, "metadata is stale or its binary changed") {
			t.Fatalf("wrong-version cache was silently reused: %s", output)
		}

		if err := os.WriteFile(destination, []byte("tampered"), 0o755); err != nil {
			t.Fatalf("tamper installed binary: %v", err)
		}
		output, err = runInstaller(t, environment, "v0.0.21", "linux", "amd64", destination, manifest)
		if err != nil {
			t.Fatalf("binary-only cache modification was not repaired: %v\n%s", err, output)
		}
		assertInstalledPair(t, destination, fakeLinuxBinary, "v0.0.21", "linux/amd64", digest)
		assertNoInstallerArtifacts(t, destination)
	})

	for _, testCase := range []struct {
		name         string
		metadataPath bool
		symlinkToDir bool
	}{
		{name: "rejects a binary destination directory"},
		{name: "rejects a metadata destination directory", metadataPath: true},
		{name: "rejects a binary destination symlink to a directory", symlinkToDir: true},
		{name: "rejects a metadata destination symlink to a directory", metadataPath: true, symlinkToDir: true},
	} {
		testCase := testCase
		s.Run(testCase.name, func() {
			t := s.T()
			tempDir := t.TempDir()
			asset := "mcpchecker-linux-amd64.zip"
			archive, digest := createArchive(t, tempDir, asset, "mcpchecker", fakeLinuxBinary)
			manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
			writeManifest(t, manifest, digest+"  "+asset)
			fakeBin := createFakeCurl(t, tempDir)
			destination := filepath.Join(tempDir, "tools", "mcpchecker")
			if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
				t.Fatalf("create destination parent: %v", err)
			}

			blockedPath := destination
			if testCase.metadataPath {
				blockedPath += ".metadata"
			}
			blockedDirectory := blockedPath
			if testCase.symlinkToDir {
				blockedDirectory = filepath.Join(tempDir, "blocked-directory")
			}
			if err := os.Mkdir(blockedDirectory, 0o755); err != nil {
				t.Fatalf("create blocked final directory: %v", err)
			}
			sentinel := filepath.Join(blockedDirectory, "sentinel")
			if err := os.WriteFile(sentinel, []byte("preserve"), 0o644); err != nil {
				t.Fatalf("write blocked-directory sentinel: %v", err)
			}

			var output string
			var err error
			if testCase.symlinkToDir {
				downloadReady := filepath.Join(tempDir, "download.ready")
				downloadRelease := filepath.Join(tempDir, "download.release")
				running := startInstaller(t,
					fakeCurlEnvironment(fakeBin, archive, filepath.Join(tempDir, "curl.log"), "", downloadReady, downloadRelease),
					"v0.0.21", "linux", "amd64", destination, manifest,
				)
				waitForFile(t, downloadReady)
				if linkErr := os.Symlink(blockedDirectory, blockedPath); linkErr != nil {
					t.Fatalf("create final-path symlink during download: %v", linkErr)
				}
				signalFile(t, downloadRelease)
				output, err = waitForInstallerResult(t, running)
			} else {
				output, err = runInstaller(t,
					fakeCurlEnvironment(fakeBin, archive, filepath.Join(tempDir, "curl.log"), "1", "", ""),
					"v0.0.21", "linux", "amd64", destination, manifest,
				)
			}
			if err == nil {
				t.Fatalf("directory-valued final path unexpectedly succeeded: %s", output)
			}
			if !strings.Contains(output, "destination must not be a directory") {
				t.Fatalf("unexpected directory-path failure output: %s", output)
			}
			if strings.Contains(output, "Installed checksum-verified") {
				t.Fatalf("directory-valued final path reported false success: %s", output)
			}
			if !testCase.symlinkToDir && strings.Contains(output, "Downloading mcpchecker") {
				t.Fatalf("pre-existing directory-valued final path was not rejected before download: %s", output)
			}
			entries, readErr := os.ReadDir(blockedDirectory)
			if readErr != nil {
				t.Fatalf("read blocked final directory: %v", readErr)
			}
			if len(entries) != 1 || entries[0].Name() != "sentinel" {
				t.Fatalf("installer moved a random publication file into the final directory: %v", entries)
			}
			assertNoInstallerArtifacts(t, destination)
		})
	}

	s.Run("serializes concurrent installs and publishes coherent pairs", func() {
		t := s.T()
		tempDir := t.TempDir()
		linuxAsset := "mcpchecker-linux-amd64.zip"
		darwinAsset := "mcpchecker-darwin-amd64.zip"
		linuxArchive, linuxDigest := createArchive(t, tempDir, linuxAsset, "mcpchecker", fakeLinuxBinary)
		darwinArchive, darwinDigest := createArchive(t, tempDir, darwinAsset, "mcpchecker", fakeDarwinBinary)
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, linuxDigest+"  "+linuxAsset, darwinDigest+"  "+darwinAsset)
		fakeBin := createFakeCurl(t, tempDir)
		commandEnvironment := createFakeInstallerCommands(t, fakeBin)
		destination := filepath.Join(tempDir, "tools", "mcpchecker")
		writeInstalledPair(t, destination, fakeDarwinBinary, "v0.0.21", "darwin/amd64", darwinDigest)

		linuxReady := filepath.Join(tempDir, "linux.ready")
		linuxRelease := filepath.Join(tempDir, "linux.release")
		firstBefore := filepath.Join(tempDir, "linux.first.before")
		firstRelease := filepath.Join(tempDir, "linux.first.release")
		firstDone := filepath.Join(tempDir, "linux.first.done")
		secondBefore := filepath.Join(tempDir, "linux.second.before")
		secondRelease := filepath.Join(tempDir, "linux.second.release")
		secondDone := filepath.Join(tempDir, "linux.second.done")
		secondReturnRelease := filepath.Join(tempDir, "linux.second.return.release")

		lockAttempts := filepath.Join(tempDir, "darwin.lock.attempts")
		lockBlocked := filepath.Join(tempDir, "darwin.lock.blocked")
		lockAcquired := filepath.Join(tempDir, "darwin.lock.acquired")
		sleepPrefix := filepath.Join(tempDir, "darwin.sleep")
		cacheReached := filepath.Join(tempDir, "darwin.cache.reached")
		darwinReady := filepath.Join(tempDir, "darwin.download.reached")
		darwinRelease := filepath.Join(tempDir, "darwin.download.release")
		publicationReached := filepath.Join(tempDir, "darwin.publication.reached")

		linuxEnvironment := append(
			fakeCurlEnvironment(fakeBin, linuxArchive, filepath.Join(tempDir, "linux.curl.log"), "", linuxReady, linuxRelease),
			commandEnvironment...,
		)
		linuxEnvironment = append(linuxEnvironment,
			"FAKE_MV_DESTINATION="+destination,
			"FAKE_MV_FIRST_BEFORE="+firstBefore,
			"FAKE_MV_FIRST_RELEASE="+firstRelease,
			"FAKE_MV_FIRST_DONE="+firstDone,
			"FAKE_MV_SECOND_BEFORE="+secondBefore,
			"FAKE_MV_SECOND_RELEASE="+secondRelease,
			"FAKE_MV_SECOND_DONE="+secondDone,
			"FAKE_MV_SECOND_RETURN_RELEASE="+secondReturnRelease,
		)
		linux := startInstaller(t,
			linuxEnvironment,
			"v0.0.21", "linux", "amd64", destination, manifest,
		)
		waitForFile(t, linuxReady)
		waitForFile(t, destination+".lock")
		signalFile(t, linuxRelease)
		waitForFile(t, firstBefore)

		darwinEnvironment := append(
			fakeCurlEnvironment(fakeBin, darwinArchive, filepath.Join(tempDir, "darwin.curl.log"), "", darwinReady, darwinRelease),
			commandEnvironment...,
		)
		darwinEnvironment = append(darwinEnvironment,
			"FAKE_LOCK_PATH="+destination+".lock",
			"FAKE_LOCK_ATTEMPTS="+lockAttempts,
			"FAKE_LOCK_BLOCKED="+lockBlocked,
			"FAKE_LOCK_ACQUIRED="+lockAcquired,
			"FAKE_SLEEP_PREFIX="+sleepPrefix,
			"FAKE_CACHE_PATH="+destination+".metadata",
			"FAKE_CACHE_REACHED="+cacheReached,
			"FAKE_MV_DESTINATION="+destination,
			"FAKE_PUBLICATION_REACHED="+publicationReached,
		)
		darwin := startInstaller(t,
			darwinEnvironment,
			"v0.0.21", "darwin", "amd64", destination, manifest,
		)
		waitForLineCount(t, lockAttempts, 1)
		waitForLineCount(t, lockBlocked, 1)
		waitForFile(t, sleepPrefix+".1.ready")
		assertFilesMissing(t, cacheReached, darwinReady, publicationReached)

		signalFile(t, firstRelease)
		waitForFile(t, firstDone)
		waitForFile(t, secondBefore)
		signalFile(t, sleepPrefix+".1.release")
		waitForLineCount(t, lockBlocked, 2)
		waitForFile(t, sleepPrefix+".2.ready")
		assertFilesMissing(t, cacheReached, darwinReady, publicationReached)

		signalFile(t, secondRelease)
		waitForFile(t, secondDone)
		signalFile(t, sleepPrefix+".2.release")
		waitForLineCount(t, lockBlocked, 3)
		waitForFile(t, sleepPrefix+".3.ready")
		assertFilesMissing(t, cacheReached, darwinReady, publicationReached)

		signalFile(t, secondReturnRelease)
		linuxOutput := waitForInstaller(t, linux)
		if !strings.Contains(linuxOutput, "Installed checksum-verified mcpchecker") {
			t.Fatalf("first installer did not report success: %s", linuxOutput)
		}
		assertInstalledPair(t, destination, fakeLinuxBinary, "v0.0.21", "linux/amd64", linuxDigest)

		signalFile(t, sleepPrefix+".3.release")
		waitForLineCount(t, lockAcquired, 1)
		waitForFile(t, cacheReached)
		waitForFile(t, darwinReady)
		assertFilesMissing(t, publicationReached)
		signalFile(t, darwinRelease)
		darwinOutput := waitForInstaller(t, darwin)
		if !strings.Contains(darwinOutput, "Installed checksum-verified mcpchecker") {
			t.Fatalf("second installer did not report success: %s", darwinOutput)
		}
		assertInstalledPair(t, destination, fakeDarwinBinary, "v0.0.21", "darwin/amd64", darwinDigest)
		assertNoInstallerArtifacts(t, destination)
	})

	s.Run("installs the supported Windows executable member", func() {
		t := s.T()
		tempDir := t.TempDir()
		asset := "mcpchecker-windows-arm64.zip"
		archive, digest := createArchive(t, tempDir, asset, "mcpchecker.exe", fakeWindowsBinary)
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, digest+"  "+asset)
		curlLog := filepath.Join(tempDir, "curl.log")
		fakeBin := createFakeCurl(t, tempDir)
		destination := filepath.Join(tempDir, "tools", "mcpchecker.exe")
		output, err := runInstaller(
			t,
			fakeCurlEnvironment(fakeBin, archive, curlLog, "", "", ""),
			"v0.0.21", "windows", "arm64", destination, manifest,
		)
		if err != nil {
			t.Fatalf("Windows install failed: %v\n%s", err, output)
		}
		assertInstalledPair(t, destination, fakeWindowsBinary, "v0.0.21", "windows/arm64", digest)
		curlCall, err := os.ReadFile(curlLog)
		if err != nil {
			t.Fatalf("read Windows curl log: %v", err)
		}
		if string(curlCall) != expectedCurlLog("v0.0.21", asset) {
			t.Fatalf("Windows installer requested an unexpected asset: %q", curlCall)
		}
		assertNoInstallerArtifacts(t, destination)
	})

	s.Run("rejects a checksum mismatch before extraction and cleans the lock", func() {
		t := s.T()
		tempDir := t.TempDir()
		archive := filepath.Join(tempDir, "untrusted.zip")
		if err := os.WriteFile(archive, []byte("not a zip archive"), 0o644); err != nil {
			t.Fatalf("write untrusted archive: %v", err)
		}
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, strings.Repeat("0", 64)+"  mcpchecker-linux-amd64.zip")
		fakeBin := createFakeCurl(t, tempDir)
		destination := filepath.Join(tempDir, "tools", "mcpchecker")
		output, err := runInstaller(t,
			fakeCurlEnvironment(fakeBin, archive, filepath.Join(tempDir, "curl.log"), "", "", ""),
			"v0.0.21", "linux", "amd64", destination, manifest,
		)
		if err == nil {
			t.Fatalf("checksum mismatch unexpectedly succeeded")
		}
		if !strings.Contains(output, "checksum mismatch") {
			t.Fatalf("unexpected checksum failure output: %s", output)
		}
		if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
			t.Fatalf("unverified binary was installed")
		}
		assertNoInstallerArtifacts(t, destination)
	})

	for _, testCase := range []struct {
		name     string
		manifest string
		expected string
	}{
		{
			name: "rejects duplicate manifest entries",
			manifest: strings.Repeat("1", 64) + "  mcpchecker-linux-amd64.zip\n" +
				strings.Repeat("2", 64) + "  mcpchecker-linux-amd64.zip\n",
			expected: "expected exactly one committed checksum",
		},
		{
			name:     "rejects a malformed manifest digest",
			manifest: "not-a-digest  mcpchecker-linux-amd64.zip\n",
			expected: "invalid SHA-256",
		},
	} {
		testCase := testCase
		s.Run(testCase.name, func() {
			t := s.T()
			tempDir := t.TempDir()
			manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
			if err := os.WriteFile(manifest, []byte(testCase.manifest), 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}
			fakeBin := createFakeCurl(t, tempDir)
			output, err := runInstaller(t,
				fakeCurlEnvironment(fakeBin, filepath.Join(tempDir, "missing.zip"), filepath.Join(tempDir, "curl.log"), "1", "", ""),
				"v0.0.21", "linux", "amd64", filepath.Join(tempDir, "mcpchecker"), manifest,
			)
			if err == nil {
				t.Fatalf("invalid manifest unexpectedly succeeded")
			}
			if !strings.Contains(output, testCase.expected) {
				t.Fatalf("unexpected invalid-manifest output: %s", output)
			}
		})
	}

	s.Run("rejects a checksum-valid archive missing the expected member", func() {
		t := s.T()
		tempDir := t.TempDir()
		asset := "mcpchecker-linux-amd64.zip"
		archive, digest := createArchive(t, tempDir, asset, "unexpected-name", fakeLinuxBinary)
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, digest+"  "+asset)
		fakeBin := createFakeCurl(t, tempDir)
		destination := filepath.Join(tempDir, "tools", "mcpchecker")
		output, err := runInstaller(t,
			fakeCurlEnvironment(fakeBin, archive, filepath.Join(tempDir, "curl.log"), "", "", ""),
			"v0.0.21", "linux", "amd64", destination, manifest,
		)
		if err == nil {
			t.Fatalf("archive missing mcpchecker unexpectedly succeeded")
		}
		if !strings.Contains(output, "does not contain mcpchecker") {
			t.Fatalf("unexpected missing-member output: %s", output)
		}
		assertNoInstallerArtifacts(t, destination)
	})

	s.Run("rejects an unverified version before download", func() {
		t := s.T()
		tempDir := t.TempDir()
		fakeBin := createFakeCurl(t, tempDir)
		output, err := runInstaller(t,
			fakeCurlEnvironment(fakeBin, filepath.Join(tempDir, "missing.zip"), filepath.Join(tempDir, "curl.log"), "1", "", ""),
			"v0.0.20", "linux", "amd64", filepath.Join(tempDir, "mcpchecker"), filepath.Join(tempDir, "mcpchecker-v0.0.20.sha256"),
		)
		if err == nil {
			t.Fatalf("unverified version unexpectedly succeeded")
		}
		if !strings.Contains(output, "no committed checksum manifest") {
			t.Fatalf("unexpected unverified-version output: %s", output)
		}
	})

	s.Run("rejects an unsupported platform before download", func() {
		t := s.T()
		tempDir := t.TempDir()
		fakeBin := createFakeCurl(t, tempDir)
		output, err := runInstaller(t,
			fakeCurlEnvironment(fakeBin, filepath.Join(tempDir, "missing.zip"), filepath.Join(tempDir, "curl.log"), "1", "", ""),
			"v0.0.21", "linux", "s390x", filepath.Join(tempDir, "mcpchecker"), filepath.Join(tempDir, "checksums"),
		)
		if err == nil {
			t.Fatalf("unsupported platform unexpectedly succeeded")
		}
		if !strings.Contains(output, "no verified release binary") {
			t.Fatalf("unexpected unsupported-platform output: %s", output)
		}
	})

	s.Run("bounds timeout cleanup and reaps blocked fixture commands", func() {
		t := s.T()
		tempDir := t.TempDir()
		asset := "mcpchecker-linux-amd64.zip"
		archive, digest := createArchive(t, tempDir, asset, "mcpchecker", fakeLinuxBinary)
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, digest+"  "+asset)
		fakeBin := createFakeCurl(t, tempDir)
		destination := filepath.Join(tempDir, "tools", "mcpchecker")
		downloadReady := filepath.Join(tempDir, "download.ready")
		running := startInstaller(t,
			fakeCurlEnvironment(
				fakeBin,
				archive,
				filepath.Join(tempDir, "curl.log"),
				"",
				downloadReady,
				filepath.Join(tempDir, "never-released"),
			),
			"v0.0.21", "linux", "amd64", destination, manifest,
		)
		waitForFile(t, downloadReady)

		output, err, timedOut, shutdownErr := waitForInstallerResultWithin(running, 50*time.Millisecond)
		if !timedOut {
			t.Fatalf("blocked installer unexpectedly completed: %v\n%s", err, output)
		}
		if shutdownErr != nil {
			t.Fatalf("timeout cleanup failed: %v", shutdownErr)
		}
		if err == nil {
			t.Fatalf("fixture shutdown unexpectedly reported installer success: %s", output)
		}
		active, readErr := os.ReadDir(running.fixtureActiveDir)
		if readErr != nil {
			t.Fatalf("read active fixture-command directory after timeout cleanup: %v", readErr)
		}
		if len(active) != 0 {
			t.Fatalf("timeout cleanup left fixture commands active: %v", active)
		}
		assertNoInstallerArtifacts(t, destination)
	})

	s.Run("times out without removing a lock it does not own", func() {
		t := s.T()
		tempDir := t.TempDir()
		asset := "mcpchecker-linux-amd64.zip"
		archive, digest := createArchive(t, tempDir, asset, "mcpchecker", fakeLinuxBinary)
		manifest := filepath.Join(tempDir, "mcpchecker-v0.0.21.sha256")
		writeManifest(t, manifest, digest+"  "+asset)
		fakeBin := createFakeCurl(t, tempDir)
		destination := filepath.Join(tempDir, "tools", "mcpchecker")
		lockDir := destination + ".lock"
		if err := os.MkdirAll(lockDir, 0o755); err != nil {
			t.Fatalf("create simulated stranded lock: %v", err)
		}
		sentinel := filepath.Join(lockDir, "owner")
		if err := os.WriteFile(sentinel, []byte("another installer"), 0o644); err != nil {
			t.Fatalf("write simulated lock owner: %v", err)
		}
		environment := append(
			fakeCurlEnvironment(fakeBin, archive, filepath.Join(tempDir, "curl.log"), "1", "", ""),
			"MCPCHECKER_LOCK_TIMEOUT_SECONDS=0",
		)
		start := time.Now()
		output, err := runInstaller(t, environment, "v0.0.21", "linux", "amd64", destination, manifest)
		if err == nil {
			t.Fatalf("installer unexpectedly ignored an existing lock")
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("zero-timeout lock failure was not bounded: %s", elapsed)
		}
		if !strings.Contains(output, "timed out after 0s") || !strings.Contains(output, "uncatchable termination") {
			t.Fatalf("lock failure did not explain bounded stranded-lock handling: %s", output)
		}
		owner, readErr := os.ReadFile(sentinel)
		if readErr != nil || string(owner) != "another installer" {
			t.Fatalf("installer removed or changed a lock it did not own: %q, %v", owner, readErr)
		}
	})
}

func TestInstallMCPChecker(t *testing.T) {
	suite.Run(t, new(InstallMCPCheckerSuite))
}
