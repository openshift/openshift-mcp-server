package build_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MCPCheckerDiagnosticsSuite struct{ suite.Suite }

func TestMCPCheckerDiagnostics(testingT *testing.T) {
	suite.Run(testingT, new(MCPCheckerDiagnosticsSuite))
}

func (suiteState *MCPCheckerDiagnosticsSuite) TestChildFileSizeLimit() {
	node, err := exec.LookPath("node")
	suiteState.Require().NoError(err)
	for _, mode := range []string{"summary", "diagnostic"} {
		for _, limit := range []int{0, 4096, 16384} {
			suiteState.Run(mode+"/"+strconv.Itoa(limit), func() {
				private := filepath.Join(suiteState.T().TempDir(), "private")
				suiteState.Require().NoError(os.Mkdir(private, 0o700))
				arguments := []string{"-c", `import os, resource, signal, sys
signal.signal(signal.SIGXFSZ, signal.SIG_IGN)
resource.setrlimit(resource.RLIMIT_FSIZE, (int(sys.argv[1]), int(sys.argv[1])))
os.execv(sys.argv[2], sys.argv[2:])`, strconv.Itoa(limit), node, workflowBuildPath(suiteState.T(), "mcpchecker-diagnostics.cjs"), private, "summary.txt"}
				filename := "summary.txt"
				if mode == "summary" {
					arguments = append(arguments, "summary")
					filename = "raw-summary.json"
				}
				input := strings.Repeat("x", 8192)
				command := exec.Command("python3", arguments...)
				command.Env = append(os.Environ(), "KUBECONFIG=/dev/null")
				command.Stdin = strings.NewReader(input)
				output, runErr := command.CombinedOutput()
				data, readErr := os.ReadFile(filepath.Join(private, filename))
				suiteState.Require().NoError(readErr)
				if limit < len(input) {
					suiteState.Error(runErr, string(output))
					suiteState.Contains(string(output), "EFBIG")
					suiteState.Len(data, limit)
				} else {
					suiteState.NoError(runErr, string(output))
					suiteState.Equal(input, string(data))
				}
			})
		}
	}
}

func (suiteState *MCPCheckerDiagnosticsSuite) TestPrivateOutputContract() {
	for _, scenario := range []string{"normal", "summary", "existing", "leaf-link", "parent-link", "relative-root", "traversal", "absolute-leaf", "outside-leaf", "directory", "public-root", "hardlink", "public-file", "nonempty-file", "exclusive-summary"} {
		suiteState.Run(scenario, func() {
			root := suiteState.T().TempDir()
			private := filepath.Join(root, "private")
			suiteState.Require().NoError(os.Mkdir(private, 0o700))
			sentinel := filepath.Join(root, "sentinel")
			suiteState.Require().NoError(os.WriteFile(sentinel, []byte("unchanged"), 0o600))
			leaf := "check.txt"
			arguments := []string{workflowBuildPath(suiteState.T(), "mcpchecker-diagnostics.cjs"), private, leaf}
			switch scenario {
			case "normal":
			case "summary":
				arguments[2] = "summary.txt"
				arguments = append(arguments, "summary")
			case "existing":
				suiteState.Require().NoError(os.WriteFile(filepath.Join(private, leaf), []byte{}, 0o600))
			case "leaf-link":
				suiteState.Require().NoError(os.Symlink(sentinel, filepath.Join(private, leaf)))
			case "parent-link":
				arguments[1] = filepath.Join(root, "alias")
				suiteState.Require().NoError(os.Symlink(private, arguments[1]))
			case "relative-root":
				arguments[1] = "private"
			case "traversal":
				arguments[2] = "../sentinel"
			case "absolute-leaf":
				arguments[2] = sentinel
			case "outside-leaf":
				arguments[2] = "unapproved.txt"
			case "directory":
				suiteState.Require().NoError(os.Mkdir(filepath.Join(private, leaf), 0o700))
			case "public-root":
				suiteState.Require().NoError(os.Chmod(private, 0o755))
			case "hardlink", "public-file", "nonempty-file":
				arguments[2] = "task-error.ABCDEFGH"
				filename := filepath.Join(private, arguments[2])
				if scenario == "hardlink" {
					suiteState.Require().NoError(os.Link(sentinel, filename))
				} else {
					mode, content := os.FileMode(0o600), "nonempty"
					if scenario == "public-file" {
						mode, content = 0o644, ""
					}
					suiteState.Require().NoError(os.WriteFile(filename, []byte(content), mode))
				}
			case "exclusive-summary":
				arguments[2] = "summary.txt"
				arguments = append(arguments, "summary")
				suiteState.Require().NoError(os.Symlink(sentinel, filepath.Join(private, "raw-summary.json")))
			}
			command := exec.Command("node", arguments...)
			command.Dir = root
			command.Env = append(os.Environ(), "KUBECONFIG=/dev/null", "MODEL_KEY=owned-diagnostic-canary")
			command.Stdin = strings.NewReader("owned-diagnostic-canary text")
			output, err := command.CombinedOutput()
			if scenario == "normal" || scenario == "summary" {
				suiteState.Require().NoError(err, string(output))
				filename := "check.txt"
				expected := "[REDACTED] text"
				if scenario == "summary" {
					filename, expected = "raw-summary.json", "owned-diagnostic-canary text"
				}
				data, readErr := os.ReadFile(filepath.Join(private, filename))
				suiteState.Require().NoError(readErr)
				suiteState.Equal(expected, string(data))
				info, statErr := os.Stat(filepath.Join(private, filename))
				suiteState.Require().NoError(statErr)
				suiteState.Equal(os.FileMode(0o600), info.Mode().Perm())
			} else {
				suiteState.Error(err, string(output))
			}
			data, readErr := os.ReadFile(sentinel)
			suiteState.Require().NoError(readErr)
			suiteState.Equal("unchanged", string(data))
		})
	}
}
