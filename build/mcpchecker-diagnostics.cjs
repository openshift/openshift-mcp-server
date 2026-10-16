'use strict';

const fs = require('node:fs');
const maximum = 1048576;
const marker = Buffer.from('\n[diagnostic truncated]\n');
const secrets = ['MODEL_KEY', 'JUDGE_API_KEY', 'OPENAI_API_KEY', 'ANTHROPIC_API_KEY', 'GOOGLE_API_KEY', 'GEMINI_API_KEY', 'AZURE_OPENAI_API_KEY'].map(name => process.env[name]).filter(value => value);
const overlap = Math.max(0, ...secrets.map(value => Buffer.byteLength(value)));
const chunks = [];
let retained = 0, total = 0;
const summary = process.argv[3] ? fs.openSync(process.argv[3], 'wx', 0o600) : null;
process.stdin.on('data', chunk => {
  if (summary !== null && total < 67108864) fs.writeSync(summary, chunk.subarray(0, 67108864 - total));
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
  fs.writeFileSync(process.argv[2], output, {mode: 0o600});
});
