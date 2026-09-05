#!/usr/bin/env node
import { readFileSync } from "node:fs";
const path = "cmd/msgdesktop/main.go";
let text;
try { text = readFileSync(path, "utf8"); } catch { console.error(`missing ${path}`); process.exit(1); }
const required = ["func (a *App) Search(", "func (a *App) ListLeads(", "func (a *App) GetLead(", "func (a *App) UpdateLeadStage(", "func (a *App) SetLeadDNC(", "func (a *App) AddLeadNote(", "func (a *App) AddLeadTag(", "func (a *App) GetKanbanBoard(", "func (a *App) MovePipelineCard(", "func (a *App) ListFollowUps("];
const missing = required.filter(s => !text.includes(s));
if (missing.length) { console.error("missing bindings: " + missing.join(", ")); process.exit(1); }
if (!text.includes("ingestSink") || !text.includes("OnPlace")) { console.error("ingest sink not wired"); process.exit(1); }
console.log("WAILS BINDINGS VERIFIED");
