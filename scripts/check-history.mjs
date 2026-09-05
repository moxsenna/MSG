#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("desktop/storage/sqlite/migrations/001_init.sql")){ console.error("missing 001_init.sql"); process.exit(1); }
const m=readFileSync("desktop/storage/sqlite/migrations/001_init.sql","utf8");
if(!m.includes("search_runs")){ console.error("search_runs missing"); process.exit(1); }
console.log("HISTORY VERIFIED");
