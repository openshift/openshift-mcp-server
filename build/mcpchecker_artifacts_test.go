package build_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type MCPCheckerArtifactsSuite struct{ suite.Suite }

func TestMCPCheckerArtifacts(testingT *testing.T) { suite.Run(testingT, new(MCPCheckerArtifactsSuite)) }

const artifactHarness = `
const helper=require(process.argv[1]), fs=require('node:fs'), path=require('node:path');
const scenario=process.argv[2].replace(/:empty$/, ''), scheduled=process.argv[3]==='schedule', selected=process.argv[4], root=process.argv[5];
const C='c'.repeat(40), H='a'.repeat(40), B='b'.repeat(40), repository='containers/kubernetes-mcp-server';
const outputs={}, comments=[], copies={}, originals={}, api=[];
const destination=path.join(root,'baselines');fs.mkdirSync(destination);process.env.RESULTS_DIRECTORY=destination;
if(scheduled&&!['valid','partial','null-metrics','failed-workflow','missing-artifacts','single-context','zero-assertions'].includes(scenario)){
 for(const suite of ['core','helm'])fs.writeFileSync(path.join(destination,'builtin-openai-'+suite+'-latest.json'),'original-'+suite);
}
function snapshot(target){
 for(const filename of fs.readdirSync(destination)){
  const full=path.join(destination,filename),stat=fs.lstatSync(full);
  target[filename]={mode:stat.mode&0o777,content:stat.isSymbolicLink()?'link:'+fs.readlinkSync(full):stat.isFile()?fs.readFileSync(full,'utf8'):'unexpected-directory'};
 }
}
snapshot(originals);
const selectedSuites=scheduled?['core','helm','kubevirt','kiali','tekton','netobserv','argocd']:[selected];
const source={id:22,run_attempt:2,workflow_id:8,repository:{full_name:repository},head_sha:B,head_branch:'main',head_repository:{full_name:repository},path:'.github/workflows/mcpchecker.yaml',event:scheduled?'schedule':'workflow_run',status:'completed',conclusion:scenario==='failed-workflow'?'failure':'success',pull_requests:[{number:518,head:{sha:C,ref:'feature',repo:{id:71}}}]};
const pr={number:518,state:'open',merge_commit_sha:'e'.repeat(40),base:{repo:{full_name:repository},ref:'main'},head:{ref:'feature',sha:C,repo:{id:71,full_name:'fork/server'}}};
source.display_title='mcpchecker source=99';
const trigger={id:99,workflow_id:9,repository:{full_name:repository},event:'pull_request_review',status:'completed',conclusion:'success',head_branch:'feature',head_sha:C,head_repository:{full_name:'fork/server'},pull_requests:source.pull_requests};
let pulls=[pr];
source.pull_requests=[];
const reviews=[{id:44,state:'COMMENTED',commit_id:C,body:'/run-mcpchecker '+selected,user:{login:'maintainer'}}];
const listPulls=()=>{}, listReviews=()=>{}, listJobs=()=>{}, listArtifacts=()=>{};
let artifacts=[], jobs=[];
for(const suite of selectedSuites){
 jobs.push({name:'Run MCP Evaluation ('+suite+')',run_attempt:2,started_at:'2026-10-16T01:00:00Z',completed_at:'2026-10-16T01:10:00Z',steps:[{name:'Upload PR context',conclusion:'success'},{name:'Run mcpchecker evaluation',conclusion:'success'}]});
 if(scenario==='missing-artifacts'||(scenario==='partial'&&suite!=='core'))continue;
 for(const prefix of ['eval-context-','mcpchecker-results-'])artifacts.push({id:artifacts.length+1,name:prefix+suite,expired:false,size_in_bytes:500,created_at:'2026-10-16T01:05:00Z',workflow_run:{id:22,head_sha:B}});
}
if(scenario==='duplicate')artifacts.push({...artifacts[0],id:99});
if(scenario==='single-context')artifacts=artifacts.filter(artifact=>artifact.name==='eval-context-core');
if(scenario==='artifact-traversal')artifacts[0].name='eval-context-../core';
if(scenario==='multicluster-artifact')artifacts.push({...artifacts[0],id:99,name:'eval-context-kiali-multicluster'});
if(scenario==='artifact-run')artifacts[0].workflow_run.id=99;
if(scenario==='artifact-head')artifacts[0].workflow_run.head_sha=C;
if(scenario==='artifact-expired')artifacts[0].expired=true;
if(scenario==='artifact-oversize')artifacts[0].size_in_bytes=70000001;
if(scenario==='artifact-attempt')jobs[0].run_attempt=1;
if(scenario==='artifact-time')artifacts[0].created_at='2026-10-15T01:05:00Z';
if(scenario==='workflow-mismatch')source.workflow_id=9;
if(scenario==='source-repo')source.repository.full_name='other/repo';
if(scenario==='source-attempt')source.run_attempt=3;
if(scenario==='source-event')source.event='pull_request';
if(scenario==='ambiguous-fork')pr.head.repo.id=72;
if(scenario==='missing-api-pr')trigger.pull_requests=[];
if(scenario==='current-head-moved'){trigger.pull_requests=[];pulls=JSON.parse(JSON.stringify(pulls));pr.head.sha=B;reviews[0].commit_id=B;}
if(scenario==='two-forks')pulls.push({...pr,number:519,head:{ref:'feature',repo:{full_name:'other/fork'}}});
if(scenario==='other-base')pulls.push({...pr,number:519,base:{...pr.base,ref:'release-1'}});
if(scenario==='no-pr')pulls=[];
if(scenario==='ambiguous-pr')pulls.push({...pr,number:519});
if(scenario==='missing-head-repo')delete trigger.head_repository;
if(scenario==='wrong-head-repo')trigger.head_repository.full_name='other/fork';
if(scenario==='wrong-branch')trigger.head_branch='other';
if(scenario==='wrong-trigger-sha')trigger.head_sha='f'.repeat(40);
if(scenario==='merge-as-trigger-sha')trigger.head_sha=pr.merge_commit_sha;
if(scenario==='unready-merge')pr.merge_commit_sha=null;
if(scenario==='closed-pr')pr.state='closed';
if(scenario==='ambiguous-api-pr')trigger.pull_requests.push(trigger.pull_requests[0]);
if(scenario==='untrusted-source-ref')source.head_branch='feature';
if(scenario==='trigger-title')source.display_title='mcpchecker source=99\n::warning::';
if(scenario==='trigger-event')trigger.event='workflow_dispatch';
if(scenario==='trigger-workflow')trigger.workflow_id=8;
if(scenario==='trigger-repo')trigger.repository.full_name='other/repo';
if(scenario==='review-missing')reviews.length=0;
if(scenario==='review-stale')reviews[0].commit_id=B;
if(scenario==='head-drift')pr.head.sha=B;
if(process.argv[2].endsWith(':empty'))trigger.pull_requests=[];
const code="env:\n  HARNESS_SHA: '"+H+"'\n";
const github={rest:{actions:{getWorkflowRun:async request=>({data:request.run_id===99?trigger:source}),getWorkflow:async request=>({data:{id:request.workflow_id==='mcpchecker.yaml'?8:9}}),listJobsForWorkflowRun:listJobs,listWorkflowRunArtifacts:listArtifacts},pulls:{list:listPulls,get:async()=>({data:pr}),listReviews},repos:{getContent:async()=>({data:{type:'file',encoding:'base64',size:Buffer.byteLength(code),content:Buffer.from(code).toString('base64')}}),getCollaboratorPermissionLevel:async()=>({data:{permission:scenario==='review-denied'?'read':'write'}})},issues:{createComment:async value=>{comments.push(value)}}},paginate:async (target,args)=>{api.push('read');return target===listPulls?pulls.filter(pr=>(!args.base||pr.base?.ref===args.base)&&(!args.state||pr.state===args.state)):target===listReviews?reviews:target===listJobs?jobs:artifacts}};
const context={repo:{owner:'containers',repo:'kubernetes-mcp-server'},runId:scheduled?22:33,eventName:scheduled?'schedule':'workflow_run',payload:{workflow_run:{id:22,run_attempt:2}}};
if(scenario==='not-schedule')context.eventName='workflow_dispatch';
const control={version:1,repository,candidate:B,harness:H,baseline:B,source_repository:repository,source_ref:'main',pr_number:0,is_pr:false,review_id:0,reviewer:'',suite:selectedSuites,run_id:22,run_attempt:2,trigger_run_id:0};
process.env.CONTROL_JSON=JSON.stringify(control); process.env.HARNESS_SHA=H;process.env.RUNNER_TEMP=root;process.env.GITHUB_RUN_ATTEMPT='2';
if(scenario==='pr-control')process.env.CONTROL_JSON=process.env.CONTROL_JSON.replace('"is_pr":false','"is_pr":true');
const core={setOutput:(key,value)=>outputs[key]=value};
async function exercise(){
 await helper.run({github,context,core,mode:scheduled?'prepare-schedule':'prepare-report'});
 process.env.ARTIFACT_ROOT=outputs['artifact-root'];process.env.ARTIFACT_PLAN=outputs['artifact-plan'];
 for(const suite of selectedSuites){
  if(!artifacts.some(artifact=>artifact.name==='eval-context-'+suite))continue;
  const contextDirectory=path.join(process.env.ARTIFACT_ROOT,'eval-context-'+suite),results=path.join(process.env.ARTIFACT_ROOT,'mcpchecker-results-'+suite);
  fs.mkdirSync(contextDirectory);fs.mkdirSync(results);
  let value={schema_version:1,repository,pr_number:scheduled?0:518,pr_sha:scheduled?B:C,candidate:scheduled?B:C,harness:H,baseline:B,run_id:22,run_attempt:2,suite,passed:true,has_diff:false,diff_applicable:false,diff_reason:'baseline-absent',check_exit:0,summary_exit:0,verify_exit:0,diff_exit:null,tasks_passed:4,tasks_total:5,task_pass_rate:0.8,assertions_passed:4,assertions_total:5,assertion_pass_rate:0.8};
  const hostile='$(touch SENTINEL);'+String.fromCharCode(96)+'touch SENTINEL'+String.fromCharCode(96)+'\n::warning:: @everyone <script>evil</script> [link](javascript:evil) " \' * %s';
  if(scenario==='wrong-type')value.tasks_passed='4';
  if(scenario==='rate')value.task_pass_rate=1.1;
  if(scenario==='negative')value.tasks_total=-5;
  if(scenario==='incoherent')value.tasks_passed=6;
  if(scenario==='invalid-pr')value.pr_number=-1;
  if(scenario==='invalid-sha')value.harness='not-sha';
  if(scenario==='run-mismatch')value.run_id=99;
  if(scenario==='repo-mismatch')value.repository='other/repo';
  if(scenario==='attempt-mismatch')value.run_attempt=1;
  if(scenario==='suite-mismatch')value.suite='../core';
  if(scenario==='status')value.check_exit=hostile;
  if(scenario==='extra-key')value.inject=hostile;
  if(scenario==='failed-status')value.check_exit=1;
  if(scenario==='partial'||scenario==='null-metrics'){value.passed=false;value.check_exit=1;}
  if(scenario==='null-metrics')for(const key of ['tasks_passed','tasks_total','task_pass_rate','assertions_passed','assertions_total','assertion_pass_rate'])value[key]=null;
  if(scenario==='single-context'){value.passed=false;value.check_exit=1;value.summary_exit=null;value.verify_exit=null;}
  if(scenario==='zero-assertions'){value.assertions_passed=0;value.assertions_total=0;value.assertion_pass_rate=0;}
  if(scenario==='below-tasks'){value.passed=false;value.verify_exit=1;value.tasks_passed=3;value.task_pass_rate=0.6;}
  if(scenario==='below-assertions'){value.passed=false;value.verify_exit=1;value.assertions_passed=3;value.assertion_pass_rate=0.6;}
  if(scenario==='runner-bound'&&suite==='core')value=JSON.parse(fs.readFileSync(process.env.INTEGRATION_CONTEXT,'utf8'));
  const contextFile=path.join(contextDirectory,'context.json');
  fs.writeFileSync(contextFile,scenario==='malformed'||scenario==='mixed-late-invalid'&&suite==='helm'?'{':scenario==='oversized'?' '.repeat(16385):JSON.stringify(value));
  fs.writeFileSync(path.join(contextDirectory,'diff-output.txt'),hostile);
  if(scenario==='diff-oversize')fs.truncateSync(path.join(contextDirectory,'diff-output.txt'),65537);
  if(scenario==='context-link'){fs.renameSync(contextFile,path.join(root,'link-target'));fs.symlinkSync(path.join(root,'link-target'),contextFile);}
  if(scenario==='unexpected-path')fs.writeFileSync(path.join(contextDirectory,'evil.sh'),hostile);
  const filename=path.join(results,'mcpchecker-'+suite+'-e2e-out.json');
  let rows=[{taskPassed:true,taskOutput:suite+hostile},{taskPassed:true},{taskPassed:true},{taskPassed:true},{taskPassed:false}];
  if(scenario==='below-tasks')rows[3].taskPassed=false;
  if(scenario==='runner-bound'&&suite==='core')rows=JSON.parse(fs.readFileSync(process.env.INTEGRATION_RESULT,'utf8'));
  if(scenario==='zero-results')rows=[];
  fs.writeFileSync(filename,scenario==='corrupt-result'?'{':JSON.stringify(rows),{mode:0o755});
  if(scenario.startsWith('diagnostic-')){
   const input=scenario==='diagnostic-expansion'?'s3'.repeat(550000):scenario==='diagnostic-split'?'x'.repeat(1048571)+'secret-model'+'x'.repeat(100):'x'.repeat(scenario==='diagnostic-below'?1048575:scenario==='diagnostic-at'?1048576:1048577);
   const output=path.join(results,'check.txt');
   const diagnosticRoot=fs.mkdtempSync(path.join(root,'diagnostic-'));fs.chmodSync(diagnosticRoot,0o700);
   const run=require('node:child_process').spawnSync('node',[path.join(path.dirname(process.argv[1]),'mcpchecker-diagnostics.cjs'),diagnosticRoot,'check.txt'],{input,env:{...process.env,MODEL_KEY:scenario==='diagnostic-expansion'?'s3':'secret-model'}});
   if(run.status!==0)throw Error('diagnostic producer failed');
   fs.copyFileSync(path.join(diagnosticRoot,'check.txt'),output);
   const diagnostic=fs.readFileSync(output);
   if(diagnostic.length>1048576||diagnostic.includes('secret-model')||scenario==='diagnostic-expansion'&&diagnostic.includes('s3'))throw Error('diagnostic contract');
   if(['diagnostic-above','diagnostic-expansion','diagnostic-split'].includes(scenario)&&!diagnostic.includes('[diagnostic truncated]'))throw Error('missing truncation marker');
  }
  if(scenario==='result-oversize')fs.truncateSync(filename,67108865);
  if(scenario==='wrong-basename')fs.renameSync(filename,path.join(results,'mcpchecker-wrong-e2e-out.json'));
  if(scenario==='result-link'){fs.renameSync(filename,path.join(root,'result-target'));fs.symlinkSync(path.join(root,'result-target'),filename);}
  if(scenario==='missing-result')fs.unlinkSync(filename);
  if(scenario==='result-script')fs.writeFileSync(path.join(results,'script.sh'),hostile);
  if(scenario==='destination-link'){fs.rmSync(path.join(destination,'builtin-openai-'+suite+'-latest.json'),{force:true});fs.symlinkSync(path.join(root,'nonexistent'),path.join(destination,'builtin-openai-'+suite+'-latest.json'));}
  if(!artifacts.some(artifact=>artifact.name==='mcpchecker-results-'+suite))fs.rmSync(results,{recursive:true});
 }
 if(artifacts.length===1){const own=path.join(process.env.ARTIFACT_ROOT,artifacts[0].name);for(const name of fs.readdirSync(own))fs.renameSync(path.join(own,name),path.join(process.env.ARTIFACT_ROOT,name));fs.rmdirSync(own);}
 if(scenario==='head-after-download')pr.head.sha=B;
 if(scenario==='review-after-download')reviews[0].commit_id=B;
 if(scenario==='suite-after-download')reviews[0].body='/run-mcpchecker helm';
 if(scenario==='extra-root')fs.mkdirSync(path.join(process.env.ARTIFACT_ROOT,'eval-context-unknown'));
 if(scenario==='root-traversal')process.env.ARTIFACT_ROOT+='/../'+path.basename(process.env.ARTIFACT_ROOT);
 if(scenario==='tamper-plan')process.env.ARTIFACT_PLAN=process.env.ARTIFACT_PLAN.replace('"run_id":22','"run_id":99');
 snapshot(originals);
 await helper.run({github,context,core,mode:scheduled?'schedule':'report'});
}
exercise().then(()=>{snapshot(copies);console.log(JSON.stringify({outputs,comments,copies,originals,api}))}).catch(()=>{snapshot(copies);console.log(JSON.stringify({rejected:true,comments,copies,originals,api}));process.exitCode=1});
`

func (suiteState *MCPCheckerArtifactsSuite) execute(scenario, mode, selected string) (map[string]any, error) {
	suiteState.T().Helper()
	root := suiteState.T().TempDir()
	command := exec.Command("node", "-e", artifactHarness, workflowBuildPath(suiteState.T(), "mcpchecker-artifacts.cjs"), scenario, mode, selected, root)
	command.Dir = root
	command.Env = append(os.Environ(), "KUBECONFIG=/dev/null")
	output, err := command.CombinedOutput()
	var data map[string]any
	suiteState.Require().NoError(json.Unmarshal(output, &data), string(output))
	suiteState.NoFileExists(filepath.Join(root, "SENTINEL"))
	return data, err
}

func (suiteState *MCPCheckerArtifactsSuite) TestAllSuiteReportConsumers() {
	for _, selected := range []string{"core", "helm", "kubevirt", "kiali", "tekton", "netobserv", "argocd", "all"} {
		suiteState.Run(selected, func() {
			data, err := suiteState.execute("valid", "report", selected)
			suiteState.Require().NoError(err, data)
			comments := data["comments"].([]any)
			suiteState.Require().Len(comments, 1)
			comment := comments[0].(map[string]any)
			suiteState.Equal(float64(518), comment["issue_number"])
			body := comment["body"].(string)
			suiteState.Contains(body, "| "+selected+" | Passed; 4/5 tasks (80.0%)")
			for _, unsafe := range []string{"touch SENTINEL", "::warning::", "@everyone", "<script>", "javascript:"} {
				suiteState.NotContains(body, unsafe)
			}
		})
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestMulticlusterRejectedBeforePublication() {
	data, err := suiteState.execute("valid", "report", "kiali-multicluster")
	suiteState.Error(err, data)
	suiteState.Empty(data["comments"])
	suiteState.Empty(data["copies"])
	for _, mode := range []string{"report", "schedule"} {
		data, err = suiteState.execute("multicluster-artifact", mode, "core")
		suiteState.Error(err, data)
		suiteState.Empty(data["comments"])
		suiteState.Equal(data["originals"], data["copies"])
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestEmptyTriggerAndTwoForkConsumers() {
	for _, scenario := range []string{"missing-api-pr", "two-forks"} {
		suiteState.Run(scenario, func() {
			data, err := suiteState.execute(scenario, "report", "core")
			suiteState.Require().NoError(err, data)
			suiteState.Len(data["comments"], 1)
		})
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestMovedHeadWithEmptyTrigger() {
	data, err := suiteState.execute("current-head-moved", "report", "core")
	suiteState.Error(err)
	suiteState.Empty(data["comments"])
}

func (suiteState *MCPCheckerArtifactsSuite) TestRESTCandidateIdentity() {
	for _, suffix := range []string{"", ":empty"} {
		for _, scenario := range []string{"valid", "other-base", "unready-merge", "wrong-trigger-sha", "merge-as-trigger-sha", "ambiguous-pr", "no-pr", "current-head-moved", "head-drift", "review-stale", "head-after-download", "review-after-download", "suite-after-download"} {
			suiteState.Run(scenario+suffix, func() {
				data, err := suiteState.execute(scenario+suffix, "report", "core")
				if scenario == "valid" || scenario == "other-base" || scenario == "unready-merge" {
					suiteState.NoError(err, data)
					suiteState.Len(data["comments"], 1)
				} else {
					suiteState.Error(err, data)
					suiteState.Empty(data["comments"])
				}
			})
		}
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestIdentityRejectionBeforeCommentOrCopy() {
	for _, scenario := range []string{"duplicate", "no-pr", "ambiguous-pr", "missing-head-repo", "wrong-head-repo", "wrong-branch", "wrong-trigger-sha", "merge-as-trigger-sha", "closed-pr", "artifact-traversal", "artifact-run", "artifact-head", "artifact-expired", "artifact-oversize", "artifact-attempt", "artifact-time", "workflow-mismatch", "source-repo", "source-attempt", "source-event", "ambiguous-fork", "ambiguous-api-pr", "untrusted-source-ref", "trigger-title", "trigger-event", "trigger-workflow", "trigger-repo", "review-missing", "review-stale", "head-drift", "review-denied", "invalid-pr", "invalid-sha", "run-mismatch", "repo-mismatch", "attempt-mismatch", "suite-mismatch", "head-after-download", "extra-root", "root-traversal", "tamper-plan"} {
		suiteState.Run(scenario, func() {
			data, err := suiteState.execute(scenario, "report", "core")
			suiteState.Error(err)
			suiteState.Empty(data["comments"])
			suiteState.Empty(data["copies"])
		})
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestMalformedDataHonestReports() {
	for _, scenario := range []string{"wrong-type", "rate", "negative", "incoherent", "status", "extra-key", "failed-status", "malformed", "oversized", "diff-oversize", "result-oversize", "context-link", "unexpected-path", "zero-results", "corrupt-result", "wrong-basename", "result-link", "missing-result", "result-script", "null-metrics", "missing-artifacts", "failed-workflow"} {
		suiteState.Run(scenario, func() {
			data, err := suiteState.execute(scenario, "report", "core")
			suiteState.Require().NoError(err, data)
			comments := data["comments"].([]any)
			suiteState.Require().Len(comments, 1)
			body := comments[0].(map[string]any)["body"].(string)
			suiteState.NotContains(body, "**Overall:** Passed")
			if scenario == "failed-workflow" {
				suiteState.Contains(body, "**Overall:** Workflow failed")
			} else {
				suiteState.NotContains(body, "| core | Passed;")
			}
		})
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestScheduleExactCopyAndPartialFailure() {
	for _, scenario := range []string{"valid", "partial", "null-metrics", "failed-workflow", "missing-artifacts"} {
		suiteState.Run(scenario, func() {
			data, err := suiteState.execute(scenario, "schedule", "core")
			suiteState.Require().NoError(err, data)
			suiteState.Empty(data["comments"])
			copies := data["copies"].(map[string]any)
			if scenario == "missing-artifacts" {
				suiteState.Empty(copies)
				return
			}
			expected := []string{"core", "helm", "kubevirt", "kiali", "tekton", "netobserv", "argocd"}
			if scenario == "partial" {
				expected = []string{"core"}
			}
			suiteState.Len(copies, len(expected))
			for _, selected := range expected {
				copy := copies["builtin-openai-"+selected+"-latest.json"].(map[string]any)
				suiteState.Equal(float64(0o644), copy["mode"])
				suiteState.Contains(copy["content"], `"taskPassed":false`)
				suiteState.Contains(copy["content"], `"taskOutput":"`+selected+`$(touch SENTINEL)`)
			}
		})
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestPinnedDownloadAndDiagnosticContracts() {
	for _, mode := range []string{"report", "schedule"} {
		for _, scenario := range []string{"valid", "missing-artifacts", "single-context", "zero-assertions", "below-tasks", "below-assertions", "diagnostic-below", "diagnostic-at", "diagnostic-above", "diagnostic-expansion", "diagnostic-split"} {
			suiteState.Run(mode+scenario, func() {
				data, err := suiteState.execute(scenario, mode, "core")
				suiteState.Require().NoError(err, data)
				if mode == "report" {
					body := data["comments"].([]any)[0].(map[string]any)["body"].(string)
					if scenario == "single-context" || scenario == "below-tasks" || scenario == "below-assertions" || scenario == "missing-artifacts" {
						suiteState.NotContains(body, "**Overall:** Passed")
					} else {
						suiteState.Contains(body, "**Overall:** Passed")
					}
					if scenario == "single-context" {
						suiteState.Contains(body, "check=1")
						suiteState.NotContains(body, "Invalid or incomplete artifact data")
					}
				} else if scenario == "single-context" || scenario == "missing-artifacts" {
					suiteState.Empty(data["copies"])
				}
			})
		}
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestScheduledRejectionNoPartialWrites() {
	for _, scenario := range []string{"mixed-late-invalid", "not-schedule", "pr-control", "duplicate", "artifact-run", "artifact-attempt", "artifact-time", "wrong-type", "negative", "status", "malformed", "oversized", "diff-oversize", "result-oversize", "context-link", "unexpected-path", "zero-results", "corrupt-result", "wrong-basename", "result-link", "missing-result", "result-script", "destination-link", "run-mismatch", "repo-mismatch", "attempt-mismatch", "extra-root", "root-traversal", "tamper-plan"} {
		suiteState.Run(scenario, func() {
			data, err := suiteState.execute(scenario, "schedule", "core")
			suiteState.Error(err, data)
			suiteState.Empty(data["comments"])
			suiteState.Equal(data["originals"], data["copies"])
		})
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestRunnerGateReportAndScheduleIntegration() {
	cases := []struct {
		name, payload                                 string
		taskPassed, assertionsPassed, assertionsTotal int
		taskRate, assertionRate                       float64
		passed                                        bool
	}{
		{"exact80", `[{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":false,"assertionPassed":false}]`, 4, 4, 5, 0.8, 0.8, true},
		{"zeroAssertions", `[{"taskPassed":true},{"taskPassed":true},{"taskPassed":true},{"taskPassed":true},{"taskPassed":false}]`, 4, 0, 0, 0.8, 0, true},
		{"tasksBelow", `[{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":false,"assertionPassed":true},{"taskPassed":false,"assertionPassed":false}]`, 3, 4, 5, 0.6, 0.8, false},
		{"assertionsBelow", `[{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":true},{"taskPassed":true,"assertionPassed":false},{"taskPassed":false,"assertionPassed":false}]`, 4, 3, 5, 0.8, 0.6, false},
	}
	for _, testCase := range cases {
		for _, failOnError := range []string{"true", "false"} {
			for _, mode := range []string{"report", "schedule"} {
				suiteState.Run(testCase.name+failOnError+mode, func() {
					fixture := newRunnerFixture(suiteState.T())
					writeActionFixture(suiteState.T(), filepath.Join(fixture.root, "payload"), testCase.payload, 0o600)
					output, values, runErr := fixture.run(suiteState.T(), "run-mcpchecker-action.sh", "FAIL_ON_ERROR="+failOnError)
					suiteState.Equal(testCase.passed || failOnError == "false", runErr == nil, output)
					outputs := map[string]string{}
					for _, line := range strings.Split(values, "\n") {
						key, value, found := strings.Cut(line, "=")
						if found {
							outputs[key] = value
						}
					}
					control := map[string]any{}
					suiteState.Require().NoError(json.Unmarshal([]byte(workflowControlJSON()), &control))
					if mode == "schedule" {
						control["candidate"] = strings.Repeat("b", 40)
						control["pr_number"] = 0
						control["is_pr"] = false
						control["source_repository"] = "containers/kubernetes-mcp-server"
						control["source_ref"] = "main"
						control["review_id"] = 0
						control["reviewer"] = ""
						control["trigger_run_id"] = 0
					}
					encoded, err := json.Marshal(control)
					suiteState.Require().NoError(err)
					directory := filepath.Join(fixture.root, "gate-context")
					environment := append(os.Environ(), "KUBECONFIG=/dev/null", "CONTROL_JSON="+string(encoded), "HARNESS_SHA="+strings.Repeat("a", 40), "GITHUB_RUN_ID=22", "GITHUB_RUN_ATTEMPT=2", "SUITE=core", "CONTEXT_DIRECTORY="+directory, "EVAL_OUTCOME=success", "EVAL_PASSED="+outputs["passed"], "CHECK_EXIT="+outputs["check-exit"], "SUMMARY_EXIT="+outputs["summary-exit"], "VERIFY_EXIT="+outputs["verify-exit"], "SETUP_VERSION=v0.0.22", "SETUP_CHECKER="+fixture.checker, "SETUP_AGENT="+fixture.agent, "RESULTS_FILE="+outputs["results-file"], "SERVER_OUTCOME=success", "REVALIDATE_OUTCOME=success", "BASELINE_OUTCOME=success", "HAS_DIFF=false", "DIFF_OUTCOME=skipped", "DIFF_REASON=baseline-absent", "DIFF_EXIT=not-run", "TASKS_TOTAL="+outputs["tasks-total"], "TASKS_PASSED="+outputs["tasks-passed"], "TASK_PASS_RATE="+outputs["task-pass-rate"], "ASSERTIONS_TOTAL="+outputs["assertions-total"], "ASSERTIONS_PASSED="+outputs["assertions-passed"], "ASSERTION_PASS_RATE="+outputs["assertion-pass-rate"])
					if runErr != nil {
						environment = append(environment, "EVAL_OUTCOME=failure")
					}
					gate := exec.Command("bash", workflowBuildPath(suiteState.T(), "mcpchecker-workflow-status.sh"))
					gate.Env = environment
					gateOutput, gateErr := gate.CombinedOutput()
					suiteState.Equal(testCase.passed, gateErr == nil, string(gateOutput))
					bytes, readErr := os.ReadFile(filepath.Join(directory, "context.json"))
					suiteState.Require().NoError(readErr)
					context := map[string]any{}
					suiteState.Require().NoError(json.Unmarshal(bytes, &context))
					suiteState.Equal(testCase.passed, context["passed"])
					suiteState.Equal(float64(testCase.taskPassed), context["tasks_passed"])
					suiteState.Equal(testCase.taskRate, context["task_pass_rate"])
					suiteState.Equal(float64(testCase.assertionsPassed), context["assertions_passed"])
					suiteState.Equal(float64(testCase.assertionsTotal), context["assertions_total"])
					suiteState.Equal(testCase.assertionRate, context["assertion_pass_rate"])
					root := suiteState.T().TempDir()
					consumer := exec.Command("node", "-e", artifactHarness, workflowBuildPath(suiteState.T(), "mcpchecker-artifacts.cjs"), "runner-bound", mode, "core", root)
					consumer.Env = append(os.Environ(), "KUBECONFIG=/dev/null", "INTEGRATION_CONTEXT="+filepath.Join(directory, "context.json"), "INTEGRATION_RESULT="+outputs["results-file"])
					consumerOutput, consumerErr := consumer.CombinedOutput()
					suiteState.Require().NoError(consumerErr, string(consumerOutput))
					data := map[string]any{}
					suiteState.Require().NoError(json.Unmarshal(consumerOutput, &data))
					if mode == "report" {
						body := data["comments"].([]any)[0].(map[string]any)["body"].(string)
						suiteState.Equal(testCase.passed, strings.Contains(body, "**Overall:** Passed"), body)
						suiteState.NotContains(body, "Invalid or incomplete artifact data")
					} else {
						copy := data["copies"].(map[string]any)["builtin-openai-core-latest.json"].(map[string]any)
						suiteState.JSONEq(testCase.payload, copy["content"].(string))
					}
				})
			}
		}
	}
}

func (suiteState *MCPCheckerArtifactsSuite) TestActualWorkflowConsumersAndPermissions() {
	for _, name := range []string{"mcpchecker-report.yaml", "mcpchecker.yaml"} {
		suiteState.Run(name, func() {
			data, err := os.ReadFile(workflowBuildPath(suiteState.T(), "../.github/workflows/"+name))
			suiteState.Require().NoError(err)
			if name == "mcpchecker.yaml" {
				suiteState.Contains(string(data), "run-name: mcpchecker source=${{ github.event_name == 'workflow_run' && github.event.workflow_run.id || 0 }}")
			}
			var document struct {
				Permissions map[string]string
				Jobs        map[string]struct {
					If          string
					Permissions map[string]string
					Steps       []struct {
						Name, Uses, Run string
						With            map[string]any
						Env             map[string]string
					}
				}
			}
			suiteState.Require().NoError(yaml.Unmarshal(data, &document))
			suiteState.Empty(document.Permissions)
			jobName := "report"
			if name == "mcpchecker.yaml" {
				jobName = "commit-results"
			}
			job := document.Jobs[jobName]
			suiteState.Equal("read", job.Permissions["actions"])
			suiteState.Equal("write", job.Permissions["pull-requests"])
			if jobName == "report" {
				suiteState.Equal("read", job.Permissions["contents"])
				suiteState.Contains(job.If, "containers/kubernetes-mcp-server")
				suiteState.Equal("${{ github.event.repository.default_branch }}", job.Steps[0].With["ref"])
				suiteState.Equal(false, job.Steps[0].With["persist-credentials"])
			} else {
				suiteState.Contains(job.If, "github.event_name == 'schedule'")
			}
			consumers := 0
			for _, step := range job.Steps {
				if strings.Contains(step.Uses, "actions/download-artifact@") {
					suiteState.Equal("actions/download-artifact@9000827ccba6bdab643e8b6fd33ac0654aef8333", step.Uses)
					suiteState.Equal(false, step.With["merge-multiple"])
					suiteState.Contains(step.With["artifact-ids"], "outputs.artifact-ids")
				}
				if script, ok := step.With["script"].(string); ok && strings.Contains(script, "mcpchecker-artifacts.cjs") {
					consumers++
					suiteState.Equal("actions/github-script@3a2844b7e9c422d3c10d287c895573f7108da1b3", step.Uses)
					suiteState.NotContains(script, "${{")
				}
				for key := range step.Env {
					suiteState.NotContains(key, "MODEL")
					suiteState.NotContains(key, "JUDGE")
				}
				suiteState.NotContains(step.Run, "ls -t")
				suiteState.NotContains(step.Run, "awk")
			}
			suiteState.Equal(2, consumers)
		})
	}
}
