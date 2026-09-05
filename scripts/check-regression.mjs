#!/usr/bin/env node
import { execSync } from "node:child_process";
try{
  execSync("go vet ./internal/msg/... ./desktop/...",{stdio:"pipe",timeout:20000});
  execSync("go test ./internal/msg/... -count=1",{stdio:"pipe",timeout:30000});
  console.log("REGRESSION VERIFIED");
}catch(e){ console.error("regression failed: "+(e.stderr?e.stderr.toString().slice(0,800):e.message)); process.exit(1); }
