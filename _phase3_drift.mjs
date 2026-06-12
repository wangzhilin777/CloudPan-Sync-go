import fs from 'node:fs';
const appJs = fs.readFileSync('web/static/app.js', 'utf8');
const lines = appJs.split('\n');

// Build map key -> [values in order]. First dictionary in file is zh-CN.
const seen = new Map();
let curNs = null;
const reNs = /^\s{4,8}([a-zA-Z_]+):\s*\{\s*$/;
const reKv = /^\s{6,}([a-zA-Z_0-9]+):\s*"((?:[^"\\]|\\.)*)",?\s*$/;
for (const ln of lines) {
  const ns = ln.match(reNs);
  if (ns) { curNs = ns[1]; continue; }
  const kv = ln.match(reKv);
  if (kv && curNs) {
    const k = curNs + '.' + kv[1];
    if (!seen.has(k)) seen.set(k, []);
    seen.get(k).push(kv[2]);
  }
}

const tRe = /\bt\(\s*"([^"]+)"\s*,\s*"((?:[^"\\]|\\.)*)"\s*\)/g;
let m;
const out = [];
while ((m = tRe.exec(appJs))) {
  const key = m[1], fb = m[2];
  const vals = seen.get(key);
  if (!vals || !vals.length) continue;
  const zh = vals[0];
  if (zh !== fb) out.push({ key, zh, fb });
}
const uniq = new Map();
for (const o of out) uniq.set(o.key + '|' + o.fb, o);
console.log('--- fallback drift (fb != zh dict), unique:', uniq.size);
[...uniq.values()].forEach((o, i) => {
  console.log(`${i + 1}  ${o.key}`);
  console.log(`     dict: ${JSON.stringify(o.zh)}`);
  console.log(`     fb  : ${JSON.stringify(o.fb)}`);
});
