#!/usr/bin/env node
import { readFileSync, existsSync } from "node:fs";
if(!existsSync("internal/msg/acquisition/adapter.go")){ console.error("missing adapter.go"); process.exit(1); }
const s = readFileSync("internal/msg/acquisition/adapter.go","utf8");
// real wiring must involve filerunner or runner + scrapemate and NOT be the demo synthetic block as primary path
// Check that it no longer returns synthetic 5 seeds for real queries as the final return
// The old synthetic block had: `// Demo synthetic generation: create 3-5 plausible leads` and `seeds := []struct`
if(s.includes("Demo synthetic generation") && s.includes("n := 3") && s.includes("seeds :=") && s.includes("return nil") && !s.includes("filerunner.New")){
  console.error("adapter still synthetic demo fallback, not real filerunner");
  process.exit(1);
}
// Check it now contains real filerunner wiring
if(!s.includes("filerunner") && !s.includes("runner.New") && !s.includes("scrapemate.New")){
  console.error("adapter missing real filerunner/scrapemate wiring");
  process.exit(1);
}
// Check it still handles FAKE: for tests
if(!s.includes("FAKE:")){
  console.error("adapter missing FAKE: handling for tests");
  process.exit(1);
}
console.log("REAL ADAPTER WIRED");
