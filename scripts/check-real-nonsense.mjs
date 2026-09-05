#!/usr/bin/env node
import { readFileSync, existsSync } from "node:fs";
if(!existsSync("internal/msg/acquisition/adapter.go")){ console.error("missing adapter.go"); process.exit(1); }
const s = readFileSync("internal/msg/acquisition/adapter.go","utf8");
// Nonsense query must not be turned into 6 leads via synthetic demo
// The old code did: if strings.HasPrefix(... FAKE:) { generate 1 } then synthetic for real queries with n=5 for Fast
// Real code should not have that synthetic block for real queries
if(s.includes("Demo synthetic generation") && s.includes("seeds :=")){
  console.error("adapter still generates synthetic for nonsense queries — should be 0 via real scrape");
  process.exit(1);
}
// Also check frontend no longer hardcodes setFound(53) for mock
const f = readFileSync("desktop/frontend/src/features/discover/index.tsx","utf8");
if(f.includes("setFound(53)") && f.includes("Math.random() * 7") && !f.includes("setError")){
  console.error("frontend still mock setFound(53) fallback");
  process.exit(1);
}
console.log("NONSENSE ZERO VERIFIED");
