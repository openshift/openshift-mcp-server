package build_test

import (
	"os"
	"os/exec"
	"regexp"
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
