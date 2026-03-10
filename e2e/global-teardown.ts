/**
 * global-teardown.ts — stop AlertLens backend after all E2E tests.
 */
import fs from 'fs';

const PID_FILE = '/tmp/alertlens-e2e.pid';

export default async function globalTeardown() {
  const proc = (globalThis as any).__alertlens_proc;
  if (proc) {
    proc.kill('SIGTERM');
    console.log('[e2e] AlertLens stopped');
  } else if (fs.existsSync(PID_FILE)) {
    const pid = parseInt(fs.readFileSync(PID_FILE, 'utf8'), 10);
    try { process.kill(pid, 'SIGTERM'); } catch { /* already gone */ }
    fs.unlinkSync(PID_FILE);
  }
}
