#!/usr/bin/env node
import { execSync } from "node:child_process";
const out=execSync("go test ./internal/msg -run TestNoDocker -v",{encoding:"utf8"});
if(!out.includes("PASS")){console.error("guard fail");process.exit(1);}
console.log("GUARD FILE VERIFIED");
