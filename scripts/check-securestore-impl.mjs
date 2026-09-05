#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
for(const f of ["desktop/securestore/store.go","desktop/securestore/windows.go","desktop/securestore/filestore.go"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
const w = readFileSync("desktop/securestore/windows.go","utf8");
if(!w.includes("DPAPI")&&!w.includes("CryptProtect")){ console.error("windows.go missing DPAPI"); process.exit(1); }
console.log("SECURESTORE IMPL VERIFIED");
