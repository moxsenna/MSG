#!/usr/bin/env node
import { existsSync } from "node:fs";
import { execSync } from "node:child_process";
const pw = process.env.LOCALAPPDATA + "\\ms-playwright";
if(!existsSync(pw) && !existsSync("C:\\Users\\" + process.env.USERNAME + "\\AppData\\Local\\ms-playwright")){ console.error("ms-playwright not found"); process.exit(1); }
// check chromium exists
try{
  execSync("npx playwright --version",{stdio:"pipe", timeout:8000});
} catch(e){ console.error("playwright version failed"); process.exit(1); }
// check go playwright module
try{
  const out = execSync("go list -m all",{encoding:"utf8"});
  if(!out.includes("playwright-go")){ console.error("playwright-go not in go.mod"); process.exit(1); }
} catch(e){ console.error("go list failed"); process.exit(1); }
console.log("PLAYWRIGHT READY");
