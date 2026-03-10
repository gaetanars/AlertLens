import { defineConfig } from '@playwright/test';
import path from 'path';

/**
 * AlertLens E2E test configuration.
 *
 * Security-focused tests (CSRF, CSP, JWT, MFA, YAML validation, RBAC)
 * run against a real AlertLens backend started via globalSetup.
 *
 * All tests use Playwright's request context (no browser required for
 * API-level security assertions).  Browser-based tests (login page)
 * use the bundled Chromium.
 */
export default defineConfig({
  testDir: './tests',
  timeout: 30_000,
  retries: 0,
  workers: 1, // Serial: tests share a single backend instance

  use: {
    // Base URL of the AlertLens backend started by globalSetup.
    baseURL: 'http://127.0.0.1:19099',
    // API tests don't need a real browser; keep for UI tests.
    headless: true,
    ignoreHTTPSErrors: false,
    extraHTTPHeaders: {
      'Accept': 'application/json',
    },
  },

  globalSetup: path.resolve('./global-setup.ts'),
  globalTeardown: path.resolve('./global-teardown.ts'),

  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
});
