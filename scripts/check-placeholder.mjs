#!/usr/bin/env node
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
const dir = "desktop/frontend/src/features";
let files = [];
try { files = readdirSync(dir, { recursive: true }).filter(f => f.endsWith("index.tsx")).map(f => join(dir, f)); } catch {}
// Node 18 recursive not supported on older, fallback
if (files.length === 0) {
  import("node:fs").then(m => {
    const walk = (d, out=[]) => { for (const e of m.readdirSync(d, { withFileTypes:true })) { const p=join(d,e.name); if(e.isDirectory()) walk(p,out); else if(e.name==="index.tsx") out.push(p);} return out; };
    files = walk(dir);
    check();
  });
} else check();
function check(){
  for (const p of files) {
    const t = readFileSync(p,"utf8");
    if (t.includes("placeholder for Phase")) { console.error(`placeholder found in ${p}`); process.exit(1); }
    if (p.includes("discover") && t.includes(">= 4.0")) { console.error("discover still contains >= not ≥"); process.exit(1); }
  }
  console.log("NO PLACEHOLDER VERIFIED");
}
if (files.length>0 && !files[0].includes("discover")) { /* sync path already handled */ }
