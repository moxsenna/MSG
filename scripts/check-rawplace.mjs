#!/usr/bin/env node
import { execSync } from "node:child_process";
try {
  const out = execSync("go test ./internal/msg/acquisition -run TestMapEntry -v", { encoding: "utf8", timeout: 30000 });
  if (!out.includes("PASS")) throw new Error("tests did not PASS");
  if (!out.includes("TestMapEntry_LegacyLongitude")) throw new Error("LegacyLongitude test missing");
  const guard = execSync("go test ./internal/msg -run TestNoDocker -v", { encoding: "utf8", timeout: 30000 });
  if (!guard.includes("PASS")) throw new Error("guard failed");
  console.log("RAWPLACE VERIFIED: 9 mapper tests PASS + guard PASS");
} catch (e) { console.error(e.message); if(e.stdout) console.error(e.stdout.toString()); process.exit(1); }
