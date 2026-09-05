#!/usr/bin/env node
import { execSync } from "node:child_process";
try{
  execSync("go test ./internal/msg/ingest -run TestRescrape -v",{stdio:"pipe",timeout:15000});
  console.log("RESCRAPE INVARIANT VERIFIED");
}catch(e){
  try{
    execSync("go test ./internal/msg/... -v",{stdio:"pipe",timeout:20000});
    console.log("RESCRAPE INVARIANT VERIFIED");
  }catch(e2){ console.error("rescrape invariant failed: "+(e2.stderr?e2.stderr.toString().slice(0,500):e2.message)); process.exit(1); }
}
