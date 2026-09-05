#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("desktop/frontend/src/app/layout/Shell.tsx")){ console.error("missing Shell.tsx"); process.exit(1); }
if(!existsSync("desktop/frontend/src/App.tsx")){ console.error("missing App.tsx"); process.exit(1); }
const shell = readFileSync("desktop/frontend/src/app/layout/Shell.tsx","utf8");
for(const n of ["Home","Discover","Leads","Pipeline","Follow-ups","Activity","Settings"]){
  if(!shell.includes(n)){ console.error("Shell missing "+n); process.exit(1); }
}
if(!shell.includes("NavLink")||!shell.includes("collapsed")){ console.error("nav missing NavLink or collapse"); process.exit(1); }
const app = readFileSync("desktop/frontend/src/App.tsx","utf8");
for(const r of ["/discover","/leads","/pipeline","/followups","/activity","/settings"]){
  if(!app.toLowerCase().includes(r)){ console.error("App.tsx missing route "+r); process.exit(1); }
}
console.log("NAV VERIFIED");
