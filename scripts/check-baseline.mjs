#!/usr/bin/env node
import { execSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
try {
  const log = execSync("git log --oneline -20", { encoding: "utf8" });
  const hasBase = log.includes("39a562b") || log.includes("39a562bdede752225a8eeb171421b087580745e1");
  const hasHardening = log.includes("2a02908");
  const baselineMd = existsSync("docs/product-v2/BASELINE.md");
  if (!hasBase) throw new Error("baseline 39a562b not found in git log");
  if (!hasHardening) throw new Error("hardening 2a02908 not found");
  if (!baselineMd) throw new Error("docs/product-v2/BASELINE.md missing");
  console.log("BASELINE VERIFIED: 39a562b + 2a02908 present, BASELINE.md exists");
} catch (e) { console.error(e.message); process.exit(1); }
