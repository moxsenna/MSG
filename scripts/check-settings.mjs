#!/usr/bin/env node
import { existsSync } from "node:fs";
if(!existsSync("desktop/frontend/src/features/settings/index.tsx")){ console.error("missing settings screen"); process.exit(1); }
console.log("SETTINGS VERIFIED");
