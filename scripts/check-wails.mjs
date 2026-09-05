#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
const checks = [
  "cmd/msgdesktop/main.go",
  "wails.json",
  "desktop/frontend/package.json",
  "desktop/frontend/src/App.tsx",
  "desktop/frontend/src/app/layout/Shell.tsx",
];
for (const f of checks) if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
const shell = readFileSync("desktop/frontend/src/app/layout/Shell.tsx","utf8");
for(const n of ["Home","Discover","Leads","Pipeline","Follow-ups","Activity","Settings"]){
  if(!shell.includes(n)){ console.error("Shell missing "+n); process.exit(1); }
}
console.log("WAILS VERIFIED");
