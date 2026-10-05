// CI-only: resolve the native binary in the already installed official npm package.
// No Node launcher is needed by the Go tests, whose child PATH can be empty.
import { accessSync, appendFileSync, constants, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { execFileSync } from 'node:child_process';

const prefix = process.argv[2];
if (!prefix || !process.env.GITHUB_ENV) {
  throw new Error('Usage in GitHub Actions: node setup-ci-codex.mjs ABSOLUTE_NPM_PREFIX');
}
const vendor = join(prefix, 'node_modules', '@openai', `codex-${process.platform}-${process.arch}`, 'vendor');
const targets = readdirSync(vendor, { withFileTypes: true }).filter(entry => entry.isDirectory());
if (targets.length !== 1) throw new Error('Expected one native Codex target in the platform package');
const binary = join(vendor, targets[0].name, 'bin', process.platform === 'win32' ? 'codex.exe' : 'codex');
accessSync(binary, constants.X_OK);
const version = execFileSync(binary, ['--version'], { encoding: 'utf8' }).trim();
if (version !== 'codex-cli 0.155.1') throw new Error(`Unexpected Codex test host version: ${version}`);
appendFileSync(process.env.GITHUB_ENV, `TODO_CONNECT_TEST_CODEX=${binary}\n`);
console.log(`${version} (${process.platform}/${process.arch}); isolated plugin-host checks enabled`);
