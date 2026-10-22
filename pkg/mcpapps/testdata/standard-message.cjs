const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

const html = fs.readFileSync(0, 'utf8');
const script = html.match(/<script>([\s\S]*?)<\/script>/)[1];

function fixture(referrer = '') {
  const sent = [], callbacks = {};
  const app = { textContent: 'Loading data', setAttribute() {}, removeAttribute() {} };
  const parent = { postMessage(message, origin) { sent.push({ message, origin }); } };
  const document = {
    referrer,
    getElementById() { return app; },
    body: { dataset: { appKind: 'table' } },
    documentElement: { dataset: {}, style: {} },
  };
  let disconnected = false;
  vm.runInNewContext(script, {
    document, parent, URL,
    window: { addEventListener(name, callback) { callbacks[name] = callback; } },
    ResizeObserver: class { observe() {} disconnect() { disconnected = true; } },
    requestAnimationFrame() { return 1; }, cancelAnimationFrame() {},
  });
  return {
    app, sent, parent, document,
    disconnected: () => disconnected,
    receive(source, origin, data) { callbacks.message({ source, origin, data }); },
  };
}

const initialize = { jsonrpc: '2.0', id: 1, result: { hostContext: { theme: 'dark' } } };
const cancel = { jsonrpc: '2.0', method: 'ui/notifications/tool-cancelled', params: { reason: 'cancelled' } };
for (const { origin, referrer } of [
  { origin: 'https://host.example', referrer: 'https://host.example/embed' },
  { origin: 'http://localhost:3000', referrer: 'http://localhost:3000/embed' },
  { origin: 'https://redacted-referrer.example', referrer: '' },
  { origin: 'null', referrer: '' },
]) {
  const state = fixture(referrer);
  assert.equal(state.sent[0].message.method, 'ui/initialize');
  assert.equal(state.sent[0].origin, referrer ? origin : '*');
  state.receive({}, 'https://attacker.example', initialize);
  assert.equal(state.sent.length, 1, 'an unrelated window cannot establish the peer');
  if (referrer) {
    state.receive(state.parent, 'https://attacker.example', initialize);
    assert.equal(state.sent.length, 1, 'browser referrer binds the peer before initialization');
  }
  state.receive(state.parent, origin, { jsonrpc: 'not-rpc' });
  state.receive(state.parent, origin, initialize);
  assert.equal(state.document.documentElement.dataset.theme, 'dark');
  assert.equal(state.sent[1].message.method, 'ui/notifications/initialized');
  const targetOrigin = origin === 'null' ? '*' : origin;
  const before = state.sent.length;
  for (const source of [{}, state.parent]) {
    const unwantedOrigin = source === state.parent ? 'https://attacker.example' : origin;
    state.receive(source, unwantedOrigin, cancel);
    state.receive(source, unwantedOrigin, { jsonrpc: '2.0', method: 'ping', id: 9 });
    state.receive(source, unwantedOrigin, { jsonrpc: '2.0', method: 'ui/resource-teardown', id: 10 });
    state.receive(source, unwantedOrigin, { jsonrpc: '2.0', method: 'ui/notifications/host-context-changed', params: { theme: 'light' } });
    state.receive(source, unwantedOrigin, { jsonrpc: '2.0', method: 'ui/notifications/tool-result', params: { isError: true, content: [{ type: 'text', text: 'injected' }] } });
  }
  assert.equal(state.app.textContent, 'Loading data', 'untrusted messages must not modify the view');
  assert.equal(state.sent.length, before, 'untrusted messages must not cause replies');
  assert.equal(state.disconnected(), false, 'untrusted messages must not tear down the view');
  assert.equal(state.document.documentElement.dataset.theme, 'dark');
  assert.equal(state.sent[1].origin, targetOrigin);
  state.receive(state.parent, origin, cancel);
  assert.match(state.app.textContent, /cancelled/);
  state.receive(state.parent, origin, { jsonrpc: '2.0', method: 'ping', id: 11 });
  assert.equal(state.sent.at(-1).message.id, 11);
  assert.equal(state.sent.at(-1).origin, targetOrigin);
  state.receive(state.parent, origin, { jsonrpc: '2.0', method: 'ui/notifications/tool-result', params: { isError: true, content: [{ type: 'text', text: 'expected error' }] } });
  assert.match(state.app.textContent, /expected error/);
  state.receive(state.parent, origin, { jsonrpc: '2.0', method: 'ui/resource-teardown', id: 12 });
  assert.equal(state.disconnected(), true);
  assert.equal(state.sent.at(-1).message.id, 12);
}
console.log('message peer boundary: PASS');
