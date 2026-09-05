#!/usr/bin/env node
import { existsSync } from "node:fs";
import { execSync } from "node:child_process";
if(!existsSync("desktop/storage/sqlite/migrate.go")){ console.error("missing migrate.go"); process.exit(1); }
try{
  execSync("go test ./desktop/storage/... -run TestMigrate -v",{stdio:"pipe",timeout:15000});
  execSync("go test ./desktop/storage/... -v",{stdio:"pipe",timeout:15000});
  console.log("MIGRATIONS VERIFIED");
}catch(e){ console.error("migration test failed: "+(e.stderr?e.stderr.toString():e.message)); process.exit(1); }
