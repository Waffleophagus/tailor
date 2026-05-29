import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import dotenv from 'dotenv';

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const envPath = path.join(webRoot, '.env');

if (fs.existsSync(envPath)) {
	dotenv.config({ path: envPath });
}
