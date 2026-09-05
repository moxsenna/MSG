#!/usr/bin/env node
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
function walk(dir, out=[]){
  try{
    for(const e of readdirSync(dir)){
      const p=join(dir,e);
      const s=statSync(p);
      if(s.isDirectory()){
        if(e==="node_modules"||e.startsWith(".")||e==="build") continue;
        walk(p,out);
      } else if(p.endsWith(".go")){
        out.push(p);
      }
    }
  }catch{}
  return out;
}
const files=[...walk("cmd/msgdesktop"), ...walk("desktop"), ...walk("internal/msg")];
let bad="";
for(const f of files){
  const c=readFileSync(f,"utf8");
  if(c.includes("alie-core")||c.includes("BullMQ")||c.includes("docker run gosom")){
    bad+=f+": sidecar\n";
  }
  if(f==="cmd/msgdesktop/main.go" && (c.includes(":8080")||c.includes("ListenAndServe"))){
    bad+=f+": 8080 sidecar\n";
  }
}
if(bad){ console.error("sidecar found:\n"+bad.slice(0,400)); process.exit(1); }
console.log("NO SIDECAR VERIFIED");
