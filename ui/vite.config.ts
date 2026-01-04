import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	build: {
		rollupOptions: {
			output: {
				// Split heavy libraries into separate chunks to avoid "large chunk" warnings
				// and improve browser caching performance.
				manualChunks: (id) => {
					if (id.includes('node_modules')) {
						// Group all CodeMirror related packages into a single vendor chunk
						if (id.includes('@codemirror') || id.includes('codemirror')) {
							return 'codemirror';
						}
					}
				}
			}
		}
	}
});
