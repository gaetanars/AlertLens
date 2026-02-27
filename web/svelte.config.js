import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({
			// Output to ../dist so Go's go:embed picks it up directly.
			pages: '../dist',
			assets: '../dist',
			fallback: 'index.html',
			precompress: false,
			strict: false
		}),
		paths: {
			base: ''
		}
	}
};

export default config;
