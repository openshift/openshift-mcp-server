package build_test

import (
	"os"
	"os/exec"
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
	suiteState.Contains(string(output), "Ran 10 tests")
	suiteState.Contains(string(output), "OK")
}
