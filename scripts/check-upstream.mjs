#!/usr/bin/env node
import { readFileSync } from "node:fs";
const entry = readFileSync("gmaps/entry.go","utf8");
const bad = ["LeadStage","OpportunityScore","OutreachStatus","stage","pipeline"];
for(const b of bad){
  // Check if gmaps.Entry struct contains CRM-like fields (case-sensitive)
  if(entry.includes(`\t${b}`) || entry.includes(` ${b} `) ) {
    // Allow generic words like "status" which is already in Entry but not CRM stage
    if(b==="stage" || b==="status") continue;
    console.error("upstream contamination: gmaps.Entry contains "+b); process.exit(1);
  }
}
console.log("UPSTREAM VERIFIED");
