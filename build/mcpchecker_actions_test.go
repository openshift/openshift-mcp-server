package build_test

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MCPCheckerActionsSuite struct{ suite.Suite }

type runnerFixture struct {
	root, work, temp, output, checker, agent, log, marker string
	environment                                           []string
}

func newRunnerFixture(testingT *testing.T) runnerFixture {
	testingT.Helper()
	root := testingT.TempDir()
	fixture := runnerFixture{root: root, work: filepath.Join(root, "work '$literal"), temp: filepath.Join(root, "temp"), output: filepath.Join(root, "output"), log: filepath.Join(root, "argv"), marker: filepath.Join(root, "SENTINEL")}
	bin := filepath.Join(fixture.temp, "mcpchecker-action.fixture/bin")
	fixture.checker, fixture.agent = filepath.Join(bin, "mcpchecker"), filepath.Join(bin, "agent")
	for _, directory := range []string{fixture.work, bin, filepath.Join(root, ".github/actions/mcpchecker-action")} {
		require.NoError(testingT, os.MkdirAll(directory, 0o700))
	}
	for _, name := range []string{"run-mcpchecker-action.sh", "run-mcpchecker-diff.sh", "mcpchecker-diagnostics.cjs", "mcpchecker-capture.py"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(installerPath(testingT)), name))
		require.NoError(testingT, err)
		writeActionFixture(testingT, filepath.Join(root, "build", name), string(data), 0o755)
	}
	checker := `#!/usr/bin/env bash
set -euo pipefail
printf '%s\0' "$@" >> "$ARGV_LOG"
printf '\0' >> "$ARGV_LOG"
case "$1/$2" in
check/*)
  result=mcpchecker-core-e2e-out.json
  case "${RESULT_MODE:-good}" in
    missing) exit "${CHECK_EXIT:-0}" ;;
    wrong) result=mcpchecker-wrong-out.json ;;
    symlink) ln -s "$PAYLOAD" "$result"; exit "${CHECK_EXIT:-0}" ;;
    malformed) printf '{bad' > "$result"; exit "${CHECK_EXIT:-0}" ;;
    zero) printf '[]' > "$result"; exit "${CHECK_EXIT:-0}" ;;
    ambiguous) cp "$PAYLOAD" mcpchecker-other-out.json ;;
  esac
  if [[ ${RESULT_MODE:-good} == timeout ]]; then
    printf '%s' "$$" > "$PARENT_PID"
    node -e 'const fs=require("fs");fs.writeFileSync(process.env.CHILD_PID,String(process.pid));process.on("SIGTERM",()=>{fs.writeFileSync(process.env.CHILD_TERM,"terminated");process.exit(0)});setInterval(()=>{},1000)' &
    child=$!
    trap 'wait "$child" || :; exit 124' TERM
    wait "$child"
    exit 99
  fi
  if [[ ${RESULT_MODE:-good} == early-parent || ${RESULT_MODE:-good} == stubborn-child ]]; then
    printf '%s' "$$" > "$PARENT_PID"
    python3 -c 'import os,signal,time; signal.signal(signal.SIGTERM,signal.SIG_IGN); open(os.environ["CHILD_PID"],"w").write(str(os.getpid())); time.sleep(3)' &
    child=$!
    while [[ ! -s $CHILD_PID ]]; do sleep 0.01; done
    cp "$PAYLOAD" "$result"
    if [[ $RESULT_MODE == stubborn-child ]]; then
      trap 'exit 0' TERM
      wait "$child"
    fi
    exit 0
  fi
  cp "$PAYLOAD" "$result"
  if [[ ${RESULT_MODE:-good} == huge-error ]]; then head -c 3145728 /dev/zero | tr '\0' x > task-error.txt; printf '%s' "$MODEL_KEY" >> task-error.txt; fi
  if [[ ${LARGE_OUTPUT:-false} == true ]]; then cat "$result"; fi
  if [[ ${RESULT_MODE:-good} != huge-error ]]; then printf '%s' "${MODEL_KEY:-diagnostic}" > task-error.txt; fi
  printf '%s' "${JUDGE_API_KEY:-check diagnostic}"
  exit "${CHECK_EXIT:-0}"
  ;;
result/summary)
  if [[ ${SUMMARY_MODE:-good} == overflow ]]; then head -c 67108865 /dev/zero | tr '\0' x; exit 0; fi
  if [[ ${SUMMARY_MODE:-good} == malformed ]]; then printf '{bad'; exit 0; fi
  if [[ ${SUMMARY_MODE:-good} == injected ]]; then printf '{"tasksTotal":"5\npassed=true"}'; exit 0; fi
  jq 'if type == "array" then . else .results end | ([.[]|select(has("assertionPassed"))]|length) as $total | ([.[]|select(.assertionPassed==true)]|length) as $passed | {tasksTotal:length,tasksPassed:([.[]|select(.taskPassed)]|length),taskPassRate:(([.[]|select(.taskPassed)]|length)/length),assertionsTotal:$total,assertionsPassed:$passed,assertionPassRate:(if $total==0 then 0 else $passed/$total end)} | if env.SUMMARY_MODE=="large" then .padding=("x"*3000000) else . end' "$3"
  exit "${SUMMARY_EXIT:-0}"
  ;;
result/verify)
  [[ ${VERIFY_EXIT:-0} == 0 ]] || exit "$VERIFY_EXIT"
  jq -e --argjson task "${4#--task=}" --argjson assertion "${5#--assertion=}" 'if type == "array" then . else .results end | ([.[]|select(has("assertionPassed"))]|length) as $total | ([.[]|select(.assertionPassed==true)]|length) as $passed | (([.[]|select(.taskPassed)]|length)/length >= $task) and ($total==0 or $passed/$total >= $assertion)' "$3" >/dev/null
  ;;
result/diff)
  printf 'MCPCHECKER_DIFF_EOF\nMCPCHECKER_DIFF_%s\n' "${RUNNER_TEMP##*/}"
  printf 'MCPCHECKER_DIFF_%s\r\n' "$(basename "$(find "$RUNNER_TEMP" -maxdepth 1 -type d -name 'mcpchecker-diff.*' | head -1)" | cut -d. -f2)"
  printf 'diff stderr' >&2
  exit "${DIFF_EXIT:-0}"
  ;;
*) exit 98 ;;
esac
`
	for _, binary := range []string{fixture.checker, fixture.agent} {
		data := checker
		if binary == fixture.agent {
			data = "#!/bin/sh\nexit 99\n"
		}
		writeActionFixture(testingT, binary, data, 0o755)
		writeActionFixture(testingT, binary+".metadata", fmt.Sprintf("binary_sha256=%x\n", sha256.Sum256([]byte(data))), 0o644)
	}
	payload := `[{"taskPassed":true,"assertionPassed":true,"taskOutput":"secret-model"},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":false,"assertionPassed":false}]`
	writeActionFixture(testingT, filepath.Join(root, "payload"), payload, 0o600)
	writeActionFixture(testingT, filepath.Join(fixture.work, "eval.yaml"), "fixture", 0o600)
	writeActionFixture(testingT, fixture.output, "", 0o600)
	writeActionFixture(testingT, filepath.Join(fixture.work, "base.json"), payload, 0o600)
	writeActionFixture(testingT, filepath.Join(fixture.work, "current.json"), payload, 0o600)
	fixture.environment = append(os.Environ(), "KUBECONFIG=/dev/null", "GITHUB_ACTION_PATH="+filepath.Join(root, ".github/actions/mcpchecker-action"), "RUNNER_TEMP="+fixture.temp,
		"GITHUB_OUTPUT="+fixture.output, "MCPCHECKER="+fixture.checker, "AGENT="+fixture.agent, "MCPCHECKER_VERSION=v0.0.22", "WORKING_DIRECTORY="+fixture.work,
		"EVAL_CONFIG=eval.yaml", "EXPECTED_RESULTS_FILE=mcpchecker-core-e2e-out.json", "TASK_THRESHOLD=0.8", "ASSERTION_THRESHOLD=0.8", "FAIL_ON_ERROR=true",
		"ARGV_LOG="+fixture.log, "PAYLOAD="+filepath.Join(root, "payload"), "BASE=base.json", "CURRENT=current.json", "MODEL_KEY=secret-model", "JUDGE_API_KEY=secret-judge")
	return fixture
}

func (fixture runnerFixture) run(testingT *testing.T, script string, overrides ...string) (string, string, error) {
	testingT.Helper()
	command := exec.Command("bash", filepath.Join(fixture.root, "build", script))
	command.Dir = fixture.root
	command.Env = append(append([]string(nil), fixture.environment...), overrides...)
	output, err := command.CombinedOutput()
	values, readErr := os.ReadFile(fixture.output)
	require.NoError(testingT, readErr)
	return string(output), string(values), err
}

func (suiteState *MCPCheckerActionsSuite) TestIndependentThresholdDataAndLargeStreams() {
	for _, testCase := range []struct {
		name    string
		payload string
		passed  bool
	}{
		{"tasksBelow", `[{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":false,"assertionPassed":true},{"taskPassed":false,"assertionPassed":false}]`, false},
		{"assertionsBelow", `[{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":false},{"taskPassed":false,"assertionPassed":false}]`, false},
		{"zeroAssertions", `[{"taskPassed":true},{"taskPassed":true},{"taskPassed":true},{"taskPassed":true},{"taskPassed":false}]`, true},
	} {
		for _, failOnError := range []string{"true", "false"} {
			suiteState.Run(testCase.name+failOnError, func() {
				fixture := newRunnerFixture(suiteState.T())
				writeActionFixture(suiteState.T(), filepath.Join(fixture.root, "payload"), testCase.payload, 0o600)
				output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", "FAIL_ON_ERROR="+failOnError)
				suiteState.Equal(testCase.passed || failOnError == "false", err == nil, output)
				suiteState.Equal(testCase.passed, strings.Contains(values, "passed=true\n"), values)
				suiteState.Contains(values, "check-exit=0\n")
				suiteState.Contains(values, "summary-exit=0\n")
				if !testCase.passed {
					suiteState.Contains(values, "verify-exit=1\n")
				}
			})
		}
	}
	for _, checkExit := range []string{"0", "7"} {
		suiteState.Run("large"+checkExit, func() {
			fixture := newRunnerFixture(suiteState.T())
			rows := []map[string]any{}
			for index := 0; index < 5; index++ {
				rows = append(rows, map[string]any{"taskPassed": index < 4, "assertionPassed": index < 4, "taskOutput": strings.Repeat("x", 600000)})
			}
			payload, err := json.Marshal(rows)
			suiteState.Require().NoError(err)
			writeActionFixture(suiteState.T(), filepath.Join(fixture.root, "payload"), string(payload), 0o600)
			output, values, runErr := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", "LARGE_OUTPUT=true", "SUMMARY_MODE=large", "CHECK_EXIT="+checkExit, "FAIL_ON_ERROR=false")
			suiteState.NoError(runErr, output)
			suiteState.Contains(values, "check-exit="+checkExit+"\n")
			suiteState.Contains(values, "summary-exit=0\nverify-exit=0\n")
			suiteState.Equal(checkExit == "0", strings.Contains(values, "passed=true\n"))
			files, globErr := filepath.Glob(filepath.Join(fixture.temp, "mcpchecker-diagnostics.*", "check.txt"))
			suiteState.Require().NoError(globErr)
			suiteState.Require().Len(files, 1)
			bytes, readErr := os.ReadFile(files[0])
			suiteState.Require().NoError(readErr)
			suiteState.LessOrEqual(len(bytes), 1048576)
			suiteState.Contains(string(bytes), "[diagnostic truncated]")
			result, readErr := os.ReadFile(filepath.Join(fixture.work, "mcpchecker-core-e2e-out.json"))
			suiteState.Require().NoError(readErr)
			suiteState.Greater(len(result), 2097152)
			suiteState.True(json.Valid(result))
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) accelerateCapture(fixture runnerFixture) {
	path := filepath.Join(fixture.root, "build", "mcpchecker-capture.py")
	data, err := os.ReadFile(path)
	suiteState.Require().NoError(err)
	suiteState.Require().Contains(string(data), "time.monotonic() + 1800")
	suiteState.Require().Contains(string(data), "grace = 30")
	text := strings.Replace(string(data), "time.monotonic() + 1800", "time.monotonic() + 0.5", 1)
	text = strings.Replace(text, "grace = 30", "grace = 0.2", 1)
	writeActionFixture(suiteState.T(), path, text, 0o600)
}

func (suiteState *MCPCheckerActionsSuite) TestCaptureInheritedPipeLifetime() {
	for _, scenario := range []string{"early-parent", "stubborn-child"} {
		for _, failOnError := range []string{"true", "false"} {
			suiteState.Run(scenario+"-"+failOnError, func() {
				fixture := newRunnerFixture(suiteState.T())
				suiteState.accelerateCapture(fixture)
				start := time.Now()
				output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", "RESULT_MODE="+scenario, "FAIL_ON_ERROR="+failOnError, "PARENT_PID="+filepath.Join(fixture.root, "parent.pid"), "CHILD_PID="+filepath.Join(fixture.root, "child.pid"))
				suiteState.Less(time.Since(start), 3*time.Second, output)
				suiteState.Equal(failOnError == "true", err != nil, output)
				suiteState.Contains(values, "check-exit=124\n")
				suiteState.NotContains(values, "passed=true\n")
				suiteState.Contains(output, "capture checker-exit=0 diagnostic-exit=0 expired=1")
				for _, name := range []string{"parent.pid", "child.pid"} {
					data, readErr := os.ReadFile(filepath.Join(fixture.root, name))
					suiteState.Require().NoError(readErr)
					pid, parseErr := strconv.Atoi(string(data))
					suiteState.Require().NoError(parseErr)
					_, statErr := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
					suiteState.True(os.IsNotExist(statErr), "owned PID remains live or zombie: %d", pid)
				}
			})
		}
	}
}

func (suiteState *MCPCheckerActionsSuite) TestCaptureDiagnosticFailureStatus() {
	fixture := newRunnerFixture(suiteState.T())
	writeActionFixture(suiteState.T(), filepath.Join(fixture.root, "build", "mcpchecker-diagnostics.cjs"), "process.stdin.resume();process.stdin.on('end',()=>process.exit(23));", 0o600)
	output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", "FAIL_ON_ERROR=false")
	suiteState.Error(err, output)
	suiteState.Contains(output, "capture checker-exit=0 diagnostic-exit=23 expired=0")
	suiteState.NotContains(values, "passed=true\n")
}

func (suiteState *MCPCheckerActionsSuite) TestStreamLimitsTimeoutAndCleanup() {
	for _, scenario := range []string{"huge-error", "overflow", "timeout", "missing-python"} {
		suiteState.Run(scenario, func() {
			fixture := newRunnerFixture(suiteState.T())
			overrides := []string{"FAIL_ON_ERROR=false"}
			switch scenario {
			case "overflow":
				overrides = append(overrides, "SUMMARY_MODE=overflow")
			case "huge-error":
				overrides = append(overrides, "RESULT_MODE=huge-error")
			}
			if scenario == "timeout" || scenario == "missing-python" {
				suiteState.accelerateCapture(fixture)
				if scenario == "missing-python" {
					bin := filepath.Join(fixture.root, "unsupported-bin")
					writeActionFixture(suiteState.T(), filepath.Join(bin, "uname"), "#!/bin/sh\necho unsupported\n", 0o755)
					overrides = append(overrides, "PATH="+bin+":"+os.Getenv("PATH"))
				}
				overrides = append(overrides, "RESULT_MODE=timeout", "PARENT_PID="+filepath.Join(fixture.root, "parent.pid"), "CHILD_PID="+filepath.Join(fixture.root, "child.pid"), "CHILD_TERM="+filepath.Join(fixture.root, "child.term"))
			}
			output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", overrides...)
			if scenario == "overflow" || scenario == "missing-python" {
				suiteState.Error(err, output)
			} else {
				suiteState.NoError(err, output)
			}
			suiteState.Equal(scenario == "huge-error", strings.Contains(values, "passed=true\n"), values)
			if scenario == "missing-python" {
				suiteState.Contains(output, "requires Linux, Node.js and Python 3")
			}
			if scenario == "timeout" {
				suiteState.Contains(values, "check-exit=124\n")
				suiteState.FileExists(filepath.Join(fixture.root, "child.term"))
				for _, name := range []string{"parent.pid", "child.pid"} {
					bytes, readErr := os.ReadFile(filepath.Join(fixture.root, name))
					suiteState.Require().NoError(readErr)
					pid, parseErr := strconv.Atoi(string(bytes))
					suiteState.Require().NoError(parseErr)
					suiteState.Eventually(func() bool { return exec.Command("kill", "-0", strconv.Itoa(pid)).Run() != nil }, 3*time.Second, 20*time.Millisecond, "owned process survives: %d", pid)
				}
			}
			diagnostics, globErr := filepath.Glob(filepath.Join(fixture.temp, "mcpchecker-diagnostics.*"))
			suiteState.Require().NoError(globErr)
			for _, directory := range diagnostics {
				suiteState.NoFileExists(filepath.Join(directory, "raw-summary.json"))
				errors, _ := filepath.Glob(filepath.Join(directory, "task-error.*"))
				if scenario == "huge-error" {
					suiteState.Require().Len(errors, 1)
					bytes, readErr := os.ReadFile(errors[0])
					suiteState.Require().NoError(readErr)
					suiteState.LessOrEqual(len(bytes), 1048576)
					suiteState.Contains(string(bytes), "[diagnostic truncated]")
					suiteState.NotContains(string(bytes), "secret-model")
				}
			}
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) TestEvaluationRunnerStatusesAndFreshness() {
	for _, testCase := range []struct {
		name      string
		overrides []string
		passed    bool
		success   bool
	}{
		{"literal80percent", nil, true, true},
		{"generic100percent", []string{"TASK_THRESHOLD=1.0", "ASSERTION_THRESHOLD=1.0"}, false, false},
		{"failedCheck", []string{"CHECK_EXIT=7"}, false, false},
		{"failedCheckCollect", []string{"CHECK_EXIT=7", "FAIL_ON_ERROR=false"}, false, true},
		{"failedVerify", []string{"VERIFY_EXIT=8"}, false, false},
		{"failedVerifyCollect", []string{"VERIFY_EXIT=8", "FAIL_ON_ERROR=false"}, false, true},
		{"failedSummary", []string{"SUMMARY_EXIT=9"}, false, false},
		{"badSummary", []string{"SUMMARY_MODE=malformed"}, false, false},
		{"injectedSummary", []string{"SUMMARY_MODE=injected"}, false, false},
		{"missing", []string{"RESULT_MODE=missing"}, false, false},
		{"wrongName", []string{"RESULT_MODE=wrong"}, false, false},
		{"symlink", []string{"RESULT_MODE=symlink"}, false, false},
		{"ambiguous", []string{"RESULT_MODE=ambiguous"}, false, false},
		{"zeroTasks", []string{"RESULT_MODE=zero"}, false, false},
		{"malformed", []string{"RESULT_MODE=malformed"}, false, false},
		{"genericDiscovery", []string{"EXPECTED_RESULTS_FILE="}, true, true},
		{"redactionFailure", []string{"MODEL_KEY=REDACTED"}, false, false},
	} {
		suiteState.Run(testCase.name, func() {
			fixture := newRunnerFixture(suiteState.T())
			output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", testCase.overrides...)
			if testCase.success {
				suiteState.NoError(err, output)
			} else {
				suiteState.Error(err, output)
			}
			suiteState.Equal(testCase.passed, strings.Contains(values, "passed=true\n"), values)
			suiteState.Contains(values, "passed=false\n")
			if testCase.passed {
				suiteState.Contains(values, "task-pass-rate=0.8\n")
				suiteState.Contains(values, "tasks-passed=4\n")
				artifacts, globErr := filepath.Glob(filepath.Join(fixture.temp, "mcpchecker-diagnostics.*"))
				suiteState.Require().NoError(globErr)
				suiteState.Require().Len(artifacts, 1)
				suiteState.FileExists(filepath.Join(artifacts[0], "mcpchecker-core-e2e-out.json"))
				redacted, readErr := os.ReadFile(filepath.Join(artifacts[0], "mcpchecker-core-e2e-out.json"))
				suiteState.Require().NoError(readErr)
				suiteState.Contains(string(redacted), "[REDACTED]")
			}
			suiteState.NotContains(output+values, "secret-model")
			suiteState.NotContains(output+values, "secret-judge")
			if testCase.name == "failedCheckCollect" {
				suiteState.Contains(values, "check-exit=7\n")
				suiteState.Contains(values, "verify-exit=0\n")
				suiteState.Contains(values, "results-file="+fixture.work+"/mcpchecker-core-e2e-out.json\n")
			}
			if testCase.name == "failedSummary" {
				suiteState.Contains(values, "summary-exit=9\n")
			}
			suiteState.NoFileExists(fixture.marker)
			require.NoError(suiteState.T(), filepath.WalkDir(fixture.temp, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil || entry.IsDir() || !strings.Contains(path, "mcpchecker-diagnostics.") {
					return walkErr
				}
				data, readErr := os.ReadFile(path)
				suiteState.NoError(readErr)
				suiteState.NotContains(string(data), "secret-model")
				suiteState.NotContains(string(data), "secret-judge")
				return nil
			}))
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) TestRunnerLiteralArgumentsAndValidation() {
	hostile := "-$(touch SENTINEL)`touch SENTINEL`'\"; space\n*[?]"
	fixture := newRunnerFixture(suiteState.T())
	writeActionFixture(suiteState.T(), filepath.Join(fixture.work, hostile), "fixture", 0o600)
	output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", "EVAL_CONFIG="+hostile, "TASK_FILTER="+hostile, "LABEL_SELECTOR="+hostile, "VERBOSE=true", "PARALLEL=7", "OUTPUT_FORMAT=text")
	suiteState.Require().NoError(err, output+values)
	arguments, err := os.ReadFile(fixture.log)
	suiteState.Require().NoError(err)
	firstCall := strings.SplitN(string(arguments), "\x00\x00", 2)[0]
	suiteState.Equal([]string{"check", "--output=text", "--parallel=7", "--run=" + hostile, "--label-selector=" + hostile, "--verbose", "--", filepath.Join(fixture.work, hostile)}, strings.Split(firstCall, "\x00"))
	suiteState.NoFileExists(filepath.Join(fixture.work, "SENTINEL"))
	for _, invalid := range []string{"VERBOSE=yes", "FAIL_ON_ERROR=", "UPLOAD_ARTIFACTS=1", "OUTPUT_FORMAT=yaml", "PARALLEL=0", "PARALLEL=1000", "PARALLEL=-1", "TASK_THRESHOLD=1.1", "ASSERTION_THRESHOLD=NaN", "TASK_THRESHOLD=-0.1", "EXPECTED_RESULTS_FILE=../x", "EXPECTED_RESULTS_FILE=mcpchecker-a\n-out.json", "MCPCHECKER=", "AGENT=/bin/true", "MCPCHECKER_VERSION=latest", "WORKING_DIRECTORY=/", "EVAL_CONFIG=missing.yaml"} {
		suiteState.Run(invalid, func() {
			fixture := newRunnerFixture(suiteState.T())
			_, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", invalid)
			suiteState.Error(err)
			suiteState.NotContains(values, "passed=true\n")
			suiteState.NoFileExists(fixture.log)
		})
	}
	for _, kind := range []string{"file", "symlink", "directory", "error"} {
		suiteState.Run("stale-"+kind, func() {
			fixture := newRunnerFixture(suiteState.T())
			path := filepath.Join(fixture.work, "mcpchecker-core-e2e-out.json")
			switch kind {
			case "file":
				writeActionFixture(suiteState.T(), path, "stale", 0o600)
			case "symlink":
				suiteState.Require().NoError(os.Symlink("missing", path))
			case "directory":
				suiteState.Require().NoError(os.Mkdir(path, 0o700))
			case "error":
				writeActionFixture(suiteState.T(), filepath.Join(fixture.work, "task-error.txt"), "stale", 0o600)
			}
			_, values, err := fixture.run(suiteState.T(), "run-mcpchecker-action.sh")
			suiteState.Error(err)
			suiteState.NotContains(values, "passed=true\n")
			suiteState.NoFileExists(fixture.log)
		})
	}
	for _, kind := range []string{"tamper", "checker-symlink", "result-object", "secret-object-key", "diff-symlink", "diff-zero", "diff-corrupt-current"} {
		suiteState.Run(kind, func() {
			fixture := newRunnerFixture(suiteState.T())
			script := "run-mcpchecker-action.sh"
			passed := false
			switch kind {
			case "tamper":
				writeActionFixture(suiteState.T(), fixture.checker, "#!/bin/sh\nexit 0\n", 0o755)
			case "checker-symlink":
				suiteState.Require().NoError(os.Rename(fixture.checker, fixture.checker+".original"))
				suiteState.Require().NoError(os.Symlink(fixture.checker+".original", fixture.checker))
			case "result-object":
				data, err := os.ReadFile(filepath.Join(fixture.root, "payload"))
				suiteState.Require().NoError(err)
				writeActionFixture(suiteState.T(), filepath.Join(fixture.root, "payload"), `{"results":`+string(data)+`}`, 0o600)
				passed = true
			case "secret-object-key":
				writeActionFixture(suiteState.T(), filepath.Join(fixture.root, "payload"), `[{"taskPassed":true,"secret-model":"value"}]`, 0o600)
			case "diff-symlink":
				script = "run-mcpchecker-diff.sh"
				suiteState.Require().NoError(os.Remove(filepath.Join(fixture.work, "base.json")))
				suiteState.Require().NoError(os.Symlink(filepath.Join(fixture.root, "payload"), filepath.Join(fixture.work, "base.json")))
			case "diff-zero":
				script = "run-mcpchecker-diff.sh"
				writeActionFixture(suiteState.T(), filepath.Join(fixture.work, "base.json"), `[]`, 0o600)
			case "diff-corrupt-current":
				script = "run-mcpchecker-diff.sh"
				writeActionFixture(suiteState.T(), filepath.Join(fixture.work, "current.json"), `{bad`, 0o600)
			}
			output, values, err := fixture.run(suiteState.T(), script)
			if passed {
				suiteState.NoError(err, output+values)
				suiteState.Contains(values, "passed=true\n")
			} else {
				suiteState.Error(err, output+values)
				suiteState.NotContains(values, "passed=true\n")
			}
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) TestDiffRunnerExitDataAndCorruptInputs() {
	for _, status := range []int{0, 7, 42} {
		suiteState.Run(fmt.Sprint(status), func() {
			fixture := newRunnerFixture(suiteState.T())
			hostile := "-base $(touch SENTINEL)`touch SENTINEL`;\n'\"*?.json"
			payload, err := os.ReadFile(filepath.Join(fixture.work, "base.json"))
			suiteState.Require().NoError(err)
			writeActionFixture(suiteState.T(), filepath.Join(fixture.work, hostile), string(payload), 0o600)
			output, values, err := fixture.run(suiteState.T(), "run-mcpchecker-diff.sh", "DIFF_EXIT="+fmt.Sprint(status), "BASE="+hostile)
			if status == 0 {
				suiteState.NoError(err, output)
			} else {
				var exitError *exec.ExitError
				suiteState.Require().ErrorAs(err, &exitError)
				suiteState.Equal(status, exitError.ExitCode())
			}
			suiteState.Contains(values, "exit-code="+fmt.Sprint(status)+"\n")
			suiteState.Contains(values, "MCPCHECKER_DIFF_EOF\n")
			arguments, readErr := os.ReadFile(fixture.log)
			suiteState.Require().NoError(readErr)
			suiteState.Equal([]string{"result", "diff", "--base", filepath.Join(fixture.work, hostile), "--current", filepath.Join(fixture.work, "current.json"), "--output", "markdown", "", ""}, strings.Split(string(arguments), "\x00"))
			entries, globErr := filepath.Glob(filepath.Join(fixture.temp, "mcpchecker-diff.*"))
			suiteState.Require().NoError(globErr)
			suiteState.Require().Len(entries, 1)
			collision := "MCPCHECKER_DIFF_" + strings.TrimPrefix(filepath.Base(entries[0]), "mcpchecker-diff.")
			suiteState.Contains(values, "diff-output<<"+collision+"_X\n")
			suiteState.FileExists(filepath.Join(entries[0], "error.txt"))
			suiteState.FileExists(filepath.Join(entries[0], "diff.txt"))
			suiteState.NoFileExists(filepath.Join(fixture.work, "SENTINEL"))
		})
	}
	for _, invalid := range []string{"BASE=missing.json", "CURRENT=missing.json", "BASE=eval.yaml", "MCPCHECKER=/bin/true", "MCPCHECKER_VERSION=v0.0.21"} {
		suiteState.Run(invalid, func() {
			fixture := newRunnerFixture(suiteState.T())
			_, values, err := fixture.run(suiteState.T(), "run-mcpchecker-diff.sh", invalid)
			suiteState.Error(err)
			suiteState.Contains(values, "exit-code=1\n")
			suiteState.NoFileExists(fixture.log)
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) TestCompositeTrustedScriptConsumers() {
	for _, action := range []string{"mcpchecker-action", "mcpchecker-diff-action"} {
		suiteState.Run(action, func() {
			data, err := os.ReadFile(filepath.Join(filepath.Dir(installerPath(suiteState.T())), "../.github/actions", action, "action.yaml"))
			suiteState.Require().NoError(err)
			var document struct {
				Inputs map[string]struct{ Default string }
				Runs   struct {
					Steps []struct {
						ID, Run, Uses string
						Env           map[string]string
						With          map[string]string
					}
				}
			}
			suiteState.Require().NoError(yaml.Unmarshal(data, &document))
			suiteState.Equal("v0.0.22", document.Inputs["mcpchecker-version"].Default)
			suiteState.Equal(`bash "$GITHUB_ACTION_PATH/../../../build/setup-mcpchecker-action.sh"`, document.Runs.Steps[0].Run)
			suiteState.Equal("${{ inputs.mcpchecker-version }}", document.Runs.Steps[0].Env["MCPCHECKER_VERSION"])
			runner := document.Runs.Steps[1]
			suiteState.Equal("${{ steps.setup.outputs.mcpchecker-path }}", runner.Env["MCPCHECKER"])
			suiteState.Equal("${{ steps.setup.outputs.agent-path }}", runner.Env["AGENT"])
			suiteState.Equal("${{ inputs.working-directory }}", runner.Env["WORKING_DIRECTORY"])
			for _, step := range document.Runs.Steps {
				suiteState.NotContains(step.Run, "${{")
				suiteState.NotContains(step.Uses, "setup-mcpchecker")
			}
			if action == "mcpchecker-action" {
				for key, input := range map[string]string{"EVAL_CONFIG": "eval-config", "TASK_FILTER": "task-filter", "LABEL_SELECTOR": "label-selector", "OUTPUT_FORMAT": "output-format", "VERBOSE": "verbose", "UPLOAD_ARTIFACTS": "upload-artifacts", "FAIL_ON_ERROR": "fail-on-error", "TASK_THRESHOLD": "task-pass-threshold", "ASSERTION_THRESHOLD": "assertion-pass-threshold", "PARALLEL": "parallel", "EXPECTED_RESULTS_FILE": "expected-results-file"} {
					suiteState.Equal("${{ inputs."+input+" }}", runner.Env[key])
				}
				suiteState.Equal("1.0", document.Inputs["task-pass-threshold"].Default)
				suiteState.Equal("1.0", document.Inputs["assertion-pass-threshold"].Default)
				suiteState.Equal(`bash "$GITHUB_ACTION_PATH/../../../build/run-mcpchecker-action.sh"`, runner.Run)
				upload := document.Runs.Steps[2]
				suiteState.Equal("actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9", upload.Uses)
				suiteState.Equal("${{ steps.run-eval.outputs.artifact-paths }}", upload.With["path"])
			} else {
				suiteState.Equal(`bash "$GITHUB_ACTION_PATH/../../../build/run-mcpchecker-diff.sh"`, runner.Run)
			}
		})
	}
}

type setupFixture struct {
	root, runnerTemp, candidate, output, pathOutput, marker, archives string
	environment                                                       []string
}

func writeActionFixture(testingT *testing.T, path, data string, mode os.FileMode) {
	testingT.Helper()
	require.NoError(testingT, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(testingT, os.WriteFile(path, []byte(data), mode))
}

func newSetupFixture(testingT *testing.T) setupFixture {
	testingT.Helper()
	temp := testingT.TempDir()
	fixture := setupFixture{
		root: filepath.Join(temp, "trusted source '$literal"), runnerTemp: filepath.Join(temp, "runner temp"),
		candidate: filepath.Join(temp, "candidate"), output: filepath.Join(temp, "output"),
		pathOutput: filepath.Join(temp, "path"), marker: filepath.Join(temp, "EXECUTED"), archives: filepath.Join(temp, "archives"),
	}
	for _, directory := range []string{fixture.runnerTemp, fixture.candidate, fixture.archives, filepath.Join(fixture.root, ".github/actions/mcpchecker-action")} {
		require.NoError(testingT, os.MkdirAll(directory, 0o755))
	}
	for _, name := range []string{"install-mcpchecker.sh", "setup-mcpchecker-action.sh"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(installerPath(testingT)), name))
		require.NoError(testingT, err)
		writeActionFixture(testingT, filepath.Join(fixture.root, "build", name), string(data), 0o755)
	}
	var entries []string
	for _, goos := range []string{"linux", "darwin", "windows"} {
		for _, goarch := range []string{"amd64", "arm64"} {
			for _, component := range []string{"mcpchecker", "agent"} {
				member := component
				if goos == "windows" {
					member += ".exe"
				}
				asset := component + "-" + goos + "-" + goarch + ".zip"
				_, digest := createArchive(testingT, fixture.archives, asset, member, fmt.Sprintf("#!/bin/sh\nprintf executed > %q\nexit 91\n", fixture.marker))
				entries = append(entries, digest+"  "+asset)
			}
		}
	}
	writeManifest(testingT, filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256"), entries...)
	transport := createFakeCurl(testingT, filepath.Join(temp, "transport"))
	bin := filepath.Join(temp, "bin")
	writeActionFixture(testingT, filepath.Join(bin, "curl"), `#!/bin/sh
set -eu
url=
for argument in "$@"; do case "$argument" in https://*) url=$argument ;; esac; done
FAKE_ARCHIVE="$FIXTURE_ARCHIVES/${url##*/}" exec "$FIXTURE_CURL" "$@"
`, 0o755)
	writeActionFixture(testingT, filepath.Join(fixture.candidate, "install-mcpchecker.sh"), "#!/bin/sh\nexit 99\n", 0o755)
	writeActionFixture(testingT, filepath.Join(fixture.candidate, "build/install-mcpchecker.sh"), "#!/bin/sh\nexit 99\n", 0o755)
	writeActionFixture(testingT, filepath.Join(fixture.candidate, "build/mcpchecker-v0.0.22.sha256"), "invalid candidate manifest\n", 0o644)
	writeActionFixture(testingT, fixture.output, "", 0o600)
	writeActionFixture(testingT, fixture.pathOutput, "", 0o600)
	fixture.environment = append(os.Environ(),
		"PATH="+fixture.candidate+string(os.PathListSeparator)+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GITHUB_ACTION_PATH="+filepath.Join(fixture.root, ".github/actions/mcpchecker-action"),
		"RUNNER_TEMP="+fixture.runnerTemp, "GITHUB_OUTPUT="+fixture.output, "GITHUB_PATH="+fixture.pathOutput,
		"RUNNER_OS=Linux", "RUNNER_ARCH=X64", "MCPCHECKER_VERSION=v0.0.22", "MCPCHECKER_LOCK_TIMEOUT_SECONDS=0",
		"FIXTURE_ARCHIVES="+fixture.archives, "FIXTURE_CURL="+filepath.Join(transport, "curl"),
		"FAKE_CURL_LOG="+filepath.Join(temp, "curl.log"),
	)
	return fixture
}

func (fixture setupFixture) run(testingT *testing.T, overrides ...string) (string, error) {
	testingT.Helper()
	command := exec.Command("bash", filepath.Join(fixture.root, "build/setup-mcpchecker-action.sh"))
	command.Dir = fixture.candidate
	command.Env = append(append([]string(nil), fixture.environment...), overrides...)
	output, err := command.CombinedOutput()
	return string(output), err
}

func (suiteState *MCPCheckerActionsSuite) TestSetupAllPlatformsBothComponents() {
	for _, platform := range []struct{ runnerOS, runnerArch, goos, goarch string }{
		{"Linux", "X64", "linux", "amd64"}, {"Linux", "ARM64", "linux", "arm64"},
		{"macOS", "X64", "darwin", "amd64"}, {"macOS", "ARM64", "darwin", "arm64"},
		{"Windows", "X64", "windows", "amd64"}, {"Windows", "ARM64", "windows", "arm64"},
	} {
		suiteState.Run(platform.goos+"/"+platform.goarch, func() {
			fixture := newSetupFixture(suiteState.T())
			output, err := fixture.run(suiteState.T(), "RUNNER_OS="+platform.runnerOS, "RUNNER_ARCH="+platform.runnerArch)
			suiteState.Require().NoError(err, output)
			data, err := os.ReadFile(fixture.output)
			suiteState.Require().NoError(err)
			values := map[string]string{}
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				key, value, found := strings.Cut(line, "=")
				suiteState.Require().True(found)
				values[key] = value
			}
			suiteState.Equal("v0.0.22", values["version"])
			suiteState.Len(values, 3)
			for _, component := range []string{"mcpchecker", "agent"} {
				destination := values[component+"-path"]
				suiteState.True(filepath.IsAbs(destination))
				suiteState.True(strings.HasPrefix(destination, fixture.runnerTemp+string(os.PathSeparator)))
				extension := ""
				if platform.goos == "windows" {
					extension = ".exe"
				}
				suiteState.Equal(component+extension, filepath.Base(destination))
				for _, expected := range []struct {
					path string
					mode os.FileMode
				}{
					{destination, 0o755}, {destination + ".metadata", 0o644},
					{filepath.Dir(destination), 0o700}, {filepath.Dir(filepath.Dir(destination)), 0o700},
				} {
					info, err := os.Stat(expected.path)
					suiteState.Require().NoError(err)
					suiteState.Equal(expected.mode, info.Mode().Perm())
				}
				assertNoInstallerArtifacts(suiteState.T(), destination)
				suiteState.Contains(output, component+"-"+platform.goos+"-"+platform.goarch+".zip")
			}
			pathData, err := os.ReadFile(fixture.pathOutput)
			suiteState.Require().NoError(err)
			suiteState.Equal(filepath.Dir(values["agent-path"])+"\n", string(pathData))
			suiteState.NoFileExists(fixture.marker)
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) TestSetupRejectsUntrustedInputsAndBadArchives() {
	for _, scenario := range []string{"version", "platform", "source", "missing-agent", "duplicate-agent", "malformed-agent", "extra-field-agent", "bad-agent-checksum", "bad-checker-checksum", "wrong-member", "symlink-output", "newline-temp"} {
		suiteState.Run(scenario, func() {
			fixture := newSetupFixture(suiteState.T())
			manifest := filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256")
			data, err := os.ReadFile(manifest)
			suiteState.Require().NoError(err)
			var overrides []string
			switch scenario {
			case "version":
				overrides = []string{"MCPCHECKER_VERSION=latest"}
			case "platform":
				overrides = []string{"RUNNER_ARCH=ARM"}
			case "source":
				directory := filepath.Join(fixture.candidate, ".github/actions/mcpchecker-action")
				suiteState.Require().NoError(os.MkdirAll(directory, 0o755))
				overrides = []string{"GITHUB_ACTION_PATH=" + directory}
			case "missing-agent":
				var kept []string
				for _, line := range strings.Split(string(data), "\n") {
					if !strings.Contains(line, "agent-linux-amd64.zip") {
						kept = append(kept, line)
					}
				}
				writeActionFixture(suiteState.T(), manifest, strings.Join(kept, "\n"), 0o644)
			case "duplicate-agent":
				for _, line := range strings.Split(string(data), "\n") {
					if strings.Contains(line, "agent-linux-amd64.zip") {
						data = append(data, []byte(line+"\n")...)
						break
					}
				}
				writeActionFixture(suiteState.T(), manifest, string(data), 0o644)
			case "malformed-agent", "extra-field-agent":
				var rows []string
				for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
					if strings.Contains(line, "agent-linux-amd64.zip") {
						if scenario == "malformed-agent" {
							line = "not-a-hash  agent-linux-amd64.zip"
						} else {
							line += " unexpected-field"
						}
					}
					rows = append(rows, line)
				}
				writeManifest(suiteState.T(), manifest, rows...)
			case "bad-agent-checksum", "bad-checker-checksum":
				component := "agent"
				if scenario == "bad-checker-checksum" {
					component = "mcpchecker"
				}
				writeActionFixture(suiteState.T(), filepath.Join(fixture.archives, component+"-linux-amd64.zip"), "untrusted archive", 0o644)
			case "wrong-member":
				_, digest := createArchive(suiteState.T(), fixture.archives, "agent-linux-amd64.zip", "unexpected-member", "not executable")
				var rows []string
				for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
					if strings.Contains(line, "agent-linux-amd64.zip") {
						line = digest + "  agent-linux-amd64.zip"
					}
					rows = append(rows, line)
				}
				writeManifest(suiteState.T(), manifest, rows...)
			case "symlink-output":
				suiteState.Require().NoError(os.Remove(fixture.output))
				suiteState.Require().NoError(os.Symlink(fixture.pathOutput, fixture.output))
			case "newline-temp":
				directory := fixture.runnerTemp + "\ninjected=value"
				suiteState.Require().NoError(os.Mkdir(directory, 0o700))
				overrides = []string{"RUNNER_TEMP=" + directory}
			}
			output, err := fixture.run(suiteState.T(), overrides...)
			suiteState.Require().Error(err, output)
			if scenario == "malformed-agent" {
				suiteState.Contains(output, "invalid SHA-256 for agent-linux-amd64.zip")
			}
			if scenario == "extra-field-agent" {
				suiteState.Contains(output, "invalid checksum manifest entry for agent-linux-amd64.zip")
			}
			outputData, err := os.ReadFile(fixture.output)
			suiteState.Require().NoError(err)
			suiteState.Empty(outputData)
			pathData, err := os.ReadFile(fixture.pathOutput)
			suiteState.Require().NoError(err)
			suiteState.Empty(pathData)
			directories, err := filepath.Glob(filepath.Join(fixture.runnerTemp, "mcpchecker-action.*"))
			suiteState.Require().NoError(err)
			suiteState.Empty(directories)
			suiteState.NoFileExists(fixture.marker)
		})
	}
}

func (suiteState *MCPCheckerActionsSuite) TestInstallerComponentAllowlistAndLiteralPaths() {
	for _, pair := range [][2]string{{"latest", "mcpchecker"}, {"main", "agent"}, {"v0.0.23", "mcpchecker"}, {"v0.0.21", "agent"}, {"v0.0.22", "unknown"}} {
		suiteState.Run(pair[0]+"/"+pair[1], func() {
			fixture := newSetupFixture(suiteState.T())
			destination := filepath.Join(fixture.runnerTemp, "binary")
			output, err := runInstaller(suiteState.T(), fixture.environment, pair[0], "linux", "amd64", destination, filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256"), pair[1])
			suiteState.Error(err, output)
			suiteState.Contains(output, "unsupported version/component")
			suiteState.NoFileExists(destination)
			assertNoInstallerArtifacts(suiteState.T(), destination)
		})
	}
	suiteState.Run("agent cache tamper distinct destinations literal path and foreign lock", func() {
		fixture := newSetupFixture(suiteState.T())
		manifest := filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256")
		checker := filepath.Join(fixture.runnerTemp, "checker")
		agent := filepath.Join(fixture.runnerTemp, "agent ' ; $(not-a-command)\nspace")
		suiteState.Require().NoError(os.Mkdir(checker+".lock", 0o700))
		for _, attempt := range []string{"install", "reuse", "tamper", "stale-metadata"} {
			if attempt == "tamper" {
				writeActionFixture(suiteState.T(), agent, "tampered", 0o755)
			}
			if attempt == "stale-metadata" {
				writeActionFixture(suiteState.T(), agent+".metadata", "stale", 0o644)
			}
			environment := fixture.environment
			if attempt == "reuse" {
				environment = append(append([]string(nil), environment...), "FAKE_CURL_FAIL=1")
			}
			output, err := runInstaller(suiteState.T(), environment, "v0.0.22", "linux", "amd64", agent, manifest, "agent")
			suiteState.Require().NoError(err, output)
			archiveData, err := os.ReadFile(filepath.Join(fixture.archives, "agent-linux-amd64.zip"))
			suiteState.Require().NoError(err)
			assertInstalledPair(suiteState.T(), agent, fmt.Sprintf("#!/bin/sh\nprintf executed > %q\nexit 91\n", fixture.marker), "v0.0.22", "linux/amd64", fmt.Sprintf("%x", sha256.Sum256(archiveData)))
			assertNoInstallerArtifacts(suiteState.T(), agent)
		}
		suiteState.DirExists(checker + ".lock")
		suiteState.NoFileExists(fixture.marker)
	})
}

func (suiteState *MCPCheckerActionsSuite) TestInstallerExtractsOnlyUniqueExpectedMember() {
	for _, duplicate := range []bool{false, true} {
		suiteState.Run(fmt.Sprint("duplicate=", duplicate), func() {
			fixture := newSetupFixture(suiteState.T())
			archive := filepath.Join(fixture.archives, "agent-linux-amd64.zip")
			file, err := os.Create(archive)
			suiteState.Require().NoError(err)
			writer := zip.NewWriter(file)
			members := []string{"agent", "../outside-marker"}
			if duplicate {
				members = append(members, "agent")
			}
			for _, member := range members {
				entry, err := writer.Create(member)
				suiteState.Require().NoError(err)
				_, err = entry.Write([]byte("inert data"))
				suiteState.Require().NoError(err)
			}
			suiteState.Require().NoError(writer.Close())
			suiteState.Require().NoError(file.Close())
			data, err := os.ReadFile(archive)
			suiteState.Require().NoError(err)
			manifest := filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256")
			writeManifest(suiteState.T(), manifest, fmt.Sprintf("%x  agent-linux-amd64.zip", sha256.Sum256(data)))
			destination := filepath.Join(fixture.runnerTemp, "agent")
			realUnzip, err := exec.LookPath("unzip")
			suiteState.Require().NoError(err)
			bin := filepath.Join(fixture.runnerTemp, "bin")
			writeActionFixture(suiteState.T(), filepath.Join(bin, "unzip"), `#!/bin/sh
set -eu
if [ "$1" = "-q" ]; then
    [ "$#" -eq 5 ] && [ "$3" = "agent" ] && [ "$4" = "-d" ] || exit 92
fi
exec "$REAL_UNZIP" "$@"
`, 0o755)
			environment := append(append([]string(nil), fixture.environment...), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "REAL_UNZIP="+realUnzip)
			for _, value := range fixture.environment {
				if strings.HasPrefix(value, "PATH=") {
					environment = append(environment, "PATH="+bin+string(os.PathListSeparator)+strings.TrimPrefix(value, "PATH="))
				}
			}
			output, err := runInstaller(suiteState.T(), environment, "v0.0.22", "linux", "amd64", destination, manifest, "agent")
			if duplicate {
				suiteState.Error(err, output)
				suiteState.NoFileExists(destination)
			} else {
				suiteState.NoError(err, output)
				suiteState.FileExists(destination)
			}
			suiteState.NoFileExists(filepath.Join(fixture.runnerTemp, "outside-marker"))
			assertNoInstallerArtifacts(suiteState.T(), destination)
		})
	}
}

func TestMCPCheckerActions(testingT *testing.T) { suite.Run(testingT, new(MCPCheckerActionsSuite)) }

func (suiteState *MCPCheckerActionsSuite) TestAgentFailureCleanupAndModes() {
	for _, scenario := range []string{"timeout", "foreign-lock", "directory", "symlink-directory", "metadata-directory", "invalid-lock"} {
		suiteState.Run(scenario, func() {
			fixture := newSetupFixture(suiteState.T())
			destination := filepath.Join(fixture.runnerTemp, "agent")
			environment := append([]string(nil), fixture.environment...)
			switch scenario {
			case "timeout":
				environment = append(environment, "FAKE_CURL_TIMEOUT=1")
			case "foreign-lock":
				suiteState.Require().NoError(os.Mkdir(destination+".lock", 0o700))
			case "directory":
				suiteState.Require().NoError(os.Mkdir(destination, 0o700))
			case "symlink-directory":
				suiteState.Require().NoError(os.Symlink(fixture.candidate, destination))
			case "metadata-directory":
				suiteState.Require().NoError(os.Mkdir(destination+".metadata", 0o700))
			case "invalid-lock":
				environment = append(environment, "MCPCHECKER_LOCK_TIMEOUT_SECONDS=301")
			}
			output, err := runInstaller(suiteState.T(), environment, "v0.0.22", "linux", "amd64", destination, filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256"), "agent")
			suiteState.Require().Error(err, output)
			if scenario == "foreign-lock" {
				suiteState.DirExists(destination + ".lock")
			} else {
				assertNoInstallerArtifacts(suiteState.T(), destination)
			}
			suiteState.NoFileExists(fixture.marker)
		})
	}
	suiteState.Run("restrictive umask and fresh action directories", func() {
		fixture := newSetupFixture(suiteState.T())
		destination := filepath.Join(fixture.runnerTemp, "agent")
		output, err := runInstallerWithUmask(suiteState.T(), fixture.environment, "0077", "v0.0.22", "linux", "amd64", destination, filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256"), "agent")
		suiteState.Require().NoError(err, output)
		for _, expected := range []struct {
			path string
			mode os.FileMode
		}{{destination, 0o755}, {destination + ".metadata", 0o644}} {
			info, err := os.Stat(expected.path)
			suiteState.Require().NoError(err)
			suiteState.Equal(expected.mode, info.Mode().Perm())
		}
		for attempt := 0; attempt < 2; attempt++ {
			output, err := fixture.run(suiteState.T())
			suiteState.Require().NoError(err, output)
		}
		directories, err := filepath.Glob(filepath.Join(fixture.runnerTemp, "mcpchecker-action.*"))
		suiteState.Require().NoError(err)
		suiteState.Len(directories, 2)
		suiteState.NoFileExists(fixture.marker)
	})
}

func (suiteState *MCPCheckerActionsSuite) TestActualMakeConsumer() {
	fixture := newSetupFixture(suiteState.T())
	root := filepath.Join(suiteState.T().TempDir(), "make-root")
	for _, relative := range []string{"Makefile", "build/evals.mk", "build/install-mcpchecker.sh"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(installerPath(suiteState.T())), "..", relative))
		suiteState.Require().NoError(err)
		writeActionFixture(suiteState.T(), filepath.Join(root, relative), string(data), 0o755)
	}
	manifest, err := os.ReadFile(filepath.Join(fixture.root, "build/mcpchecker-v0.0.22.sha256"))
	suiteState.Require().NoError(err)
	for _, version := range []string{"v0.0.21", "v0.0.22"} {
		writeActionFixture(suiteState.T(), filepath.Join(root, "build/mcpchecker-"+version+".sha256"), string(manifest), 0o644)
	}
	var environment []string
	for _, value := range fixture.environment {
		if !strings.HasPrefix(value, "MCPCHECKER_VERSION=") {
			environment = append(environment, value)
		}
	}
	for _, version := range []string{"", "v0.0.21"} {
		arguments := []string{"mcpchecker", "MCPCHECKER_GO_OS=linux", "MCPCHECKER_GO_ARCH=amd64"}
		if version != "" {
			arguments = append(arguments, "MCPCHECKER_VERSION="+version)
		}
		command := exec.Command("make", arguments...)
		command.Dir, command.Env = root, environment
		output, err := command.CombinedOutput()
		if version == "" {
			suiteState.Require().NoError(err, string(output))
			suiteState.Contains(string(output), "mcpchecker v0.0.22")
			suiteState.Contains(string(output), "agent v0.0.22")
			for _, component := range []string{"mcpchecker", "agent"} {
				suiteState.FileExists(filepath.Join(root, "_output/tools/bin", component))
				assertNoInstallerArtifacts(suiteState.T(), filepath.Join(root, "_output/tools/bin", component))
			}
		} else {
			suiteState.Require().Error(err, string(output))
			suiteState.Contains(string(output), "unsupported version/component v0.0.21/agent")
		}
	}
	suiteState.NoFileExists(fixture.marker)
}
