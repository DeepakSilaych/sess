#!/usr/bin/env node
'use strict';

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const crypto = require('node:crypto');
const { spawnSync } = require('node:child_process');
const release = require('./releases.json');

function target(platform = process.platform, arch = process.arch) {
  const mapped = { x64: 'amd64', arm64: 'arm64' }[arch];
  if (!['darwin', 'linux'].includes(platform) || !mapped) {
    throw new Error(`Unsupported platform ${platform}/${arch}. sess supports macOS and Linux on ARM64 and AMD64.`);
  }
  return `sess-${platform}-${mapped}.tar.gz`;
}

function verify(bytes, expected) {
  if (crypto.createHash('sha256').update(bytes).digest('hex') !== expected) {
    throw new Error('Release checksum mismatch. Download discarded; please retry.');
  }
}

async function install() {
  const asset = target();
  const digest = release.sha256[asset];
  if (!digest) throw new Error(`No pinned checksum for ${asset}`);
  const base = process.env.XDG_CACHE_HOME || path.join(os.homedir(), '.cache');
  const dir = path.join(base, 'sess', 'npm', release.version, digest);
  const binary = path.join(dir, 'sess');
  if (fs.existsSync(binary)) return binary;
  fs.mkdirSync(dir, { recursive: true, mode: 0o700 });
  const temp = fs.mkdtempSync(path.join(dir, '.download-'));
  try {
    process.stderr.write(`sess: downloading v${release.version} for ${process.platform}/${process.arch}\n`);
    const url = `https://github.com/DeepakSilaych/sess/releases/download/v${release.version}/${asset}`;
    const response = await fetch(url, { signal: AbortSignal.timeout(120000) });
    if (!response.ok) throw new Error(`Download failed: HTTP ${response.status}`);
    const bytes = Buffer.from(await response.arrayBuffer());
    verify(bytes, digest);
    const archive = path.join(temp, asset);
    fs.writeFileSync(archive, bytes, { mode: 0o600 });
    const extracted = spawnSync('tar', ['-xzf', archive, '-C', temp, 'sess'], { encoding: 'utf8' });
    if (extracted.error || extracted.status !== 0) {
      throw new Error(`Cannot extract release; ensure tar is installed. ${extracted.error?.message || extracted.stderr}`);
    }
    fs.chmodSync(path.join(temp, 'sess'), 0o755);
    fs.renameSync(path.join(temp, 'sess'), binary);
    return binary;
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
}

async function main() {
  const binary = await install();
  // Replace Node when available so SSH receives the original terminal and signals.
  if (process.execve) process.execve(binary, [binary, ...process.argv.slice(2)], process.env);
  const child = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit' });
  if (child.error) throw child.error;
  if (child.signal) {
    process.kill(process.pid, child.signal);
    return;
  }
  process.exitCode = child.status ?? 1;
}

module.exports = { target, verify };
if (require.main === module) {
  main().catch(error => {
    process.stderr.write(`sess: ${error.message}\n`);
    process.exitCode = 1;
  });
}
