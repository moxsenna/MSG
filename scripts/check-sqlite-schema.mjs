#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
for(const f of ["desktop/storage/sqlite/db.go","desktop/storage/sqlite/migrate.go","desktop/storage/sqlite/migrations/001_init.sql"]){
  if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
}
const db = readFileSync("desktop/storage/sqlite/db.go","utf8");
for(const k of ["WAL","foreign_keys","busy_timeout"]){
  if(!db.toLowerCase().includes(k.toLowerCase())){ console.error("db.go missing pragma "+k); process.exit(1); }
}
const mig = readFileSync("desktop/storage/sqlite/migrations/001_init.sql","utf8");
for(const t of ["search_runs","source_snapshots","businesses","lead_states"]){
  if(!mig.includes(t)){ console.error("001_init.sql missing table "+t); process.exit(1); }
}
console.log("SQLITE SCHEMA VERIFIED");
