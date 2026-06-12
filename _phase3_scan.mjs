import fs from 'node:fs';
import path from 'node:path';

const repo = process.cwd();
const testDir = path.join(repo, 'internal', 'app');

// 1) 收集 internal/app 里所有 strings.Contains(body, "...") 锚点字符串
const anchors = new Set();
function walk(dir) {
  for (const name of fs.readdirSync(dir)) {
    const p = path.join(dir, name);
    const st = fs.statSync(p);
    if (st.isDirectory()) walk(p);
    else if (name.endsWith('.go')) {
      const text = fs.readFileSync(p, 'utf8');
      const re = /strings\.Contains\(\s*body\s*,\s*"((?:[^"\\]|\\.)*)"\s*\)/g;
      let m;
      while ((m = re.exec(text))) {
        const raw = m[1].replace(/\\"/g, '"').replace(/\\\\/g, '\\');
        anchors.add(raw);
      }
    }
  }
}
walk(testDir);

console.log('--- anchors count:', anchors.size);
const sample = [...anchors].filter((s) => /[\u4e00-\u9fff]/.test(s)).slice(0, 50);
console.log('--- chinese anchors sample (first 50):');
for (const s of sample) console.log('  ', JSON.stringify(s));

// 2) 扫 web/static/app.js 里所有 t("ns.key", "fallback") 的 fallback 默认值
const appJs = fs.readFileSync(path.join(repo, 'web', 'static', 'app.js'), 'utf8');
const tRe = /\bt\(\s*"([^"]+)"\s*,\s*"((?:[^"\\]|\\.)*)"\s*\)/g;
const fallbacks = [];
let m2;
while ((m2 = tRe.exec(appJs))) {
  fallbacks.push({ key: m2[1], val: m2[2] });
}
console.log('--- fallback total:', fallbacks.length);

// 3) 找出 fallback 含英文运营词且不是技术专名的候选
const TECH = /(JSON|Cookie|OpenList|Alist|Smoke Matrix|Provider 默认风控校准|providerData|uploadId|nextPart|retrySummary|retryClass|retryLimit|riskOverride|retry_queue_auto_retry|upload_checkpoint|upload checkpoint|upload session|uploadid|provider_session_missing|pending_manual)/i;
const candidatesRaw = fallbacks.filter((f) => /[\u4e00-\u9fff]/.test(f.val) && /[A-Za-z]{4,}/.test(f.val) && !TECH.test(f.val));
const seen = new Set();
const candidates = [];
for (const c of candidatesRaw) {
  const k = c.key + '|' + c.val;
  if (seen.has(k)) continue;
  seen.add(k);
  candidates.push(c);
}
console.log('--- mixed-zh-en fallback candidates (dedup):', candidates.length);
candidates.forEach((c, i) => {
  const blocked = anchors.has(c.val);
  console.log(`  ${i + 1} ${blocked ? 'ANCHOR' : 'free  '}  ${c.key}  =>  ${JSON.stringify(c.val)}`);
});
