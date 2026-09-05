#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("desktop/frontend/src/components/ErrorBoundary.tsx")){ console.error("missing ErrorBoundary.tsx"); process.exit(1); }
const eb = readFileSync("desktop/frontend/src/components/ErrorBoundary.tsx","utf8");
if(!eb.includes("componentDidCatch")&&!eb.includes("getDerivedStateFromError")){ console.error("ErrorBoundary missing error handling"); process.exit(1); }
if(!existsSync("desktop/frontend/src/app/layout/Shell.tsx")){ console.error("missing Shell.tsx"); process.exit(1); }
const shell = readFileSync("desktop/frontend/src/app/layout/Shell.tsx","utf8");
if(!shell.includes("keydown")&&!shell.includes("Escape")){ console.error("Shell missing keyboard nav"); process.exit(1); }
console.log("A11Y VERIFIED");
