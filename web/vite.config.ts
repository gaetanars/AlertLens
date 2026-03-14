import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type Plugin } from 'vitest/config';
import tailwindcss from '@tailwindcss/vite';
import { fileURLToPath, URL } from 'node:url';
import { createHash } from 'node:crypto';
import { readFileSync, writeFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';

/**
 * cspHashPlugin — Vite plugin that runs after the SvelteKit static build.
 *
 * SvelteKit's adapter-static always emits an inline bootstrapper <script> in
 * dist/index.html.  Because that script contains a build-time-generated
 * identifier (__sveltekit_XXXXXX) the hash changes on every build.
 *
 * This plugin:
 *   1. Reads dist/index.html after the build completes.
 *   2. Extracts every inline <script> block.
 *   3. Computes the SHA-256 hash of each block's text content.
 *   4. Writes the hash(es) as a single line to dist/csp-hash.txt
 *      in the format expected by the CSP script-src directive:
 *        'sha256-<base64>' ['sha256-<base64>' …]
 *
 * The Go server embeds dist/ and reads csp-hash.txt at startup to construct
 * the Content-Security-Policy header dynamically, so the hash is always
 * current without hard-coding it in Go source code.
 */
function cspHashPlugin(): Plugin {
	return {
		name: 'csp-hash',
		// closeBundle fires after all output files have been written to disk.
		closeBundle() {
			const configDir = dirname(fileURLToPath(import.meta.url));
			const distDir   = resolve(configDir, '../dist');
			const htmlPath  = resolve(distDir, 'index.html');
			const hashPath  = resolve(distDir, 'csp-hash.txt');

			let html: string;
			try {
				html = readFileSync(htmlPath, 'utf8');
			} catch {
				// During `vite dev` or unit-test runs the dist/ directory may not
				// exist yet — skip silently so non-build commands are unaffected.
				return;
			}

			// Extract the text content of every inline <script> block.
			// The regex captures everything between <script> and </script> that
			// does not have a src= attribute (i.e. inline scripts only).
			const scriptRe = /<script(?![^>]*\bsrc\s*=)[^>]*>([\s\S]*?)<\/script>/gi;
			const hashes: string[] = [];
			let match: RegExpExecArray | null;

			while ((match = scriptRe.exec(html)) !== null) {
				const content = match[1];
				if (content.trim().length === 0) continue;
				const digest = createHash('sha256').update(content).digest('base64');
				hashes.push(`'sha256-${digest}'`);
			}

			if (hashes.length === 0) {
				console.warn('[csp-hash] No inline scripts found in dist/index.html — csp-hash.txt will be empty.');
			}

			writeFileSync(hashPath, hashes.join(' '), 'utf8');
			console.info(`[csp-hash] Wrote ${hashes.length} hash(es) to dist/csp-hash.txt`);
		}
	};
}

export default defineConfig({
	plugins: [tailwindcss(), sveltekit(), cspHashPlugin()],
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:9000',
				changeOrigin: true
			}
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		passWithNoTests: false,
		globals: true,
		environment: 'jsdom',
		setupFiles: ['./src/test-setup.ts'],
		// Svelte 5 uses separate server/browser entry points.
		// Alias svelte to the browser-compatible entry so `mount()` is available in jsdom.
		alias: [
			// Svelte 5 defaults to the server entry in Node.js environments.
			// Point it to the browser client entry so `mount()` is available in jsdom.
			{
				find: /^svelte$/,
				replacement: fileURLToPath(
					new URL('./node_modules/svelte/src/index-client.js', import.meta.url)
				)
			}
		]
	}
});
