#!/usr/bin/env node
import { execSync } from "node:child_process";
import { existsSync } from "node:fs";
for(const f of ["desktop/securestore/store.go","desktop/securestore/memory.go"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
try{
  execSync("go test ./desktop/securestore/... -v",{stdio:"pipe", timeout:30000});
  console.log("SECURESTORE VERIFIED");
}catch(e){ console.error(e.message); process.exit(1); }
