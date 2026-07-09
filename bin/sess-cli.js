#!/usr/bin/env node
// npx wrapper — delegates to the real bash script
const { execFileSync } = require('child_process');
const path = require('path');

// The bash script is next to this file
const script = path.join(__dirname, 'sess');
try {
  execFileSync(script, process.argv.slice(2), {
    stdio: 'inherit',
    env: { ...process.env }
  });
} catch (e) {
  process.exit(e.status || 1);
}