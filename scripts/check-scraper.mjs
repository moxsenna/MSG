#!/usr/bin/env node
import { execSync } from "node:child_process";
import { existsSync } from "node:fs";
for(const f of ["internal/msg/acquisition/adapter.go","internal/msg/acquisition/types.go","internal/msg/ingest/service.go"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
try{
  execSync("go test ./internal/msg/acquisition -run TestAdapter -v",{stdio:"pipe", timeout:30000});
  execSync("go test ./internal/msg/ingest -v",{stdio:"pipe", timeout:30000});
  console.log("SCRAPER VERIFIED");
}catch(e){ console.error(e.message); if(e.stdout) console.error(e.stdout.toString()); process.exit(1); }
