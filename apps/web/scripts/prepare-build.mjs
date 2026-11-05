import { existsSync, mkdirSync, renameSync } from 'node:fs';
import { join, resolve } from 'node:path';

// On Windows the safe-delete shim can reject adapter-node's recursive cleanup
// when an old build directory is present. Move it aside before Vite starts.
const projectRoot = resolve(import.meta.dirname, '..');
const buildDir = join(projectRoot, 'build');
if (existsSync(buildDir)) {
  const tempDir = resolve(projectRoot, '..', '..', '.workbuddy', 'tmp');
  mkdirSync(tempDir, { recursive: true });
  renameSync(buildDir, join(tempDir, `web-build-${Date.now()}`));
}
