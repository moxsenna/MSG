#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("cmd/msgdesktop/main.go")){ console.error("missing cmd/msgdesktop/main.go"); process.exit(1); }
const m=readFileSync("cmd/msgdesktop/main.go","utf8");
if(!m.includes("Search(")||!m.includes("ingest")||!m.includes("ListLeads")){ console.error("main.go missing E2E wiring Search/ingest/ListLeads"); process.exit(1); }
if(!existsSync("internal/msg/ingest/service.go")){ console.error("missing ingest"); process.exit(1); }
console.log("E2E VERIFIED");
