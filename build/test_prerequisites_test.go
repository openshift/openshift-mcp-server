package build_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestPrerequisitesSuite struct{ suite.Suite }

func TestTestPrerequisites(testingT *testing.T) {
	suite.Run(testingT, new(TestPrerequisitesSuite))
}

func (suiteState *TestPrerequisitesSuite) TestOfflineBootstrapAndMakeConsumer() {
	command := exec.Command("python3", workflowBuildPath(suiteState.T(), "test_prerequisites_test.py"), "-v")
	command.Env = append(os.Environ(), "KUBECONFIG=/dev/null", "PYTHONDONTWRITEBYTECODE=1")
	output, err := command.CombinedOutput()
	suiteState.NoError(err, string(output))
	suiteState.True(successfulPythonSuite(string(output)), string(output))
}

func (suiteState *TestPrerequisitesSuite) TestFixtureTrafficIgnoresInheritedProxies() {
	var requests atomic.Int64
	proxy := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		suiteState.Equal(http.MethodConnect, request.Method)
		suiteState.Equal("official.invalid:443", request.Host)
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	environment := append(os.Environ(), "KUBECONFIG=/dev/null", "PYTHONDONTWRITEBYTECODE=1", "NO_PROXY=", "no_proxy=", "REQUEST_METHOD=")
	for _, name := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		environment = append(environment, name+"="+proxy.URL)
	}
	command := exec.Command("python3", workflowBuildPath(suiteState.T(), "test_prerequisites_test.py"), "-v")
	command.Env = environment
	output, err := command.CombinedOutput()
	suiteState.NoError(err, string(output))
	suiteState.True(successfulPythonSuite(string(output)), string(output))
	suiteState.Contains(string(output), "test_default_make_consumer_preserves_command_and_environment")
	suiteState.Zero(requests.Load(), string(output))
	command = exec.Command("python3", "-c", `import importlib.util,sys,urllib.error
spec=importlib.util.spec_from_file_location("production",sys.argv[1])
module=importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
try:
    module.open_release("https://official.invalid/release",10)
except urllib.error.URLError as error:
    assert "502" in str(error),str(error)
else:
    raise AssertionError("expected the owned proxy to refuse CONNECT")
`, workflowBuildPath(suiteState.T(), "test-prerequisites.py"))
	command.Env = environment
	output, err = command.CombinedOutput()
	suiteState.NoError(err, string(output))
	suiteState.Equal(int64(1), requests.Load(), string(output))
}

func (suiteState *TestPrerequisitesSuite) TestEvalServerStartup() {
	command := exec.Command("python3", workflowBuildPath(suiteState.T(), "eval_startup_test.py"), "-v")
	command.Env = append(os.Environ(), "KUBECONFIG=/dev/null", "PYTHONDONTWRITEBYTECODE=1")
	output, err := command.CombinedOutput()
	suiteState.NoError(err, string(output))
	suiteState.True(successfulPythonSuite(string(output)), string(output))
}

func successfulPythonSuite(output string) bool {
	return regexp.MustCompile(`(?m)^Ran [1-9][0-9]* tests? in [0-9]+(?:\.[0-9]+)?s\r?$`).MatchString(output) &&
		regexp.MustCompile(`\nOK\r?\n?$`).MatchString(output) &&
		!regexp.MustCompile(`(?im)^FAILED\b|^ERROR\b|\bskipped\b`).MatchString(output)
}

func (suiteState *TestPrerequisitesSuite) TestPythonCompletionSummary() {
	for _, output := range []string{"Ran 1 test in 0.1s\n\nOK\n", "Ran 30 tests in 1.23s\n\nOK\n"} {
		suiteState.True(successfulPythonSuite(output), output)
	}
	for _, output := range []string{"Ran 0 tests in 0.1s\n\nOK\n", "Ran 1 test in 0.1s\n\nFAILED (failures=1)\n", "Ran 1 test in 0.1s\n\nOK (skipped=1)\n", "test ... skipped 'offline'\nRan 1 test in 0.1s\n\nOK\n", "Ran many tests\n\nOK\n", "Ran 1 test in 0.1s\nnot OK\n", "OK\n"} {
		suiteState.False(successfulPythonSuite(output), output)
	}
}
