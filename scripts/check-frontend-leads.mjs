#!/usr/bin/env node
import { readFileSync } from "node:fs";
const p = "desktop/frontend/src/features/leads/index.tsx";
let t;
try { t = readFileSync(p, "utf8"); } catch { console.error(`missing ${p}`); process.exit(1); }
const need = ["ListLeads", "UpdateLeadStage", "SetLeadDNC", "AddLeadNote", "useEffect", "pageSize"];
const miss = need.filter(s => !t.includes(s));
if (miss.length) { console.error("leads missing: " + miss.join(", ")); process.exit(1); }
if (t.includes('useState<LeadItem[]>([\n    { id: "1"') && !t.includes("mockLeads")) {
  console.error("leads still hardcode mock without fallback guard");
  process.exit(1);
}
console.log("FRONTEND LEADS VERIFIED");
