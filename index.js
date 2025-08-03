#!/usr/bin/env node
const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const binaryPath = path.join(__dirname, 'mcp-grafana');
if (!fs.existsSync(binaryPath)) {
  console.log('Building Go binary...');
  const build = spawn('go', ['build', '-o', 'mcp-grafana', '.'], { stdio: 'inherit', cwd: __dirname });
  build.on('close', (code) => {
    if (code === 0) {
      const child = spawn(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
      child.on('close', (code) => process.exit(code));
    } else {
      console.error('Build failed');
      process.exit(1);
    }
  });
} else {
  const child = spawn(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
  child.on('close', (code) => process.exit(code));
}
