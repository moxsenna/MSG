#!/usr/bin/env node
import { readFileSync, existsSync, readdirSync, statSync } from "node:fs";
import { execSync } from "node:child_process";
import { join } from "node:path";
try {
  const mod = readFileSync("go.mod","utf8");
  if (!mod.includes("github.com/wailsapp/wails/v2")) { console.error("wails not in go.mod"); process.exit(1); }
  execSync("go vet ./internal/msg/... ./desktop/... ./cmd/msgdesktop/...", { stdio:"pipe", timeout:60000 });
  function walk(dir, out=[]) {
    for (const e of readdirSync(dir, { withFileTypes:true })) {
      const p = join(dir, e.name);
      if (e.isDirectory()) walk(p, out);
      else if (e.name.endsWith(".go") && e.name !== "guard_test.go") out.push(p);
    }
    return out;
  }
  for (const dir of ["internal/msg", "desktop", "cmd/msgdesktop"]) {
    for (const f of walk(dir)) {
      const txt = readFileSync(f, "utf8");
      if (txt.includes("docker run") && txt.includes("gosom/google-maps-scraper")) {
        console.error(`docker run found in product code ${f}`);
        process.exit(1);
      }
    }
  }
  console.log("RUNTIME VERIFIED");
} catch(e){
  console.error((e.stdout||Buffer.alloc(0)).toString() + (e.stderr||Buffer.alloc(0)).toString() || e.message);
  process.exit(1);
}
