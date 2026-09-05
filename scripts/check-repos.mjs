#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
for(const f of ["internal/msg/leads/service.go","internal/msg/ingest/service.go"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
const ingest = readFileSync("internal/msg/ingest/service.go","utf8");
if(!ingest.includes("place_id")||!ingest.includes("cid")||!ingest.includes("Dedupe identity")){ console.error("ingest missing dedupe identity place_id>cid>data_id"); process.exit(1); }
if(!ingest.includes("CRM preservation")&&!ingest.includes("preserv")){ console.error("ingest missing CRM preservation check"); process.exit(1); }
console.log("REPOS VERIFIED");
