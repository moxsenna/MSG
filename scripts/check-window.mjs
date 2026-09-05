#!/usr/bin/env node
import { readFileSync, existsSync } from "node:fs";
if(!existsSync("cmd/msgdesktop/main.go")){ console.error("missing cmd/msgdesktop/main.go"); process.exit(1); }
const main = readFileSync("cmd/msgdesktop/main.go","utf8");
for(const k of ["Width:     1280","Height:    800","MinWidth","MinHeight","BackgroundColour","AssetServer"]){
  if(!main.includes(k)){ console.error("main.go missing window config "+k); process.exit(1); }
}
if(!existsSync("desktop/frontend/src/app/layout/Shell.tsx")){ console.error("missing Shell.tsx"); process.exit(1); }
console.log("WINDOW VERIFIED");
