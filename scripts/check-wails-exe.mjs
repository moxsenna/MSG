#!/usr/bin/env node
import { existsSync, readFileSync, statSync } from "node:fs";
const main = readFileSync("cmd/msgdesktop/main.go","utf8");
if (!main.includes("frontend.Assets")) { console.error("main.go not using frontend.Assets embed"); process.exit(1); }
if (!existsSync("desktop/frontend/assets.go")) { console.error("missing desktop/frontend/assets.go with //go:embed dist"); process.exit(1); }
const assetSrc = readFileSync("desktop/frontend/assets.go","utf8");
if (!assetSrc.includes("//go:embed dist")) { console.error("assets.go missing //go:embed dist"); process.exit(1); }
if (!existsSync("desktop/frontend/dist/index.html")) { console.error("missing desktop/frontend/dist/index.html — run npm run build"); process.exit(1); }
if (existsSync("build/bin/MSGDesktop.exe")) {
  const sz = statSync("build/bin/MSGDesktop.exe").size;
  if (sz < 10*1024*1024) { console.error(`exe too small ${sz} bytes`); process.exit(1); }
}
console.log("EXE VERIFIED");
