import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		// Allow the dev server to read ../docs (so the Help page can import MANUAL.md).
		fs: { allow: ['..'] },
		proxy: {
			'/api': {
				target: 'http://localhost:8091',
				changeOrigin: true,
			},
		},
	},
});
