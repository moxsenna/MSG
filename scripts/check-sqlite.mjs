#!/usr/bin/env node
import { execSync } from "node:child_process";
import { existsSync } from "node:fs";
for(const f of ["desktop/storage/appdata.go","desktop/storage/sqlite/db.go","desktop/storage/sqlite/migrate.go","desktop/storage/sqlite/migrations/001_init.sql"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
try{
  execSync("go test ./desktop/storage/... -v",{stdio:"pipe", timeout:30000});
  console.log("SQLITE VERIFIED");
}catch(e){ console.error(e.message); process.exit(1); }
