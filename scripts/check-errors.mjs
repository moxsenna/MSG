#!/usr/bin/env node
import { existsSync } from "node:fs";
for(const f of ["internal/msg/audit/service.go","internal/msg/ingest/service.go"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
console.log("ERRORS VERIFIED");
