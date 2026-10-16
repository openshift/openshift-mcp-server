'use strict';

const fs = require('node:fs');
const path = require('node:path');
const suites = Object.freeze(['core', 'helm', 'kubevirt', 'kiali', 'tekton', 'netobserv', 'argocd', 'all']);
const positive = value => Number.isSafeInteger(value) && value > 0;
const count = value => Number.isSafeInteger(value) && value >= 0 && value <= 2147483647;
const sha = value => typeof value === 'string' && /^[a-f0-9]{40}$/.test(value);
const requireData = condition => { if (!condition) throw new Error('Invalid evaluation artifact data or source binding'); };
const exactKeys = (value, keys) => requireData(value && typeof value === 'object' && !Array.isArray(value) && Object.keys(value).sort().join() === [...keys].sort().join());
const basename = suite => `mcpchecker-${suite}-e2e-out.json`;
const contextKeys = ['schema_version', 'repository', 'pr_number', 'pr_sha', 'candidate', 'harness', 'baseline', 'run_id', 'run_attempt', 'suite', 'passed', 'has_diff', 'diff_applicable', 'diff_reason', 'check_exit', 'summary_exit', 'verify_exit', 'diff_exit', 'tasks_passed', 'tasks_total', 'task_pass_rate', 'assertions_passed', 'assertions_total', 'assertion_pass_rate'];
function validateContext(value, plan, suite) {
  exactKeys(value, contextKeys);
  requireData(value.schema_version === 1 && value.repository === plan.repository && value.run_id === plan.run_id && value.run_attempt === plan.run_attempt && value.suite === suite);
  requireData(value.pr_number === plan.pr_number && value.candidate === plan.candidate && value.pr_sha === value.candidate && sha(value.candidate) && sha(value.harness) && sha(value.baseline));
  requireData(plan.event === 'schedule' ? value.pr_number === 0 && value.candidate === value.baseline : positive(value.pr_number) && value.harness !== value.candidate);
  for (const key of ['passed', 'has_diff', 'diff_applicable']) requireData(typeof value[key] === 'boolean');
  requireData(['baseline-present', 'baseline-absent', 'not-run', ''].includes(value.diff_reason));
  for (const key of ['check_exit', 'summary_exit', 'verify_exit', 'diff_exit']) requireData(value[key] === null || (Number.isSafeInteger(value[key]) && value[key] >= 0 && value[key] <= 999));
  requireData(!value.has_diff || (value.diff_applicable && value.diff_exit === 0 && value.diff_reason === 'baseline-present'));
  const metrics = ['tasks_passed', 'tasks_total', 'task_pass_rate', 'assertions_passed', 'assertions_total', 'assertion_pass_rate'];
  if (metrics.every(key => value[key] === null)) requireData(!value.passed);
  else {
    for (const key of ['tasks_passed', 'tasks_total', 'assertions_passed', 'assertions_total']) requireData(count(value[key]));
    for (const key of ['task_pass_rate', 'assertion_pass_rate']) requireData(typeof value[key] === 'number' && Number.isFinite(value[key]) && value[key] >= 0 && value[key] <= 1);
    requireData(value.tasks_total > 0 && value.tasks_passed <= value.tasks_total && value.assertions_passed <= value.assertions_total && value.task_pass_rate === value.tasks_passed / value.tasks_total);
    requireData(value.assertions_total === 0 ? value.assertion_pass_rate === 0 || value.assertion_pass_rate === 1 : value.assertion_pass_rate === value.assertions_passed / value.assertions_total);
  }
  requireData(!value.passed || (value.check_exit === 0 && value.summary_exit === 0 && value.verify_exit === 0 && value.task_pass_rate >= 0.8 && (value.assertions_total === 0 || value.assertion_pass_rate >= 0.8) && (value.has_diff || (!value.diff_applicable && value.diff_reason === 'baseline-absent'))));
  return value;
}
function directory(filename) {
  requireData(path.isAbsolute(filename) && fs.realpathSync(filename) === filename);
  const stat = fs.lstatSync(filename);
  requireData(stat.isDirectory() && !stat.isSymbolicLink() && stat.uid === process.getuid());
  return filename;
}
function readRegular(filename, maximum) {
  const descriptor = fs.openSync(filename, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW);
  try {
    const stat = fs.fstatSync(descriptor);
    requireData(stat.isFile() && stat.uid === process.getuid() && stat.size > 0 && stat.size <= maximum && stat.nlink === 1);
    const bytes = fs.readFileSync(descriptor);
    requireData(bytes.length === stat.size);
    return bytes;
  } finally { fs.closeSync(descriptor); }
}
function artifactDirectory(root, name, suite, contextArtifact) {
  const target = directory(path.join(root, name));
  const files = fs.readdirSync(target);
  requireData(files.length > 0 && files.length <= 128);
  for (const file of files) {
    requireData(contextArtifact ? ['context.json', 'diff-output.txt'].includes(file) : file === basename(suite) || ['check.txt', 'summary.txt', 'verify.txt'].includes(file) || /^task-error\.[A-Za-z0-9]+$/.test(file));
    const stat = fs.lstatSync(path.join(target, file));
    requireData(stat.isFile() && !stat.isSymbolicLink() && stat.uid === process.getuid() && stat.nlink === 1 && stat.size <= (file === basename(suite) ? 67108864 : 1048576));
  }
  return target;
}
function resultData(bytes) {
  const value = JSON.parse(bytes.toString('utf8'));
  let nodes = 0;
  function bounded(data, depth) {
    requireData(++nodes <= 1000000 && depth <= 64);
    if (typeof data === 'string') requireData(data.length <= 1048576);
    else if (typeof data === 'number') requireData(Number.isFinite(data));
    else if (data && typeof data === 'object') {
      requireData(Object.keys(data).length <= 10000);
      for (const [key, child] of Object.entries(data)) { requireData(key.length <= 1024); bounded(child, depth + 1); }
    }
  }
  bounded(value, 0);
  const rows = Array.isArray(value) ? value : value?.results;
  requireData(Array.isArray(rows) && rows.length > 0 && rows.length <= 10000 && rows.every(row => row && typeof row === 'object' && !Array.isArray(row) && typeof row.taskPassed === 'boolean'));
  return {total: rows.length, passed: rows.filter(row => row.taskPassed).length};
}
async function identity(github, context, scheduled) {
  const {owner, repo} = context.repo;
  const repository = `${owner}/${repo}`;
  requireData(/^[A-Za-z0-9_.-]{1,100}\/[A-Za-z0-9_.-]{1,100}$/.test(repository));
  const runID = scheduled ? context.runId : context.payload.workflow_run?.id;
  requireData(positive(runID) && (!scheduled || context.eventName === 'schedule'));
  const {data: run} = await github.rest.actions.getWorkflowRun({owner, repo, run_id: runID});
  const {data: workflow} = await github.rest.actions.getWorkflow({owner, repo, workflow_id: 'mcpchecker.yaml'});
  requireData(run.id === runID && positive(run.run_attempt) && run.repository?.full_name === repository && run.workflow_id === workflow.id && run.path === '.github/workflows/mcpchecker.yaml' && sha(run.head_sha));
  requireData(run.event === (scheduled ? 'schedule' : 'workflow_run'));
  requireData(run.head_branch === 'main' && run.head_repository?.full_name === repository);
  if (scheduled) requireData(run.run_attempt === Number(process.env.GITHUB_RUN_ATTEMPT) && run.head_branch === 'main' && run.head_repository?.full_name === repository);
  else requireData(context.eventName === 'workflow_run' && run.status === 'completed' && ['success', 'failure', 'cancelled', 'timed_out', 'action_required', 'neutral', 'skipped', 'stale', 'startup_failure'].includes(run.conclusion) && context.payload.workflow_run.run_attempt === run.run_attempt);
  let prNumber = 0, triggerRunID = 0, candidate = run.head_sha, harness, baseline = null, selected = suites.filter(suite => suite !== 'all');
  if (scheduled) {
    const raw = process.env.CONTROL_JSON || '';
    requireData(Buffer.byteLength(raw) <= 16384);
    const control = require('./mcpchecker-control.cjs').validate(JSON.parse(raw), context);
    requireData(!control.is_pr && control.suite.join() === selected.join());
    candidate = control.candidate; harness = control.harness; baseline = control.baseline;
  } else {
    const {data: source} = await github.rest.repos.getContent({owner, repo, ref: run.head_sha, path: '.github/workflows/mcpchecker.yaml'});
    requireData(source.type === 'file' && source.encoding === 'base64' && positive(source.size) && source.size <= 65536 && typeof source.content === 'string' && source.content.length <= 90000);
    const bytes = Buffer.from(source.content, 'base64');
    requireData(bytes.length === source.size);
    const pins = [...bytes.toString('utf8').matchAll(/^  HARNESS_SHA: '([a-f0-9]{40})'$/gm)];
    requireData(pins.length === 1);
    harness = pins[0][1];
  }
  if (!scheduled) {
    requireData(typeof run.display_title === 'string' && /^mcpchecker source=[1-9][0-9]{0,15}$/.test(run.display_title));
    triggerRunID = Number(run.display_title.slice('mcpchecker source='.length));
    requireData(positive(triggerRunID));
    const {data: trigger} = await github.rest.actions.getWorkflowRun({owner, repo, run_id: triggerRunID});
    const {data: triggerWorkflow} = await github.rest.actions.getWorkflow({owner, repo, workflow_id: 'mcpchecker-trigger.yaml'});
    requireData(trigger.id === triggerRunID && trigger.workflow_id === triggerWorkflow.id && trigger.repository?.full_name === repository && trigger.event === 'pull_request_review' && trigger.status === 'completed' && trigger.conclusion === 'success');
    requireData(Array.isArray(trigger.pull_requests) && trigger.pull_requests.length === 1 && positive(trigger.pull_requests[0].number) && sha(trigger.pull_requests[0].head?.sha));
    const recorded = trigger.pull_requests[0];
    const {data: pr} = await github.rest.pulls.get({owner, repo, pull_number: recorded.number});
    requireData(pr.state === 'open' && pr.number === recorded.number && pr.base?.repo?.full_name === repository && pr.base.ref === 'main' && typeof pr.head?.ref === 'string' && pr.head.ref.length > 0 && pr.head.ref.length <= 255 && /^[A-Za-z0-9_.-]{1,100}\/[A-Za-z0-9_.-]{1,100}$/.test(pr.head?.repo?.full_name));
    requireData(pr.head.sha === recorded.head.sha && pr.head.ref === recorded.head.ref && pr.head.repo.id === recorded.head.repo?.id && positive(pr.head.repo.id));
    const reviews = await github.paginate(github.rest.pulls.listReviews, {owner, repo, pull_number: pr.number, per_page: 100});
    requireData(Array.isArray(reviews) && reviews.length <= 5000);
    const commands = reviews.filter(review => review.state !== 'DISMISSED' && typeof review.body === 'string' && review.body.includes('/run-mcpchecker')).sort((left, right) => right.id - left.id);
    const review = commands[0];
    requireData(review && positive(review.id) && ['APPROVED', 'CHANGES_REQUESTED', 'COMMENTED'].includes(review.state) && review.commit_id === pr.head.sha && review.body.length <= 65536 && /^[A-Za-z0-9-]{1,39}$/.test(review.user?.login));
    const lines = review.body.split(/\r?\n/).map(line => line.trim()).filter(line => line.includes('/run-mcpchecker'));
    requireData(lines.length === 1);
    const command = /^\/run-mcpchecker(?: (core|helm|kubevirt|kiali|tekton|netobserv|argocd|all))?$/i.exec(lines[0]);
    requireData(command);
    const {data: permission} = await github.rest.repos.getCollaboratorPermissionLevel({owner, repo, username: review.user.login});
    requireData(['write', 'admin'].includes(permission.permission));
    prNumber = pr.number; candidate = pr.head.sha; selected = [(command[1] || 'core').toLowerCase()];
  }
  const jobs = await github.paginate(github.rest.actions.listJobsForWorkflowRun, {owner, repo, run_id: runID, filter: 'latest', per_page: 100});
  requireData(Array.isArray(jobs) && jobs.length <= 1000);
  const artifacts = await github.paginate(github.rest.actions.listWorkflowRunArtifacts, {owner, repo, run_id: runID, per_page: 100});
  requireData(Array.isArray(artifacts) && artifacts.length <= 1000);
  const accepted = [];
  for (const artifact of artifacts) {
    if (!/^(eval-context-|mcpchecker-results-)/.test(artifact.name)) continue;
    const match = /^(eval-context|mcpchecker-results)-(core|helm|kubevirt|kiali|tekton|netobserv|argocd|all)$/.exec(artifact.name);
    requireData(match && selected.includes(match[2]) && positive(artifact.id) && artifact.expired === false && positive(artifact.size_in_bytes) && artifact.size_in_bytes <= 70000000 && artifact.workflow_run?.id === runID && artifact.workflow_run.head_sha === run.head_sha);
    const job = jobs.filter(entry => entry.run_attempt === run.run_attempt && entry.name.startsWith(`Run MCP Evaluation (${match[2]})`));
    requireData(job.length === 1 && Array.isArray(job[0].steps));
    requireData(job[0].steps.some(step => step.name === (match[1] === 'eval-context' ? 'Upload PR context' : 'Run mcpchecker evaluation') && step.conclusion === 'success'));
    const start = Date.parse(job[0].started_at), end = Date.parse(job[0].completed_at), created = Date.parse(artifact.created_at);
    requireData(Number.isFinite(start) && Number.isFinite(end) && Number.isFinite(created) && created >= start && created < end + 1000);
    requireData(!accepted.some(entry => entry.name === artifact.name || entry.id === artifact.id));
    accepted.push({id: artifact.id, name: artifact.name});
  }
  return {version: 1, repository, run_id: runID, run_attempt: run.run_attempt, trigger_run_id: triggerRunID, event: run.event, conclusion: run.conclusion, pr_number: prNumber, candidate, harness, baseline, suite: selected, artifacts: accepted};
}
async function run({github, context, core, mode}) {
  requireData(['prepare-report', 'report', 'prepare-schedule', 'schedule'].includes(mode));
  const scheduled = mode.endsWith('schedule');
  const plan = await identity(github, context, scheduled);
  if (mode.startsWith('prepare-')) {
    const encoded = JSON.stringify(plan);
    requireData(Buffer.byteLength(encoded) <= 32768);
    core.setOutput('artifact-ids', plan.artifacts.map(artifact => artifact.id).join(','));
    core.setOutput('artifact-plan', encoded);
    const root = fs.mkdtempSync(path.join(directory(process.env.RUNNER_TEMP), 'mcpchecker-artifacts.'));
    fs.chmodSync(root, 0o700);
    core.setOutput('artifact-root', root);
    return;
  }
  const raw = process.env.ARTIFACT_PLAN || '';
  requireData(Buffer.byteLength(raw) <= 32768);
  requireData(JSON.stringify(JSON.parse(raw)) === JSON.stringify(plan));
  const root = directory(process.env.ARTIFACT_ROOT);
  if (plan.artifacts.length === 1) {
    const artifact = plan.artifacts[0];
    const entries = fs.readdirSync(root);
    if (!entries.includes(artifact.name)) {
      const suite = plan.suite.find(suite => artifact.name === `eval-context-${suite}` || artifact.name === `mcpchecker-results-${suite}`);
      requireData(suite);
      const contextArtifact = artifact.name === `eval-context-${suite}`;
      requireData(entries.length > 0 && entries.length <= 128);
      for (const file of entries) {
        requireData(contextArtifact ? ['context.json', 'diff-output.txt'].includes(file) : file === basename(suite) || ['check.txt', 'summary.txt', 'verify.txt'].includes(file) || /^task-error\.[A-Za-z0-9]+$/.test(file));
        const stat = fs.lstatSync(path.join(root, file));
        requireData(stat.isFile() && !stat.isSymbolicLink() && stat.uid === process.getuid() && stat.nlink === 1);
      }
      fs.mkdirSync(path.join(root, artifact.name), {mode: 0o700});
      for (const file of entries) fs.renameSync(path.join(root, file), path.join(root, artifact.name, file));
    }
  }
  const names = fs.readdirSync(root);
  requireData(names.length === plan.artifacts.length && names.every(name => plan.artifacts.some(artifact => artifact.name === name)));
  const diagnostics = [], copies = [];
  let rejected = false, common = null;
  for (const suite of plan.suite) {
    const contextName = `eval-context-${suite}`, resultName = `mcpchecker-results-${suite}`;
    if (!names.includes(contextName)) {
      diagnostics.push(`| ${suite} | Missing context |`);
      if (names.includes(resultName)) rejected = true;
      continue;
    }
    let parsed;
    try {
      const contextPath = artifactDirectory(root, contextName, suite, true);
      parsed = JSON.parse(readRegular(path.join(contextPath, 'context.json'), 16384));
    } catch {
      rejected = true;
      diagnostics.push(`| ${suite} | Invalid or incomplete artifact data |`);
      continue;
    }
    requireData(parsed && parsed.repository === plan.repository && parsed.run_id === plan.run_id && parsed.run_attempt === plan.run_attempt && parsed.suite === suite && parsed.pr_number === plan.pr_number && parsed.candidate === plan.candidate && parsed.pr_sha === plan.candidate && parsed.harness === plan.harness && (plan.baseline === null || parsed.baseline === plan.baseline));
    try {
      const contextPath = artifactDirectory(root, contextName, suite, true);
      const value = validateContext(parsed, plan, suite);
      const binding = `${value.harness}/${value.baseline}`;
      requireData(common === null || binding === common); common = binding;
      if (fs.existsSync(path.join(contextPath, 'diff-output.txt'))) readRegular(path.join(contextPath, 'diff-output.txt'), 65536);
      if (names.includes(resultName)) {
        const resultPath = artifactDirectory(root, resultName, suite, false);
        if (fs.existsSync(path.join(resultPath, basename(suite)))) {
          const bytes = readRegular(path.join(resultPath, basename(suite)), 67108864);
          const metrics = resultData(bytes);
          requireData(value.tasks_total === null || value.tasks_total === metrics.total && value.tasks_passed === metrics.passed);
          copies.push({suite, bytes});
        } else requireData(!value.passed);
      } else requireData(!value.passed);
      const summary = value.tasks_total === null ? 'Metrics unavailable' : `${value.tasks_passed}/${value.tasks_total} tasks (${(value.task_pass_rate * 100).toFixed(1)}%); ${value.assertions_passed}/${value.assertions_total} assertions`;
      diagnostics.push(`| ${suite} | ${value.passed ? 'Passed' : 'Failed'}; ${summary}; check=${value.check_exit ?? 'not-run'}, summary=${value.summary_exit ?? 'not-run'}, verify=${value.verify_exit ?? 'not-run'}, diff=${value.diff_exit ?? 'not-run'} |`);
    } catch {
      rejected = true;
      diagnostics.push(`| ${suite} | Invalid or incomplete artifact data |`);
    }
  }
  if (scheduled) {
    requireData(!rejected);
    const destination = directory(process.env.RESULTS_DIRECTORY);
    for (const copy of copies) {
      const target = path.join(destination, `builtin-openai-${copy.suite}-latest.json`);
      let stat;
      try { stat = fs.lstatSync(target); } catch (error) { if (error.code !== 'ENOENT') throw error; }
      if (stat) requireData(stat.isFile() && !stat.isSymbolicLink() && stat.nlink === 1);
    }
    for (const copy of copies) {
      const temporary = fs.mkdtempSync(path.join(destination, '.mcpchecker-'));
      try {
        const filename = path.join(temporary, 'result.json');
        fs.writeFileSync(filename, copy.bytes, {mode: 0o644, flag: 'wx'}); fs.chmodSync(filename, 0o644);
        fs.renameSync(filename, path.join(destination, `builtin-openai-${copy.suite}-latest.json`));
      } finally { fs.rmSync(temporary, {recursive: true}); }
    }
    core.setOutput('copied-suites', copies.map(copy => copy.suite).join(','));
    return;
  }
  if (!plan.artifacts.length) diagnostics.push('| all | No evaluation artifacts were produced |');
  const overall = plan.conclusion !== 'success' ? 'Workflow failed' : rejected || diagnostics.some(line => !line.includes('| Passed;')) ? 'Failed or incomplete' : 'Passed';
  const body = `## mcpchecker MCP Evaluation Results\n\n**Commit:** \`${plan.candidate.slice(0, 7)}\`\n\n**Overall:** ${overall}\n\n| Suite | Result |\n|---|---|\n${diagnostics.join('\n')}\n\nDiff text and task output are untrusted data; inspect the run artifacts separately.\n\n[View full results](https://github.com/${plan.repository}/actions/runs/${plan.run_id})`;
  requireData(body.length <= 12000);
  await identity(github, context, false).then(current => requireData(JSON.stringify(current) === JSON.stringify(plan)));
  await github.rest.issues.createComment({...context.repo, issue_number: plan.pr_number, body});
  core.setOutput('reported', 'true');
}
module.exports = {run};
