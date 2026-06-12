import fs from 'node:fs';
const appJs = fs.readFileSync('web/static/app.js', 'utf8');
const lines = appJs.split('\n');

// zh-CN block: 67..1088 (0-based 66..1087), en-US: 1089.. (find end by matching brace depth is hard; use next top-level marker)
// We know "zh-CN": { at line 67, "en-US": { at 1089. Find line where en block closes -> the dict closes with "  },".
const zhStart = lines.findIndex((l) => l.includes('"zh-CN": {'));
const enStart = lines.findIndex((l) => l.includes('"en-US": {'));
// en block ends at the matching dedented "  }" or "  },". Scan from enStart for a line === "  }," or "  }" at 2-space indent.
let enEnd = lines.length;
for (let i = enStart + 1; i < lines.length; i++) {
  if (/^  \},?\s*$/.test(lines[i])) { enEnd = i; break; }
}

const reNs = /^\s{4,6}([a-zA-Z_]+):\s*\{\s*$/;
const reKv = /^\s{6,}([a-zA-Z_0-9]+):\s*"((?:[^"\\]|\\.)*)",?\s*$/;

function parse(start, end) {
  const map = new Map();
  let ns = null;
  for (let i = start; i < end; i++) {
    const ln = lines[i];
    const nm = ln.match(reNs);
    if (nm) { ns = nm[1]; continue; }
    const kv = ln.match(reKv);
    if (kv && ns) map.set(ns + '.' + kv[1], kv[2]);
  }
  return map;
}

const zh = parse(zhStart + 1, enStart);
const en = parse(enStart + 1, enEnd);

console.log('zh keys:', zh.size, ' en keys:', en.size);

// 1) keys in zh missing from en  -> English mode would have no translation (uses fallback, may be Chinese)
const missingInEn = [...zh.keys()].filter((k) => !en.has(k));
console.log('\n=== keys in zh-CN but MISSING in en-US:', missingInEn.length);
for (const k of missingInEn) console.log('  ', k, '=>', JSON.stringify(zh.get(k)));

// 2) en values that are identical to zh value AND contain CJK -> untranslated (English mode shows Chinese)
const enStillChinese = [...en.entries()].filter(([k, v]) => /[一-鿿]/.test(v));
console.log('\n=== en-US values containing CJK (untranslated):', enStillChinese.length);
for (const [k, v] of enStillChinese) console.log('  ', k, '=>', JSON.stringify(v));

// 3) zh values mixed zh+en (4+ letter english word), excluding tech tokens
const TECH = /(JSON|Cookie|OpenList|Alist|Smoke Matrix|providerData|uploadId|nextPart|retryClass|retryLimit|riskOverride|pre_scan_flat|leaf_first_lazy|record_only|manual_token|auth_only|fast_upload|upload checkpoint|provider_session_missing|pending_manual|risk_control|auth_expired|rate_limited|local_file_missing|large_file|multipart|nested_directory|subtree|retry_recovery|checkpoint|resume|providerSmokeProviders|blocked action|lane|root|parentId|fileId|localPath|selectedRoots|domainId|driveId|pwdId|Token|success|failure)/i;
const zhMixed = [...zh.entries()].filter(([k, v]) => /[一-鿿]/.test(v) && /[A-Za-z]{4,}/.test(v) && !TECH.test(v));
console.log('\n=== zh-CN values mixed zh+en (non-tech):', zhMixed.length);
for (const [k, v] of zhMixed) console.log('  ', k, '=>', JSON.stringify(v));
