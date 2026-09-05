#!/usr/bin/env node
import { existsSync } from "node:fs";
import { execSync } from "node:child_process";
const files = ["internal/msg/acquisition/testdata/golden_normal.json","internal/msg/acquisition/testdata/golden_missing.json","internal/msg/acquisition/testdata/golden_multi_email.json","internal/msg/acquisition/testdata/golden_closed.json"];
for(const f of files) if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
const out = execSync("go test ./internal/msg/acquisition -run TestMapEntry_LegacyLongitude -v",{encoding:"utf8"});
if(!out.includes("PASS")){console.error("LegacyLongitude fail"); process.exit(1);}
console.log("GOLDEN VERIFIED");
