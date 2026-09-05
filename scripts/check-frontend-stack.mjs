#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
for(const f of ["desktop/frontend/package.json","desktop/frontend/vite.config.ts","desktop/frontend/tsconfig.json","desktop/frontend/src/main.tsx","desktop/frontend/src/App.tsx"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
const pkg = JSON.parse(readFileSync("desktop/frontend/package.json","utf8"));
const deps = {...(pkg.dependencies||{}), ...(pkg.devDependencies||{})};
for(const d of ["react","react-dom","react-router-dom","@fluentui/react-components","vite","typescript"]){
  if(!deps[d]){ console.error("missing dep "+d); process.exit(1); }
}
const ts = readFileSync("desktop/frontend/tsconfig.json","utf8");
if(!ts.includes('"strict": true')){ console.error("tsconfig strict not true"); process.exit(1); }
if(!existsSync("desktop/frontend/src/app/layout/Shell.tsx")){ console.error("missing Shell.tsx"); process.exit(1); }
console.log("FRONTEND STACK VERIFIED");
