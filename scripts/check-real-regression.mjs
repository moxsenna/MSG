#!/usr/bin/env node
import { execSync } from "node:child_process";
try{
  execSync("go vet ./internal/msg/... ./desktop/...",{stdio:"pipe", timeout:20000});
} catch(e){ console.error("vet failed: "+(e.stderr?e.stderr.toString().slice(0,400):e.message)); process.exit(1); }
try{
  execSync("go test ./internal/msg -run TestNoDockerInvocationInProductCode -count=1",{stdio:"pipe", timeout:15000});
} catch(e){ console.error("guard test failed: "+(e.stderr?e.stderr.toString().slice(0,400):e.message)); process.exit(1); }
try{
  execSync("go test ./desktop/storage/... -count=1",{stdio:"pipe", timeout:15000});
} catch(e){ console.error("storage test failed"); process.exit(1); }
console.log("REGRESSION VERIFIED");
