#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
import { execSync } from "node:child_process";
const must = ["cmd/msgdesktop/main.go","wails.json","desktop/frontend/assets.go"];
for(const f of must) if(!existsSync(f)){ console.error("missing "+f); process.exit(1); }
const wails = JSON.parse(readFileSync("wails.json","utf8"));
if(wails["main"] !== "./cmd/msgdesktop"){ console.error("wails.json main should be ./cmd/msgdesktop got "+wails["main"]); process.exit(1); }
const main = readFileSync("cmd/msgdesktop/main.go","utf8");
if(main.includes("docker run")||main.includes("Redis")||main.includes("postgres")){ console.error("cmd/msgdesktop should not import docker/redis/postgres"); process.exit(1); }
if(!main.includes("frontend.Assets")||!main.includes("wails.Run")){ console.error("main.go missing frontend.Assets or wails.Run"); process.exit(1); }
if(!existsSync("desktop/frontend/dist/index.html")){ console.error("missing desktop/frontend/dist/index.html - run npm run build"); process.exit(1); }
try{ execSync("go vet ./cmd/msgdesktop/...",{stdio:"pipe",timeout:15000}); }catch(e){ console.error("go vet failed: "+e.message); process.exit(1); }
console.log("WAILS BUILD VERIFIED");
