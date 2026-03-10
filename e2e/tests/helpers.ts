/**
 * Shared helpers for AlertLens E2E security tests.
 */
import { APIRequestContext } from '@playwright/test';

export const BASE = 'http://127.0.0.1:19099';

// TOTP secret matching the one configured in global-setup.ts for the MFA user.
export const MFA_TOTP_SECRET = 'JBSWY3DPEHPK3PXP';

/** Login with password (and optional TOTP code). Returns the JWT token. */
export async function login(
  request: APIRequestContext,
  password: string,
  totpCode?: string,
): Promise<string> {
  // First prime the CSRF token.
  const prime = await request.get(`${BASE}/api/health`);
  const csrfToken = prime.headers()['x-csrf-token'] ?? '';

  const body: Record<string, string> = { password };
  if (totpCode) body['totp_code'] = totpCode;

  const res = await request.post(`${BASE}/api/auth/login`, {
    data: body,
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
  });
  if (!res.ok()) {
    throw new Error(`Login failed ${res.status()}: ${await res.text()}`);
  }
  const json = await res.json();
  return json.token as string;
}

/**
 * Generate a TOTP code for the given base32 secret using Node.js crypto.
 * Implements RFC 6238 (TOTP = HOTP with counter = floor(time/30)).
 */
export async function generateTOTP(secret: string): Promise<string> {
  const { createHmac } = await import('crypto');

  // Decode base32 secret (RFC 4648, no padding).
  const base32Chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
  let bits = '';
  for (const c of secret.toUpperCase()) {
    const idx = base32Chars.indexOf(c);
    if (idx < 0) continue;
    bits += idx.toString(2).padStart(5, '0');
  }
  const bytes = new Uint8Array(Math.floor(bits.length / 8));
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(bits.slice(i * 8, i * 8 + 8), 2);
  }
  const keyBuffer = Buffer.from(bytes);

  // Counter = floor(unix_seconds / 30)
  const counter = Math.floor(Date.now() / 1000 / 30);
  const counterBuf = Buffer.alloc(8);
  counterBuf.writeBigUInt64BE(BigInt(counter));

  const hmac = createHmac('sha1', keyBuffer);
  hmac.update(counterBuf);
  const digest = hmac.digest();

  // Dynamic truncation
  const offset = digest[digest.length - 1] & 0x0f;
  const code = (
    ((digest[offset]     & 0x7f) << 24) |
    ((digest[offset + 1] & 0xff) << 16) |
    ((digest[offset + 2] & 0xff) <<  8) |
     (digest[offset + 3] & 0xff)
  ) % 1_000_000;

  return code.toString().padStart(6, '0');
}
