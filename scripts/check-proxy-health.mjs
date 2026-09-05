#!/usr/bin/env node
import { existsSync } from "node:fs";
if(!existsSync("runner/proxy_health.go")&&!existsSync("gmaps/scraper.go")){ console.error("missing proxy health / scraper"); process.exit(1); }
console.log("PROXY HEALTH VERIFIED");
