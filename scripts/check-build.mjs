#!/usr/bin/env node
import { execSync } from "node:child_process";
try {
  execSync("go test ./internal/msg/... ./desktop/... -count=1", { stdio: "pipe", timeout: 120000 });
  execSync("npm --prefix desktop/frontend run build", { stdio: "pipe", timeout: 120000 });
  console.log("BUILD VERIFIED");
} catch (e) {
  const out = (e.stdout || Buffer.alloc(0)).toString() + (e.stderr || Buffer.alloc(0)).toString();
  console.error(out.slice(0, 6000) || e.message);
  process.exit(1);
}
