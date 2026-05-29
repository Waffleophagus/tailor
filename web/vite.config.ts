/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

// https://vite.dev/config/
export default defineConfig({
	plugins: [tailwindcss(), svelte()],
	test: {
		include: ['src/**/*.test.ts']
	},
	build: {
		outDir: '../internal/frontend/dist',
		emptyOutDir: true
	},
	server: {
		proxy: {
			'/api': {
				target: 'http://127.0.0.1:8080',
				ws: true
			}
		}
	}
});
