#!/usr/bin/env node
import { readFileSync } from "node:fs";
const p = "desktop/frontend/src/features/discover/index.tsx";
let t;
try { t = readFileSync(p, "utf8"); } catch { console.error(`missing ${p}`); process.exit(1); }
const need = ["Search", "location_text", "radius_meters", "speed_preset", "go?.main?.App?.Search"];
const miss = need.filter(s => !t.includes(s));
if (miss.length) { console.error("discover missing: " + miss.join(", ")); process.exit(1); }
if (!t.includes("setFound") || !t.includes("setRunning")) { console.error("discover progress not wired"); process.exit(1); }
console.log("FRONTEND DISCOVER VERIFIED");
