#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("cmd/msgdesktop/main.go")){ console.error("missing main.go"); process.exit(1); }
const m=readFileSync("cmd/msgdesktop/main.go","utf8");
if(!m.includes("noopProgress")&&!m.includes("OnProgress")){ console.error("main.go missing progress sink"); process.exit(1); }
console.log("PROGRESS VERIFIED");
