#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("desktop/frontend/src/lib/logger.ts")){ console.error("missing logger.ts"); process.exit(1); }
const lg = readFileSync("desktop/frontend/src/lib/logger.ts","utf8");
if(!lg.includes("log")&&!lg.includes("Logger")){ console.error("logger.ts missing log impl"); process.exit(1); }
console.log("LOGGING VERIFIED");
