#!/usr/bin/env node
// Exports the legacy CV data (maulanaiskak-nextjs/src/data/*.ts) as JSON in
// the shape cmd/seedcv expects. Uses the TypeScript compiler already
// installed in that project, so nothing is downloaded.
//
//   node scripts/export-cv-data.mjs ../maulanaiskak.github.io/maulanaiskak-nextjs > legacy-cv.json
//   go run ./cmd/seedcv -file legacy-cv.json

import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import vm from 'node:vm';

const projectDir = path.resolve(process.argv[2] ?? '../maulanaiskak.github.io/maulanaiskak-nextjs');
const require = createRequire(path.join(projectDir, 'package.json'));
const ts = require('typescript');

// Each data file only exports a literal array and imports types, so it can
// be transpiled and evaluated in an empty sandbox with a stub require.
function load(name, exportName) {
  const file = path.join(projectDir, 'src', 'data', `${name}.ts`);
  const { outputText } = ts.transpileModule(readFileSync(file, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 },
  });
  const module = { exports: {} };
  vm.runInNewContext(outputText, { module, exports: module.exports, require: () => ({}) }, { filename: file });
  const value = module.exports[exportName];
  if (!Array.isArray(value)) throw new Error(`${file}: export "${exportName}" is not an array`);
  return value;
}

const out = {
  experiences: load('experience', 'experience'),
  projects: load('projects', 'projects'),
  education: load('education', 'education'),
  skills: load('skills', 'skills'),
};

process.stdout.write(JSON.stringify(out, null, 2) + '\n');
