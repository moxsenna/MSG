#!/usr/bin/env node
import { readFileSync } from "node:fs";
const p = "desktop/frontend/src/features/pipeline/index.tsx";
let t;
try { t = readFileSync(p, "utf8"); } catch { console.error(`missing ${p}`); process.exit(1); }
const need = ["GetKanbanBoard", "MovePipelineCard", "useEffect", "STAGES"];
const miss = need.filter(s => !t.includes(s));
if (miss.length) { console.error("pipeline missing: " + miss.join(", ")); process.exit(1); }
if (t.includes('useState<Record<string, PipelineCard[]>>({') && !t.includes("mockBoard")) {
  console.error("pipeline still hardcode without fallback guard");
  process.exit(1);
}
console.log("FRONTEND PIPELINE VERIFIED");
