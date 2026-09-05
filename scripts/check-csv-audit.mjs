#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
const p="docs/product-v2/CSV_CONTRACT_AUDIT.md";
if(!existsSync(p)){console.error(p+" missing");process.exit(1);}
const t=readFileSync(p,"utf8");
if(!t.includes("review_rating")||!t.includes("link")||!t.includes("mismatch")){console.error("CSV audit incomplete");process.exit(1);}
console.log("CSV AUDIT VERIFIED");
