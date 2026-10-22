package build_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type MCPCheckerWorkflowSuite struct{ suite.Suite }

func TestMCPCheckerWorkflow(testingT *testing.T) { suite.Run(testingT, new(MCPCheckerWorkflowSuite)) }

func workflowBuildPath(testingT *testing.T, name string) string {
	testingT.Helper()
	return filepath.Join(filepath.Dir(installerPath(testingT)), name)
}

func workflowControlJSON() string {
	return `{"version":1,"repository":"containers/kubernetes-mcp-server","candidate":"cccccccccccccccccccccccccccccccccccccccc","harness":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","baseline":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","source_repository":"fork/server","source_ref":"feature","pr_number":518,"is_pr":true,"review_id":44,"reviewer":"maintainer","suite":["core"],"run_id":22,"run_attempt":2,"trigger_run_id":99}`
}

const controlHarness = `
const helper = require(process.argv[1]);
const mode = process.argv[2], scenario = process.argv[3].replace(/:empty$/, '');
const C = 'c'.repeat(40), B = 'b'.repeat(40);
const outputs = {}, calls = [];
const repository = 'containers/kubernetes-mcp-server';
const pr = {number:518,state:'open',merge_commit_sha:'e'.repeat(40),base:{ref:'main',repo:{full_name:repository}},head:{sha:C,ref:'feature',repo:{id:71,full_name:'fork/server'}}};
let reviews = [{id:44,state:'COMMENTED',body:'/run-mcpchecker core',commit_id:C,user:{login:'maintainer'}}];
const trigger = {id:99,workflow_id:8,repository:{full_name:repository},event:'pull_request_review',status:'completed',conclusion:'success',head_branch:'feature',head_sha:C,head_repository:{full_name:'fork/server'},pull_requests:[{number:518,head:JSON.parse(JSON.stringify(pr.head))}]};
const artifact = {id:123,name:'mcpchecker-candidate-server',expired:false,size_in_bytes:2048,workflow_run:{id:22}};
const environment = {protection_rules:[{type:'required_reviewers',prevent_self_review:true,reviewers:[{type:'Team',reviewer:{id:4}}]}],deployment_branch_policy:{custom_branch_policies:true,protected_branches:false}};
const policies = {total_count:1,branch_policies:[{name:'main',type:'branch'}]};
const job = {name:'Build Candidate',run_attempt:2,conclusion:'success',steps:[{name:'Upload candidate binary',conclusion:'success'}]};
let permission = 'write', pulls = [pr], baseline = '[{"taskPassed":true}]';
if(scenario.startsWith('baseline-large'))baseline=require('node:fs').readFileSync(process.env.LARGE_BASELINE_FILE,'utf8')+(scenario==='baseline-large-newline'?'\n':'');
switch(scenario) {
case 'two-forks': pulls = [pr,{...pr,number:519,head:{ref:'feature',repo:{full_name:'other/fork'}}}]; break;
case 'other-base': pulls = [pr,{...pr,number:519,base:{...pr.base,ref:'release-1'}}]; break;
case 'ambiguous': pulls = [pr,{...pr,number:519}]; break;
case 'empty-trigger': trigger.pull_requests=[]; break;
case 'current-head-moved': trigger.pull_requests=[]; pulls=JSON.parse(JSON.stringify(pulls)); pr.head.sha='d'.repeat(40); reviews[0].commit_id=pr.head.sha; break;
case 'missing-head-repo': delete trigger.head_repository; break;
case 'wrong-head-repo': trigger.head_repository.full_name='other/fork'; break;
case 'wrong-trigger-sha': trigger.head_sha='f'.repeat(40); break;
case 'merge-as-trigger-sha': trigger.head_sha=pr.merge_commit_sha; break;
case 'unready-merge': pr.merge_commit_sha=null; break;
case 'closed-pr': pr.state='closed'; break;
case 'wrong-branch': trigger.head_branch='other'; break;
case 'no-pr': pulls = []; break;
case 'missing-review': reviews = []; break;
case 'dismissed': reviews[0].state='DISMISSED'; break;
case 'pending': reviews[0].state='PENDING'; break;
case 'stale': reviews[0].commit_id='d'.repeat(40); break;
case 'denied': permission='read'; break;
case 'command-injection': reviews[0].body='/run-mcpchecker core; touch SENTINEL'; break;
case 'command-prefix': reviews[0].body='prose /run-mcpchecker core'; break;
case 'command-duplicate': reviews[0].body='/run-mcpchecker core\n/run-mcpchecker helm'; break;
case 'multicluster-review': reviews[0].body='/run-mcpchecker kiali-multicluster'; break;
case 'latest-invalid': reviews.push({...reviews[0],id:45,body:'/run-mcpchecker bogus'}); break;
case 'wrong-trigger': trigger.workflow_id=9; break;
case 'wrong-event': trigger.event='push'; break;
case 'wrong-trigger-repo': trigger.repository.full_name='other/server'; break;
case 'wrong-pr': trigger.pull_requests=[{number:519}]; break;
case 'head-drift': pr.head.sha='d'.repeat(40); reviews[0].commit_id=pr.head.sha; break;
case 'fork-drift': pr.head.repo.full_name='other/fork'; break;
case 'review-drift': reviews[0].id=45; break;
case 'reviewer-drift': reviews[0].user.login='other'; break;
case 'suite-drift': reviews[0].body='/run-mcpchecker helm'; break;
case 'artifact-name': artifact.name='../../other'; break;
case 'artifact-run': artifact.workflow_run.id=23; break;
case 'artifact-expired': artifact.expired=true; break;
case 'artifact-size': artifact.size_in_bytes=268435457; break;
case 'job-failed': job.conclusion='failure'; break;
case 'job-attempt': job.run_attempt=1; break;
case 'baseline-corrupt': baseline='{bad'; break;
case 'baseline-empty': baseline='[]'; break;
case 'unprotected': environment.protection_rules=[]; break;
case 'self-approval': environment.protection_rules[0].prevent_self_review=false; break;
case 'no-reviewers': environment.protection_rules[0].reviewers=[]; break;
case 'branch-policy': policies.branch_policies[0].name='*'; break;
}
if(process.argv[3].endsWith(':empty'))trigger.pull_requests=[];
if(scenario==='baseline-network')baseline='network';
const list = function list(){}, listReviews=function listReviews(){}, listJobs=function listJobs(){};
const github = {rest:{pulls:{list,listReviews,get:async()=>({data:pr})},repos:{
 getCommit:async args=>{calls.push(['commit',args.ref]);return {data:{sha:args.ref==='main'?B:C}}},
 getCollaboratorPermissionLevel:async()=>({data:{permission}}),
 getEnvironment:async()=>({data:environment}),listDeploymentBranchPolicies:async()=>({data:policies}),
 getContent:async args=>{calls.push(['baseline',args.ref,args.path]);if(scenario==='baseline-absent')throw {status:404};if(scenario==='baseline-denied')throw {status:403};if(scenario==='baseline-network')throw Error('network unavailable');if(scenario==='baseline-media')return {data:'unexpected raw representation'}; if(args.mediaType.format!=='object'||args.ref!==B||args.path!=='evals/results/builtin-openai-core-latest.json')throw Error('identity');const bytes=Buffer.from(baseline);return {data:{type:scenario==='baseline-symlink'?'symlink':'file',path:scenario==='baseline-path'?'wrong/path':args.path,sha:scenario==='baseline-sha'?'invalid':'e'.repeat(40),encoding:bytes.length>1048576?'none':'base64',size:scenario==='baseline-size'?67108865:bytes.length,content:bytes.length>1048576?'':bytes.toString('base64')}}}
 },git:{getBlob:async args=>{if(args.file_sha!=='e'.repeat(40)||args.owner!=='containers'||args.repo!=='kubernetes-mcp-server')throw Error('blob identity');if(scenario==='blob-404')throw {status:404};if(scenario==='blob-network')throw Error('network unavailable');const bytes=scenario==='blob-oversize'?Buffer.alloc(67108865,120):Buffer.from(baseline);return {data:{sha:scenario==='blob-sha'?'f'.repeat(40):args.file_sha,size:scenario==='blob-size'?bytes.length+1:scenario==='blob-oversize'?Buffer.byteLength(baseline):bytes.length,encoding:scenario==='blob-encoding'?'none':'base64',content:scenario==='blob-base64'?'Zh==':scenario==='blob-media'?{}:bytes.toString('base64')}}}},actions:{getWorkflowRun:async()=>({data:trigger}),getWorkflow:async()=>({data:{id:8}}),getArtifact:async()=>({data:artifact}),listJobsForWorkflowRun:listJobs}},
 paginate:async (target,args)=>target===list?pulls.filter(pr=>(!args.base||pr.base?.ref===args.base)&&(!args.state||pr.state===args.state)):target===listReviews?reviews:[job]};
const context={repo:{owner:'containers',repo:'kubernetes-mcp-server'},runId:22,eventName:'workflow_run',ref:'refs/heads/main',payload:{workflow_run:{id:99}}};
if(scenario==='schedule')context.eventName='schedule';
if(scenario==='dispatch')context.eventName='workflow_dispatch';
if(scenario==='untrusted-dispatch'){context.eventName='workflow_dispatch';context.ref='refs/heads/feature'}
if(scenario==='wrong-run-binding')context.runId=23;
const core={setOutput:(key,value)=>outputs[key]=value,summary:{addHeading(){return this},addRaw(text){outputs.audit=text;return this},async write(){}}};
helper.run({github,context,core,mode}).then(result=>{if(mode==='baseline'&&outputs['has-diff']==='true'){const fs=require('node:fs'),file=outputs['base-results'],stat=fs.lstatSync(file);if(!stat.isFile()||stat.isSymbolicLink()||stat.uid!==process.getuid()||fs.readFileSync(file,'utf8')!==baseline)throw Error('baseline persistence');outputs['baseline-bytes']=stat.size}console.log(JSON.stringify({outputs,calls,result}))}).catch(()=>{console.log(JSON.stringify({outputs,calls,error:'rejected'}));process.exitCode=1});
`

func (suiteState *MCPCheckerWorkflowSuite) runControl(mode, scenario string, overrides ...string) (map[string]json.RawMessage, string, error) {
	command := exec.Command("node", "-e", controlHarness, workflowBuildPath(suiteState.T(), "mcpchecker-control.cjs"), mode, scenario)
	command.Env = append(os.Environ(), "HARNESS_SHA="+strings.Repeat("a", 40), "GITHUB_RUN_ATTEMPT=2", "CONTROL_JSON="+workflowControlJSON(), "INPUT_SUITE=core", "CANDIDATE_ARTIFACT_ID=123", "SUITE=core", "RUNNER_TEMP="+suiteState.T().TempDir(), "KUBECONFIG=/dev/null")
	command.Env = append(command.Env, overrides...)
	output, err := command.CombinedOutput()
	var decoded map[string]json.RawMessage
	suiteState.Require().NoError(json.Unmarshal(output, &decoded), string(output))
	return decoded, string(output), err
}

func (suiteState *MCPCheckerWorkflowSuite) TestControlCurrentAPIEligibilityAndRevalidation() {
	for _, scenario := range []string{"valid", "schedule", "dispatch"} {
		suiteState.Run(scenario, func() {
			decoded, output, err := suiteState.runControl("control", scenario)
			suiteState.Require().NoError(err, output)
			var outputs map[string]any
			suiteState.Require().NoError(json.Unmarshal(decoded["outputs"], &outputs))
			suiteState.Equal("true", outputs["should-run"])
			var metadata map[string]any
			suiteState.Require().NoError(json.Unmarshal([]byte(outputs["control-json"].(string)), &metadata))
			suiteState.Equal(strings.Repeat("a", 40), metadata["harness"])
			suiteState.Equal(strings.Repeat("b", 40), metadata["baseline"])
			suiteState.Equal(float64(22), metadata["run_id"])
			if scenario == "valid" {
				suiteState.Equal("fork/server", metadata["source_repository"])
				suiteState.Equal(strings.Repeat("c", 40), metadata["candidate"])
			} else {
				suiteState.Equal(strings.Repeat("b", 40), metadata["candidate"])
			}
			suiteState.Contains(outputs["matrix"], "mcpchecker-core-e2e-out.json")
		})
	}
	for _, scenario := range []string{"ambiguous", "no-pr", "missing-head-repo", "wrong-head-repo", "wrong-trigger-sha", "merge-as-trigger-sha", "closed-pr", "wrong-branch", "head-drift", "current-head-moved", "missing-review", "dismissed", "pending", "stale", "denied", "command-injection", "command-prefix", "command-duplicate", "latest-invalid", "wrong-trigger", "wrong-event", "wrong-trigger-repo", "wrong-pr", "untrusted-dispatch", "unprotected", "self-approval", "no-reviewers", "branch-policy"} {
		suiteState.Run(scenario, func() {
			_, output, err := suiteState.runControl("control", scenario)
			suiteState.Error(err, output)
			suiteState.Contains(output, `"should-run":"false"`)
		})
	}
	for _, scenario := range []string{"empty-trigger", "two-forks"} {
		suiteState.Run(scenario, func() {
			_, output, err := suiteState.runControl("control", scenario)
			suiteState.NoError(err, output)
		})
	}
	_, output, err := suiteState.runControl("revalidate", "valid")
	suiteState.NoError(err, output)
	for _, scenario := range []string{"head-drift", "fork-drift", "review-drift", "reviewer-drift", "suite-drift", "missing-review", "denied", "wrong-run-binding"} {
		suiteState.Run("revalidate-"+scenario, func() {
			_, output, err := suiteState.runControl("revalidate", scenario)
			suiteState.Error(err, output)
		})
	}
	for _, invalid := range []string{"HARNESS_SHA=", "HARNESS_SHA=main", "CONTROL_JSON=" + strings.Replace(workflowControlJSON(), `"run_id":22`, `"run_id":"22"`, 1), "CONTROL_JSON=" + strings.Replace(workflowControlJSON(), `"version":1`, `"version":1,"extra":true`, 1), "CONTROL_JSON=" + strings.Repeat(" ", 16385)} {
		suiteState.Run("schema-"+invalid[:min(len(invalid), 70)], func() {
			_, output, err := suiteState.runControl("revalidate", "valid", invalid)
			suiteState.Error(err, output)
		})
	}
}

func (suiteState *MCPCheckerWorkflowSuite) TestRESTCandidateIdentity() {
	for _, suffix := range []string{"", ":empty"} {
		for _, mode := range []string{"control", "revalidate"} {
			for _, scenario := range []string{"valid", "other-base", "unready-merge", "wrong-trigger-sha", "merge-as-trigger-sha", "ambiguous", "no-pr", "current-head-moved", "head-drift", "stale", "review-drift", "reviewer-drift", "suite-drift"} {
				if mode == "revalidate" && (scenario == "wrong-trigger-sha" || scenario == "merge-as-trigger-sha" || scenario == "ambiguous" || scenario == "no-pr") {
					continue
				}
				if mode == "control" && (scenario == "review-drift" || scenario == "reviewer-drift" || scenario == "suite-drift") {
					continue
				}
				suiteState.Run(mode+"/"+scenario+suffix, func() {
					_, output, err := suiteState.runControl(mode, scenario+suffix)
					if scenario == "valid" || scenario == "other-base" || scenario == "unready-merge" {
						suiteState.NoError(err, output)
						if mode == "control" {
							suiteState.Contains(output, `"should-run":"true"`)
						}
					} else {
						suiteState.Error(err, output)
						if mode == "control" {
							suiteState.Contains(output, `"should-run":"false"`)
						}
					}
				})
			}
		}
	}
}

func (suiteState *MCPCheckerWorkflowSuite) TestArtifactAPIAndBaselineData() {
	baseline, marshalErr := json.Marshal([]map[string]any{{"taskPassed": true, "taskOutput": strings.Repeat("x", 1100000)}})
	suiteState.Require().NoError(marshalErr)
	baselineFile := filepath.Join(suiteState.T().TempDir(), "large-baseline.json")
	suiteState.Require().NoError(os.WriteFile(baselineFile, baseline, 0o600))
	for _, scenario := range []string{"baseline-large", "baseline-large-newline"} {
		suiteState.Run(scenario, func() {
			expectedBytes := len(baseline)
			if scenario == "baseline-large-newline" {
				expectedBytes++
			}
			suiteState.Greater(expectedBytes, 1048576)
			decoded, largeOutput, largeErr := suiteState.runControl("baseline", scenario, "LARGE_BASELINE_FILE="+baselineFile)
			suiteState.NoError(largeErr, largeOutput)
			var outputs map[string]json.RawMessage
			suiteState.Require().NoError(json.Unmarshal(decoded["outputs"], &outputs))
			var persistedBytes int
			suiteState.Require().NoError(json.Unmarshal(outputs["baseline-bytes"], &persistedBytes))
			suiteState.Equal(expectedBytes, persistedBytes)
		})
	}
	_, controlOutput, controlErr := suiteState.runControl("control", "valid")
	suiteState.NoError(controlErr, controlOutput)
	suiteState.Contains(controlOutput, "REST environment response does not verify that setting")
	for _, mode := range []string{"artifact", "baseline"} {
		_, output, err := suiteState.runControl(mode, "valid")
		suiteState.NoError(err, output)
	}
	for _, scenario := range []string{"artifact-name", "artifact-run", "artifact-expired", "artifact-size", "job-failed", "job-attempt"} {
		suiteState.Run(scenario, func() {
			_, output, err := suiteState.runControl("artifact", scenario)
			suiteState.Error(err, output)
		})
	}
	for _, scenario := range []string{"baseline-network", "baseline-media", "baseline-path", "baseline-sha", "baseline-size", "blob-404", "blob-network", "blob-sha", "blob-size", "blob-oversize", "blob-encoding", "blob-base64", "blob-media", "baseline-absent", "baseline-corrupt", "baseline-empty", "baseline-denied", "baseline-symlink"} {
		suiteState.Run(scenario, func() {
			_, output, err := suiteState.runControl("baseline", scenario)
			if scenario == "baseline-absent" {
				suiteState.NoError(err, output)
				suiteState.Contains(output, `"has-diff":"false"`)
				suiteState.Contains(output, `baseline-absent`)
			} else {
				suiteState.Error(err, output)
			}
		})
	}
}

func (suiteState *MCPCheckerWorkflowSuite) TestIndependentLiteralSuiteMap() {
	expected := map[string][]string{
		"core":      {"eval-core.yaml", "core,config", "mcpchecker-core-e2e-out.json"},
		"helm":      {"eval-helm.yaml", "core,config,helm", "mcpchecker-helm-e2e-out.json"},
		"kubevirt":  {"eval-kubevirt.yaml", "core,config,kubevirt,tekton", "mcpchecker-kubevirt-e2e-out.json"},
		"kiali":     {"eval-kiali.yaml", "core,config,kiali", "mcpchecker-kiali-e2e-out.json"},
		"tekton":    {"eval-tekton.yaml", "core,config,tekton", "mcpchecker-tekton-e2e-out.json"},
		"netobserv": {"eval-netobserv.yaml", "core,config,netobserv", "mcpchecker-netobserv-e2e-out.json"},
		"argocd":    {"eval-argo.yaml", "core,config", "mcpchecker-argocd-e2e-out.json"},
		"all":       {"eval-all.yaml", "core,config,helm,kiali,kubevirt,tekton,netobserv", "mcpchecker-all-e2e-out.json"},
	}
	command := exec.Command("node", "-e", `const helper=require(process.argv[1]);console.log(JSON.stringify(Object.fromEntries(['core','helm','kubevirt','kiali','tekton','netobserv','argocd','all'].map(suite=>[suite,helper.suiteEntry(suite)]))))`, workflowBuildPath(suiteState.T(), "mcpchecker-control.cjs"))
	output, err := command.CombinedOutput()
	suiteState.Require().NoError(err, string(output))
	var actual map[string]map[string]string
	suiteState.Require().NoError(json.Unmarshal(output, &actual))
	suiteState.Require().Len(actual, 8)
	for suiteName, contract := range expected {
		suiteState.Equal("evals/core-eval-testing/builtin-openai/"+contract[0], actual[suiteName]["eval-config"])
		suiteState.Equal(contract[1], actual[suiteName]["toolsets"])
		suiteState.Equal(contract[2], actual[suiteName]["expected-results-file"])
	}
	_, text, err := suiteState.runControl("control", "dispatch", "INPUT_SUITE=core;touch SENTINEL")
	suiteState.Error(err, text)
}

func (suiteState *MCPCheckerWorkflowSuite) TestMulticlusterRemainsOperatorOnly() {
	_, output, err := suiteState.runControl("control", "multicluster-review")
	suiteState.Error(err, output)
	suiteState.Contains(output, `"should-run":"false"`)
	_, output, err = suiteState.runControl("control", "dispatch", "INPUT_SUITE=kiali-multicluster")
	suiteState.Error(err, output)
	suiteState.Contains(output, `"should-run":"false"`)
	metadata := strings.Replace(workflowControlJSON(), `"suite":["core"]`, `"suite":["kiali-multicluster"]`, 1)
	for _, mode := range []string{"revalidate", "artifact", "baseline"} {
		_, output, err = suiteState.runControl(mode, "valid", "CONTROL_JSON="+metadata, "SUITE=kiali-multicluster")
		suiteState.Error(err, mode+": "+output)
	}
	decoded, output, err := suiteState.runControl("control", "schedule")
	suiteState.Require().NoError(err, output)
	var outputs map[string]json.RawMessage
	suiteState.Require().NoError(json.Unmarshal(decoded["outputs"], &outputs))
	var encodedMatrix string
	suiteState.Require().NoError(json.Unmarshal(outputs["matrix"], &encodedMatrix))
	var matrix struct {
		Include []struct{ Suite string }
	}
	suiteState.Require().NoError(json.Unmarshal([]byte(encodedMatrix), &matrix))
	actual := make([]string, 0, len(matrix.Include))
	for _, entry := range matrix.Include {
		actual = append(actual, entry.Suite)
	}
	suiteState.ElementsMatch([]string{"core", "helm", "kubevirt", "kiali", "tekton", "netobserv", "argocd"}, actual)
	for _, name := range []string{"mcpchecker.yaml", "mcpchecker-report.yaml"} {
		data, readErr := os.ReadFile(workflowBuildPath(suiteState.T(), "../.github/workflows/"+name))
		suiteState.Require().NoError(readErr)
		suiteState.NotContains(strings.ToLower(string(data)), "multicluster", name)
	}
	data, err := os.ReadFile(workflowBuildPath(suiteState.T(), "kiali.mk"))
	suiteState.Require().NoError(err)
	for _, target := range []string{"write-kiali-multicluster-mcp-config:", "setup-kiali-multicluster:", "redeploy-kiali-multicluster-dev:", "kind-delete-multicluster:", "run-evals-multicluster: mcpchecker"} {
		suiteState.Contains(string(data), target)
	}
	for _, agent := range []string{"builtin-openai", "builtin-anthropic", "builtin-google", "acp-anthropic", "acp-google"} {
		data, err = os.ReadFile(workflowBuildPath(suiteState.T(), "../evals/core-eval-testing/"+agent+"/eval-kiali-multicluster.yaml"))
		suiteState.Require().NoError(err, agent)
		suiteState.Contains(string(data), "multicluster", agent)
	}
	for _, task := range []string{"mesh-status-multicluster", "metrics-west-reviews", "resource-list-clusters", "resource-remote-workload", "topology-west-cluster-graph", "traces-west-reviews"} {
		_, err = os.Stat(workflowBuildPath(suiteState.T(), "../evals/tasks/kiali/multicluster/"+task+"/"+task+".yaml"))
		suiteState.NoError(err, task)
	}
}

func fixtureELF() []byte { return append([]byte{0x7f, 'E', 'L', 'F'}, make([]byte, 60)...) }

func candidateEnvironment(root string) []string {
	return append(os.Environ(), "KUBECONFIG=/dev/null", "CANDIDATE_SHA="+strings.Repeat("c", 40), "SOURCE_REPOSITORY=fork/server", "GITHUB_RUN_ID=22", "GITHUB_RUN_ATTEMPT=2", "ARTIFACT_DIRECTORY="+filepath.Join(root, "artifact"), "TRUSTED_DIRECTORY="+filepath.Join(root, "trusted"))
}

func writeCandidateMetadata(testingT *testing.T, directory string, binary []byte) {
	testingT.Helper()
	metadata := fmt.Sprintf(`{"version":1,"candidate":"%s","repository":"fork/server","sha256":"%x","size":%d,"run_id":22,"run_attempt":2,"job":"candidate-build"}`, strings.Repeat("c", 40), sha256.Sum256(binary), len(binary))
	writeActionFixture(testingT, filepath.Join(directory, "metadata.json"), metadata, 0o600)
}

func (suiteState *MCPCheckerWorkflowSuite) TestCandidateBuildFixedRecipeAndArtifactConsumer() {
	for _, scenario := range []string{"valid", "extra-script", "symlink", "metadata-symlink", "source-mismatch", "digest-mismatch", "run-mismatch", "oversize", "script", "traversal"} {
		suiteState.Run(scenario, func() {
			root := suiteState.T().TempDir()
			directory := filepath.Join(root, "artifact")
			binary := fixtureELF()
			writeActionFixture(suiteState.T(), filepath.Join(directory, "kubernetes-mcp-server"), string(binary), 0o600)
			writeCandidateMetadata(suiteState.T(), directory, binary)
			suiteState.Require().NoError(os.Mkdir(filepath.Join(root, "trusted"), 0o700))
			switch scenario {
			case "extra-script":
				writeActionFixture(suiteState.T(), filepath.Join(directory, "extra.sh"), "exit 0", 0o600)
			case "symlink":
				suiteState.Require().NoError(os.Rename(filepath.Join(directory, "kubernetes-mcp-server"), filepath.Join(root, "other")))
				suiteState.Require().NoError(os.Symlink(filepath.Join(root, "other"), filepath.Join(directory, "kubernetes-mcp-server")))
			case "metadata-symlink":
				suiteState.Require().NoError(os.Rename(filepath.Join(directory, "metadata.json"), filepath.Join(root, "other")))
				suiteState.Require().NoError(os.Symlink(filepath.Join(root, "other"), filepath.Join(directory, "metadata.json")))
			case "source-mismatch", "digest-mismatch", "run-mismatch":
				metadata, err := os.ReadFile(filepath.Join(directory, "metadata.json"))
				suiteState.Require().NoError(err)
				old, replacement := strings.Repeat("c", 40), strings.Repeat("d", 40)
				switch scenario {
				case "digest-mismatch":
					old, replacement = fmt.Sprintf("%x", sha256.Sum256(binary)), strings.Repeat("0", 64)
				case "run-mismatch":
					old, replacement = `"run_id":22`, `"run_id":23`
				}
				writeActionFixture(suiteState.T(), filepath.Join(directory, "metadata.json"), strings.Replace(string(metadata), old, replacement, 1), 0o600)
			case "oversize":
				suiteState.Require().NoError(os.Truncate(filepath.Join(directory, "kubernetes-mcp-server"), 268435457))
				file, openErr := os.Open(filepath.Join(directory, "kubernetes-mcp-server"))
				suiteState.Require().NoError(openErr)
				digest := sha256.New()
				size, copyErr := io.Copy(digest, file)
				suiteState.Require().NoError(copyErr)
				suiteState.Require().NoError(file.Close())
				metadata := fmt.Sprintf(`{"version":1,"candidate":"%s","repository":"fork/server","sha256":"%x","size":%d,"run_id":22,"run_attempt":2,"job":"candidate-build"}`, strings.Repeat("c", 40), digest.Sum(nil), size)
				writeActionFixture(suiteState.T(), filepath.Join(directory, "metadata.json"), metadata, 0o600)
			case "script":
				binary = []byte(strings.Repeat("#!/bin/sh\n", 10))
				writeActionFixture(suiteState.T(), filepath.Join(directory, "kubernetes-mcp-server"), string(binary), 0o600)
				writeCandidateMetadata(suiteState.T(), directory, binary)
			}
			command := exec.Command("bash", workflowBuildPath(suiteState.T(), "mcpchecker-candidate.sh"), "verify")
			command.Env = candidateEnvironment(root)
			if scenario == "traversal" {
				command.Env = append(command.Env, "ARTIFACT_DIRECTORY="+directory+"/../artifact")
			}
			output, err := command.CombinedOutput()
			if scenario == "valid" {
				suiteState.Require().NoError(err, string(output))
				installed, readErr := os.ReadFile(filepath.Join(root, "trusted/kubernetes-mcp-server"))
				suiteState.NoError(readErr)
				suiteState.Equal(binary, installed)
				info, statErr := os.Stat(filepath.Join(root, "trusted/kubernetes-mcp-server"))
				suiteState.Require().NoError(statErr)
				suiteState.Equal(os.FileMode(0o755), info.Mode().Perm())
			} else {
				suiteState.Error(err, string(output))
				suiteState.NoFileExists(filepath.Join(root, "trusted/kubernetes-mcp-server"))
			}
		})
	}
	root := suiteState.T().TempDir()
	candidate := filepath.Join(root, "candidate")
	suiteState.Require().NoError(os.Mkdir(candidate, 0o700))
	writeActionFixture(suiteState.T(), filepath.Join(candidate, "Makefile"), "$(error CANDIDATE_MAKE_EXECUTED)\n", 0o600)
	writeActionFixture(suiteState.T(), filepath.Join(root, "fixture-elf"), string(fixtureELF()), 0o600)
	bin := filepath.Join(root, "bin")
	writeActionFixture(suiteState.T(), filepath.Join(bin, "git"), "#!/bin/sh\nprintf '%s\\n' '"+strings.Repeat("c", 40)+"'\n", 0o755)
	writeActionFixture(suiteState.T(), filepath.Join(bin, "go"), `#!/bin/bash
set -eu
[[ $1 == build && $2 == -ldflags && $4 == -o && $6 == ./cmd/kubernetes-mcp-server && $# == 6 ]]
[[ -z ${MODEL_KEY:-} && -z ${JUDGE_API_KEY:-} && $GOENV == off && $GOWORK == off && -z $GOFLAGS ]]
printf '%s\0' "$@" > "$FIXTURE_ROOT/argv"
cp "$FIXTURE_ROOT/fixture-elf" "$5"
`, 0o755)
	command := exec.Command("bash", workflowBuildPath(suiteState.T(), "mcpchecker-candidate.sh"), "build")
	command.Env = append(candidateEnvironment(root), "CANDIDATE_DIRECTORY="+candidate, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FIXTURE_ROOT="+root, "MODEL_KEY=", "JUDGE_API_KEY=")
	output, err := command.CombinedOutput()
	suiteState.Require().NoError(err, string(output))
	suiteState.FileExists(filepath.Join(root, "artifact/metadata.json"))
	suiteState.FileExists(filepath.Join(root, "artifact/kubernetes-mcp-server"))
	arguments, err := os.ReadFile(filepath.Join(root, "argv"))
	suiteState.Require().NoError(err)
	suiteState.Contains(string(arguments), "-X github.com/containers/kubernetes-mcp-server/pkg/version.CommitHash="+strings.Repeat("c", 40))
}

func (suiteState *MCPCheckerWorkflowSuite) TestFinalGateAndTypedContext() {
	for _, scenario := range []string{"pass", "optional-absent", "eval-failure", "eval-skipped", "setup-missing", "check-failure", "summary-failure", "verify-failure", "diff-failure", "baseline-corrupt", "server-failure", "revalidation-failure", "missing-metrics", "injected-metrics", "wrong-run"} {
		suiteState.Run(scenario, func() {
			root := suiteState.T().TempDir()
			checker, agent, result := filepath.Join(root, "checker"), filepath.Join(root, "agent"), filepath.Join(root, "mcpchecker-core-e2e-out.json")
			for _, executable := range []string{checker, agent} {
				writeActionFixture(suiteState.T(), executable, "fixture", 0o755)
			}
			writeActionFixture(suiteState.T(), result, `[{"taskPassed":true}]`, 0o600)
			environment := append(os.Environ(), "CONTROL_JSON="+workflowControlJSON(), "HARNESS_SHA="+strings.Repeat("a", 40), "GITHUB_RUN_ID=22", "GITHUB_RUN_ATTEMPT=2", "SUITE=core", "CONTEXT_DIRECTORY="+filepath.Join(root, "context"), "EVAL_OUTCOME=success", "EVAL_PASSED=true", "CHECK_EXIT=0", "SUMMARY_EXIT=0", "VERIFY_EXIT=0", "SETUP_VERSION=v0.0.22", "SETUP_CHECKER="+checker, "SETUP_AGENT="+agent, "RESULTS_FILE="+result, "SERVER_OUTCOME=success", "REVALIDATE_OUTCOME=success", "BASELINE_OUTCOME=success", "HAS_DIFF=true", "DIFF_OUTCOME=success", "DIFF_EXIT=0", "DIFF_REASON=baseline-present", "TASKS_TOTAL=5", "TASKS_PASSED=4", "TASK_PASS_RATE=0.8", "ASSERTIONS_TOTAL=5", "ASSERTIONS_PASSED=4", "ASSERTION_PASS_RATE=0.8", "KUBECONFIG=/dev/null")
			overrides := map[string][]string{
				"optional-absent": {"HAS_DIFF=false", "DIFF_OUTCOME=skipped", "DIFF_EXIT=not-run", "DIFF_REASON=baseline-absent"}, "eval-failure": {"EVAL_PASSED=false"}, "eval-skipped": {"EVAL_OUTCOME=skipped"}, "setup-missing": {"SETUP_AGENT="}, "check-failure": {"CHECK_EXIT=7"}, "summary-failure": {"SUMMARY_EXIT=9"}, "verify-failure": {"VERIFY_EXIT=1"}, "diff-failure": {"DIFF_EXIT=42", "DIFF_OUTCOME=failure"}, "baseline-corrupt": {"BASELINE_OUTCOME=failure", "HAS_DIFF=", "DIFF_REASON="}, "server-failure": {"SERVER_OUTCOME=failure"}, "revalidation-failure": {"REVALIDATE_OUTCOME=failure"}, "missing-metrics": {"TASKS_TOTAL=", "TASKS_PASSED=", "TASK_PASS_RATE=", "ASSERTIONS_TOTAL=", "ASSERTIONS_PASSED=", "ASSERTION_PASS_RATE="}, "injected-metrics": {"TASKS_TOTAL=5\npassed=true"}, "wrong-run": {"GITHUB_RUN_ID=23"},
			}
			command := exec.Command("bash", workflowBuildPath(suiteState.T(), "mcpchecker-workflow-status.sh"))
			command.Env = append(environment, overrides[scenario]...)
			output, err := command.CombinedOutput()
			passed := scenario == "pass" || scenario == "optional-absent"
			if passed {
				suiteState.NoError(err, string(output))
			} else {
				suiteState.Error(err, string(output))
			}
			if scenario == "wrong-run" {
				suiteState.NoFileExists(filepath.Join(root, "context/context.json"))
				return
			}
			data, readErr := os.ReadFile(filepath.Join(root, "context/context.json"))
			suiteState.Require().NoError(readErr)
			var context map[string]any
			suiteState.Require().NoError(json.Unmarshal(data, &context))
			suiteState.Equal(passed, context["passed"])
			suiteState.Equal(float64(22), context["run_id"])
			suiteState.Equal(strings.Repeat("c", 40), context["candidate"])
			suiteState.Equal("core", context["suite"])
			if scenario == "diff-failure" {
				suiteState.Equal(float64(42), context["diff_exit"])
			}
		})
	}
}

func workflowServerRootReady(serverRootPath, trusted string) bool {
	serverRoot, err := os.ReadFile(serverRootPath)
	return err == nil && string(serverRoot) == trusted+"\n"
}

func workflowServerPIDReady(serverPIDPath string) bool {
	data, err := os.ReadFile(serverPIDPath)
	if err != nil || !strings.HasSuffix(string(data), "\n") {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSuffix(string(data), "\n"))
	return err == nil && pid > 0
}

func (suiteState *MCPCheckerWorkflowSuite) TestServerPIDReadiness() {
	serverPIDPath := filepath.Join(suiteState.T().TempDir(), "server-pid")
	suiteState.False(workflowServerPIDReady(serverPIDPath))
	for _, fixture := range []struct {
		name, content string
		ready         bool
	}{
		{name: "empty"},
		{name: "partial", content: "12"},
		{name: "malformed", content: "not-a-pid\n"},
		{name: "zero", content: "0\n"},
		{name: "negative", content: "-1\n"},
		{name: "multiple", content: "12\n34\n"},
		{name: "complete", content: "1234\n", ready: true},
	} {
		suiteState.Run(fixture.name, func() {
			suiteState.Require().NoError(os.WriteFile(serverPIDPath, []byte(fixture.content), 0o600))
			suiteState.Equal(fixture.ready, workflowServerPIDReady(serverPIDPath))
		})
	}
}

func (suiteState *MCPCheckerWorkflowSuite) TestServerRootReadiness() {
	root := suiteState.T().TempDir()
	serverRootPath, trusted := filepath.Join(root, "server-root"), filepath.Join(root, "trusted")
	suiteState.False(workflowServerRootReady(serverRootPath, trusted))
	for _, fixture := range []struct {
		name, content string
		ready         bool
	}{
		{name: "empty"},
		{name: "partial", content: trusted[:len(trusted)-1]},
		{name: "missing-newline", content: trusted},
		{name: "complete", content: trusted + "\n", ready: true},
	} {
		suiteState.Run(fixture.name, func() {
			suiteState.Require().NoError(os.WriteFile(serverRootPath, []byte(fixture.content), 0o600))
			suiteState.Equal(fixture.ready, workflowServerRootReady(serverRootPath, trusted))
		})
	}
}

func (suiteState *MCPCheckerWorkflowSuite) TestActualTrustedMakeServerConsumer() {
	if runtime.GOOS != "linux" {
		suiteState.T().Skip("fixture ownership requires the workflow's Linux child-subreaper support")
	}
	for _, scenario := range []string{"normal", "early-failure", "timeout"} {
		suiteState.Run(scenario, func() {
			testingT := suiteState.T()
			cleanupAssertions := assert.New(testingT)
			root := suiteState.T().TempDir()
			trusted, candidate := filepath.Join(root, "trusted"), filepath.Join(root, "candidate")
			diagnostics := filepath.Join(root, "diagnostics")
			suiteState.Require().NoError(os.Mkdir(diagnostics, 0o700))
			makefile, err := os.ReadFile(workflowBuildPath(suiteState.T(), "../Makefile"))
			suiteState.Require().NoError(err)
			writeActionFixture(suiteState.T(), filepath.Join(trusted, "Makefile"), string(makefile), 0o600)
			evals, err := os.ReadFile(workflowBuildPath(suiteState.T(), "evals.mk"))
			suiteState.Require().NoError(err)
			writeActionFixture(suiteState.T(), filepath.Join(trusted, "build/evals.mk"), string(evals), 0o600)
			writeActionFixture(suiteState.T(), filepath.Join(trusted, "build/assert-fixture.mk"), "build:\n\t@echo BUILD_MUST_NOT_RUN; exit 97\n", 0o600)
			writeActionFixture(suiteState.T(), filepath.Join(candidate, "Makefile"), "$(error CANDIDATE_MAKE_EXECUTED)\n", 0o600)
			writeActionFixture(suiteState.T(), filepath.Join(candidate, "build/evil.mk"), "$(error CANDIDATE_INCLUDE_EXECUTED)\n", 0o600)
			writeActionFixture(suiteState.T(), filepath.Join(trusted, "dev/config/mcp-configs/fixture.toml"), "fixture = true\n", 0o600)
			bin := filepath.Join(root, "bin")
			writeActionFixture(suiteState.T(), filepath.Join(bin, "go"), "#!/bin/sh\nexit 0\n", 0o755)
			writeActionFixture(suiteState.T(), filepath.Join(bin, "git"), "#!/bin/sh\nexit 0\n", 0o755)
			writeActionFixture(suiteState.T(), filepath.Join(bin, "curl"), "#!/bin/sh\nexit 0\n", 0o755)
			writeActionFixture(suiteState.T(), filepath.Join(bin, "kubectl"), "#!/bin/sh\nexit 99\n", 0o755)
			server := `#!/bin/bash
set -eu
[[ -z ${MODEL_KEY:-} && -z ${JUDGE_API_KEY:-} ]]
printf '%s\n' "$PWD" > "$FIXTURE_ROOT/server-root"
printf '%s\0' "$@" > "$FIXTURE_ROOT/server-argv"
trap 'printf stopped > "$FIXTURE_ROOT/server-stopped"; exit 0' TERM
while :; do sleep 0.1; done
`
			writeActionFixture(suiteState.T(), filepath.Join(trusted, "kubernetes-mcp-server"), server, 0o755)
			environment := append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "FIXTURE_ROOT="+root, "KUBECONFIG=/dev/null", "MCP_EVAL_KUBECONFIG=/dev/null", "MODEL_KEY=", "JUDGE_API_KEY=")
			for _, name := range []string{"mcpchecker-capture.py", "mcpchecker-diagnostics.cjs"} {
				data, readErr := os.ReadFile(workflowBuildPath(suiteState.T(), name))
				suiteState.Require().NoError(readErr)
				if name == "mcpchecker-capture.py" {
					suiteState.Require().Contains(string(data), "time.monotonic() + 1800")
					suiteState.Require().Contains(string(data), "grace = 30")
					data = []byte(strings.Replace(strings.Replace(string(data), "time.monotonic() + 1800", "time.monotonic() + 3", 1), "grace = 30", "grace = 0.2", 1))
				}
				writeActionFixture(suiteState.T(), filepath.Join(root, name), string(data), 0o600)
			}
			arguments := []string{"make", "-C", trusted, "-o", "build", "run-server", "TOOLSETS=core,config", "MCP_HEALTH_TIMEOUT=2"}
			if scenario == "early-failure" {
				arguments = append([]string{"bash", "-c", `"$@"; result=$?; [[ $result != 0 ]] || result=47; exit "$result"`, "fixture"}, arguments...)
			}
			arguments = append([]string{filepath.Join(root, "mcpchecker-capture.py"), diagnostics, "check.txt", ""}, arguments...)
			command := exec.Command("python3", arguments...)
			command.Dir, command.Env = candidate, environment
			mockProbe := exec.Command("bash", "-c", `[[ $(command -v kubectl) == "$FIXTURE_ROOT/bin/kubectl" && $KUBECONFIG == /dev/null ]]`)
			mockProbe.Env = environment
			probeOutput, probeErr := mockProbe.CombinedOutput()
			suiteState.Require().NoError(probeErr, string(probeOutput))
			log, err := os.Create(filepath.Join(root, "owner-status"))
			suiteState.Require().NoError(err)
			command.Stdout, command.Stderr = log, log
			suiteState.Require().NoError(command.Start())
			done := make(chan struct{})
			var ownerErr error
			go func() {
				ownerErr = command.Wait()
				close(done)
			}()
			testingT.Cleanup(func() {
				pidData, pidErr := os.ReadFile(filepath.Join(trusted, ".mcp-server.pid"))
				pid, parseErr := strconv.Atoi(strings.TrimSpace(string(pidData)))
				cleanupAssertions.NoError(pidErr)
				cleanupAssertions.NoError(parseErr)
				if scenario != "timeout" {
					stop := exec.Command("make", "-C", trusted, "stop-server")
					stop.Dir, stop.Env = candidate, environment
					output, stopErr := stop.CombinedOutput()
					cleanupAssertions.NoError(stopErr, string(output))
					cleanupAssertions.NoFileExists(filepath.Join(trusted, ".mcp-server.pid"))
				}
				select {
				case <-done:
				case <-time.After(20 * time.Second):
					cleanupAssertions.Fail("bounded fixture capture owner did not finish")
					return
				}
				cleanupAssertions.NoError(log.Close())
				cleanupAssertions.NoError(ownerErr)
				status, statusErr := os.ReadFile(filepath.Join(root, "owner-status"))
				cleanupAssertions.NoError(statusErr)
				expected := map[string]string{"normal": "0 0 0\n", "early-failure": "47 0 0\n", "timeout": "0 0 1\n"}
				cleanupAssertions.Equal(expected[scenario], string(status))
				data, readErr := os.ReadFile(filepath.Join(diagnostics, "check.txt"))
				cleanupAssertions.NoError(readErr)
				cleanupAssertions.NotContains(string(data), "BUILD_MUST_NOT_RUN")
				cleanupAssertions.FileExists(filepath.Join(root, "server-stopped"))
				if pidErr == nil && parseErr == nil && pid > 0 {
					_, statErr := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
					cleanupAssertions.True(os.IsNotExist(statErr), "owned server survives live or zombie: %d", pid)
				}
				_, statErr := os.Stat(filepath.Join("/proc", strconv.Itoa(command.Process.Pid)))
				cleanupAssertions.True(os.IsNotExist(statErr), "owned supervisor survives live or zombie: %d", command.Process.Pid)
				testingT.Logf("owned cleanup scenario=%s server=%d supervisor=%d absent=%t status=%q", scenario, pid, command.Process.Pid, os.IsNotExist(statErr), status)
			})
			suiteState.Require().Eventually(func() bool { return workflowServerPIDReady(filepath.Join(trusted, ".mcp-server.pid")) }, 2*time.Second, 10*time.Millisecond)
			pidData, err := os.ReadFile(filepath.Join(trusted, ".mcp-server.pid"))
			suiteState.Require().NoError(err)
			pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
			suiteState.Require().NoError(err)
			_, statErr := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
			suiteState.Require().NoError(statErr)
			suiteState.T().Logf("owned before cleanup scenario=%s server=%d present=true", scenario, pid)
			suiteState.Eventually(func() bool { return workflowServerRootReady(filepath.Join(root, "server-root"), trusted) }, 2*time.Second, 10*time.Millisecond)
			serverRoot, err := os.ReadFile(filepath.Join(root, "server-root"))
			suiteState.Require().NoError(err)
			suiteState.Equal(trusted+"\n", string(serverRoot))
			configuration, err := os.ReadFile(filepath.Join(trusted, "_output/eval-config.d/zz-eval-overrides.toml"))
			suiteState.Require().NoError(err)
			suiteState.Contains(string(configuration), `toolsets = ["core", "config"]`)
		})
	}
}

func (suiteState *MCPCheckerWorkflowSuite) TestWorkflowTrustStructureAndLiteralHarness() {
	data, err := os.ReadFile(workflowBuildPath(suiteState.T(), "../.github/workflows/mcpchecker.yaml"))
	suiteState.Require().NoError(err)
	var workflow struct {
		Env  map[string]string
		Jobs map[string]struct {
			Name        string
			Needs       any
			If          string
			Environment map[string]string
			Permissions map[string]string
			Steps       []struct {
				Name, ID, Uses, Run, If string
				Env                     map[string]string
				With                    map[string]any
			}
		}
	}
	suiteState.Require().NoError(yaml.Unmarshal(data, &workflow))
	const reviewedHarness = "aa2cde6a978111cd162b79082084175bf5b0e820"
	suiteState.Equal(reviewedHarness, workflow.Env["HARNESS_SHA"])
	gate := workflow.Jobs["check-trigger"].Steps[0].Run
	for _, pin := range []string{"", "__REVIEWED_H_SHA__", "main", strings.Repeat("a", 39), strings.Repeat("g", 40), workflow.Env["HARNESS_SHA"]} {
		command := exec.Command("bash", "-c", gate)
		command.Env = append(os.Environ(), "HARNESS_SHA="+pin)
		output, err := command.CombinedOutput()
		if pin == reviewedHarness {
			suiteState.NoError(err, string(output))
		} else {
			suiteState.Error(err, string(output))
		}
	}
	suiteState.Contains(workflow.Jobs["check-trigger"].If, "github.repository == 'containers/kubernetes-mcp-server'")
	suiteState.Equal("write", workflow.Jobs["check-trigger"].Permissions["pull-requests"])
	suiteState.Equal(map[string]string{"contents": "read"}, workflow.Jobs["candidate-build"].Permissions)
	evaluation := workflow.Jobs["run-evaluation"]
	suiteState.Equal("mcpchecker-reviewed", evaluation.Environment["name"])
	suiteState.Contains(evaluation.Name, "Run MCP Evaluation (")
	allowed := map[string]bool{"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1": true, "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e": true, "actions/upload-artifact@cf430e030ddbb5b0abf93d22962f4752f3646cd9": true, "actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333": true, "actions/github-script@3a2844b7e9c422d3c10d287c895573f7108da1b3": true, "./trusted/.github/actions/mcpchecker-action": true, "./trusted/.github/actions/mcpchecker-diff-action": true}
	for jobName, job := range workflow.Jobs {
		for _, step := range job.Steps {
			if step.Uses != "" {
				suiteState.True(allowed[step.Uses], step.Uses)
			}
			for key, value := range step.Env {
				if strings.Contains(value, "secrets.MODEL") || strings.Contains(value, "secrets.JUDGE") {
					suiteState.Equal("run-evaluation", jobName)
					suiteState.Equal("mcpchecker", step.ID, key)
				}
			}
			if jobName != "commit-results" {
				suiteState.NotContains(step.Run, "${{")
				suiteState.NotContains(step.Run, "ls -t")
				if step.Run != "" {
					for _, tool := range []string{"bash", "shellcheck"} {
						arguments := []string{"-n"}
						if tool == "shellcheck" {
							arguments = []string{"--shell=bash", "-"}
						}
						command := exec.Command(tool, arguments...)
						command.Stdin = strings.NewReader(step.Run)
						output, err := command.CombinedOutput()
						suiteState.NoError(err, tool+" "+step.Name+": "+string(output))
					}
				}
				suiteState.NotContains(step.Uses, "cache")
				if strings.HasPrefix(step.Uses, "actions/checkout@") {
					suiteState.Equal(false, step.With["persist-credentials"])
				}
			}
			if jobName == "run-evaluation" && strings.HasPrefix(step.Uses, "actions/checkout@") {
				suiteState.Equal("redhat-chai-bot/openshift_openshift-mcp-server", step.With["repository"])
				suiteState.Equal("trusted", step.With["path"])
				suiteState.Equal(reviewedHarness, step.With["ref"])
			}
			if step.ID == "mcpchecker" {
				suiteState.Equal("0.8", step.With["task-pass-threshold"])
				suiteState.Equal("0.8", step.With["assertion-pass-threshold"])
				suiteState.Equal("false", step.With["fail-on-error"])
				suiteState.Equal("${{ matrix.expected-results-file }}", step.With["expected-results-file"])
			}
			if step.ID == "final-gate" {
				suiteState.Equal("always()", step.If)
			}
		}
	}
	suiteState.NotContains(string(data), "pull_request_target")
	suiteState.NotContains(string(data), "__REVIEWED_H_SHA__")
	checkoutPaths := map[string]string{"check-trigger": "control-trusted", "candidate-build": "trusted-build", "run-evaluation": "trusted"}
	for _, jobName := range []string{"check-trigger", "candidate-build", "run-evaluation"} {
		trustedCheckouts := 0
		for _, step := range workflow.Jobs[jobName].Steps {
			if step.With["path"] == checkoutPaths[jobName] {
				trustedCheckouts++
				suiteState.Equal("actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", step.Uses)
				suiteState.Equal("redhat-chai-bot/openshift_openshift-mcp-server", step.With["repository"])
				suiteState.Equal(reviewedHarness, step.With["ref"])
				suiteState.Equal(false, step.With["persist-credentials"])
			}
			if step.With["path"] == "candidate" {
				suiteState.Equal("candidate-build", jobName)
				suiteState.Equal("${{ needs.check-trigger.outputs.source-repository }}", step.With["repository"])
				suiteState.Equal("${{ needs.check-trigger.outputs.pr-sha }}", step.With["ref"])
			}
		}
		suiteState.Equal(1, trustedCheckouts, jobName)
	}
	for _, mode := range []string{"control", "revalidate", "artifact", "baseline"} {
		decoded, output, runErr := suiteState.runControl(mode, "valid", "HARNESS_SHA="+workflow.Env["HARNESS_SHA"], "CONTROL_JSON="+strings.ReplaceAll(workflowControlJSON(), strings.Repeat("a", 40), reviewedHarness))
		suiteState.Require().NoError(runErr, mode+": "+output)
		if mode == "control" {
			var outputs map[string]any
			suiteState.Require().NoError(json.Unmarshal(decoded["outputs"], &outputs))
			var metadata map[string]any
			suiteState.Require().NoError(json.Unmarshal([]byte(outputs["control-json"].(string)), &metadata))
			suiteState.Equal(reviewedHarness, metadata["harness"])
			suiteState.Equal(strings.Repeat("c", 40), metadata["candidate"])
			suiteState.NotEqual(metadata["harness"], metadata["candidate"])
		}
	}
	module, err := os.ReadFile(workflowBuildPath(suiteState.T(), "../go.mod"))
	suiteState.Require().NoError(err)
	suiteState.Contains(string(module), "\ngo 1.26.4\n")
	for _, step := range evaluation.Steps {
		if strings.HasPrefix(step.Uses, "actions/setup-go@") {
			suiteState.Equal("1.26.4", step.With["go-version"])
			suiteState.Equal(false, step.With["cache"])
		}
	}
	composite, err := os.ReadFile(workflowBuildPath(suiteState.T(), "../.github/actions/mcpchecker-action/action.yaml"))
	suiteState.Require().NoError(err)
	var action struct {
		Runs struct {
			Steps []struct {
				ID  string
				Env map[string]string
			}
		}
	}
	suiteState.Require().NoError(yaml.Unmarshal(composite, &action))
	for _, index := range []int{0, 2} {
		for _, key := range []string{"MODEL_BASE_URL", "MODEL_KEY", "JUDGE_BASE_URL", "JUDGE_API_KEY", "JUDGE_MODEL_NAME"} {
			value, exists := action.Runs.Steps[index].Env[key]
			suiteState.True(exists, key)
			suiteState.Equal("", value, key)
		}
	}
}
