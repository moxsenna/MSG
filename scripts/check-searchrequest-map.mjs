#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("internal/msg/acquisition/types.go")){ console.error("missing types.go"); process.exit(1); }
const t=readFileSync("internal/msg/acquisition/types.go","utf8");
for(const k of ["Query","LocationText","RadiusMeters","SearchRequest"]){
  if(!t.includes(k)){ console.error("types.go missing "+k); process.exit(1); }
}
console.log("SEARCHREQUEST MAP VERIFIED");
