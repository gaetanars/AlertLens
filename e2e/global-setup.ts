/**
 * global-setup.ts — start AlertLens backend before all E2E tests.
 *
 * The test binary path is read from ALERTLENS_BIN (default: /tmp/alertlens-bin).
 * A minimal config is written to /tmp/alertlens-e2e-config.yaml:
 *   - Port 19099 (avoids conflict with any real instance on 9000)
 *   - Single admin user password "e2e-admin-pass"
 *   - A viewer user "e2e-viewer-pass"
 *   - A MFA user with a known TOTP secret (for MFA tests)
 *   - A fake alertmanager URL (we test auth/security layers, not AM proxy)
 */
import { execSync, spawn } from 'child_process';
import fs from 'fs';
import path from 'path';

const BINARY   = process.env.ALERTLENS_BIN ?? '/tmp/alertlens-bin';
const CONFIG   = '/tmp/alertlens-e2e-config.yaml';
const PID_FILE = '/tmp/alertlens-e2e.pid';
const PORT     = 19099;

// Fixed TOTP secret for MFA tests (pre-generated base32 secret).
// Corresponding TOTP codes are generated in the tests using the same secret.
export const MFA_TOTP_SECRET = 'JBSWY3DPEHPK3PXP'; // "Hello!" in base32

const CONFIG_YAML = `
server:
  host: "127.0.0.1"
  port: ${PORT}
auth:
  admin_password: "e2e-admin-pass"
  users:
    - password: "e2e-viewer-pass"
      role: "viewer"
    - password: "e2e-mfa-pass"
      role: "admin"
      totp_secret: "${MFA_TOTP_SECRET}"
alertmanagers:
  - name: "test-am"
    url: "http://127.0.0.1:19999"
`.trim();

export default async function globalSetup() {
  // Build the binary if it doesn't exist.
  if (!fs.existsSync(BINARY)) {
    console.log('[e2e] Building AlertLens binary...');
    execSync('go build -o ' + BINARY + ' .', {
      cwd: path.resolve(__dirname, '..'),
      stdio: 'inherit',
    });
  }

  // Write test config.
  fs.writeFileSync(CONFIG, CONFIG_YAML, 'utf8');

  // Start the backend.
  console.log(`[e2e] Starting AlertLens on port ${PORT}...`);
  const proc = spawn(BINARY, ['-config', CONFIG], {
    detached: false,
    stdio: ['ignore', 'pipe', 'pipe'],
  });

  // Write PID so globalTeardown can kill it.
  if (proc.pid === undefined) {
    throw new Error('[e2e] Failed to spawn AlertLens — is ALERTLENS_BIN set correctly? (' + BINARY + ')');
  }
  fs.writeFileSync(PID_FILE, String(proc.pid), 'utf8');

  proc.stderr?.on('data', (d: Buffer) => {
    if (process.env.ALERTLENS_E2E_VERBOSE) process.stderr.write(d);
  });
  proc.stdout?.on('data', (d: Buffer) => {
    if (process.env.ALERTLENS_E2E_VERBOSE) process.stdout.write(d);
  });

  // Store process reference for globalTeardown.
  (globalThis as any).__alertlens_proc = proc;

  // Wait until the health endpoint responds (up to 10 s).
  const baseURL = `http://127.0.0.1:${PORT}`;
  const deadline = Date.now() + 10_000;
  while (Date.now() < deadline) {
    try {
      const res = await fetch(`${baseURL}/api/health`);
      if (res.ok) {
        console.log(`[e2e] AlertLens ready at ${baseURL}`);
        return;
      }
    } catch {
      // Not ready yet.
    }
    await sleep(200);
  }
  throw new Error('[e2e] AlertLens did not become ready within 10 s');
}

function sleep(ms: number) {
  return new Promise(resolve => setTimeout(resolve, ms));
}
