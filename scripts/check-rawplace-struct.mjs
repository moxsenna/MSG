#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
import { execSync } from "node:child_process";
const rp = "internal/msg/domain/rawplace.go";
if (!existsSync(rp)) { console.error(rp+" missing"); process.exit(1); }
const t = readFileSync(rp,"utf8");
if (!t.includes('RawPlaceVersion = "v1"')) { console.error("version not v1"); process.exit(1); }
if (!t.includes("Longitude")) { console.error("longitude field missing"); process.exit(1); }
if (t.includes("Longtitude")) { console.error("domain must not have typo Longtitude"); process.exit(1); }
const out = execSync("go test ./internal/msg/acquisition -run TestMapEntry -v",{encoding:"utf8"});
if(!out.includes("PASS")) process.exit(1);
console.log("RAWPLACE STRUCT VERIFIED");
