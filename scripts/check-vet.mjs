#!/usr/bin/env node
import { execSync } from "node:child_process";
try {
  execSync("go vet ./internal/msg/... ./desktop/... ./cmd/msgdesktop/...", { stdio: "pipe", timeout: 60000 });
  console.log("VET VERIFIED");
} catch (e) {
  const out = (e.stdout || Buffer.alloc(0)).toString() + (e.stderr || Buffer.alloc(0)).toString();
  console.error(out.slice(0, 4000) || e.message);
  process.exit(1);
}
