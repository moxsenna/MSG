#!/usr/bin/env node
import { execSync } from "node:child_process";
try {
  const out = execSync("go test ./internal/msg -run TestNoDocker -v", { encoding: "utf8", timeout: 30000 });
  if (!out.includes("PASS")) throw new Error("guard not PASS");
  console.log("GUARD VERIFIED: no docker run gosom/google-maps-scraper in product code");
} catch (e) { console.error(e.message); process.exit(1); }
