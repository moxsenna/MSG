#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
const p="UPSTREAM_TOUCHES_V2.md";
if(!existsSync(p)){console.error(p+" missing");process.exit(1);}
const t=readFileSync(p,"utf8");
if(!t.includes("2a02908")||!t.includes("CentralWriter")){console.error("ledger missing hardening");process.exit(1);}
if(t.includes("gmaps.Entry") && t.includes("CRM")){ /* product side only check */ }
console.log("UPSTREAM LEDGER VERIFIED");
