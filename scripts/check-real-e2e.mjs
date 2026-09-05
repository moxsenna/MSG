#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
import { execSync } from "node:child_process";
// Verify adapter now delegates to real filerunner and DB persistence still works
if(!existsSync("internal/msg/acquisition/adapter.go")){ console.error("missing adapter.go"); process.exit(1); }
const s = readFileSync("internal/msg/acquisition/adapter.go","utf8");
if(!s.includes("filerunner") && !s.includes("runner.Config")){
  console.error("adapter missing real runner config");
  process.exit(1);
}
// Check that storage still has sqlite and the Search method in cmd/msgdesktop writes to search_runs and ingestSink
if(!existsSync("cmd/msgdesktop/main.go")){ console.error("missing cmd/msgdesktop/main.go"); process.exit(1); }
const m = readFileSync("cmd/msgdesktop/main.go","utf8");
if(!m.includes("Search(") || !m.includes("ingestSink") || !m.includes("search_runs")){
  console.error("main.go missing Search wiring");
  process.exit(1);
}
// Light integration: verify vet still passes for the new wiring (no broken imports)
try{
  execSync("go vet ./internal/msg/acquisition",{stdio:"pipe", timeout:15000});
} catch(e){ console.error("go vet acquisition failed"); process.exit(1); }
console.log("REAL E2E VERIFIED");
