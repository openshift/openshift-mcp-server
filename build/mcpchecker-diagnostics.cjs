'use strict';

const fs = require('node:fs');
const path = require('node:path');
const [root, leaf, summaryMode] = process.argv.slice(2);
if (process.argv.length < 4 || process.argv.length > 5 || !root || !path.isAbsolute(root) || fs.realpathSync(root) !== root) throw new Error('Invalid diagnostic root');
const directory = fs.lstatSync(root);
if (!directory.isDirectory() || directory.uid !== process.getuid() || (directory.mode & 0o7777) !== 0o700) throw new Error('Diagnostic root must be private');
if (!/^(check|verify|summary)\.txt$|^task-error\.[A-Za-z0-9]{8}$/.test(leaf) || (summaryMode !== undefined && summaryMode !== 'summary')) throw new Error('Invalid diagnostic leaf');
if (summaryMode && leaf !== 'summary.txt') throw new Error('Invalid summary leaf');
const destination = path.join(root, summaryMode ? 'raw-summary.json' : leaf);
const existing = !summaryMode && /^task-error\./.test(leaf);
const descriptor = fs.openSync(destination, fs.constants.O_WRONLY | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK | (existing ? 0 : fs.constants.O_CREAT | fs.constants.O_EXCL), 0o600);
const outputInfo = fs.fstatSync(descriptor);
if (!outputInfo.isFile() || outputInfo.uid !== process.getuid() || outputInfo.nlink !== 1 || outputInfo.size !== 0 || (outputInfo.mode & 0o7777) !== 0o600) {
  fs.closeSync(descriptor);
  throw new Error('Invalid diagnostic file');
}
const maximum = 1048576;
const marker = Buffer.from('\n[diagnostic truncated]\n');
const secrets = ['MODEL_KEY', 'JUDGE_API_KEY', 'OPENAI_API_KEY', 'ANTHROPIC_API_KEY', 'GOOGLE_API_KEY', 'GEMINI_API_KEY', 'AZURE_OPENAI_API_KEY'].map(name => process.env[name]).filter(value => value);
const overlap = Math.max(0, ...secrets.map(value => Buffer.byteLength(value)));
const chunks = [];
let retained = 0, total = 0;
const summary = summaryMode ? descriptor : null;
process.stdin.on('data', chunk => {
  if (summary !== null && total < 67108864) fs.writeFileSync(summary, chunk.subarray(0, 67108864 - total));
  total += chunk.length;
  const prefix = chunk.subarray(0, Math.max(0, maximum + overlap - retained));
  if (prefix.length) chunks.push(Buffer.from(prefix));
  retained += prefix.length;
});
process.stdin.on('end', () => {
  if (summary !== null) {
    fs.closeSync(summary);
    if (total > 67108864) process.exitCode = 1;
    return;
  }
  let text = Buffer.concat(chunks).toString('utf8');
  if (total > retained) {
    let suffix = 0;
    for (const secret of secrets) {
      for (let length = Math.min(secret.length - 1, text.length); length > suffix; length--) {
        if (text.endsWith(secret.slice(0, length))) { suffix = length; break; }
      }
    }
    if (suffix) text = text.slice(0, -suffix);
  }
  for (const secret of secrets) text = text.split(secret).join('[REDACTED]');
  const bytes = Buffer.from(text);
  const truncated = total > retained || bytes.length > maximum;
  const output = truncated ? Buffer.concat([bytes.subarray(0, maximum - marker.length), marker]) : bytes;
  if (secrets.some(secret => output.includes(Buffer.from(secret)))) throw new Error('Diagnostic redaction failed');
  fs.writeFileSync(descriptor, output);
  fs.closeSync(descriptor);
});
