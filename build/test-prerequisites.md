# Default test prerequisites

`make test` prepares only missing or incompatible Node.js, jq and ShellCheck in
an owned 0700 user cache, then runs the unchanged complete Go test command.
Existing usable tools remain on PATH: Node >=18 with Object.hasOwn, jq >=1.6
supporting the tested all/type query, and ShellCheck >=0.9. Python >=3.9, Bash, Make, tar,
sha256sum and unzip must already exist. Python is not downloaded. The actual
Prow src image provides Python 3.9; its complete inventory still needs validation.
Capture/process-reaping coverage requires Linux child-subreaper support.

The missing-tool bootstrap supports Linux x86_64. Other platforms without the
required tools fail explicitly, not by skipping or tagging tests. No system
packages, root access, Go version changes, or cross-repository edits are used.
An explicit `XDG_CACHE_HOME` must be a safe physical owned writable path; invalid
explicit settings fail closed. Without it, the bootstrap tries the home cache
and falls back to a unique owned 0700 temporary child when home is unavailable
or unsafe (including `HOME=/`). Temporary children survive the command and are
cleaned afterward; they do not promise reuse across invocations. A symlink
`/tmp` is rejected unless a safe physical `TMPDIR` is provided. Installed
executables are rehashed before reuse, and corruption fails closed. It is not a
sandbox against malicious processes running under the same user. Initial
downloads need HTTPS network access. Requests have a 10-second socket timeout,
120-second streaming budget and 80MiB archive cap; selected executables are capped
at 256MiB, including the official Node binary. Lock acquisition is capped at 300s.

The committed manifest pins official release assets, archive SHA256 and selected
executable SHA256. No downloaded checksum, latest URL or install script is trusted
at runtime. Node 24.22.0 is the selected Node 24 LTS patch; its official BUILDING.md
requires Linux kernel >=4.18 and glibc >=2.28, compatible with RHEL9's glibc 2.34.
Runtime compatibility still must be exercised in the actual CI image.

Checksum provenance:

- Node: https://nodejs.org/dist/v24.22.0/SHASUMS256.txt
- jq: https://github.com/jqlang/jq/releases/download/jq-1.7.1/sha256sum.txt
- ShellCheck: official HTTPS v0.10.0 release asset linked in the manifest; its
  archive digest was computed from that asset, not an independently signed
  checksum publication. Selected binary digests were computed only after archive
  verification. Release-origin HTTPS and reviewed literal pins are the initial
  trust boundary; signatures were not independently verified.

Archives are verified before parsing and only one exact regular-file member is
read, without filesystem extraction. Files are installed atomically after both
checksums match. Partial preparation never runs tests with missing tools.

The caller's PATH, loader configuration, proxy/TLS settings and explicit command
are trusted inputs. Fixed version probes and generic CLI commands use argv, not
shell interpretation; this wrapper intentionally executes the caller's selected
program and does not restrict it to Go. Offline regressions test the actual Make
recipe using test-only platform overrides and fake release assets, with native
Linux x86_64 positives and unsupported-platform negatives retained. They are not
official-download evidence or proof of native macOS/ARM execution. Command-exec
scanner warnings require security-owner disposition; no suppression or blanket
false-positive/clearance claim follows from these boundary tests.
