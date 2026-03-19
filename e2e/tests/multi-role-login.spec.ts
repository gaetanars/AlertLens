/**
 * Multi-role login UX tests for AlertLens — AC-6 coverage.
 *
 * Tests: Navbar "Sign in" label, role badge, logout visibility, role-gated nav items.
 *
 * All tests use Playwright's browser API (page.goto / page.locator) because
 * they verify rendered UI state, not HTTP layer behaviour.
 *
 * DESIGN: Each test performs its own login so browser state is never shared
 * across tests.  Tests run serially (workers: 1 in playwright.config.ts), which
 * naturally spreads the logins over time and avoids the rate-limiter burst limit.
 */
import { test, expect, Page } from '@playwright/test';
import { BASE } from './helpers';
import {
  E2E_VIEWER_PASS,
  E2E_SILENCER_PASS,
  E2E_CONFIG_EDITOR_PASS,
  E2E_ADMIN_PASS,
} from '../global-setup';

// ─── Browser login helper ────────────────────────────────────────────────────

/**
 * Log in via the browser UI login form and wait for the redirect to /alerts.
 * Uses the real form so CSRF, cookie, and store wiring are all exercised.
 */
async function loginViaUI(page: Page, password: string): Promise<void> {
  await page.goto(`${BASE}/login`);
  await page.fill('#password', password);
  await page.click('button[type="submit"]');
  await page.waitForURL(`${BASE}/alerts`, { timeout: 10_000 });
}

// ─── Tests ───────────────────────────────────────────────────────────────────

test.describe('Multi-role login UX', () => {

  // ── 1. Unauthenticated navbar ──────────────────────────────────────────────

  test('unauthenticated: navbar shows "Sign in", not "Admin"', async ({ page }) => {
    await page.goto(`${BASE}/alerts`);

    // "Sign in" login link must be present.
    await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible();

    // The old "Admin" label must not appear anywhere in the header.
    const header = page.locator('header');
    await expect(header.getByText('Admin')).not.toBeVisible();
  });

  // ── 2. Viewer ─────────────────────────────────────────────────────────────

  test('viewer: role badge, logout visible, Config nav absent', async ({ page }) => {
    await loginViaUI(page, E2E_VIEWER_PASS);

    const header = page.locator('header');

    // Role badge shows "viewer".
    await expect(header.getByText('viewer')).toBeVisible();

    // "Sign out" button is visible.
    await expect(page.getByRole('button', { name: /sign out/i })).toBeVisible();

    // Config nav item is not visible for viewer.
    await expect(page.getByRole('link', { name: 'Config' })).not.toBeVisible();
  });

  // ── 3. Silencer ───────────────────────────────────────────────────────────

  test('silencer: role badge, logout visible, silence creation UI visible, Config nav absent', async ({ page }) => {
    await loginViaUI(page, E2E_SILENCER_PASS);

    const header = page.locator('header');

    // Role badge shows "silencer".
    await expect(header.getByText('silencer')).toBeVisible();

    // "Sign out" button is visible.
    await expect(page.getByRole('button', { name: /sign out/i })).toBeVisible();

    // Config nav item is not visible for silencer.
    await expect(page.getByRole('link', { name: 'Config' })).not.toBeVisible();

    // Silence creation UI is visible for silencer.
    // Use client-side nav (click the Silences link) to preserve the in-memory
    // auth token — page.goto() would cause a full reload and clear the store.
    await page.getByRole('link', { name: 'Silences' }).click();
    await page.waitForURL(`${BASE}/silences`);
    await expect(page.getByRole('button', { name: /new silence/i })).toBeVisible();
  });

  // ── 4. Config-editor ──────────────────────────────────────────────────────

  test('config-editor: role badge and Config nav visible', async ({ page }) => {
    await loginViaUI(page, E2E_CONFIG_EDITOR_PASS);

    const header = page.locator('header');

    // Role badge shows "config-editor".
    await expect(header.getByText('config-editor')).toBeVisible();

    // Config nav item is visible for config-editor.
    await expect(page.getByRole('link', { name: 'Config' })).toBeVisible();
  });

  // ── 5. Admin ──────────────────────────────────────────────────────────────

  test('admin: role badge and Config nav visible', async ({ page }) => {
    await loginViaUI(page, E2E_ADMIN_PASS);

    const header = page.locator('header');

    // Role badge shows "admin".
    await expect(header.getByText('admin')).toBeVisible();

    // Config nav item is visible for admin.
    await expect(page.getByRole('link', { name: 'Config' })).toBeVisible();
  });

  // ── 6. Logout flow ────────────────────────────────────────────────────────

  test('logout: returns to unauthenticated state with Sign in link', async ({ page }) => {
    await loginViaUI(page, E2E_VIEWER_PASS);

    // Verify we are authenticated (logout button visible).
    await expect(page.getByRole('button', { name: /sign out/i })).toBeVisible();

    // Click the logout button.
    await page.getByRole('button', { name: /sign out/i }).click();

    // After logout the "Sign in" link must reappear.
    await expect(page.getByRole('link', { name: 'Sign in' })).toBeVisible();

    // The logout button must be gone.
    await expect(page.getByRole('button', { name: /sign out/i })).not.toBeVisible();
  });

});
