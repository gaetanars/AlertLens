import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';
import { fileURLToPath, URL } from 'node:url';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
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
