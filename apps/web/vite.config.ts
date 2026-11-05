import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [sveltekit({ adapter: adapter() })],
  test: {
    include: ['src/**/*.test.ts']
  }
});
