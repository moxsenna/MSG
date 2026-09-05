#!/usr/bin/env node
import { existsSync, readFileSync } from "node:fs";
if(!existsSync("internal/msg/acquisition/adapter.go")){ console.error("missing adapter.go"); process.exit(1); }
const s=readFileSync("internal/msg/acquisition/adapter.go","utf8");
if(!s.includes("Search(ctx")||!s.includes("RawPlaceSink")||!s.includes("ProgressSink")){ console.error("adapter missing Search interface"); process.exit(1); }
console.log("ADAPTER VERIFIED");
