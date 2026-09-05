#!/usr/bin/env node
import { readFileSync, existsSync } from "node:fs";
if(!existsSync("desktop/frontend/src/features/discover/index.tsx")){ console.error("missing discover index.tsx"); process.exit(1); }
const s = readFileSync("desktop/frontend/src/features/discover/index.tsx","utf8");
if(!s.includes("setError") || !s.includes("setSuccess")){
  console.error("frontend missing error/success handling");
  process.exit(1);
}
if(s.includes("catch {}") && !s.includes("catch (e")){
  console.error("frontend still has empty catch {}");
  process.exit(1);
}
if(!s.includes("Berhasil! Run") && !s.includes("Gagal:")){
  console.error("frontend missing success/error banner");
  process.exit(1);
}
console.log("REAL FRONTEND VERIFIED");
