const { test } = require('node:test');
const assert = require('node:assert/strict');
const { createHash } = require('node:crypto');
const { target, verify } = require('./sess.cjs');

test('maps all supported platforms and rejects unsupported ones', () => {
  for (const platform of ['darwin', 'linux']) {
    assert.equal(target(platform, 'x64'), `sess-${platform}-amd64.tar.gz`);
    assert.equal(target(platform, 'arm64'), `sess-${platform}-arm64.tar.gz`);
  }
  assert.throws(() => target('win32', 'x64'), /Unsupported platform/);
  assert.throws(() => target('linux', 'ia32'), /Unsupported platform/);
});

test('rejects modified downloads before extraction', () => {
  const bytes = Buffer.from('release');
  const digest = createHash('sha256').update(bytes).digest('hex');
  verify(bytes, digest);
  assert.throws(() => verify(Buffer.from('modified'), digest), /checksum mismatch/);
});
