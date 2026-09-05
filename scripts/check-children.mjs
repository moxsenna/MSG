#!/usr/bin/env node
import { execSync } from "node:child_process";
const checks = [
  ["node scripts/check-wails-build.mjs","WAILS BUILD VERIFIED"],
  ["node scripts/check-frontend-stack.mjs","FRONTEND STACK VERIFIED"],
  ["node scripts/check-appdata.mjs","APPDATA VERIFIED"],
  ["node scripts/check-adapter.mjs","ADAPTER VERIFIED"],
];
for(const [cmd,expect] of checks){
  try{
    const out=execSync(cmd,{encoding:"utf8"});
    if(!out.includes(expect)){ console.error(cmd+" missing "+expect); process.exit(1); }
  }catch(e){ console.error(cmd+" failed: "+(e.stderr||e.message)); process.exit(1); }
}
console.log("CHILDREN VERIFIED");
