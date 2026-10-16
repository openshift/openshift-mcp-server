'use strict';

const fs = require('node:fs');
const path = require('node:path');
const sha = value => typeof value === 'string' && /^[a-f0-9]{40}$/.test(value);
const integer = value => Number.isSafeInteger(value) && value > 0;
const repository = value => typeof value === 'string' && /^[A-Za-z0-9_.-]{1,100}\/[A-Za-z0-9_.-]{1,100}$/.test(value);
const suites = Object.freeze({
  core: ['core', 'core,config'], helm: ['helm', 'core,config,helm'],
  kubevirt: ['kubevirt', 'core,config,kubevirt,tekton'], kiali: ['kiali', 'core,config,kiali'],
  tekton: ['tekton', 'core,config,tekton'], netobserv: ['netobserv', 'core,config,netobserv'],
  argocd: ['argo', 'core,config'], all: ['all', 'core,config,helm,kiali,kubevirt,tekton,netobserv'],
});
const requireValue = condition => { if (!condition) throw new Error('Invalid evaluation identity or metadata'); };
function suiteEntry(suite) {
  requireValue(Object.hasOwn(suites, suite));
  return {suite, 'eval-config': `evals/core-eval-testing/builtin-openai/eval-${suites[suite][0]}.yaml`,
    toolsets: suites[suite][1], 'expected-results-file': `mcpchecker-${suite}-e2e-out.json`};
}
function validate(value, context) {
  requireValue(value && typeof value === 'object' && !Array.isArray(value));
  const keys = ['version', 'repository', 'candidate', 'harness', 'baseline', 'source_repository', 'source_ref', 'pr_number', 'is_pr', 'review_id', 'reviewer', 'suite', 'run_id', 'run_attempt', 'trigger_run_id'];
  requireValue(Object.keys(value).sort().join() === keys.sort().join());
  requireValue(value.version === 1 && sha(value.candidate) && sha(value.harness) && sha(value.baseline));
  requireValue(repository(value.repository) && repository(value.source_repository));
  requireValue(value.repository === `${context.repo.owner}/${context.repo.repo}` && value.run_id === context.runId && value.run_attempt === Number(process.env.GITHUB_RUN_ATTEMPT));
  requireValue(integer(value.run_id) && integer(value.run_attempt) && typeof value.is_pr === 'boolean');
  requireValue(typeof value.source_ref === 'string' && value.source_ref.length > 0 && value.source_ref.length <= 255 && !/[\x00-\x20\x7f]/.test(value.source_ref));
  requireValue(Array.isArray(value.suite) && value.suite.length > 0 && value.suite.length <= 7 && new Set(value.suite).size === value.suite.length);
  value.suite.forEach(suiteEntry);
  requireValue(value.harness === process.env.HARNESS_SHA);
  requireValue(!value.is_pr || value.harness !== value.candidate);
  requireValue(value.is_pr ? integer(value.pr_number) && integer(value.review_id) && integer(value.trigger_run_id) && /^[A-Za-z0-9-]{1,39}$/.test(value.reviewer) : value.pr_number === 0 && value.review_id === 0 && value.trigger_run_id === 0 && value.reviewer === '' && value.source_repository === value.repository && value.candidate === value.baseline && value.source_ref === 'main');
  return value;
}
function parse(context) {
  const raw = process.env.CONTROL_JSON || '';
  requireValue(Buffer.byteLength(raw) <= 16384);
  return validate(JSON.parse(raw), context);
}
async function currentReview(github, context, number) {
  const {owner, repo} = context.repo;
  const {data: pr} = await github.rest.pulls.get({owner, repo, pull_number: number});
  requireValue(pr.state === 'open' && pr.number === number && integer(pr.number) && pr.base.repo.full_name === `${owner}/${repo}` && pr.base.ref === 'main');
  requireValue(sha(pr.head.sha) && repository(pr.head.repo?.full_name) && pr.head.ref && pr.head.ref.length <= 255);
  const reviews = await github.paginate(github.rest.pulls.listReviews, {owner, repo, pull_number: number, per_page: 100});
  requireValue(Array.isArray(reviews) && reviews.length <= 5000);
  const commands = reviews.filter(review => review.state !== 'DISMISSED' && typeof review.body === 'string' && review.body.includes('/run-mcpchecker'));
  commands.sort((left, right) => right.id - left.id);
  const review = commands[0];
  requireValue(review && integer(review.id) && ['APPROVED', 'CHANGES_REQUESTED', 'COMMENTED'].includes(review.state) && review.body.length <= 65536 && /^[A-Za-z0-9-]{1,39}$/.test(review.user?.login));
  const lines = review.body.split(/\r?\n/).map(line => line.trim()).filter(line => line.includes('/run-mcpchecker'));
  requireValue(lines.length === 1);
  const match = /^\/run-mcpchecker(?: (core|helm|kubevirt|kiali|tekton|netobserv|argocd|all))?$/i.exec(lines[0]);
  requireValue(match && review.commit_id === pr.head.sha);
  const {data: permission} = await github.rest.repos.getCollaboratorPermissionLevel({owner, repo, username: review.user.login});
  requireValue(['write', 'admin'].includes(permission.permission));
  return {pr, review, suite: (match[1] || 'core').toLowerCase()};
}
async function protectedEnvironment(github, context) {
  const {owner, repo} = context.repo;
  const environment_name = 'mcpchecker-reviewed';
  const {data: environment} = await github.rest.repos.getEnvironment({owner, repo, environment_name});
  requireValue(environment.protection_rules?.some(rule => rule.type === 'required_reviewers' && rule.prevent_self_review === true && Array.isArray(rule.reviewers) && rule.reviewers.length > 0));
  requireValue(environment.deployment_branch_policy?.custom_branch_policies === true && environment.deployment_branch_policy.protected_branches === false);
  const {data: policies} = await github.rest.repos.listDeploymentBranchPolicies({owner, repo, environment_name, per_page: 100});
  requireValue(policies.total_count === 1 && policies.branch_policies?.length === 1 && policies.branch_policies[0].name === 'main' && policies.branch_policies[0].type === 'branch');
}
async function run({github, context, core, mode = 'control'}) {
  const {owner, repo} = context.repo;
  requireValue(sha(process.env.HARNESS_SHA) && integer(context.runId));
  if (mode === 'control') {
    core.setOutput('should-run', 'false');
    await protectedEnvironment(github, context);
    const {data: baseline} = await github.rest.repos.getCommit({owner, repo, ref: 'main'});
    requireValue(sha(baseline.sha));
    let selected = context.eventName === 'schedule' ? Object.keys(suites).filter(name => name !== 'all') : [process.env.INPUT_SUITE || 'core'];
    let identity = {candidate: baseline.sha, source_repository: `${owner}/${repo}`, source_ref: 'main', pr_number: 0, is_pr: false, review_id: 0, reviewer: '', trigger_run_id: 0};
    if (context.eventName === 'workflow_run') {
      const triggerID = context.payload.workflow_run?.id;
      requireValue(integer(triggerID));
      const {data: trigger} = await github.rest.actions.getWorkflowRun({owner, repo, run_id: triggerID});
      const {data: workflow} = await github.rest.actions.getWorkflow({owner, repo, workflow_id: 'mcpchecker-trigger.yaml'});
      requireValue(trigger.id === triggerID && trigger.workflow_id === workflow.id && trigger.repository.full_name === `${owner}/${repo}` && trigger.event === 'pull_request_review' && trigger.status === 'completed' && trigger.conclusion === 'success');
      const pulls = await github.paginate(github.rest.pulls.list, {owner, repo, state: 'open', per_page: 100});
      requireValue(Array.isArray(pulls) && pulls.length <= 5000 && typeof trigger.head_branch === 'string');
      const matches = pulls.filter(pr => pr.head.ref === trigger.head_branch);
      requireValue(matches.length === 1);
      const current = await currentReview(github, context, matches[0].number);
      requireValue(current.pr.head.ref === trigger.head_branch);
      if (trigger.pull_requests?.length) requireValue(trigger.pull_requests.length === 1 && trigger.pull_requests[0].number === current.pr.number);
      identity = {candidate: current.pr.head.sha, source_repository: current.pr.head.repo.full_name, source_ref: current.pr.head.ref, pr_number: current.pr.number, is_pr: true, review_id: current.review.id, reviewer: current.review.user.login, trigger_run_id: triggerID};
      selected = [current.suite];
    } else {
      requireValue(['schedule', 'workflow_dispatch'].includes(context.eventName) && context.ref === 'refs/heads/main');
    }
    const metadata = validate({version: 1, repository: `${owner}/${repo}`, harness: process.env.HARNESS_SHA, baseline: baseline.sha, ...identity, suite: selected, run_id: context.runId, run_attempt: Number(process.env.GITHUB_RUN_ATTEMPT)}, context);
    const encoded = JSON.stringify(metadata);
    requireValue(Buffer.byteLength(encoded) <= 16384);
    for (const [key, value] of Object.entries({'control-json': encoded, matrix: JSON.stringify({include: selected.map(suiteEntry)}), 'pr-number': metadata.pr_number, 'pr-sha': metadata.candidate, 'is-pr': String(metadata.is_pr), baseline: metadata.baseline, 'source-repository': metadata.source_repository})) core.setOutput(key, value);
    core.setOutput('should-run', 'true');
    if (core.summary) await core.summary.addHeading('Independent privileged-execution approval required').addRaw(`Eligibility is not approval. Review whole C=${metadata.candidate}, H=${metadata.harness}, suites=${metadata.suite.join(',')}. Same-account execution is not a sandbox; do not approve if this risk is unacceptable. External deployment prerequisite: a maintainer must audit and disable administrative bypass of this environment before deployment and each approval; the REST environment response does not verify that setting.`).write();
    return metadata;
  }
  const metadata = parse(context);
  if (mode === 'revalidate') {
    await protectedEnvironment(github, context);
    if (metadata.is_pr) {
      const current = await currentReview(github, context, metadata.pr_number);
      requireValue(current.pr.head.sha === metadata.candidate && current.pr.head.repo.full_name === metadata.source_repository && current.pr.head.ref === metadata.source_ref && current.review.id === metadata.review_id && current.review.user.login === metadata.reviewer && current.suite === metadata.suite[0]);
    } else {
      const {data: commit} = await github.rest.repos.getCommit({owner, repo, ref: metadata.candidate});
      requireValue(commit.sha === metadata.candidate);
    }
    return metadata;
  }
  if (mode === 'artifact') {
    const artifactID = Number(process.env.CANDIDATE_ARTIFACT_ID);
    requireValue(integer(artifactID));
    const {data: artifact} = await github.rest.actions.getArtifact({owner, repo, artifact_id: artifactID});
    requireValue(artifact.id === artifactID && artifact.name === 'mcpchecker-candidate-server' && artifact.expired === false && integer(artifact.size_in_bytes) && artifact.size_in_bytes <= 268435456 && artifact.workflow_run?.id === metadata.run_id);
    const jobs = await github.paginate(github.rest.actions.listJobsForWorkflowRun, {owner, repo, run_id: metadata.run_id, filter: 'latest', per_page: 100});
    const build = jobs.filter(job => job.name === 'Build Candidate' && job.run_attempt === metadata.run_attempt);
    requireValue(build.length === 1 && build[0].conclusion === 'success' && build[0].steps?.some(step => step.name === 'Upload candidate binary' && step.conclusion === 'success'));
    return metadata;
  }
  if (mode === 'baseline') {
    const suite = process.env.SUITE;
    requireValue(metadata.suite.includes(suite));
    const directory = fs.realpathSync(process.env.RUNNER_TEMP);
    const output = fs.mkdtempSync(path.join(directory, 'mcpchecker-baseline.'));
    fs.chmodSync(output, 0o700);
    let response;
    try {
      response = await github.rest.repos.getContent({owner, repo, ref: metadata.baseline, path: `evals/results/builtin-openai-${suite}-latest.json`, mediaType: {format: 'object'}});
    } catch (error) {
      if (error.status !== 404) throw error;
      core.setOutput('has-diff', 'false'); core.setOutput('reason', 'baseline-absent'); return;
    }
    const data = response.data;
    requireValue(!Array.isArray(data) && data.type === 'file' && data.path === `evals/results/builtin-openai-${suite}-latest.json` && sha(data.sha) && integer(data.size) && data.size <= 67108864);
    const {data: blob} = await github.rest.git.getBlob({owner, repo, file_sha: data.sha});
    requireValue(blob.sha === data.sha && blob.size === data.size && blob.encoding === 'base64' && typeof blob.content === 'string' && blob.content.length <= 92274688);
    const encoded = blob.content.replace(/\n/g, '');
    requireValue(/^[A-Za-z0-9+/]*={0,2}$/.test(encoded));
    const bytes = Buffer.from(encoded, 'base64');
    requireValue(bytes.length === data.size && bytes.length <= 67108864 && bytes.toString('base64') === encoded);
    const parsed = JSON.parse(bytes.toString('utf8'));
    const rows = Array.isArray(parsed) ? parsed : parsed.results;
    requireValue(Array.isArray(rows) && rows.length > 0 && rows.every(row => row && typeof row.taskPassed === 'boolean'));
    const file = path.join(output, 'baseline.json');
    fs.writeFileSync(file, bytes, {mode: 0o600, flag: 'wx'});
    core.setOutput('base-results', file); core.setOutput('has-diff', 'true'); core.setOutput('reason', 'baseline-present'); return;
  }
  throw new Error('Unknown evaluation control operation');
}

module.exports = {run, validate, suiteEntry};
