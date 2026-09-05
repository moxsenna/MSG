#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("desktop/storage/appdata.go")){ console.error("missing desktop/storage/appdata.go"); process.exit(1); }
const s = readFileSync("desktop/storage/appdata.go","utf8");
for(const k of ["AppDataDir","DBPath","LOCALAPPDATA","msg.db"]){
  if(!s.includes(k)){ console.error("appdata.go missing "+k); process.exit(1); }
}
console.log("APPDATA VERIFIED");
