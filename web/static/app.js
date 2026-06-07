const state = {
  authenticated: localStorage.getItem("cloudpan_console_session") === "ok",
  providers: [],
  providerCapabilityDetails: {},
  selectedProviderCapabilityKey: "",
  profiles: [],
  tasks: [],
  preview: null,
  selectedTaskId: null,
  focusedProfileId: null,
  evidence: null,
  statuses: [],
  report: null,
  reportHistory: [],
  selectedReportId: "",
  providerSmokes: [],
  providerSmokeSummary: [],
  providerSmokeMatrix: [],
  providerSmokeMatrixFilter: "all",
  providerSmokeRecordFilters: {
    query: "",
    protocolGroup: "",
    result: "",
  },
  selectedProviderSmokeId: "",
  selectedProviderSmokeMarkdown: "",
  autoRecoverFilters: {
    mode: "",
    protocolGroup: "",
    providerKey: "",
    profileId: "",
    retryClass: "",
    blockedAction: "",
    recoverState: "",
    strategy: "",
    limit: "",
    limitPerMode: "",
    limitPerLane: "",
    limitPerProtocolGroup: "",
    limitPerProvider: "",
    limitPerProfile: "",
  },
  autoRecoverLastResult: null,
  taskActionPending: false,
  treeGroupsCollapsed: {},
  treeFilters: {
    taskDirectory: { query: "", status: "", leafOnly: false, problemOnly: false },
    taskPending: { query: "", reason: "", leafOnly: false },
    taskRetry: { query: "", retryClass: "", retryState: "" },
    statusDirectory: { query: "", status: "", leafOnly: false, problemOnly: false },
    statusPending: { query: "", reason: "", leafOnly: false },
    statusRetry: { query: "", retryClass: "", retryState: "" },
  },
};

const treeGroupsStorageKey = "cloudpan_console_tree_groups_collapsed";

function $(selector) {
  return document.querySelector(selector);
}

function loadTreeGroupsCollapsed() {
  try {
    const raw = localStorage.getItem(treeGroupsStorageKey);
    if (!raw) {
      return {};
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return {};
    }
    return Object.fromEntries(Object.entries(parsed).filter(([, value]) => Boolean(value)));
  } catch (error) {
    return {};
  }
}

function saveTreeGroupsCollapsed() {
  try {
    localStorage.setItem(treeGroupsStorageKey, JSON.stringify(state.treeGroupsCollapsed));
  } catch (error) {
    // Ignore storage quota / privacy mode failures.
  }
}

Object.assign(state.treeGroupsCollapsed, loadTreeGroupsCollapsed());

function formatJSON(value) {
  return JSON.stringify(value, null, 2);
}

function stringifyValue(value, fallback = "-") {
  if (value === undefined || value === null || value === "") {
    return fallback;
  }
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

function renderSourceDeletePolicy(value, fallback = "record_only") {
  const policy = stringifyValue(value, fallback);
  if (policy === "record_only") {
    return "record_only（只记录，不删目标端）";
  }
  return policy;
}

function firstNonEmpty(...values) {
  for (const value of values) {
    if (value !== undefined && value !== null && String(value).trim() !== "") {
      return String(value).trim();
    }
  }
  return "";
}

function summarizePathList(paths, limit = 6) {
  if (!Array.isArray(paths) || !paths.length) {
    return "-";
  }
  const items = paths.map((item) => String(item || "").trim()).filter(Boolean);
  if (!items.length) {
    return "-";
  }
  const shown = items.slice(0, Math.max(limit, 1));
  const suffix = items.length > shown.length ? ` …(+${items.length - shown.length})` : "";
  return `${shown.join(" -> ")}${suffix}`;
}

function renderRuntimePathChips(title, paths, scope, kind) {
  const items = Array.isArray(paths) ? paths.map((item) => String(item || "").trim()).filter(Boolean) : [];
  if (!items.length) {
    return `
      <div class="insight-card checkpoint-card">
        <strong>${escapeHTML(title)}</strong>
        <span>-</span>
      </div>
    `;
  }
  const limit = kind === "scan" ? 8 : 6;
  const shown = items.slice(0, limit);
  return `
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(title)}</strong>
      <div class="actions compact path-chip-row">
        ${shown
          .map(
            (item) => `
              <button
                type="button"
                class="ghost path-chip"
                data-runtime-focus-path="${escapeHTML(item)}"
                data-runtime-focus-scope="${escapeHTML(scope)}"
                data-runtime-focus-kind="${escapeHTML(kind)}"
              >${escapeHTML(item)}</button>
            `,
          )
          .join("")}
      </div>
      ${
        items.length > shown.length
          ? `<div class="muted">还有 ${items.length - shown.length} 项未展开。</div>`
          : ""
      }
    </div>
  `;
}

function renderRiskResolutionSummary(resolution) {
  if (!resolution || typeof resolution !== "object") {
    return "-";
  }
  const providerKey = stringifyValue(resolution.providerKey, "-");
  const profileSource = stringifyValue(resolution.profileDefaultSource, "provider default only");
  const profileSourceKind = stringifyValue(resolution.profileDefaultSourceKind, "-");
  const profileDefaultBias = stringifyValue(resolution.profileDefaultBias, "same_as_provider");
  const profileDefaultFields = Array.isArray(resolution.profileDefaultFields)
    ? resolution.profileDefaultFields.filter(Boolean)
    : [];
  const overrideFields = Array.isArray(resolution.overrideFields) ? resolution.overrideFields.filter(Boolean) : [];
  const steps = [`provider ${providerKey}`];
  steps.push(`profile ${profileSource}`);
  if (profileSourceKind !== "-") {
    steps.push(`profile-kind ${profileSourceKind}`);
  }
  if (profileDefaultBias !== "same_as_provider") {
    steps.push(`profile-bias ${profileDefaultBias}`);
  }
  if (profileDefaultFields.length) {
    steps.push(`profile fields ${profileDefaultFields.join(", ")}`);
  }
  steps.push(`override ${overrideFields.length ? overrideFields.join(", ") : "none"}`);
  steps.push(`final ${stringifyValue(resolution.applied?.mode, stringifyValue(resolution.calibrated?.mode, "balanced"))}`);
  return steps.join(" -> ");
}

function renderRiskResolutionFlow(resolution) {
  if (!resolution || typeof resolution !== "object") {
    return `
      <div class="insight-card">
        <strong>风控链路</strong>
        <span>-</span>
      </div>
    `;
  }
  const profileSourceKind = stringifyValue(resolution.profileDefaultSourceKind, "-");
  const profileDefaultBias = stringifyValue(resolution.profileDefaultBias, "same_as_provider");
  const profileDefaultFields = Array.isArray(resolution.profileDefaultFields)
    ? resolution.profileDefaultFields.filter(Boolean)
    : [];
  const overrideFields = Array.isArray(resolution.overrideFields) ? resolution.overrideFields.filter(Boolean) : [];
  return `
    <div class="insight-card">
      <strong>Provider 基线</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.base))}</span>
    </div>
    <div class="insight-card">
      <strong>Provider 校准后</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.calibrated))}</span>
    </div>
    <div class="insight-card">
      <strong>账号默认注入</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.profileApplied))}</span>
      <div class="muted">source ${escapeHTML(stringifyValue(resolution.profileDefaultSource, "provider default only"))} / kind ${escapeHTML(profileSourceKind)} / bias ${escapeHTML(profileDefaultBias)} / fields ${escapeHTML(profileDefaultFields.join(", ") || "-")}</div>
    </div>
    <div class="insight-card">
      <strong>任务覆盖</strong>
      <span>${overrideFields.length ? escapeHTML(overrideFields.join(", ")) : "无"}</span>
    </div>
    <div class="insight-card">
      <strong>最终生效</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.applied))}</span>
    </div>
  `;
}

function renderRiskProfileCompact(profile) {
  if (!profile || typeof profile !== "object") {
    return "-";
  }
  return [
    `mode ${stringifyValue(profile.mode, "-")}`,
    `req ${stringifyValue(profile.requestIntervalMs, "0")}ms`,
    `page ${stringifyValue(profile.pageSize, "0")}`,
    `dir ${stringifyValue(profile.directoryIntervalMs, "0")}ms`,
    `cooldown ${stringifyValue(profile.cooldownSeconds, "0")}s`,
    `retry ${stringifyValue(profile.retryLimit, "0")}`,
    `conc ${stringifyValue(profile.maxConcurrent, "0")}`,
  ].join(" / ");
}

function renderRecoverBudgetCompact(policy) {
  if (!policy || typeof policy !== "object") {
    return "-";
  }
  return [
    `group ${stringifyValue(policy.protocolGroupBudget, "0")}`,
    `provider ${stringifyValue(policy.providerBudget, "0")}`,
    `profile ${stringifyValue(policy.profileBudget, "0")}`,
  ].join(" / ");
}
function isSensitiveRecoverBudgetTemplate(policy, providerKey = "") {
  if (!policy || typeof policy !== "object") {
    return false;
  }
  const sensitiveProviders = Array.isArray(policy.sensitiveProviders) ? policy.sensitiveProviders.filter(Boolean) : [];
  const normalized = String(providerKey || "").trim();
  return sensitiveProviders.length > 0 && (!normalized || sensitiveProviders.includes(normalized));
}

function renderRecoverBudgetAdvice(policy, providerKey = "") {
  if (!policy || typeof policy !== "object") {
    return "未返回账号级预算建议。";
  }
  const reason = String(policy.reason || "").trim();
  if (isSensitiveRecoverBudgetTemplate(policy, providerKey)) {
    return `高风险 provider 建议单账号串行推进：${renderRecoverBudgetCompact(policy)}${reason ? `；${reason}` : ""}`;
  }
  if (Number(policy.profileBudget || 0) <= 1 && Number(policy.providerBudget || 0) > 0) {
    return `建议按账号轮转控制补传并发：${renderRecoverBudgetCompact(policy)}${reason ? `；${reason}` : ""}`;
  }
  return `建议恢复预算：${renderRecoverBudgetCompact(policy)}${reason ? `；${reason}` : ""}`;
}

function findProviderEntry(providerKey) {
  const normalized = String(providerKey || "").trim();
  if (!normalized) {
    return null;
  }
  return (state.providers || []).find((entry) => entry?.meta?.key === normalized) || null;
}

function renderProviderCapabilityCompact(capability) {
  if (!capability || typeof capability !== "object") {
    return "-";
  }
  const enabled = [];
  if (capability.supportsAuthValidation) enabled.push("auth");
  if (capability.supportsList) enabled.push("list");
  if (capability.supportsMetadata) enabled.push("metadata");
  if (capability.supportsCreateDir) enabled.push("create_dir");
  if (capability.supportsFastUpload) enabled.push("fast_check");
  if (capability.supportsUpload) enabled.push("upload");
  return enabled.length ? enabled.join(", ") : "-";
}

function renderProviderRiskTemplateDetail(template, { title = "默认风控模板", compact = false } = {}) {
  if (!template || typeof template !== "object") {
    return `
      <div class="insight-card">
        <strong>${escapeHTML(title)}</strong>
        <span>-</span>
      </div>
    `;
  }
  const providerHints = Array.isArray(template.providerRiskHints) ? template.providerRiskHints.filter(Boolean) : [];
  const providerTraits = Array.isArray(template.providerRiskTraits) ? template.providerRiskTraits.filter(Boolean) : [];
  const reasons = Array.isArray(template.calibrationReasons) ? template.calibrationReasons.filter(Boolean) : [];
  const parts = [
    `<div class="insight-card">`,
    `<strong>${escapeHTML(title)}</strong>`,
    `<span>${escapeHTML(renderRiskProfileCompact(template.calibrated))}</span>`,
    `<div class="muted">auto retry window ${escapeHTML(renderRiskWindow(template.calibrated))}</div>`,
    `<div class="muted">window source ${escapeHTML(renderAutoRetryWindowSource(template.autoRetryWindowSource))}</div>`,
    `<div class="muted">calibration coverage ${escapeHTML(stringifyValue(template.calibrationCoverage, "-"))}</div>
    <div class="muted">calibration covered ${escapeHTML(stringifyValue(template.calibrationCoveredCount, "0"))}/${escapeHTML(stringifyValue(template.calibrationTargetCount, "0"))} / missing ${escapeHTML(stringifyValue(template.calibrationMissingCount, "0"))}</div>
    <div class="muted">calibration covered fields ${escapeHTML((template.calibrationCoveredFields || []).join(", ") || "-")}</div>
    <div class="muted">calibration readiness ${escapeHTML(stringifyValue(template.calibrationReadiness, "-"))}</div>`,
    `<div class="muted">recommended ${escapeHTML(stringifyValue(template.recommendedMode, "-"))}</div>`,
    `<div class="muted">recover budget ${escapeHTML(renderRecoverBudgetCompact(template.recoverBudget))}</div>`,
    `<div class="muted">budget advice ${escapeHTML(renderRecoverBudgetAdvice(template.recoverBudget, template.providerKey || ""))}</div>`,
  ];
  if (!compact) {
    parts.push(`<div class="muted">base ${escapeHTML(renderRiskProfileCompact(template.base))}</div>`);
    parts.push(`<div class="muted">reasons ${escapeHTML(reasons.join(" / ") || "-")}</div>`);
    parts.push(`<div class="muted">risk hints ${escapeHTML(providerHints.join(" / ") || "-")}</div>`);
    parts.push(`<div class="muted">risk traits ${escapeHTML(providerTraits.join(", ") || "-")}</div>`);
    parts.push(`<div class="muted">calibration missing ${escapeHTML((template.calibrationMissing || []).join(", ") || "-")}</div>`);
    parts.push(`<div class="muted">priority calibration: ${escapeHTML(stringifyValue(template.calibrationPriorityAction, "-"))}</div>`);
    parts.push(`<div class="muted">window advice ${escapeHTML(stringifyValue(template.autoRetryWindowAdvice, "-"))}</div>`);
    parts.push(`<div class="muted">advice ${escapeHTML(stringifyValue(template.recommendedReason, "-"))}</div>`);
    parts.push(`<div class="muted">warning ${escapeHTML(stringifyValue(template.aggressiveRiskWarning, "-"))}</div>`);
  }
  parts.push(`</div>`);
  return parts.join("");
}

function renderProviderCapabilityDetail() {
  const wrap = $("#provider-capability-detail");
  if (!wrap) {
    return;
  }
  const providerKey = state.selectedProviderCapabilityKey || "";
  const entry = findProviderEntry(providerKey);
  const detail = state.providerCapabilityDetails[providerKey] || null;
  if (!providerKey || !entry) {
    wrap.innerHTML = `<div class="muted">点击任一 provider 卡片，查看能力声明、默认风控模板和恢复预算。</div>`;
    return;
  }
  if (!detail) {
    wrap.innerHTML = `
      <div class="insight-grid">
        <div class="insight-card">
          <strong>${escapeHTML(entry.meta.displayName)}</strong>
          <span>正在加载能力详情...</span>
        </div>
      </div>
    `;
    return;
  }
  const provider = detail.provider || entry.meta || {};
  const capability = detail.capabilities || entry.capability || {};
  wrap.innerHTML = `
    <div class="section-head">
      <h3>${escapeHTML(stringifyValue(provider.displayName, provider.key || "-"))}</h3>
      <span class="muted">${escapeHTML(stringifyValue(provider.key, "-"))} / ${escapeHTML(stringifyValue(provider.protocolGroup, "-"))}</span>
    </div>
    <div class="insight-grid">
      <div class="insight-card">
        <strong>能力声明</strong>
        <span>${escapeHTML(renderProviderCapabilityCompact(capability))}</span>
      </div>
      <div class="insight-card">
        <strong>鉴权模式</strong>
        <span>${escapeHTML((provider.authModes || []).join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>冲突策略</strong>
        <span>${escapeHTML((provider.conflictPolicies || []).join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>回退策略</strong>
        <span>${escapeHTML((provider.fallbackModes || []).join(", ") || "-")}</span>
      </div>
      ${renderProviderRiskTemplateDetail({ ...(provider.defaultRiskTemplate || {}), providerKey: provider.key || providerKey }, { title: "默认风控模板" })}
    </div>
  `;
}

function syncTargetProviderInsight() {
  const wrap = $("#plan-target-provider-insight");
  if (!wrap) {
    return;
  }
  const providerKey = $("#plan-target-provider")?.value || "";
  const entry = findProviderEntry(providerKey);
  if (!providerKey || !entry) {
    wrap.innerHTML = `<div class="muted">选择目标 provider 后，这里会显示默认风控模板、推荐档位和恢复预算。</div>`;
    return;
  }
  wrap.innerHTML = `
    <div class="section-head">
      <h3>${escapeHTML(stringifyValue(entry.meta.displayName, providerKey))}</h3>
      <span class="muted">${escapeHTML(providerKey)} / ${escapeHTML(stringifyValue(entry.meta.protocolGroup, "-"))}</span>
    </div>
    <div class="insight-grid">
      <div class="insight-card">
        <strong>推荐风控档位</strong>
        <span>${escapeHTML(stringifyValue(entry.meta.defaultRiskTemplate?.recommendedMode, "-"))}</span>
      </div>
      <div class="insight-card">
        <strong>能力摘要</strong>
        <span>${escapeHTML(renderProviderCapabilityCompact(entry.capability))}</span>
      </div>
      ${renderProviderRiskTemplateDetail({ ...(entry.meta.defaultRiskTemplate || {}), providerKey }, { title: "Provider 默认模板", compact: true })}
    </div>
    <div class="actions compact-actions">
      <button type="button" class="ghost" id="apply-provider-default-risk">采用 provider 推荐风控</button>
      <button type="button" class="ghost" id="open-target-provider-capability">查看 provider 能力详情</button>
    </div>
  `;
  $("#apply-provider-default-risk").onclick = () => {
    const recommended = entry.meta.defaultRiskTemplate?.recommendedMode || "";
    if (!recommended) {
      showFlash("当前 provider 没有可用的推荐风控档位", true);
      return;
    }
    setSelectValueIfPresent("#plan-risk-mode", recommended);
    showFlash(`已采用 provider 推荐风控：${recommended}`);
  };
  $("#open-target-provider-capability").onclick = async () => {
    try {
      await loadProviderCapabilityDetail(providerKey);
      activateTab("providers");
      showFlash(`已打开 ${providerKey} provider 能力详情`);
    } catch (error) {
      showFlash(error.message, true);
    }
  };
}

function syncTargetProfileInsight() {
  const wrap = $("#plan-target-profile-insight");
  if (!wrap) {
    return;
  }
  const profileID = $("#plan-target-profile")?.value || "";
  const profile = (state.profiles || []).find((item) => item?.id === profileID);
  if (!profile) {
    wrap.innerHTML = `<div class="muted">选择目标授权档案后，这里会显示账号默认风控模板。</div>`;
    return;
  }
  const riskDefaults = parseProfileRiskDefaultsFromExtra(profile.extra);
  const riskDefaultSource = parseProfileRiskDefaultsSourceFromExtra(profile.extra);
  const providerEntry = findProviderEntry(profile.providerKey || "");
  const recoverBudget = providerEntry?.meta?.defaultRiskTemplate?.recoverBudget || null;
  const extraKeys = profile.extra && typeof profile.extra === "object" ? Object.keys(profile.extra).filter(Boolean) : [];
  const profileDefaultFields = riskDefaults && typeof riskDefaults === "object" ? Object.keys(riskDefaults).filter(Boolean) : [];
  wrap.innerHTML = `
    <div class="section-head">
      <h3>${escapeHTML(stringifyValue(profile.displayName, profileID))}</h3>
      <span class="muted">${escapeHTML(stringifyValue(profile.providerKey, "-"))} / ${escapeHTML(stringifyValue(profile.authMode, "-"))}</span>
    </div>
    <div class="insight-grid">
      <div class="insight-card">
        <strong>账号默认风控</strong>
        <span>${escapeHTML(renderRiskProfileCompact(riskDefaults))}</span>
        <div class="muted">可直接写入本次任务覆盖，便于在此基础上再细调。</div>
      </div>
      <div class="insight-card">
        <strong>来源</strong>
        <span>${riskDefaults ? escapeHTML(riskDefaultSource || "auth profile riskDefaults") : "未配置，使用 provider 默认模板"}</span>
        <div class="muted">${escapeHTML(renderProfileRiskDefaultSourceAdvice(riskDefaultSource || ""))}</div>
      </div>
      <div class="insight-card">
        <strong>Extra Keys</strong>
        <span>${escapeHTML(extraKeys.join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>命中字段</strong>
        <span>${escapeHTML(profileDefaultFields.join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>账号恢复预算建议</strong>
        <span>${escapeHTML(renderRecoverBudgetAdvice(recoverBudget, profile.providerKey || ""))}</span>
      </div>
    </div>
    <div class="actions compact-actions">
      <button type="button" class="ghost" id="apply-profile-default-risk"${riskDefaults ? "" : " disabled"}>应用账号默认到任务覆盖</button>
      <button type="button" class="ghost" id="clear-profile-default-risk">改回账号默认</button>
    </div>
  `;
  const applyButton = $("#apply-profile-default-risk");
  if (applyButton) {
    applyButton.onclick = () => {
      if (!riskDefaults) {
        showFlash("当前授权档案没有账号默认风控可写入", true);
        return;
      }
      hydrateRiskOverrideForm(riskDefaults);
      $("#plan-risk-override").value = JSON.stringify(riskDefaults, null, 2);
      setSelectValueIfPresent("#plan-risk-mode", "custom");
      showFlash("已将账号默认风控写入任务覆盖，可继续按任务单独微调");
    };
  }
  const clearButton = $("#clear-profile-default-risk");
  if (clearButton) {
    clearButton.onclick = () => {
      hydrateRiskOverrideForm(null);
      $("#plan-risk-override").value = "";
      showFlash("已清空任务覆盖，将回到账号默认 / provider 默认链路");
    };
  }
}

async function loadProviderCapabilityDetail(providerKey, { force = false } = {}) {
  const normalized = String(providerKey || "").trim();
  if (!normalized) {
    return;
  }
  state.selectedProviderCapabilityKey = normalized;
  renderProviders();
  if (!force && state.providerCapabilityDetails[normalized]) {
    renderProviderCapabilityDetail();
    return;
  }
  const data = await api(`/api/providers/${encodeURIComponent(normalized)}/capabilities`);
  state.providerCapabilityDetails[normalized] = data;
  renderProviders();
  renderProviderCapabilityDetail();
}

function renderRiskResolutionDetail(resolution) {
  if (!resolution || typeof resolution !== "object") {
    return "";
  }
  const reasons = Array.isArray(resolution.calibrationReasons) ? resolution.calibrationReasons.filter(Boolean) : [];
  const profileSourceKind = stringifyValue(resolution.profileDefaultSourceKind, "-");
  const profileDefaultBias = stringifyValue(resolution.profileDefaultBias, "same_as_provider");
  const profileDefaultFields = Array.isArray(resolution.profileDefaultFields)
    ? resolution.profileDefaultFields.filter(Boolean)
    : [];
  const overrideFields = Array.isArray(resolution.overrideFields) ? resolution.overrideFields.filter(Boolean) : [];
  const providerHints = Array.isArray(resolution.providerRiskHints) ? resolution.providerRiskHints.filter(Boolean) : [];
  const providerTraits = Array.isArray(resolution.providerRiskTraits) ? resolution.providerRiskTraits.filter(Boolean) : [];
  const recoverBudget = resolution.recoverBudget && typeof resolution.recoverBudget === "object" ? resolution.recoverBudget : null;
  const sensitiveProviders = Array.isArray(recoverBudget?.sensitiveProviders)
    ? recoverBudget.sensitiveProviders.filter(Boolean)
    : [];
  return `
    <div class="muted">FLOW ${escapeHTML(renderRiskResolutionSummary(resolution))}</div>
    <div class="muted">BASE ${escapeHTML(renderRiskProfileCompact(resolution.base))}</div>
    <div class="muted">CALIBRATED ${escapeHTML(renderRiskProfileCompact(resolution.calibrated))}</div>
    <div class="muted">PROFILE DEFAULT SOURCE ${escapeHTML(stringifyValue(resolution.profileDefaultSource, "-"))}</div>
    <div class="muted">PROFILE DEFAULT SOURCE KIND ${escapeHTML(profileSourceKind)}</div>
    <div class="muted">PROFILE DEFAULT BIAS ${escapeHTML(profileDefaultBias)}</div>
    <div class="muted">PROFILE DEFAULT ${escapeHTML(renderRiskProfileCompact(resolution.profileApplied))}</div>
    <div class="muted">APPLIED ${escapeHTML(renderRiskProfileCompact(resolution.applied))}</div>
    <div class="muted">RECOVER BUDGET ${escapeHTML(renderRecoverBudgetCompact(recoverBudget))}</div>
    <div class="muted">RECOVER REASON ${escapeHTML(stringifyValue(recoverBudget?.reason, "-"))}</div>
    <div class="muted">SENSITIVE PROVIDERS ${escapeHTML(sensitiveProviders.join(", ") || "-")}</div>
    <div class="muted">PROVIDER HINTS ${escapeHTML(providerHints.join(" / ") || "-")}</div>
    <div class="muted">PROVIDER TRAITS ${escapeHTML(providerTraits.join(", ") || "-")}</div>
    <div class="muted">CALIBRATION REASONS ${escapeHTML(reasons.join(" / ") || "-")}</div>
    <div class="muted">PROFILE DEFAULT FIELDS ${escapeHTML(profileDefaultFields.join(", ") || "-")}</div>
    <div class="muted">OVERRIDE FIELDS ${escapeHTML(overrideFields.join(", ") || "-")}</div>
  `;
}

function renderRiskResolutionMetaCards(resolution) {
  if (!resolution || typeof resolution !== "object") {
    return "";
  }
  const recoverBudget = resolution.recoverBudget && typeof resolution.recoverBudget === "object" ? resolution.recoverBudget : {};
  const profileSource = stringifyValue(resolution.profileDefaultSource, "provider default only");
  const profileSourceKind = stringifyValue(resolution.profileDefaultSourceKind, "-");
  const profileDefaultBias = stringifyValue(resolution.profileDefaultBias, "same_as_provider");
  const profileDefaultFields = Array.isArray(resolution.profileDefaultFields)
    ? resolution.profileDefaultFields.filter(Boolean)
    : [];
  return `
    <div class="insight-card">
      <strong>账号默认来源</strong>
      <span>${escapeHTML(profileSource)}</span>
      <div class="muted">${escapeHTML(renderProfileRiskDefaultSourceAdvice(profileSource))}</div>
    </div>
    <div class="insight-card">
      <strong>来源类型 / 偏向</strong>
      <span>${escapeHTML(`${profileSourceKind} / ${profileDefaultBias}`)}</span>
    </div>
    <div class="insight-card">
      <strong>账号默认字段</strong>
      <span>${escapeHTML(profileDefaultFields.join(", ") || "-")}</span>
    </div>
    <div class="insight-card">
      <strong>恢复预算理由</strong>
      <span>${escapeHTML(stringifyValue(recoverBudget.reason, "-"))}</span>
    </div>
  `;
}

function renderRiskWindow(profile) {
  if (!profile || typeof profile !== "object") {
    return "-";
  }
  const start = Number(profile.autoRetryStartHour || 0);
  const end = Number(profile.autoRetryEndHour || 0);
  if (start <= 0 && end <= 0) {
    return "always_on";
  }
  return `${start}:00-${end}:00 UTC`;
}

function renderAutoRetryWindowSource(source) {
  switch (String(source || "").trim()) {
    case "provider_default":
      return "provider default";
    case "empty_until_profile_or_override":
      return "provider default empty";
    default:
      return stringifyValue(source, "-");
  }
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function normalizeDirectoryStates(states) {
  if (!Array.isArray(states)) {
    return [];
  }
  return states
    .filter((item) => item && typeof item === "object" && item.path)
    .map((item) => ({
      path: String(item.path || ""),
      rootPath: String(item.rootPath || ""),
      status: String(item.status || "pending"),
      totalItems: Number(item.totalItems || 0),
      processedItems: Number(item.processedItems || 0),
      doneItems: Number(item.doneItems || 0),
      skippedItems: Number(item.skippedItems || 0),
      failedItems: Number(item.failedItems || 0),
      lastItemPath: String(item.lastItemPath || ""),
    }));
}

function normalizePendingTree(nodes) {
  if (!Array.isArray(nodes)) {
    return [];
  }
  return nodes
    .filter((item) => item && typeof item === "object" && item.path)
    .map((item) => ({
      path: String(item.path || ""),
      name: String(item.name || item.path || ""),
      nodeType: String(item.nodeType || "directory"),
      status: String(item.status || "pending_manual"),
      rootPath: String(item.rootPath || ""),
      itemCount: Number(item.itemCount || 0),
      reason: String(item.reason || ""),
      providerStatus: String(item.providerStatus || ""),
      children: normalizePendingTree(item.children),
    }));
}

function normalizeRetryQueue(items) {
  if (!Array.isArray(items)) {
    return [];
  }
  return items
    .filter((item) => item && typeof item === "object" && item.path)
    .map((item) => ({
      path: String(item.path || ""),
      rootPath: String(item.rootPath || ""),
      providerStatus: String(item.providerStatus || ""),
      strategy: String(item.strategy || ""),
      retryClass: String(item.retryClass || ""),
      retryAction: String(item.retryAction || ""),
      attemptCount: Number(item.attemptCount || 0),
      retryLimit: Number(item.retryLimit || 0),
      remainingCount: Number(item.remainingCount || 0),
      cooldownTier: String(item.cooldownTier || ""),
      cooldownSeconds: Number(item.cooldownSeconds || 0),
      eligibleAt: String(item.eligibleAt || ""),
      retryable: Boolean(item.retryable),
      blocked: Boolean(item.blocked),
      exhausted: Boolean(item.exhausted),
      reason: String(item.reason || ""),
    }));
}

function inferDisplayName(path) {
  if (!path || path === "/") {
    return "/";
  }
  const normalized = String(path).replaceAll("\\", "/");
  const index = normalized.lastIndexOf("/");
  if (index >= 0 && index < normalized.length - 1) {
    return normalized.slice(index + 1);
  }
  return normalized;
}

function treeGroupCollapseKey(scope, panel, path) {
  return `${scope}:${panel}:${path}`;
}

function isTreeGroupCollapsed(scope, panel, path) {
  return Boolean(state.treeGroupsCollapsed[treeGroupCollapseKey(scope, panel, path)]);
}

function setTreeGroupCollapsed(scope, panel, path, collapsed) {
  const key = treeGroupCollapseKey(scope, panel, path);
  if (collapsed) {
    state.treeGroupsCollapsed[key] = true;
  } else {
    delete state.treeGroupsCollapsed[key];
  }
  saveTreeGroupsCollapsed();
}

function setTreeGroupsCollapsedForPaths(scope, panel, paths, collapsed) {
  (paths || []).forEach((path) => {
    setTreeGroupCollapsed(scope, panel, path, collapsed);
  });
}

function setTreeGroupsCollapsedForTree(scope, panel, tree, collapsed) {
  const paths = Array.isArray(tree)
    ? tree.map((root) => root?.rootPath || root?.path).filter(Boolean)
    : [];
  setTreeGroupsCollapsedForPaths(scope, panel, paths, collapsed);
}

function currentTreeRootsForPanel(scope, panel) {
  const detail = currentSelectedTaskDetail();
  const runtimePayload = recentRuntimePayload();
  const runtime = scope === "task" ? detail?.runtime || detail?.plan?.metadata?.runtime || {} : runtimePayload?.runtime || runtimePayload || {};
  if (panel === "pending") {
    return Array.isArray(runtime.pendingTree) ? runtime.pendingTree : [];
  }
  return Array.isArray(runtime.directoryStates) ? runtime.directoryStates : [];
}

function wireTreeBulkActions(scope, panel) {
  document.querySelectorAll(`[data-tree-bulk-scope="${scope}"][data-tree-bulk-panel="${panel}"]`).forEach((button) => {
    button.onclick = () => {
      const collapsed = button.dataset.treeBulkAction === "collapse";
      const roots = currentTreeRootsForPanel(scope, panel);
      setTreeGroupsCollapsedForTree(scope, panel, roots, collapsed);
      if (scope === "task") {
        updateTaskTreePanels(currentSelectedTaskDetail());
      } else {
        updateStatusTreePanels(recentRuntimePayload());
      }
    };
  });
}

function buildDirectoryStateTree(states) {
  const normalized = normalizeDirectoryStates(states);
  if (!normalized.length) {
    return [];
  }

  const nodes = new Map();
  const rootOrder = [];

  normalized.forEach((item) => {
    nodes.set(item.path, {
      path: item.path,
      name: inferDisplayName(item.path),
      nodeType: item.path === (item.rootPath || item.path) ? "root" : "directory",
      status: item.status,
      rootPath: item.rootPath || item.path,
      itemCount: item.totalItems,
      processedItems: item.processedItems,
      doneItems: item.doneItems,
      skippedItems: item.skippedItems,
      failedItems: item.failedItems,
      lastItemPath: item.lastItemPath,
      children: [],
    });
  });

  normalized.forEach((item) => {
    const current = nodes.get(item.path);
    if (!current) {
      return;
    }
    const parentPath = String(item.path).replaceAll("\\", "/").split("/").slice(0, -1).join("/") || "/";
    const rootPath = item.rootPath || item.path;
    if (item.path === rootPath || !nodes.has(parentPath) || parentPath === item.path) {
      rootOrder.push(item.path);
      return;
    }
    nodes.get(parentPath).children.push(current);
  });

  const sortNodes = (items) => {
    items.sort((left, right) => left.path.localeCompare(right.path));
    items.forEach((item) => sortNodes(item.children));
    return items;
  };

  return sortNodes(rootOrder.map((path) => nodes.get(path)).filter(Boolean));
}

function countTreeNodes(nodes) {
  if (!Array.isArray(nodes) || !nodes.length) {
    return 0;
  }
  return nodes.reduce((total, node) => total + 1 + countTreeNodes(node.children || []), 0);
}

function countTreeLeafNodes(nodes, mode = "directory") {
  if (!Array.isArray(nodes) || !nodes.length) {
    return 0;
  }
  return nodes.reduce((total, node) => {
    const children = Array.isArray(node.children) ? node.children : [];
    if (!children.length) {
      return total + 1;
    }
    if (mode === "pending" && node.nodeType === "file") {
      return total + 1;
    }
    return total + countTreeLeafNodes(children, mode);
  }, 0);
}

function deepestTreePath(nodes, fallback = "") {
  let bestPath = String(fallback || "");
  let bestDepth = bestPath ? inferPathDepth(bestPath) : -1;
  const visit = (items) => {
    (items || []).forEach((node) => {
      const path = String(node?.path || "").trim();
      const depth = inferPathDepth(path);
      if (path && depth >= bestDepth) {
        bestDepth = depth;
        bestPath = path;
      }
      if (Array.isArray(node?.children) && node.children.length) {
        visit(node.children);
      }
    });
  };
  visit(nodes);
  return bestPath;
}

function inferPathDepth(path) {
  const normalized = normalizeComparePath(path);
  if (!normalized || normalized === "/") {
    return 0;
  }
  return normalized.split("/").filter(Boolean).length;
}

function parentTreePath(path) {
  const normalized = normalizeComparePath(path);
  if (!normalized || normalized === "/") {
    return "";
  }
  const parts = normalized.split("/").filter(Boolean);
  if (parts.length <= 1) {
    return "/";
  }
  return `/${parts.slice(0, -1).join("/")}`;
}

function directoryNodeHasProblem(node) {
  if (!node || typeof node !== "object") {
    return false;
  }
  const status = String(node.status || "").trim().toLowerCase();
  if (status === "blocked" || status === "running" || status === "pending") {
    return true;
  }
  const failedItems = Number(node.failedItems || 0);
  const totalItems = Number(node.itemCount || node.totalItems || 0);
  const processedItems = Number(node.processedItems || 0);
  return failedItems > 0 || processedItems < totalItems;
}

function summarizeVisibleTree(result, mode = "directory") {
  const nodes = Array.isArray(result?.nodes) ? result.nodes : [];
  const rootCount = nodes.length;
  const leafCount = countTreeLeafNodes(nodes, mode);
  const deepestPath = deepestTreePath(nodes, "");
  const maxDepth = inferPathDepth(deepestPath);
  let problemCount = 0;
  if (mode === "directory") {
    const visit = (items) => {
      (items || []).forEach((node) => {
        if (directoryNodeHasProblem(node)) {
          problemCount += 1;
        }
        if (Array.isArray(node.children) && node.children.length) {
          visit(node.children);
        }
      });
    };
    visit(nodes);
  }
  return { rootCount, leafCount, deepestPath, maxDepth, problemCount };
}

function includesFilterText(values, text) {
  if (!text) {
    return true;
  }
  return values.some((value) => String(value || "").toLowerCase().includes(text));
}

function filterDirectoryTree(states, filters = {}) {
  const tree = Array.isArray(states) && states[0]?.children ? states : buildDirectoryStateTree(states);
  const query = String(filters.query || "").trim().toLowerCase();
  const status = String(filters.status || "").trim().toLowerCase();
  const leafOnly = Boolean(filters.leafOnly);
  const problemOnly = Boolean(filters.problemOnly);
  const filterActive = Boolean(query || status || leafOnly || problemOnly);
  const prune = (nodes) =>
    nodes.flatMap((node) => {
      const children = prune(node.children || []);
      const isLeaf = !node.children?.length;
      const selfMatch =
        includesFilterText([node.path, node.name, node.rootPath, node.lastItemPath], query) &&
        includesFilterText([node.status], status) &&
        (!leafOnly || isLeaf) &&
        (!problemOnly || directoryNodeHasProblem(node));
      if (!selfMatch && !children.length) {
        return [];
      }
      return [{ ...node, children }];
    });

  const filtered = prune(tree);
  return {
    nodes: filtered,
    totalNodes: countTreeNodes(tree),
    visibleNodes: countTreeNodes(filtered),
    filterActive,
  };
}

function filterPendingTree(nodes, filters = {}) {
  const tree = Array.isArray(nodes) && nodes[0]?.children !== undefined ? nodes : normalizePendingTree(nodes);
  const query = String(filters.query || "").trim().toLowerCase();
  const reason = String(filters.reason || "").trim().toLowerCase();
  const leafOnly = Boolean(filters.leafOnly);
  const filterActive = Boolean(query || reason || leafOnly);
  const prune = (items) =>
    items.flatMap((node) => {
      const children = prune(node.children || []);
      const isLeaf = !node.children?.length || node.nodeType === "file";
      const selfMatch =
        includesFilterText([node.path, node.name, node.rootPath], query) &&
        includesFilterText([node.reason, node.providerStatus, node.status], reason) &&
        (!leafOnly || isLeaf);
      if (!selfMatch && !children.length) {
        return [];
      }
      return [{ ...node, children }];
    });

  const filtered = prune(tree);
  return {
    nodes: filtered,
    totalNodes: countTreeNodes(tree),
    visibleNodes: countTreeNodes(filtered),
    filterActive,
  };
}

function renderTreeFilterSummary(result, label, mode = "directory") {
  if (!result.totalNodes) {
    return `暂无${label}。`;
  }
  const summary = summarizeVisibleTree(result, mode);
  const suffix = [
    `roots ${summary.rootCount}`,
    `leaf ${summary.leafCount}`,
    `maxDepth ${summary.maxDepth}`,
    mode === "directory" ? `problem ${summary.problemCount}` : "",
    summary.deepestPath ? `deepest ${summary.deepestPath}` : "",
  ]
    .filter(Boolean)
    .join(" / ");
  if (!result.filterActive) {
    return `显示全部 ${result.visibleNodes} 个${label}。${suffix}`;
  }
  return `当前显示 ${result.visibleNodes} / ${result.totalNodes} 个${label}。${suffix}`;
}

function resetTreeFilterSection(section) {
  if (!state.treeFilters[section]) {
    return;
  }
  Object.keys(state.treeFilters[section]).forEach((key) => {
    state.treeFilters[section][key] = typeof state.treeFilters[section][key] === "boolean" ? false : "";
  });
}

function filterRetryQueue(items, filters = {}) {
  const queue = normalizeRetryQueue(items);
  const query = String(filters.query || "").trim().toLowerCase();
  const retryClass = String(filters.retryClass || "").trim().toLowerCase();
  const retryState = String(filters.retryState || "").trim().toLowerCase();
  const filterActive = Boolean(query || retryClass || retryState);
  const visible = queue.filter((item) => {
    const matchesQuery = includesFilterText(
      [item.path, item.rootPath, item.reason, item.providerStatus, item.retryAction, item.strategy, item.uploadCheckpoint?.uploadId],
      query,
    );
    const matchesClass = includesFilterText([item.retryClass], retryClass);
    const stateValue = item.exhausted ? "exhausted" : item.blocked ? "blocked" : item.retryable ? "retryable" : "queued";
    const matchesState = includesFilterText([stateValue], retryState);
    return matchesQuery && matchesClass && matchesState;
  });
  return {
    items: visible,
    totalItems: queue.length,
    visibleItems: visible.length,
    filterActive,
  };
}

function renderRetryQueueSummary(result) {
  if (!result.totalItems) {
    return `<div class="directory-empty">当前没有需要后续重试的队列项。</div>`;
  }
  const retryable = result.items.filter((item) => item.retryable && !item.exhausted).length;
  const blocked = result.items.filter((item) => item.blocked && !item.exhausted).length;
  const exhausted = result.items.filter((item) => item.exhausted).length;
  return `
    <div class="retry-summary-grid">
      <div class="retry-card">
        <strong>当前显示</strong>
        <span>${result.visibleItems} / ${result.totalItems}</span>
      </div>
      <div class="retry-card">
        <strong>Retryable</strong>
        <span>${retryable}</span>
      </div>
      <div class="retry-card">
        <strong>Blocked</strong>
        <span>${blocked}</span>
      </div>
      <div class="retry-card">
        <strong>Exhausted</strong>
        <span>${exhausted}</span>
      </div>
    </div>
  `;
}

function renderRetryQueue(items, filters = {}) {
  const result = filterRetryQueue(items, filters);
  if (!result.totalItems) {
    return {
      html: `<div class="directory-empty">当前没有需要后续重试的队列项。</div>`,
      summaryText: "当前没有重试队列项。",
    };
  }
  const rows = result.items
    .map((item) => {
      const stateValue = item.exhausted ? "exhausted" : item.blocked ? "blocked" : item.retryable ? "retryable" : "queued";
      return `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(item.retryClass || "unknown")}</strong>
            <code>${escapeHTML(item.path)}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">${escapeHTML(stateValue)}</span>
            ${item.retryAction ? `<span class="pill">${escapeHTML(item.retryAction)}</span>` : ""}
            ${item.providerStatus ? `<span class="pill">${escapeHTML(item.providerStatus)}</span>` : ""}
            ${item.strategy ? `<span class="pill">${escapeHTML(item.strategy)}</span>` : ""}
          </div>
          <div class="muted">attempt ${item.attemptCount} / limit ${item.retryLimit || 0} / remaining ${item.remainingCount}</div>
          ${item.cooldownTier || item.cooldownSeconds ? `<div class="muted">cooldown: <code>${escapeHTML(item.cooldownTier || "custom")}</code> / ${escapeHTML(String(item.cooldownSeconds || 0))}s</div>` : ""}
          ${item.eligibleAt ? `<div class="muted">eligibleAt: <code>${escapeHTML(item.eligibleAt)}</code></div>` : ""}
          ${item.rootPath ? `<div class="muted">root: <code>${escapeHTML(item.rootPath)}</code></div>` : ""}
          ${item.reason ? `<div class="muted">reason: <code>${escapeHTML(item.reason)}</code></div>` : ""}
          <div class="muted">next-step: ${escapeHTML(renderBlockedSummary(item.retryAction, item.reason, item.eligibleAt || ""))}</div>
          ${item.uploadCheckpoint ? `<div class="muted">checkpoint: upload ${escapeHTML(stringifyValue(item.uploadCheckpoint.uploadId, "-"))} / next part ${escapeHTML(stringifyValue(item.uploadCheckpoint.nextPartNumber, "-"))} / uploaded ${escapeHTML(stringifyValue(item.uploadCheckpoint.uploadedPartCount, "0"))}</div>` : ""}
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-retry-focus-pending="${escapeHTML(item.rootPath || item.path)}"
            >定位待补传树</button>
            <button
              type="button"
              class="ghost"
              data-retry-focus-class="${escapeHTML(item.retryClass)}"
              data-retry-focus-state="${escapeHTML(stateValue)}"
            >只看同类队列</button>
          </div>
        </div>
      `;
    })
    .join("");
  const summaryText = result.filterActive
    ? `当前显示 ${result.visibleItems} / ${result.totalItems} 个重试队列项。`
    : `显示全部 ${result.totalItems} 个重试队列项。`;
  return {
    html: `${renderRetryQueueSummary(result)}${rows || `<div class="directory-empty">筛选后没有命中的重试队列项。</div>`}`,
    summaryText,
  };
}

function renderRetrySummaryBreakdown(summary) {
  if (!summary || typeof summary !== "object") {
    return "-";
  }
  const parts = [
    `ready ${stringifyValue(summary.retryableNowCount, "0")}`,
    `cooldown ${stringifyValue(summary.cooldownCount, "0")}`,
    `checkpoint ${stringifyValue(summary.uploadCheckpointEligible, "0")}`,
    `manual ${stringifyValue(summary.pendingManualCount, "0")}`,
    `auth ${stringifyValue(summary.authExpiredCount, "0")}`,
    `local ${stringifyValue(summary.localMissingCount, "0")}`,
    `exhausted ${stringifyValue(summary.exhaustedCount, "0")}`,
  ];
  return parts.join(" / ");
}

function renderAutoRecoverMode(summary) {
  if (!summary || typeof summary !== "object") {
    return "-";
  }
  if (summary.autoRecoverMode) {
    return String(summary.autoRecoverMode);
  }
  if (summary.autoRecoverEligible) {
    return "eligible";
  }
  return "manual_only";
}

function renderBlockedSummary(action, advice, nextRetryAt = "", autoRecoverAdvice = "") {
  const normalizedAction = String(action || "").trim();
  const normalizedAdvice = String(advice || "").trim();
  const normalizedNextRetryAt = String(nextRetryAt || "").trim();
  const normalizedAutoRecoverAdvice = String(autoRecoverAdvice || "").trim();
  if (!normalizedAction && !normalizedAdvice && !normalizedAutoRecoverAdvice) {
    return "-";
  }
  const actionLabelMap = {
    refresh_auth_profile: "刷新授权后继续",
    restore_local_source_file: "补回本地文件后继续",
    manual_intervention_required: "修复 provider 会话后继续",
    wait_for_cooldown: normalizedNextRetryAt ? `等待冷却到 ${normalizedNextRetryAt}` : "等待冷却结束后继续",
    wait_for_retry_window: normalizedNextRetryAt ? `等待时间窗到 ${normalizedNextRetryAt}` : "等待时间窗开放后继续",
    manual_confirmation_required: "人工确认后继续",
    review_and_reset_retry_strategy: "调整重试策略后继续",
  };
  const primary = actionLabelMap[normalizedAction] || normalizedAdvice || normalizedAutoRecoverAdvice || normalizedAction;
  if (normalizedAdvice && normalizedAdvice !== primary) {
    return `${primary} | ${normalizedAdvice}`;
  }
  if (normalizedAutoRecoverAdvice && normalizedAutoRecoverAdvice !== primary && normalizedAutoRecoverAdvice !== normalizedAdvice) {
    return `${primary} | ${normalizedAutoRecoverAdvice}`;
  }
  return primary;
}

function renderRuntimeCheckpoint(runtime, metadata = null, scope = "task") {
  if (!runtime || typeof runtime !== "object") {
    return `
      <div class="insight-card checkpoint-card">
        <strong>运行检查点</strong>
        <span>暂无运行时信息</span>
      </div>
    `;
  }
  return `
    <div class="insight-card checkpoint-card">
      <strong>执行状态</strong>
      <span>${stringifyValue(runtime.executionState)}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>暂停请求</strong>
      <span>${runtime.pauseRequested ? "waiting_current_item" : "-"}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>请求时间</strong>
      <span>${stringifyValue(runtime.pauseRequestedAt, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>当前根目录</strong>
      <span><code>${escapeHTML(stringifyValue(runtime.currentRoot, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>当前目录</strong>
      <span><code>${escapeHTML(stringifyValue(runtime.currentDirectory, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>上次完成</strong>
      <span><code>${escapeHTML(stringifyValue(runtime.lastCompletedPath, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>处理进度</strong>
      <span>${stringifyValue(runtime.processedCount, "0")} / next ${stringifyValue(runtime.nextSequence, "1")}</span>
    </div>
    ${metadata && typeof metadata === "object" ? renderRuntimePathChips("Selected Roots", metadata.selectedRoots || [], scope, "roots") : ""}
    ${metadata && typeof metadata === "object" ? renderRuntimePathChips("Scan Trace", metadata.scanTrace || [], scope, "scan") : ""}
    <div class="insight-card checkpoint-card">
      <strong>结果计数</strong>
      <span>done ${stringifyValue(runtime.doneCount, "0")} / skipped ${stringifyValue(runtime.skippedCount, "0")} / failed ${stringifyValue(runtime.failedCount, "0")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>源端删除记录</strong>
      <span>${stringifyValue(runtime.sourceDeletionCount, "0")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>风控命中</strong>
      <span>${stringifyValue(runtime.riskHitCount, "0")} / last ${stringifyValue(runtime.lastRiskStatus, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>重试队列</strong>
      <span>retryable ${stringifyValue(runtime.retryableCount, "0")} / blocked ${stringifyValue(runtime.blockedRetryCount, "0")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>阻塞原因</strong>
      <span>${stringifyValue(runtime.blockedReason, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>处理动作</strong>
      <span>${stringifyValue(runtime.blockedAction, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>处理建议</strong>
      <span>${stringifyValue(runtime.blockedAdvice, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>阻塞摘要</strong>
      <span>${escapeHTML(
        renderBlockedSummary(
          runtime.blockedAction,
          runtime.blockedAdvice,
          runtime.nextRetryAt,
          metadata?.retrySummary?.autoRecoverAdvice,
        ),
      )}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>下次自动补传</strong>
      <span>${stringifyValue(runtime.nextRetryAt, "-")}</span>
    </div>
    ${
      metadata?.retrySummary
        ? `
          <div class="insight-card checkpoint-card">
            <strong>后台补传候选</strong>
            <span>${escapeHTML(renderAutoRecoverMode(metadata.retrySummary))}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>队列拆分</strong>
            <span>${escapeHTML(renderRetrySummaryBreakdown(metadata.retrySummary))}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>自动补传提示</strong>
            <span>${escapeHTML(stringifyValue(metadata.retrySummary.autoRecoverAdvice, "-"))}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>恢复等待 - Auth 刷新</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingAuthRefreshTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>恢复等待 - 本地恢复</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingLocalRestoreTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>恢复等待 - Provider 会话</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingProviderSessionTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>恢复等待 - 手动确认</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingManualTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>恢复等待 - 限额超限</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingRetryLimitTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>恢复等待 - 时间窗</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingRetryWindowTasks, metadata.retrySummary.windowBlocked ? "1" : "0")}</span>
          </div>
        `
        : ""
    }
    ${renderSourceDeletionSummary(runtime.sourceDeletionRecords || metadata?.sourceDeletionRecords || [], runtime.sourceDeletionCount || metadata?.deletedEntryCount || 0, scope, scope)}
    ${renderUploadCheckpoint(runtime.uploadCheckpoint)}
  `;
}

function sourceDeletionRecordPaths(records) {
  const seen = new Set();
  return (Array.isArray(records) ? records : [])
    .map((item) => normalizeComparePath(firstNonEmpty(item?.path, "")))
    .filter((path) => {
      if (!path || seen.has(path)) {
        return false;
      }
      seen.add(path);
      return true;
    });
}

function renderSourceDeletionSummary(records, count = 0, scope = "task", prefillScope = null) {
  const items = Array.isArray(records) ? records.filter((item) => item && typeof item === "object") : [];
  const resolvedCount = Number(count || items.length || 0);
  if (!resolvedCount && !items.length) {
    return "";
  }
  const paths = sourceDeletionRecordPaths(items);
  const canPrefill = Boolean(prefillScope && paths.length);
  const shown = items.slice(0, 4);
  const rows = shown
    .map((item) => {
      const path = firstNonEmpty(item.path, "-");
      const reason = firstNonEmpty(item.deleteReason, item.reason, "-");
      const deletedAt = firstNonEmpty(item.deletedAt, "-");
      return `
        <div class="muted source-deletion-row">
          <div class="actions compact">
            <button
              type="button"
              class="ghost path-chip"
              data-runtime-focus-path="${escapeHTML(path)}"
              data-runtime-focus-scope="${escapeHTML(scope)}"
              data-runtime-focus-kind="roots"
            >${escapeHTML(path)}</button>
            ${canPrefill && path !== "-" ? `
              <button
                type="button"
                class="ghost"
                data-source-delete-prefill-path="${escapeHTML(path)}"
                data-source-delete-prefill-scope="${escapeHTML(prefillScope)}"
              >用此删除记录重建向导</button>
            ` : ""}
          </div>
          <span> | reason ${escapeHTML(reason)} | deletedAt ${escapeHTML(deletedAt)}</span>
        </div>
      `;
    })
    .join("");
  return `
    <div class="insight-card checkpoint-card">
      <strong>删除记录摘要</strong>
      <div>${resolvedCount} 条，默认只记录，不会自动删除目标端真实文件。</div>
      ${canPrefill ? `
        <div class="actions compact">
          <button
            type="button"
            class="ghost"
            data-source-delete-prefill-paths="${escapeHTML(JSON.stringify(paths))}"
            data-source-delete-prefill-scope="${escapeHTML(prefillScope)}"
            data-source-delete-prefill-label="全部删除记录"
          >按全部删除记录重建向导</button>
        </div>
      ` : ""}
      ${rows || '<div class="muted">暂无可展开样本。</div>'}
      ${items.length > shown.length ? `<div class="muted">还有 ${items.length - shown.length} 条未展开。</div>` : ""}
    </div>
  `;
}

function renderUploadCheckpointPartSummary(uploadedParts) {
  if (!Array.isArray(uploadedParts) || !uploadedParts.length) {
    return "-";
  }
  return uploadedParts
    .slice(0, 3)
    .map((item) => {
      if (!item || typeof item !== "object") {
        return "?";
      }
      const partNumber = stringifyValue(item.partNumber, item.part_index ?? "?");
      const etag = firstNonEmpty(item.etag, item.eTag, item.partEtag, item.partETag);
      return etag ? `${partNumber}:${etag}` : String(partNumber);
    })
    .join(" / ");
}

function renderUploadCheckpointProviderDataSummary(providerData) {
  if (!providerData || typeof providerData !== "object") {
    return "-";
  }
  const entries = Object.entries(providerData)
    .filter(([key]) => String(key || "").trim() !== "")
    .slice(0, 4)
    .map(([key, value]) => {
      if (value && typeof value === "object") {
        const nestedKeys = Object.keys(value).filter(Boolean).slice(0, 3);
        return `${key}{${nestedKeys.join(", ") || "..."}}`;
      }
      return `${key}=${stringifyValue(value, "-")}`;
    });
  return entries.length ? entries.join(" / ") : "-";
}

function renderUploadCheckpointResumeState(checkpoint) {
  if (!checkpoint || typeof checkpoint !== "object") {
    return "-";
  }
  const resumable =
    firstNonEmpty(checkpoint.uploadId, checkpoint.fileId) ||
    Number(checkpoint.partCount || 0) > 0 ||
    Number(checkpoint.nextPartNumber || 0) > 0 ||
    Number(checkpoint.uploadedPartCount || 0) > 0 ||
    (Array.isArray(checkpoint.uploadedParts) && checkpoint.uploadedParts.length > 0) ||
    (checkpoint.providerData && Object.keys(checkpoint.providerData).length > 0);
  return resumable ? "可继续续传" : "仍需重新建会话";
}

function renderUploadCheckpointReadiness(evidence) {
  if (evidence?.uploadCheckpointResumeReadiness) {
    return evidence.uploadCheckpointResumeReadiness;
  }
  const resumeCount = Number(evidence?.uploadCheckpointResumeTaskCount || 0);
  if (resumeCount <= 0) {
    return "pending";
  }
  const hasUploadID = Boolean(String(evidence?.uploadCheckpointResumeSampleUploadId || "").trim());
  const hasPartEvidence = Number(evidence?.uploadCheckpointResumeSampleNextPart || 0) > 0
    || Number(evidence?.uploadCheckpointResumeSampleUploaded || 0) > 0
    || Number(evidence?.uploadCheckpointResumeSamplePartCount || 0) > 0;
  return hasUploadID && hasPartEvidence ? "ready" : "partial";
}

function renderUploadCheckpointPriorityAction(evidence) {
  if (evidence?.uploadCheckpointResumePriorityAction) {
    return evidence.uploadCheckpointResumePriorityAction;
  }
  if (Number(evidence?.uploadCheckpointTaskCount || 0) <= 0) {
    return "优先形成 1 条 upload checkpoint 失败样本";
  }
  if (Number(evidence?.uploadCheckpointResumeTaskCount || 0) <= 0) {
    return "优先补 1 条 upload checkpoint 自动续传成功样本";
  }
  const hasUploadID = Boolean(String(evidence?.uploadCheckpointResumeSampleUploadId || "").trim());
  const hasPartEvidence = Number(evidence?.uploadCheckpointResumeSampleNextPart || 0) > 0
    || Number(evidence?.uploadCheckpointResumeSampleUploaded || 0) > 0
    || Number(evidence?.uploadCheckpointResumeSamplePartCount || 0) > 0;
  if (!hasUploadID) {
    return "优先补 uploadId / upload session 证据";
  }
  if (!hasPartEvidence) {
    return "优先补 nextPart / uploadedParts 分片进度证据";
  }
  return "complete";
}

function renderAutoRecoverReadiness(evidence) {
  const hasRunnable = Number(evidence?.autoRecoverRunnableTasks || 0) > 0;
  const hasBlockingWait = Number(evidence?.autoRecoverWaitingProviderSessionTasks || 0) > 0
    || Number(evidence?.autoRecoverWaitingAuthRefreshTasks || 0) > 0
    || Number(evidence?.autoRecoverWaitingLocalRestoreTasks || 0) > 0
    || Number(evidence?.autoRecoverWaitingManualTasks || 0) > 0
    || Number(evidence?.autoRecoverWaitingRetryLimitTasks || 0) > 0;
  const hasSoftWait = Number(evidence?.autoRecoverWaitingRetryWindowTasks || 0) > 0
    || Number(evidence?.autoRecoverWaitingCooldownTasks || 0) > 0;
  const hasEvidenceGap = Number(evidence?.uploadCheckpointTaskCount || 0) > 0
    && Number(evidence?.uploadCheckpointResumeTaskCount || 0) <= 0;
  const hasFairnessGap = Array.isArray(evidence?.autoRecoverPool) && evidence.autoRecoverPool.length > 0;
  if (!hasRunnable && !hasBlockingWait && !hasSoftWait && !hasEvidenceGap && !hasFairnessGap) {
    return "ready";
  }
  if (hasRunnable && !hasBlockingWait && !hasEvidenceGap) {
    return "ready";
  }
  if (hasBlockingWait || hasEvidenceGap) {
    return "pending";
  }
  return "partial";
}

function renderAutoRecoverFairnessReadiness(evidence) {
  if (evidence?.autoRecoverFairnessReadiness) {
    return evidence.autoRecoverFairnessReadiness;
  }
  const pool = Array.isArray(evidence?.autoRecoverPool) ? evidence.autoRecoverPool : [];
  if (!pool.length) {
    return "pending";
  }
  const hasMultiProvider = pool.some((item) => Number(item?.providerCount || 0) > 1);
  const hasMultiProfile = pool.some((item) => Number(item?.profileCount || 0) > 1);
  const hasMultiProtocolGroup = pool.some((item) => Array.isArray(item?.protocolGroups) && item.protocolGroups.length > 1);
  if (hasMultiProvider && hasMultiProfile) {
    return "ready";
  }
  if (hasMultiProvider || hasMultiProfile || hasMultiProtocolGroup) {
    return "partial";
  }
  return "pending";
}
function renderAutoRecoverFairnessPriorityAction(evidence) {
  if (evidence?.autoRecoverFairnessPriorityAction) {
    return evidence.autoRecoverFairnessPriorityAction;
  }
  const pool = Array.isArray(evidence?.autoRecoverPool) ? evidence.autoRecoverPool : [];
  if (!pool.length) {
    return "优先形成 1 条自动补传候选池样本";
  }
  const hasMultiProvider = pool.some((item) => Number(item?.providerCount || 0) > 1);
  const hasMultiProfile = pool.some((item) => Number(item?.profileCount || 0) > 1);
  const hasMultiProtocolGroup = pool.some((item) => Array.isArray(item?.protocolGroups) && item.protocolGroups.length > 1);
  if (!hasMultiProvider) {
    return "优先补多 provider 自动补传候选池样本";
  }
  if (!hasMultiProfile) {
    return "优先补多账号自动补传候选池样本";
  }
  if (!hasMultiProtocolGroup) {
    return "优先补多协议组自动补传候选池样本";
  }
  return "complete";
}
function renderAutoRecoverFairnessMissing(evidence) {
  const missing = Array.isArray(evidence?.autoRecoverFairnessMissing) ? evidence.autoRecoverFairnessMissing : [];
  return missing.length ? missing.join(", ") : "complete";
}
function renderAutoRecoverPriorityAction(evidence) {
  if (Number(evidence?.autoRecoverWaitingProviderSessionTasks || 0) > 0) {
    return "优先重建 provider 会话缺口";
  }
  if (Number(evidence?.autoRecoverWaitingAuthRefreshTasks || 0) > 0) {
    return "优先刷新授权档案后再恢复任务";
  }
  if (Number(evidence?.autoRecoverWaitingLocalRestoreTasks || 0) > 0) {
    return "优先补回本地缺失文件";
  }
  if (Number(evidence?.autoRecoverWaitingManualTasks || 0) > 0) {
    return "优先处理 pending_manual / 人工确认任务";
  }
  if (Number(evidence?.autoRecoverWaitingRetryLimitTasks || 0) > 0) {
    return "优先处理重试耗尽任务";
  }
  if (Number(evidence?.autoRecoverWaitingRetryWindowTasks || 0) > 0) {
    return "优先评估自动补传时间窗是否需要放宽";
  }
  if (Number(evidence?.autoRecoverWaitingCooldownTasks || 0) > 0) {
    return "优先等待冷却结束后继续自动重试";
  }
  if (Number(evidence?.autoRecoverRunnableTasks || 0) > 0) {
    return "优先放行当前可立即自动补传的任务";
  }
  if (Number(evidence?.uploadCheckpointTaskCount || 0) > 0 && Number(evidence?.uploadCheckpointResumeTaskCount || 0) <= 0) {
    return "优先补 1 条 upload checkpoint 自动续传成功样本";
  }
  if (Array.isArray(evidence?.autoRecoverPool) && evidence.autoRecoverPool.length > 0) {
    return "优先继续补多 provider / 多账号公平性样本";
  }
  return "complete";
}


function renderUploadCheckpoint(checkpoint) {
  if (!checkpoint || typeof checkpoint !== "object") {
    return "";
  }
  const uploadedPartCount = stringifyValue(checkpoint.uploadedPartCount, "0");
  const partCount = stringifyValue(checkpoint.partCount, "0");
  const uploadedPartsLen = Array.isArray(checkpoint.uploadedParts) ? checkpoint.uploadedParts.length : 0;
  return `
    <div class="insight-card checkpoint-card">
      <strong>上传恢复文件</strong>
      <span><code>${escapeHTML(stringifyValue(checkpoint.itemPath, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>上传会话</strong>
      <span>${stringifyValue(checkpoint.uploadId, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>续传就绪</strong>
      <span>${escapeHTML(renderUploadCheckpointResumeState(checkpoint))}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>上传分片进度</strong>
      <span>${uploadedPartCount} / ${partCount}，证据 ${uploadedPartsLen} 段</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>已传分片摘要</strong>
      <span>${escapeHTML(renderUploadCheckpointPartSummary(checkpoint.uploadedParts))}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>失败分片</strong>
      <span>${stringifyValue(checkpoint.failedPartNumber, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>下一个分片</strong>
      <span>${stringifyValue(checkpoint.nextPartNumber, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>Provider 恢复线索</strong>
      <span>${escapeHTML(renderUploadCheckpointProviderDataSummary(checkpoint.providerData))}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>上传状态</strong>
      <span>${stringifyValue(checkpoint.providerStatus, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>检查点时间</strong>
      <span>${stringifyValue(checkpoint.updatedAt, "-")}</span>
    </div>
  `;
}

function renderTreeNodes(nodes, options = {}) {
  const {
    mode = "directory",
    emptyMessage = "暂无数据。",
    normalized = false,
    scope = "global",
    panel = "tree",
  } = options;
  const tree = normalized ? nodes : mode === "pending" ? normalizePendingTree(nodes) : buildDirectoryStateTree(nodes);
  if (!tree.length) {
    return `<div class="directory-empty">${escapeHTML(emptyMessage)}</div>`;
  }

  const renderNode = (node) => {
    const childrenList = Array.isArray(node.children) ? node.children : [];
    const hasChildren = childrenList.length > 0;
    const collapsed = hasChildren && isTreeGroupCollapsed(scope, panel, node.path);
    const leafCount = countTreeLeafNodes(hasChildren ? childrenList : [node], mode);
    const descendantCount = countTreeNodes(childrenList);
    const parentPath = parentTreePath(node.path);
    const metrics =
      mode === "pending"
        ? `
            <div class="directory-metrics">
              <span class="pill">${escapeHTML(node.nodeType)}</span>
              <span class="pill">pending ${node.itemCount}</span>
              <span class="pill">leaf ${leafCount}</span>
              ${descendantCount ? `<span class="pill">children ${descendantCount}</span>` : ""}
              ${node.providerStatus ? `<span class="pill">${escapeHTML(node.providerStatus)}</span>` : ""}
            </div>
            ${
              node.reason
                ? `<div class="muted">reason: <code>${escapeHTML(node.reason)}</code></div>`
                : ""
            }
          `
        : `
            <div class="directory-metrics">
              <span class="pill">${escapeHTML(node.status)}</span>
              <span class="pill">processed ${node.processedItems}/${node.itemCount}</span>
              <span class="pill">done ${node.doneItems}</span>
              <span class="pill">skipped ${node.skippedItems}</span>
              <span class="pill">failed ${node.failedItems}</span>
              <span class="pill">leaf ${leafCount}</span>
              ${descendantCount ? `<span class="pill">children ${descendantCount}</span>` : ""}
            </div>
            <div class="muted">last item: <code>${escapeHTML(stringifyValue(node.lastItemPath, "-"))}</code></div>
          `;

    const children = hasChildren
      ? `<div class="directory-children ${collapsed ? "is-collapsed" : ""}">${childrenList.map((child) => renderNode(child)).join("")}</div>`
      : "";
    const syncLabel = panel === "directory" ? "待补传树" : "目录树";

    return `
      <div class="directory-row tree-node ${collapsed ? "is-collapsed" : ""}">
        <div class="directory-row-header">
          <strong>${escapeHTML(node.name || node.path)}</strong>
          <code>${escapeHTML(node.path)}</code>
        </div>
        ${metrics}
        <div class="actions compact">
          ${
            hasChildren
              ? `
                <button
                  type="button"
                  class="ghost tree-group-toggle"
                  data-tree-group-toggle
                  data-tree-group-scope="${escapeHTML(scope)}"
                  data-tree-group-panel="${escapeHTML(panel)}"
                  data-tree-group-path="${escapeHTML(node.path)}"
                  aria-expanded="${collapsed ? "false" : "true"}"
                >${collapsed ? "展开子树" : "收起子树"}</button>
              `
              : ""
          }
          ${
            scope === "task"
              ? `
                <button
                  type="button"
                  class="ghost"
                  data-tree-prefill-path="${escapeHTML(node.path)}"
                  data-tree-prefill-scope="${escapeHTML(scope)}"
                  data-tree-prefill-panel="${escapeHTML(panel)}"
                >按当前路径重建向导</button>
              `
              : ""
          }
          ${
            (scope === "task" || scope === "status") && panel === "pending"
              ? `
                <button
                  type="button"
                  class="ghost"
                  data-tree-retry-path="${escapeHTML(node.path)}"
                  data-tree-retry-scope="${escapeHTML(scope)}"
                  data-tree-retry-panel="${escapeHTML(panel)}"
                >重试当前路径</button>
              `
              : ""
          }
          ${
            scope === "task" || scope === "status"
              ? `
                <button
                  type="button"
                  class="ghost"
                  data-tree-auto-recover-path="${escapeHTML(node.path)}"
                  data-tree-auto-recover-panel="${escapeHTML(panel)}"
                >后台补传当前路径</button>
              `
              : ""
          }
          <button
            type="button"
            class="ghost"
            data-tree-copy-path="${escapeHTML(node.path)}"
            data-tree-copy-scope="${escapeHTML(scope)}"
            data-tree-copy-panel="${escapeHTML(panel)}"
          >复制当前子树</button>
          ${
            parentPath
              ? `
                <button
                  type="button"
                  class="ghost"
                  data-tree-parent-path="${escapeHTML(node.path)}"
                  data-tree-parent-scope="${escapeHTML(scope)}"
                  data-tree-parent-panel="${escapeHTML(panel)}"
                >只看父级</button>
              `
              : ""
          }
          <button
            type="button"
            class="ghost"
            data-tree-focus-path="${escapeHTML(node.path)}"
            data-tree-focus-scope="${escapeHTML(scope)}"
            data-tree-focus-panel="${escapeHTML(panel)}"
          >只看当前路径</button>
          <button
            type="button"
            class="ghost"
            data-tree-sync-path="${escapeHTML(node.path)}"
            data-tree-sync-scope="${escapeHTML(scope)}"
            data-tree-sync-panel="${escapeHTML(panel)}"
          >同步到${escapeHTML(syncLabel)}</button>
        </div>
        ${children}
      </div>
    `;
  };

  return tree
    .map(
      (root) => {
        const rootPath = root.rootPath || root.path;
        const collapsed = isTreeGroupCollapsed(scope, panel, rootPath);
        const summary =
          mode === "pending"
            ? `
                <div class="directory-group-summary">
                  <span class="pill">pending ${stringifyValue(root.itemCount, "0")}</span>
                  <span class="pill">children ${stringifyValue(countTreeNodes(root.children || []), "0")}</span>
                  ${root.providerStatus ? `<span class="pill">${escapeHTML(root.providerStatus)}</span>` : ""}
                </div>
              `
            : `
                <div class="directory-group-summary">
                  <span class="pill">${escapeHTML(root.status)}</span>
                  <span class="pill">processed ${stringifyValue(root.processedItems, "0")}/${stringifyValue(root.itemCount, "0")}</span>
                  <span class="pill">done ${stringifyValue(root.doneItems, "0")}</span>
                  <span class="pill">skipped ${stringifyValue(root.skippedItems, "0")}</span>
                  <span class="pill">failed ${stringifyValue(root.failedItems, "0")}</span>
                </div>
              `;
        return `
          <section class="directory-group ${collapsed ? "is-collapsed" : ""}" data-tree-group-key="${escapeHTML(treeGroupCollapseKey(scope, panel, rootPath))}">
            <div class="directory-group-header">
              <div class="directory-group-title">
                <h4>Root <code>${escapeHTML(rootPath)}</code></h4>
                ${summary}
              </div>
              <div class="actions compact">
                ${
                  scope === "task"
                    ? `
                      <button
                        type="button"
                        class="ghost"
                        data-tree-prefill-path="${escapeHTML(rootPath)}"
                        data-tree-prefill-scope="${escapeHTML(scope)}"
                        data-tree-prefill-panel="${escapeHTML(panel)}"
                      >按当前 root 重建向导</button>
                    `
                    : ""
                }
                ${
                  (scope === "task" || scope === "status") && panel === "pending"
                    ? `
                      <button
                        type="button"
                        class="ghost"
                        data-tree-retry-path="${escapeHTML(rootPath)}"
                        data-tree-retry-scope="${escapeHTML(scope)}"
                        data-tree-retry-panel="${escapeHTML(panel)}"
                      >重试当前 root</button>
                    `
                    : ""
                }
                ${
                  scope === "task" || scope === "status"
                    ? `
                      <button
                        type="button"
                        class="ghost"
                        data-tree-auto-recover-path="${escapeHTML(rootPath)}"
                        data-tree-auto-recover-panel="${escapeHTML(panel)}"
                      >后台补传当前 root</button>
                    `
                    : ""
                }
                <button
                  type="button"
                  class="ghost"
                  data-tree-focus-path="${escapeHTML(rootPath)}"
                  data-tree-focus-scope="${escapeHTML(scope)}"
                  data-tree-focus-panel="${escapeHTML(panel)}"
                >只看 root</button>
                <button
                  type="button"
                  class="ghost"
                  data-tree-sync-path="${escapeHTML(rootPath)}"
                  data-tree-sync-scope="${escapeHTML(scope)}"
                  data-tree-sync-panel="${escapeHTML(panel)}"
                >同步另一棵树</button>
                <button
                  type="button"
                  class="ghost tree-group-toggle"
                  data-tree-group-toggle
                  data-tree-group-scope="${escapeHTML(scope)}"
                  data-tree-group-panel="${escapeHTML(panel)}"
                  data-tree-group-path="${escapeHTML(rootPath)}"
                  aria-expanded="${collapsed ? "false" : "true"}"
                >${collapsed ? "展开" : "收起"}</button>
              </div>
            </div>
            <div class="directory-group-body">
              ${renderNode(root)}
            </div>
          </section>
        `;
      },
    )
    .join("");
}

function renderDirectoryStates(states) {
  const normalized = normalizeDirectoryStates(states);
  if (!normalized.length) {
    return `<div class="directory-empty">暂无目录状态。</div>`;
  }
  return renderTreeNodes(states, { mode: "directory", emptyMessage: "暂无目录状态。" });
}

function renderPendingTree(nodes) {
  return renderTreeNodes(nodes, { mode: "pending", emptyMessage: "暂无待补传项。" });
}

function currentSelectedTaskDetail() {
  return state.tasks.find((item) => item.task.id === state.selectedTaskId) || null;
}

function currentStatusTaskContext() {
  const runtimePayload = recentRuntimePayload();
  const taskId = String(runtimePayload?.taskId || runtimePayload?.runtime?.taskId || "").trim();
  if (!taskId) {
    return null;
  }
  const detail = state.tasks.find((item) => item?.task?.id === taskId) || null;
  const providerKey = String(
    runtimePayload?.providerKey || runtimePayload?.targetProvider || runtimePayload?.runtime?.targetProvider || detail?.task?.targetProvider || "",
  ).trim();
  return {
    taskId,
    providerKey,
    detail,
  };
}

function updateTaskTreePanels(detail) {
  const runtime = detail?.runtime || detail?.plan?.metadata?.runtime || {};
  const directoryResult = filterDirectoryTree(runtime.directoryStates || [], state.treeFilters.taskDirectory);
  const pendingResult = filterPendingTree(runtime.pendingTree || [], state.treeFilters.taskPending);
  $("#task-directory-states").innerHTML = renderTreeNodes(directoryResult.nodes, {
    mode: "directory",
    emptyMessage: "暂无目录状态。",
    normalized: true,
    scope: "task",
    panel: "directory",
  });
  $("#task-pending-tree").innerHTML = renderTreeNodes(pendingResult.nodes, {
    mode: "pending",
    emptyMessage: "暂无待补传项。",
    normalized: true,
    scope: "task",
    panel: "pending",
  });
  $("#task-directory-filter-summary").textContent = detail
    ? renderTreeFilterSummary(directoryResult, "目录节点", "directory")
    : "等待任务数据...";
  $("#task-pending-filter-summary").textContent = detail
    ? renderTreeFilterSummary(pendingResult, "待补传节点", "pending")
    : "等待任务数据...";
  wireTreeBulkActions("task", "directory");
  wireTreeBulkActions("task", "pending");
  wireTreeGroupToggles("task", "directory");
  wireTreeGroupToggles("task", "pending");
}

function updateStatusTreePanels(runtimePayload) {
  const runtime = runtimePayload?.runtime || runtimePayload || {};
  const directoryResult = filterDirectoryTree(runtime.directoryStates || [], state.treeFilters.statusDirectory);
  const pendingResult = filterPendingTree(runtime.pendingTree || [], state.treeFilters.statusPending);
  $("#status-directory-states").innerHTML = renderTreeNodes(directoryResult.nodes, {
    mode: "directory",
    emptyMessage: "暂无目录状态。",
    normalized: true,
    scope: "status",
    panel: "directory",
  });
  $("#status-pending-tree").innerHTML = renderTreeNodes(pendingResult.nodes, {
    mode: "pending",
    emptyMessage: "暂无待补传项。",
    normalized: true,
    scope: "status",
    panel: "pending",
  });
  $("#status-directory-filter-summary").textContent = renderTreeFilterSummary(directoryResult, "目录节点", "directory");
  $("#status-pending-filter-summary").textContent = renderTreeFilterSummary(pendingResult, "待补传节点", "pending");
  wireTreeBulkActions("status", "directory");
  wireTreeBulkActions("status", "pending");
  wireTreeGroupToggles("status", "directory");
  wireTreeGroupToggles("status", "pending");
}

function updateTaskRetryQueue(detail) {
  const runtime = detail?.runtime || detail?.plan?.metadata?.runtime || {};
  const rendered = renderRetryQueue(runtime.retryQueue || [], state.treeFilters.taskRetry);
  $("#task-retry-queue").innerHTML = rendered.html;
  $("#task-retry-filter-summary").textContent = detail ? rendered.summaryText : "等待任务数据...";
  wireRetryQueueActions("task");
}

function updateStatusRetryQueue(runtimePayload) {
  const runtime = runtimePayload?.runtime || runtimePayload || {};
  const rendered = renderRetryQueue(runtime.retryQueue || [], state.treeFilters.statusRetry);
  $("#status-retry-queue").innerHTML = rendered.html;
  $("#status-retry-filter-summary").textContent = rendered.summaryText;
  wireRetryQueueActions("status");
}

function flattenVisibleTreePaths(nodes, mode) {
  if (!Array.isArray(nodes) || !nodes.length) {
    return [];
  }
  const paths = [];
  const visit = (items) => {
    items.forEach((node) => {
      if (!node || !node.path) {
        return;
      }
      if (mode === "pending") {
        if (node.nodeType === "file" || !Array.isArray(node.children) || node.children.length === 0) {
          paths.push(node.path);
        }
      } else {
        paths.push(node.path);
      }
      if (Array.isArray(node.children) && node.children.length) {
        visit(node.children);
      }
    });
  };
  visit(nodes);
  return Array.from(new Set(paths));
}

function currentVisibleTreeNodes(scope, panel) {
  const detail = currentSelectedTaskDetail();
  const runtimePayload = recentRuntimePayload();
  if (scope === "task") {
    const runtime = detail?.runtime || detail?.plan?.metadata?.runtime || {};
    if (panel === "pending") {
      return filterPendingTree(runtime.pendingTree || [], state.treeFilters.taskPending).nodes;
    }
    return filterDirectoryTree(runtime.directoryStates || [], state.treeFilters.taskDirectory).nodes;
  }
  const runtime = runtimePayload?.runtime || runtimePayload || {};
  if (panel === "pending") {
    return filterPendingTree(runtime.pendingTree || [], state.treeFilters.statusPending).nodes;
  }
  return filterDirectoryTree(runtime.directoryStates || [], state.treeFilters.statusDirectory).nodes;
}

function visibleTreePaths(scope, panel) {
  return flattenVisibleTreePaths(currentVisibleTreeNodes(scope, panel), panel === "pending" ? "pending" : "directory");
}

function visibleRetryPaths(scope) {
  const detail = currentSelectedTaskDetail();
  const runtimePayload = recentRuntimePayload();
  const runtime = scope === "task" ? detail?.runtime || detail?.plan?.metadata?.runtime || {} : runtimePayload?.runtime || runtimePayload || {};
  const filters = scope === "task" ? state.treeFilters.taskRetry : state.treeFilters.statusRetry;
  const result = filterRetryQueue(runtime.retryQueue || [], filters);
  return Array.from(new Set((result.items || []).map((item) => item.path).filter(Boolean)));
}

function findTreeNodeByPath(nodes, path) {
  const normalizedPath = normalizeComparePath(path);
  if (!normalizedPath) {
    return null;
  }
  const visit = (items) => {
    for (const node of items || []) {
      if (normalizeComparePath(node?.path) === normalizedPath) {
        return node;
      }
      if (Array.isArray(node?.children) && node.children.length) {
        const found = visit(node.children);
        if (found) {
          return found;
        }
      }
    }
    return null;
  };
  return visit(nodes);
}

function flattenTreeNodePaths(node, mode = "directory") {
  if (!node || !node.path) {
    return [];
  }
  const paths = [];
  const visit = (current) => {
    if (!current || !current.path) {
      return;
    }
    const children = Array.isArray(current.children) ? current.children : [];
    if (mode === "pending") {
      if (current.nodeType === "file" || !children.length) {
        paths.push(current.path);
      }
    } else {
      paths.push(current.path);
    }
    children.forEach(visit);
  };
  visit(node);
  return Array.from(new Set(paths));
}

async function copyVisibleTreePaths(scope, panel) {
  const paths = visibleTreePaths(scope, panel);
  if (!paths.length) {
    showFlash(`当前${panel === "pending" ? "待补传树" : "目录树"}没有可复制的路径`, true);
    return;
  }
  await copyTextToClipboard(paths.join("\n"));
  showFlash(`已复制 ${paths.length} 条${panel === "pending" ? "待补传" : "目录"}路径`);
}

async function copyTreeNodePaths(scope, panel, path) {
  const nodes = currentVisibleTreeNodes(scope, panel);
  const node = findTreeNodeByPath(nodes, path);
  if (!node) {
    showFlash("当前筛选结果里未找到对应子树", true);
    return;
  }
  const paths = flattenTreeNodePaths(node, panel === "pending" ? "pending" : "directory");
  if (!paths.length) {
    showFlash("当前子树没有可复制的路径", true);
    return;
  }
  await copyTextToClipboard(paths.join("\n"));
  showFlash(`已复制 ${paths.length} 条${panel === "pending" ? "待补传" : "目录"}子树路径`);
}

function focusTreeParentPath(scope, panel, path) {
  const parentPath = parentTreePath(path);
  if (!parentPath) {
    showFlash("当前已经是最上层路径", true);
    return;
  }
  focusTreePanelByPath(scope, panel, parentPath);
  showFlash(`已按父级路径 ${parentPath} 收敛${panel === "pending" ? "待补传树" : "目录树"}`);
}

function setFilterControlValue(selector, value) {
  const element = $(selector);
  if (!element) {
    return;
  }
  if (element.type === "checkbox") {
    element.checked = Boolean(value);
    return;
  }
  element.value = value;
}

function treePanelFilterSelector(scope, panel) {
  if (scope === "task") {
    return panel === "pending" ? "#task-pending-filter-query" : "#task-directory-filter-query";
  }
  return panel === "pending" ? "#status-pending-filter-query" : "#status-directory-filter-query";
}

function rerenderTreeScope(scope) {
  if (scope === "task") {
    updateTaskTreePanels(currentSelectedTaskDetail());
    return;
  }
  updateStatusTreePanels(recentRuntimePayload());
}

function focusTreePanelByPath(scope, panel, path) {
  const normalized = String(path || "").trim();
  if (!normalized) {
    return;
  }
  const selector = treePanelFilterSelector(scope, panel);
  if (scope === "task") {
    state.treeFilters[panel === "pending" ? "taskPending" : "taskDirectory"].query = normalized;
  } else {
    state.treeFilters[panel === "pending" ? "statusPending" : "statusDirectory"].query = normalized;
  }
  setFilterControlValue(selector, normalized);
  rerenderTreeScope(scope);
  showFlash(`已按 ${normalized} 收敛${panel === "pending" ? "待补传树" : "目录树"}`);
}

function syncTreePanelPath(scope, fromPanel, path) {
  const targetPanel = fromPanel === "directory" ? "pending" : "directory";
  focusTreePanelByPath(scope, targetPanel, path);
}

function focusPendingTreeFromRetry(scope, path) {
  if (!path) {
    return;
  }
  if (scope === "task") {
    state.treeFilters.taskPending.query = path;
    setFilterControlValue("#task-pending-filter-query", path);
    updateTaskTreePanels(currentSelectedTaskDetail());
    showFlash("已按当前重试项定位待补传树");
    return;
  }
  state.treeFilters.statusPending.query = path;
  setFilterControlValue("#status-pending-filter-query", path);
  updateStatusTreePanels(recentRuntimePayload());
  showFlash("已按当前重试项定位最近待补传树");
}

function focusRetryClass(scope, retryClass, retryState) {
  if (scope === "task") {
    state.treeFilters.taskRetry.retryClass = retryClass || "";
    state.treeFilters.taskRetry.retryState = retryState || "";
    setFilterControlValue("#task-retry-filter-class", retryClass || "");
    setFilterControlValue("#task-retry-filter-state", retryState || "");
    updateTaskRetryQueue(currentSelectedTaskDetail());
    showFlash("已收敛到当前同类重试队列");
    return;
  }
  state.treeFilters.statusRetry.retryClass = retryClass || "";
  state.treeFilters.statusRetry.retryState = retryState || "";
  setFilterControlValue("#status-retry-filter-class", retryClass || "");
  setFilterControlValue("#status-retry-filter-state", retryState || "");
  updateStatusRetryQueue(recentRuntimePayload());
  showFlash("已收敛到最近同类重试队列");
}

function focusTaskRetryByBlockedAction(action) {
  const preset = blockedActionFilterPreset(action);
  state.treeFilters.taskRetry.retryClass = preset.retryClass;
  state.treeFilters.taskRetry.retryState = preset.retryState;
  setFilterControlValue("#task-retry-filter-class", preset.retryClass);
  setFilterControlValue("#task-retry-filter-state", preset.retryState);
  updateTaskRetryQueue(currentSelectedTaskDetail());
  showFlash("已按当前 blocked action 收敛任务重试队列");
}

function firstPendingFocusPath(detail) {
  const runtime = detail?.runtime || detail?.plan?.metadata?.runtime || {};
  if (Array.isArray(runtime.retryQueue) && runtime.retryQueue.length) {
    const candidate = runtime.retryQueue.find((item) => item.rootPath || item.path);
    if (candidate) {
      return candidate.rootPath || candidate.path;
    }
  }
  if (Array.isArray(runtime.pendingTree) && runtime.pendingTree.length) {
    return runtime.pendingTree[0].rootPath || runtime.pendingTree[0].path;
  }
  if (detail?.plan?.metadata?.retryPendingPaths?.length) {
    return detail.plan.metadata.retryPendingPaths[0];
  }
  if (detail?.plan?.metadata?.selectedRoots?.length) {
    return detail.plan.metadata.selectedRoots[0];
  }
  return "";
}

function focusTaskPendingByDetail(detail) {
  const path = firstPendingFocusPath(detail);
  if (!path) {
    showFlash("当前任务没有可定位的待补传路径", true);
    return;
  }
  state.treeFilters.taskPending.query = path;
  setFilterControlValue("#task-pending-filter-query", path);
  updateTaskTreePanels(detail || currentSelectedTaskDetail());
  showFlash("已按当前任务定位待补传树");
}

function blockedActionFilterPreset(action) {
  switch (action) {
    case "refresh_auth_profile":
      return { retryClass: "auth_expired", retryState: "blocked" };
    case "restore_local_source_file":
      return { retryClass: "local_file_missing", retryState: "blocked" };
    case "manual_intervention_required":
      return { retryClass: "provider_session_missing", retryState: "blocked" };
    case "wait_for_retry_window":
      return { retryClass: "", retryState: "blocked" };
    case "wait_for_cooldown":
      return { retryClass: "rate_limited", retryState: "blocked" };
    case "manual_confirmation_required":
      return { retryClass: "pending_manual", retryState: "blocked" };
    case "review_and_reset_retry_strategy":
      return { retryClass: "", retryState: "exhausted" };
    default:
      return { retryClass: "", retryState: "" };
  }
}

async function openTaskByID(taskID) {
  if (!taskID) {
    showFlash("当前 blocked 摘要没有可用样本任务", true);
    return;
  }
  activateTab("tasks");
  if (!state.tasks.some((item) => item.task.id === taskID)) {
    await loadTasks();
  }
  if (!state.tasks.some((item) => item.task.id === taskID)) {
    showFlash("未找到对应样本任务，可能已被清理", true);
    return;
  }
  state.selectedTaskId = taskID;
  renderTasks();
  renderSelectedTask();
  showFlash("已打开 blocked 摘要对应的样本任务");
}

function focusBlockedActionSummary(action) {
  const preset = blockedActionFilterPreset(action);
  activateTab("status");
  state.treeFilters.statusRetry.retryClass = preset.retryClass;
  state.treeFilters.statusRetry.retryState = preset.retryState;
  setFilterControlValue("#status-retry-filter-class", preset.retryClass);
  setFilterControlValue("#status-retry-filter-state", preset.retryState);
  state.autoRecoverFilters.blockedAction = action || "";
  setFilterControlValue("#auto-recover-blocked-action", action || "");
  updateStatusRetryQueue(recentRuntimePayload());
  $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
    filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
    state.evidence?.autoRecoverPool || [],
  );
  $("#auto-recover-summary").innerHTML = renderAutoRecoverSummary(state.evidence?.autoRecoverPool || []);
  wireAutoRecoverSummary();
  showFlash("已按 blocked action 收敛最近重试队列");
}

async function runBlockedActionRecovery(action) {
  activateTab("status");
  state.autoRecoverFilters.blockedAction = action || "";
  setFilterControlValue("#auto-recover-blocked-action", action || "");
  await triggerAutoRecover();
}

function wireRetryQueueActions(scope) {
  const wrap = scope === "task" ? $("#task-retry-queue") : $("#status-retry-queue");
  if (!wrap) {
    return;
  }
  wrap.querySelectorAll("[data-retry-focus-pending]").forEach((button) => {
    button.addEventListener("click", () => {
      focusPendingTreeFromRetry(scope, button.dataset.retryFocusPending);
    });
  });
  wrap.querySelectorAll("[data-retry-focus-class]").forEach((button) => {
    button.addEventListener("click", () => {
      focusRetryClass(scope, button.dataset.retryFocusClass, button.dataset.retryFocusState);
    });
  });
}

function wireTreeGroupToggles(scope, panel) {
  const wrap =
    scope === "task"
      ? panel === "directory"
        ? $("#task-directory-states")
        : $("#task-pending-tree")
      : panel === "directory"
        ? $("#status-directory-states")
        : $("#status-pending-tree");
  if (!wrap) {
    return;
  }
  wrap.querySelectorAll("[data-tree-group-toggle]").forEach((button) => {
    button.addEventListener("click", () => {
      const path = button.dataset.treeGroupPath;
      const groupScope = button.dataset.treeGroupScope || scope;
      const groupPanel = button.dataset.treeGroupPanel || panel;
      const key = treeGroupCollapseKey(groupScope, groupPanel, path);
      const next = !state.treeGroupsCollapsed[key];
      setTreeGroupCollapsed(groupScope, groupPanel, path, next);
      if (scope === "task") {
        updateTaskTreePanels(currentSelectedTaskDetail());
      } else {
        updateStatusTreePanels(recentRuntimePayload());
      }
    });
  });
  wrap.querySelectorAll("[data-tree-focus-path]").forEach((button) => {
    button.addEventListener("click", () => {
      focusTreePanelByPath(
        button.dataset.treeFocusScope || scope,
        button.dataset.treeFocusPanel || panel,
        button.dataset.treeFocusPath || "",
      );
    });
  });
  wrap.querySelectorAll("[data-tree-sync-path]").forEach((button) => {
    button.addEventListener("click", () => {
      syncTreePanelPath(
        button.dataset.treeSyncScope || scope,
        button.dataset.treeSyncPanel || panel,
        button.dataset.treeSyncPath || "",
      );
    });
  });
  wrap.querySelectorAll("[data-tree-prefill-path]").forEach((button) => {
    button.addEventListener("click", () => {
      prefillWizardFromTaskPath(currentSelectedTaskDetail(), button.dataset.treePrefillPath || "");
    });
  });
  wrap.querySelectorAll("[data-tree-retry-path]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await retryTaskPath(button.dataset.treeRetryScope || scope, button.dataset.treeRetryPath || "");
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-tree-auto-recover-path]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await autoRecoverTaskPath(
          scope,
          button.dataset.treeAutoRecoverPath || "",
          button.dataset.treeAutoRecoverPanel || "directory",
        );
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-tree-copy-path]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await copyTreeNodePaths(
          button.dataset.treeCopyScope || scope,
          button.dataset.treeCopyPanel || panel,
          button.dataset.treeCopyPath || "",
        );
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-tree-parent-path]").forEach((button) => {
    button.addEventListener("click", () => {
      focusTreeParentPath(
        button.dataset.treeParentScope || scope,
        button.dataset.treeParentPanel || panel,
        button.dataset.treeParentPath || "",
      );
    });
  });
}

function wireBlockedActionsSummary() {
  const wrap = $("#blocked-actions-summary");
  if (!wrap) {
    return;
  }
  wrap.querySelectorAll("[data-blocked-open-task]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await openTaskByID(button.dataset.blockedOpenTask);
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-blocked-focus-action]").forEach((button) => {
    button.addEventListener("click", () => {
      focusBlockedActionSummary(button.dataset.blockedFocusAction);
    });
  });
  wrap.querySelectorAll("[data-blocked-run-action]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await runBlockedActionRecovery(button.dataset.blockedRunAction);
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
}

function recentRuntimePayload() {
  const recentProbe = state.evidence?.recentProbes?.[0];
  if (recentProbe && recentProbe.payload && typeof recentProbe.payload === "object") {
    return recentProbe.payload;
  }
  const statusWithRuntime = (state.statuses || []).find((item) => item.snapshotSummary?.runtime);
  return statusWithRuntime?.snapshotSummary || null;
}

function parseJSONInput(raw, fallback) {
  const text = raw.trim();
  if (!text) {
    return fallback;
  }
  return JSON.parse(text);
}

function optionalNumberValue(selector) {
  const value = $(selector).value.trim();
  if (value === "") {
    return null;
  }
  return Number(value);
}

function collectRiskProfileFromForm(prefix) {
  const override = {};
  const numberFields = [
    ["request-interval", "requestIntervalMs"],
    ["directory-interval", "directoryIntervalMs"],
    ["page-size", "pageSize"],
    ["cooldown-seconds", "cooldownSeconds"],
    ["retry-limit", "retryLimit"],
    ["max-concurrent", "maxConcurrent"],
    ["auto-retry-start-hour", "autoRetryStartHour"],
    ["auto-retry-end-hour", "autoRetryEndHour"],
  ];
  numberFields.forEach(([suffix, key]) => {
    const value = optionalNumberValue(`#${prefix}-${suffix}`);
    if (value !== null && Number.isFinite(value)) {
      override[key] = value;
    }
  });
  const keywords = $(`#${prefix}-keywords`).value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  if (keywords.length > 0) {
    override.riskKeywords = keywords;
  }
  return Object.keys(override).length > 0 ? override : null;
}

function hydrateRiskProfileForm(prefix, value) {
  const override = value && typeof value === "object" ? value : {};
  $(`#${prefix}-request-interval`).value = override.requestIntervalMs ?? "";
  $(`#${prefix}-directory-interval`).value = override.directoryIntervalMs ?? "";
  $(`#${prefix}-page-size`).value = override.pageSize ?? "";
  $(`#${prefix}-cooldown-seconds`).value = override.cooldownSeconds ?? "";
  $(`#${prefix}-retry-limit`).value = override.retryLimit ?? "";
  $(`#${prefix}-max-concurrent`).value = override.maxConcurrent ?? "";
  $(`#${prefix}-auto-retry-start-hour`).value = override.autoRetryStartHour ?? "";
  $(`#${prefix}-auto-retry-end-hour`).value = override.autoRetryEndHour ?? "";
  $(`#${prefix}-keywords`).value = Array.isArray(override.riskKeywords) ? override.riskKeywords.join(",") : "";
}

function collectRiskOverrideFromForm() {
  return collectRiskProfileFromForm("risk");
}

function hydrateRiskOverrideForm(value) {
  hydrateRiskProfileForm("risk", value);
}

function syncRiskOverrideJSON() {
  const override = collectRiskOverrideFromForm();
  $("#plan-risk-override").value = override ? JSON.stringify(override, null, 2) : "";
  return override;
}

function parseProfileRiskDefaultsFromExtra(extra) {
  if (!extra || typeof extra !== "object") {
    return null;
  }
  const raw = String(extra.riskDefaults || "").trim();
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" ? parsed : null;
  } catch (error) {
    return null;
  }
}

function parseProfileRiskDefaultsSourceFromExtra(extra) {
  if (!extra || typeof extra !== "object") {
    return "";
  }
  const display = String(extra.riskDefaultsSourceDisplay || "").trim();
  if (display) {
    return display;
  }
  const raw = String(extra.riskDefaultsSource || "").trim();
  if (!raw) {
    return "";
  }
  if (raw.startsWith("smoke_matrix:")) {
    const parts = raw.split(":");
    if (parts.length >= 3) {
      return `Smoke Matrix ${parts[1]} (${parts[2]})`;
    }
  }
  return raw;
}

function renderRiskDefaultsSourceBadge(source) {
  const normalized = String(source || "").trim();
  if (!normalized) {
    return "provider_default_only";
  }
  const match = normalized.match(/^Smoke Matrix\s+(.+?)\s+\((accepted|in_progress|pending)\)$/i);
  if (!match) {
    return normalized;
  }
  return `smoke ${String(match[1] || "").trim()} ${String(match[2] || "").trim().toLowerCase()}`;
}
function renderProfileRiskDefaultSourceAdvice(source) {
  const normalized = String(source || "").trim();
  if (!normalized) {
    return "当前未附带真实样本来源说明，将沿用 provider 默认模板。";
  }
  const match = normalized.match(/^Smoke Matrix\s+(.+?)\s+\((accepted|in_progress|pending)\)$/i);
  if (!match) {
    return `当前账号默认模板来源：${normalized}`;
  }
  const protocolGroup = String(match[1] || "").trim() || "unknown";
  const status = String(match[2] || "").trim().toLowerCase();
  if (status === "accepted") {
    return `真实样本矩阵显示 ${protocolGroup} 已验收，建议优先沿用这套账号默认模板。`;
  }
  if (status === "in_progress") {
    return `真实样本矩阵显示 ${protocolGroup} 仍在补样中，建议沿用当前模板并继续补齐上传成功或异常样本。`;
  }
  return `真实样本矩阵显示 ${protocolGroup} 仍待补齐，建议先按保守模板试跑，再回填真实样本继续校准。`;
}

function mergeProfileRiskDefaultsIntoExtra(extra, riskDefaults) {
  const merged = extra && typeof extra === "object" ? { ...extra } : {};
  if (riskDefaults && typeof riskDefaults === "object" && Object.keys(riskDefaults).length > 0) {
    merged.riskDefaults = JSON.stringify(riskDefaults);
  } else {
    delete merged.riskDefaults;
  }
  return merged;
}

function resetProfileForm() {
  const form = $("#profile-form");
  form.reset();
  $("#profile-id").value = "";
  $("#profile-extra").value = "";
  hydrateRiskProfileForm("profile-risk", null);
  $("#profile-submit").textContent = "创建授权档案";
  syncAuthModes();
}

function setProfileFormEditing(profile) {
  if (!profile || typeof profile !== "object") {
    resetProfileForm();
    return;
  }
  $("#profile-id").value = profile.id || "";
  setSelectValueIfPresent("#profile-provider", profile.providerKey || "");
  syncAuthModes();
  setSelectValueIfPresent("#profile-auth-mode", profile.authMode || "");
  $("#profile-display-name").value = profile.displayName || "";
  $("#profile-token").value = profile.token || "";
  $("#profile-cookie").value = profile.cookie || "";
  const extra = profile.extra && typeof profile.extra === "object" ? profile.extra : {};
  $("#profile-extra").value = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : "";
  hydrateRiskProfileForm("profile-risk", parseProfileRiskDefaultsFromExtra(extra));
  $("#profile-submit").textContent = "更新授权档案";
}

function showFlash(message, isError = false) {
  const flash = $("#flash");
  flash.textContent = message;
  flash.classList.remove("hidden");
  flash.style.background = isError ? "rgba(127, 29, 29, 0.96)" : "rgba(31, 27, 23, 0.94)";
  clearTimeout(showFlash.timer);
  showFlash.timer = setTimeout(() => flash.classList.add("hidden"), 2600);
}

async function api(path, options = {}) {
  const config = {
    method: options.method || "GET",
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
  };
  if (options.body !== undefined) {
    config.body = JSON.stringify(options.body);
  }
  const response = await fetch(path, config);
  const payload = await response.json();
  if (!response.ok || !payload.ok) {
    throw new Error(payload?.error?.message || `Request failed: ${response.status}`);
  }
  return payload.data;
}

function syncSessionState() {
  $("#session-state").textContent = state.authenticated ? "已登录" : "未登录";
}

function setupTabs() {
  document.querySelectorAll(".tab").forEach((button) => {
    button.addEventListener("click", () => {
      activateTab(button.dataset.view);
    });
  });
}

function activateTab(view) {
  document.querySelectorAll(".tab").forEach((item) => item.classList.toggle("active", item.dataset.view === view));
  document.querySelectorAll(".panel").forEach((item) => item.classList.toggle("active", item.dataset.panel === view));
}

function setSelectValueIfPresent(selector, value) {
  const element = $(selector);
  if (!element || value === undefined || value === null || value === "") {
    return;
  }
  if ([...element.options].some((option) => option.value === value)) {
    element.value = value;
    element.dispatchEvent(new Event("change"));
  }
}

function setInputValueIfPresent(selector, value) {
  const element = $(selector);
  if (!element || value === undefined || value === null) {
    return;
  }
  element.value = String(value);
  element.dispatchEvent(new Event("change"));
}

async function copyTextToClipboard(text) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const probe = document.createElement("textarea");
  probe.value = text;
  probe.setAttribute("readonly", "readonly");
  probe.style.position = "fixed";
  probe.style.opacity = "0";
  document.body.appendChild(probe);
  probe.select();
  document.execCommand("copy");
  probe.remove();
}

function buildCreatePayloadFromTask(detail) {
  if (!detail || !detail.task || !detail.plan) {
    return null;
  }
  return {
    sourceProvider: detail.task.sourceProvider,
    sourceProfileId: detail.sourceProfileId || "",
    targetProvider: detail.task.targetProvider,
    targetProfileId: detail.targetProfileId || "",
    thresholdMB: Number(detail.plan.thresholdMB || 0),
    riskMode: detail.plan.metadata?.riskProfile?.mode || "balanced",
    riskOverride: detail.plan.metadata?.riskOverride || null,
    executionMode: detail.plan.metadata?.executionMode || "leaf_first_lazy",
    sourceDeletePolicy: detail.plan.metadata?.sourceDeletePolicy || "record_only",
    conflictPolicy: detail.conflictPolicy || "auto_rename_new",
    selectedRoots: detail.plan.metadata?.selectedRoots || ["/"],
    entries: detail.sourceEntries || [],
  };
}

function normalizeComparePath(path) {
  const value = String(path || "").trim();
  if (!value) {
    return "";
  }
  if (value === "/") {
    return "/";
  }
  return value.startsWith("/") ? value.replace(/\/+$/, "") || "/" : `/${value}`.replace(/\/+$/, "") || "/";
}

function pathMatchesSubtree(candidatePath, rootPath) {
  const candidate = normalizeComparePath(candidatePath);
  const root = normalizeComparePath(rootPath);
  if (!candidate || !root) {
    return false;
  }
  if (root === "/") {
    return true;
  }
  return candidate === root || candidate.startsWith(`${root}/`);
}

function buildCreatePayloadFromPayloadPath(payload, path, options = {}) {
  const normalizedPath = normalizeComparePath(path);
  if (!payload || !normalizedPath) {
    return payload;
  }
  const narrowedEntries = (payload.entries || []).filter((entry) => pathMatchesSubtree(entry?.path, normalizedPath));
  const narrowedRoots = (payload.selectedRoots || []).filter((root) => pathMatchesSubtree(normalizedPath, root) || pathMatchesSubtree(root, normalizedPath));
  payload.selectedRoots = options.exactRoots
    ? [normalizedPath]
    : narrowedRoots.length
      ? narrowedRoots.filter((root) => pathMatchesSubtree(normalizedPath, root))
      : [normalizedPath];
  payload.entries = narrowedEntries.length ? narrowedEntries : payload.entries;
  return payload;
}

function buildCreatePayloadFromPayloadPaths(payload, paths, options = {}) {
  const normalizedPaths = Array.from(new Set((Array.isArray(paths) ? paths : []).map((path) => normalizeComparePath(path)).filter(Boolean)));
  if (!payload || !normalizedPaths.length) {
    return payload;
  }
  const narrowedEntries = (payload.entries || []).filter((entry) =>
    normalizedPaths.some((path) => pathMatchesSubtree(entry?.path, path)),
  );
  const narrowedRoots = (payload.selectedRoots || []).filter((root) =>
    normalizedPaths.some((path) => pathMatchesSubtree(path, root) || pathMatchesSubtree(root, path)),
  );
  payload.selectedRoots = options.exactRoots ? normalizedPaths : narrowedRoots.length ? narrowedRoots : normalizedPaths;
  payload.entries = narrowedEntries.length ? narrowedEntries : payload.entries;
  return payload;
}

function buildCreatePayloadFromTaskPath(detail, path, options = {}) {
  return buildCreatePayloadFromPayloadPath(buildCreatePayloadFromTask(detail), path, options);
}

function buildCreatePayloadFromTaskPaths(detail, paths, options = {}) {
  return buildCreatePayloadFromPayloadPaths(buildCreatePayloadFromTask(detail), paths, options);
}

function prefillWizardFromTask(detail) {
  if (!detail || !detail.task || !detail.plan) {
    return;
  }
  setSelectValueIfPresent("#plan-source-provider", detail.task.sourceProvider);
  syncSourceProfiles();
  setSelectValueIfPresent("#plan-source-profile", detail.sourceProfileId || "");
  setSelectValueIfPresent("#plan-target-provider", detail.task.targetProvider);
  syncTargetProfiles();
  setSelectValueIfPresent("#plan-target-profile", detail.targetProfileId || "");
  setSelectValueIfPresent("#plan-execution-mode", detail.plan.metadata?.executionMode || "leaf_first_lazy");
  setSelectValueIfPresent("#plan-source-delete-policy", detail.plan.metadata?.sourceDeletePolicy || "record_only");
  setSelectValueIfPresent("#plan-risk-mode", detail.plan.metadata?.riskProfile?.mode || "balanced");
  setSelectValueIfPresent("#plan-conflict-policy", detail.conflictPolicy || "auto_rename_new");
  setInputValueIfPresent("#plan-threshold", detail.plan.thresholdMB || 0);
  hydrateRiskOverrideForm(detail.plan.metadata?.riskOverride || null);
  $("#plan-risk-override").value = detail.plan.metadata?.riskOverride
    ? JSON.stringify(detail.plan.metadata.riskOverride, null, 2)
    : "";
  $("#plan-selected-roots").value = JSON.stringify(detail.plan.metadata?.selectedRoots || ["/"], null, 2);
  $("#plan-entries").value = JSON.stringify(detail.sourceEntries || [], null, 2);
  syncExecutionModeHint();
}

function focusProfile(profileId) {
  state.focusedProfileId = profileId || null;
  renderProfiles();
  if (!profileId) {
    return;
  }
  requestAnimationFrame(() => {
    const row = document.querySelector(`[data-profile-row="${profileId}"]`);
    row?.scrollIntoView({ block: "center", behavior: "smooth" });
  });
}

function wireTaskQuickActions(detail) {
  const prefillButton = $("#task-prefill-wizard");
  const copyButton = $("#task-copy-payload");
  const payload = buildCreatePayloadFromTask(detail);
  const disabled = !payload;

  if (prefillButton) {
    prefillButton.disabled = disabled;
    prefillButton.onclick = disabled
      ? () => showFlash("请先选择任务", true)
      : () => {
          activateTab("wizard");
          prefillWizardFromTask(detail);
          showFlash("已按当前任务重建向导参数");
        };
  }

  if (copyButton) {
    copyButton.disabled = disabled;
    copyButton.onclick = disabled
      ? () => showFlash("请先选择任务", true)
      : async () => {
          try {
            await copyTextToClipboard(formatJSON(payload));
            showFlash("任务创建参数已复制到剪贴板");
          } catch (error) {
            showFlash(`复制失败：${error.message}`, true);
          }
        };
  }
}

function syncTaskActionButtons() {
  const hasTask = Boolean(state.selectedTaskId);
  const pending = Boolean(state.taskActionPending);
  ["#task-run", "#task-pause", "#task-resume", "#task-retry"].forEach((selector) => {
    const button = $(selector);
    if (!button) {
      return;
    }
    button.disabled = pending || !hasTask;
  });
}

function renderTaskResolutionGuide(detail) {
  const metadata = detail?.plan?.metadata || {};
  const runtime = detail?.runtime || {};
  const retrySummary = metadata.retrySummary || {};
  const providerKey = detail?.task?.targetProvider || "";
  const profileId = detail?.targetProfileId || "";
  const nextRetryAt = runtime.nextRetryAt || retrySummary.nextRetryAt || "";
  const action = runtime.blockedAction || retrySummary.blockedAction || "";
  const advice = runtime.blockedAdvice || retrySummary.blockedAdvice || "";
  if (!action) {
    if (retrySummary.autoRecoverMode === "upload_checkpoint_auto_resume" || retrySummary.autoRecoverMode === "retry_queue_auto_retry" || Number(retrySummary.uploadCheckpointEligible || 0) > 0 || Boolean(retrySummary.autoRecoverEligible)) {
      return `
        <div class="provider-card">
          <h3>${retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? "等待后台自动补传接管" : "等待上传会话自动续跑"}</h3>
          <div class="meta-row">
            <span class="pill">${escapeHTML(stringifyValue(retrySummary.autoRecoverMode, "upload_checkpoint_auto_resume"))}</span>
            ${nextRetryAt ? `<span class="pill">${escapeHTML(nextRetryAt)}</span>` : ""}
          </div>
          <ol class="checklist">
            <li>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? "当前队列满足后台自动补传条件，系统会在后续 tick 中自动尝试继续执行。" : "当前失败队列携带可恢复的 upload checkpoint，单机 worker 会优先尝试续跑上传会话。")}</li>
            <li>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? "先到状态矩阵查看后台补传候选池，确认该任务是否已经进入 retry_queue_auto_retry lane。" : "先到状态矩阵查看后台补传候选池，确认该任务是否已经进入 upload checkpoint 自动续跑 lane。")}</li>
            <li>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? "如果长时间未自动推进，再检查 retrySummary、provider 返回状态和风险窗口是否把它留在等待态。" : "如果长时间未自动推进，再检查 providerData / uploadId / nextPartNumber 等恢复线索是否完整。")}</li>
          </ol>
          <div class="muted">${escapeHTML(retrySummary.autoRecoverAdvice || "当前失败队列都带可恢复的 upload checkpoint，单机 worker 会优先尝试续跑上传会话。")}</div>
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-task-guide-view="status"
              data-task-guide-intent="focus_status_auto_recover_mode"
              data-task-guide-mode="${escapeHTML(stringifyValue(retrySummary.autoRecoverMode, "upload_checkpoint_auto_resume"))}"
              data-task-guide-provider="${escapeHTML(providerKey)}"
              data-task-guide-profile="${escapeHTML(profileId)}"
            >${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? "只看自动补传候选" : "只看自动续跑候选")}</button>
            <button
              type="button"
              class="ghost"
              data-task-guide-view="status"
              data-task-guide-intent="focus_status_open"
              data-task-guide-provider="${escapeHTML(providerKey)}"
              data-task-guide-profile="${escapeHTML(profileId)}"
            >打开状态矩阵</button>
          </div>
        </div>
      `;
    }
    return `
      <div class="insight-card">
        <strong>下一步处理</strong>
        <span>当前任务没有 blocked 人工处理动作，可直接继续运行或观察状态矩阵。</span>
      </div>
    `;
  }

  const stepsByAction = {
    refresh_auth_profile: {
      title: "刷新授权档案",
      steps: [
        "切到“Provider / 授权”面板，定位当前目标端授权档案。",
        "更新 token/cookie 后先执行 Validate，确认授权恢复正常。",
        "回到任务详情页，再执行 Resume 或 Retry。",
      ],
      buttons: [
        { label: "打开授权面板", view: "providers", providerKey, profileId, intent: "focus_profile" },
        { label: "只看授权失效队列", view: "tasks", intent: "focus_task_retry" },
        { label: "按当前阻塞打开状态矩阵", view: "status", intent: "focus_status_blocked" },
      ],
    },
    restore_local_source_file: {
      title: "补回本地回退文件",
      steps: [
        "先补回源文件或校正本地 fallback 路径，确保 localPath 对应文件真实存在。",
        "如果路径配置有误，建议回到任务向导核对 entries / selectedRoots。",
        "补齐后返回任务详情页重新 Retry。",
      ],
      buttons: [
        { label: "定位待补传树", view: "tasks", intent: "focus_task_pending" },
        { label: "只看本地缺失队列", view: "tasks", intent: "focus_task_retry" },
        { label: "打开任务向导", view: "wizard", providerKey, intent: "prefill_wizard" },
        { label: "按当前阻塞打开状态矩阵", view: "status", intent: "focus_status_blocked" },
      ],
    },
    manual_intervention_required: {
      title: "修复 provider 会话缺口",
      steps: [
        "当前 retryClass 是 provider_session_missing，说明 provider 返回体缺少 uploadid / upload session 这类关键会话字段。",
        "先核对 provider 返回体、上传会话构建逻辑和目标端授权档案，确认是否需要重新生成会话或刷新授权。",
        "修复后回到状态矩阵，确认该类 blocked 项已经收敛，再执行 Retry。",
      ],
      buttons: [
        { label: "只看会话缺口队列", view: "tasks", intent: "focus_task_retry" },
        { label: "打开授权面板", view: "providers", providerKey, profileId, intent: "focus_profile" },
        { label: "按当前阻塞打开状态矩阵", view: "status", intent: "focus_status_blocked" },
      ],
    },
    wait_for_cooldown: {
      title: "等待冷却窗口结束",
      steps: [
        nextRetryAt ? `当前最早自动补传时间是 ${nextRetryAt}。` : "当前处于风控冷却窗口。",
        "冷却期间无需手动重试，系统会在窗口结束后自动尝试补传。",
        "如果想确认整体阻塞分布，可切到状态矩阵查看 blocked 聚合看板。",
      ],
      buttons: [
        { label: "只看冷却队列", view: "tasks", intent: "focus_task_retry" },
        { label: "按当前阻塞打开状态矩阵", view: "status", intent: "focus_status_blocked" },
      ],
    },
    wait_for_retry_window: {
      title: "等待自动补传时间窗",
      steps: [
        nextRetryAt ? `当前下一次允许自动补传的时间是 ${nextRetryAt}。` : "当前不在允许的自动补传时间窗内。",
        "这类任务仍会留在自动补传候选池里，但在时间窗开始前不会被 worker 实际执行。",
        "如果需要排查影响范围，可切到状态矩阵按 blocked action 或 lane 直接聚焦。",
      ],
      buttons: [
        { label: "只看时间窗等待态", view: "status" },
        { label: "只看当前任务重试队列", view: "tasks", intent: "focus_task_retry" },
      ],
    },
    manual_confirmation_required: {
      title: "等待人工确认",
      steps: [
        "当前任务存在 pending_manual 项，说明 provider 仍需要人工确认或后续 fallback 运行时能力。",
        "先在状态矩阵和待补传树里确认影响范围，再决定是否拆分任务或等待后续链路补齐。",
        "确认后再回到任务详情执行 Retry。",
      ],
      buttons: [
        { label: "定位待补传树", view: "tasks", intent: "focus_task_pending" },
        { label: "只看待确认队列", view: "tasks", intent: "focus_task_retry" },
        { label: "按当前阻塞打开状态矩阵", view: "status", intent: "focus_status_blocked" },
        { label: "留在任务详情", view: "tasks" },
      ],
    },
    review_and_reset_retry_strategy: {
      title: "调整重试策略",
      steps: [
        "当前任务已经达到 retryLimit，继续原样 Retry 不会再推进。",
        "建议回到任务向导调整 riskOverride / retryLimit / 执行策略，必要时拆成更小任务。",
        "创建新任务后，用状态矩阵对比新的 blocked 分布是否收敛。",
      ],
      buttons: [
        { label: "只看 exhausted 队列", view: "tasks", intent: "focus_task_retry" },
        { label: "打开任务向导", view: "wizard", providerKey, intent: "prefill_wizard" },
        { label: "按当前阻塞打开状态矩阵", view: "status", intent: "focus_status_blocked" },
      ],
    },
  };

  const config = stepsByAction[action] || {
    title: "人工处理建议",
    steps: [advice || "请根据 blocked 原因检查授权、源文件和 provider 返回状态。"],
    buttons: [{ label: "打开状态矩阵", view: "status" }],
  };

  return `
    <div class="provider-card">
      <h3>${escapeHTML(config.title)}</h3>
      <div class="meta-row">
        <span class="pill">${escapeHTML(action)}</span>
        ${nextRetryAt ? `<span class="pill">${escapeHTML(nextRetryAt)}</span>` : ""}
      </div>
      <ol class="checklist">
        ${config.steps.map((step) => `<li>${escapeHTML(step)}</li>`).join("")}
      </ol>
      ${advice && advice !== config.steps[0] ? `<div class="muted">${escapeHTML(advice)}</div>` : ""}
      <div class="actions compact">
        ${config.buttons
          .map(
            (button) => `
              <button
                type="button"
                class="ghost"
                data-task-guide-view="${escapeHTML(button.view)}"
                data-task-guide-intent="${escapeHTML(button.intent || "")}"
                data-task-guide-provider="${escapeHTML(button.providerKey || "")}"
                data-task-guide-profile="${escapeHTML(button.profileId || "")}"
              >${escapeHTML(button.label)}</button>
            `,
          )
          .join("")}
      </div>
    </div>
  `;
}

function wireTaskResolutionGuide(detail) {
  document.querySelectorAll("[data-task-guide-view]").forEach((button) => {
    button.addEventListener("click", () => {
      const view = button.dataset.taskGuideView;
      const intent = button.dataset.taskGuideIntent;
      activateTab(view);
      if (view === "providers") {
        setSelectValueIfPresent("#profile-provider", button.dataset.taskGuideProvider);
        if (intent === "focus_profile") {
          focusProfile(button.dataset.taskGuideProfile);
          showFlash("已定位到当前授权档案");
        }
      }
      if (view === "tasks") {
        state.selectedTaskId = detail?.task?.id || state.selectedTaskId;
        renderTasks();
        renderSelectedTask();
        if (intent === "focus_task_retry") {
          focusTaskRetryByBlockedAction(detail?.runtime?.blockedAction || detail?.plan?.metadata?.retrySummary?.blockedAction || "");
        }
        if (intent === "focus_task_pending") {
          focusTaskPendingByDetail(detail);
        }
      }
      if (view === "status") {
        if (intent === "focus_status_blocked") {
          focusBlockedActionSummary(detail?.runtime?.blockedAction || detail?.plan?.metadata?.retrySummary?.blockedAction || "");
        }
        if (intent === "focus_status_auto_recover_mode") {
          activateTab("status");
          state.autoRecoverFilters.mode = button.dataset.taskGuideMode || "";
          setFilterControlValue("#auto-recover-mode", button.dataset.taskGuideMode || "");
          state.autoRecoverFilters.blockedAction = "";
          setFilterControlValue("#auto-recover-blocked-action", "");
          $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
            filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
            state.evidence?.autoRecoverPool || [],
          );
          $("#auto-recover-summary").innerHTML = renderAutoRecoverSummary(state.evidence?.autoRecoverPool || []);
          wireAutoRecoverSummary();
          showFlash(`已按 ${button.dataset.taskGuideMode || "-"} 收敛后台补传候选`);
        }
        if (intent === "focus_status_open") {
          activateTab("status");
          showFlash("已打开状态矩阵");
        }
      }
      if (view === "wizard") {
        if (intent === "prefill_wizard") {
          prefillWizardFromTask(detail);
          showFlash("已按当前任务预填任务向导参数");
        } else {
          setSelectValueIfPresent("#plan-target-provider", detail?.task?.targetProvider || button.dataset.taskGuideProvider);
          setSelectValueIfPresent("#plan-target-profile", detail?.targetProfileId || button.dataset.taskGuideProfile);
          setSelectValueIfPresent("#plan-source-provider", detail?.task?.sourceProvider);
        }
      }
    });
  });
}

function renderProviders() {
  const providerSelect = $("#profile-provider");
  const sourceSelect = $("#plan-source-provider");
  const targetSelect = $("#plan-target-provider");
  const providerCards = $("#providers-grid");
  const selectedProfileProvider = providerSelect.value || state.providers[0]?.meta?.key || "";
  const selectedSourceProvider = sourceSelect.value || state.providers[0]?.meta?.key || "";
  const selectedTargetProvider = targetSelect.value || state.providers[0]?.meta?.key || "";

  const options = state.providers
    .map((entry) => `<option value="${entry.meta.key}">${entry.meta.displayName}</option>`)
    .join("");
  providerSelect.innerHTML = options;
  sourceSelect.innerHTML = options;
  targetSelect.innerHTML = options;
  if ([...providerSelect.options].some((option) => option.value === selectedProfileProvider)) {
    providerSelect.value = selectedProfileProvider;
  }
  if ([...sourceSelect.options].some((option) => option.value === selectedSourceProvider)) {
    sourceSelect.value = selectedSourceProvider;
  }
  if ([...targetSelect.options].some((option) => option.value === selectedTargetProvider)) {
    targetSelect.value = selectedTargetProvider;
  }

  providerCards.innerHTML = state.providers
    .map(
      (entry) => {
        const riskHints = Array.isArray(entry.meta.riskHints) ? entry.meta.riskHints.filter(Boolean) : [];
        const riskTraits = Array.isArray(entry.meta.riskTraits) ? entry.meta.riskTraits.filter(Boolean) : [];
        const defaultRiskTemplate =
          entry.meta.defaultRiskTemplate && typeof entry.meta.defaultRiskTemplate === "object"
            ? entry.meta.defaultRiskTemplate
            : null;
        const profileSource = parseProfileRiskDefaultsSourceFromExtra(entry.profile?.extra || {});
        const fallbackModes = Array.isArray(entry.meta.fallbackModes) ? entry.meta.fallbackModes.filter(Boolean) : [];
        const conflictPolicies = Array.isArray(entry.meta.conflictPolicies) ? entry.meta.conflictPolicies.filter(Boolean) : [];
        const active = entry.meta.key === state.selectedProviderCapabilityKey;
        return `
        <article class="provider-card${active ? " active" : ""}">
          <h3>${entry.meta.displayName}</h3>
          <div class="meta-row">
            <span class="pill">${entry.meta.key}</span>
            <span class="pill">${entry.meta.protocolGroup}</span>
            <span class="pill">${entry.meta.status}</span>
          </div>
          <div class="meta-row">
            ${entry.meta.authModes.map((mode) => `<span class="pill">${mode}</span>`).join("")}
          </div>
          <div class="muted">fallback: ${escapeHTML(fallbackModes.join(", ") || "-")}</div>
          <div class="muted">conflict: ${escapeHTML(conflictPolicies.join(", ") || "-")}</div>
          <div class="muted">risk traits: ${escapeHTML(riskTraits.join(", ") || "-")}</div>
          <div class="muted">risk hints: ${escapeHTML(riskHints.join(" / ") || "-")}</div>
          <div class="muted">default risk: ${escapeHTML(renderRiskProfileCompact(defaultRiskTemplate?.calibrated))}</div>
          <div class="muted">recommended risk: ${escapeHTML(stringifyValue(defaultRiskTemplate?.recommendedMode, "-"))}</div>
          <div class="muted">risk calibration: ${escapeHTML((defaultRiskTemplate?.calibrationReasons || []).join(" / ") || "-")}</div>
          <div class="muted">calibration readiness: ${escapeHTML(stringifyValue(defaultRiskTemplate?.calibrationReadiness, "-"))}</div>
          <div class="muted">priority calibration: ${escapeHTML(stringifyValue(defaultRiskTemplate?.calibrationPriorityAction, "-"))}</div>
          <div class="muted">recover budget: ${escapeHTML(renderRecoverBudgetCompact(defaultRiskTemplate?.recoverBudget))}</div>
          <div class="muted">profile risk source: ${escapeHTML(renderRiskDefaultsSourceBadge(profileSource))}</div>
          <div class="muted">profile risk advice: ${escapeHTML(renderProfileRiskDefaultSourceAdvice(profileSource))}</div>
          <div class="actions compact">
            <button type="button" class="ghost" data-provider-detail-open="${escapeHTML(entry.meta.key)}">查看能力详情</button>
          </div>
        </article>
      `;
      },
    )
    .join("");

  if (!state.selectedProviderCapabilityKey || !findProviderEntry(state.selectedProviderCapabilityKey)) {
    state.selectedProviderCapabilityKey = state.providers[0]?.meta?.key || "";
  }

  syncAuthModes();
  syncSourceProfiles();
  syncTargetProfiles();
  syncAutoRecoverProviders();
  syncAutoRecoverProtocolGroups();
  syncAutoRecoverProfiles();
  syncAutoRecoverBlockedActions();
  syncExecutionModeHint();
  renderProviderCapabilityDetail();
  syncTargetProviderInsight();
}

function prefillWizardFromTaskPath(detail, path, options = {}) {
  const payload = buildCreatePayloadFromTaskPath(detail, path, options);
  if (!payload) {
    showFlash("请先选择任务", true);
    return;
  }
  prefillWizardFromPayload(payload);
  showFlash(`已按 ${path} 重建向导范围`);
}

function prefillWizardFromPayload(payload) {
  activateTab("wizard");
  setSelectValueIfPresent("#plan-source-provider", payload.sourceProvider);
  syncSourceProfiles();
  setSelectValueIfPresent("#plan-source-profile", payload.sourceProfileId || "");
  setSelectValueIfPresent("#plan-target-provider", payload.targetProvider);
  syncTargetProfiles();
  setSelectValueIfPresent("#plan-target-profile", payload.targetProfileId || "");
  setSelectValueIfPresent("#plan-execution-mode", payload.executionMode || "leaf_first_lazy");
  setSelectValueIfPresent("#plan-source-delete-policy", payload.sourceDeletePolicy || "record_only");
  setSelectValueIfPresent("#plan-risk-mode", payload.riskMode || "balanced");
  setSelectValueIfPresent("#plan-conflict-policy", payload.conflictPolicy || "auto_rename_new");
  setInputValueIfPresent("#plan-threshold", payload.thresholdMB || 0);
  hydrateRiskOverrideForm(payload.riskOverride || null);
  $("#plan-risk-override").value = payload.riskOverride ? JSON.stringify(payload.riskOverride, null, 2) : "";
  $("#plan-selected-roots").value = JSON.stringify(payload.selectedRoots || ["/"], null, 2);
  $("#plan-entries").value = JSON.stringify(payload.entries || [], null, 2);
  syncExecutionModeHint();
}

function prefillWizardFromTaskPaths(detail, paths, label = "当前筛选", options = {}) {
  const payload = buildCreatePayloadFromTaskPaths(detail, paths, options);
  if (!payload) {
    showFlash("请先选择任务", true);
    return;
  }
  prefillWizardFromPayload(payload);
  showFlash(`已按${label}重建向导范围`);
}

function prefillWizardFromPreviewPaths(paths, label = "全部删除记录") {
  let payload;
  try {
    payload = buildPlanPayload();
  } catch (error) {
    showFlash(`当前向导参数无法解析：${error.message}`, true);
    return;
  }
  prefillWizardFromPayload(buildCreatePayloadFromPayloadPaths(payload, paths, { exactRoots: true }));
  showFlash(`已按${label}重建向导范围`);
}

function wireSourceDeletionSummary(scope, selector = null) {
  const wrap = selector ? $(selector) : scope === "task" ? $("#task-runtime") : $("#status-runtime-checkpoints");
  if (!wrap) {
    return;
  }
  wrap.dataset.sourceDeletionScope = scope;
  if (wrap.dataset.sourceDeletionWired === "true") {
    return;
  }
  wrap.dataset.sourceDeletionWired = "true";
  wrap.addEventListener("click", (event) => {
    const pathButton = event.target.closest("[data-source-delete-prefill-path]");
    const pathsButton = event.target.closest("[data-source-delete-prefill-paths]");
    const button = pathButton || pathsButton;
    if (!button || !wrap.contains(button)) {
      return;
    }
    const focusScope = button.dataset.sourceDeletePrefillScope || wrap.dataset.sourceDeletionScope || scope;
    if (pathButton) {
      const path = pathButton.dataset.sourceDeletePrefillPath || "";
      if (!path) {
        showFlash("缺少可重建路径", true);
        return;
      }
      if (focusScope === "preview") {
        prefillWizardFromPreviewPaths([path], "此删除记录");
        return;
      }
      const context = taskContextByScope(focusScope);
      if (!context?.detail) {
        showFlash(focusScope === "task" ? "请先选择任务" : "当前状态样本没有可用任务", true);
        return;
      }
      prefillWizardFromTaskPath(context.detail, path, { exactRoots: true });
      return;
    }

    let paths = [];
    try {
      paths = JSON.parse(pathsButton.dataset.sourceDeletePrefillPaths || "[]");
    } catch {
      paths = [];
    }
    const normalizedPaths = Array.isArray(paths) ? paths.map((path) => normalizeComparePath(path)).filter(Boolean) : [];
    if (!normalizedPaths.length) {
      showFlash("当前没有可重建的删除记录", true);
      return;
    }
    if (focusScope === "preview") {
      prefillWizardFromPreviewPaths(normalizedPaths, pathsButton.dataset.sourceDeletePrefillLabel || "全部删除记录");
      return;
    }
    const context = taskContextByScope(focusScope);
    if (!context?.detail) {
      showFlash(focusScope === "task" ? "请先选择任务" : "当前状态样本没有可用任务", true);
      return;
    }
    prefillWizardFromTaskPaths(context.detail, normalizedPaths, pathsButton.dataset.sourceDeletePrefillLabel || "全部删除记录", { exactRoots: true });
  });
}

function syncAuthModes() {
  const providerKey = $("#profile-provider").value;
  const provider = state.providers.find((item) => item.meta.key === providerKey);
  const authModeSelect = $("#profile-auth-mode");
  if (!provider) {
    authModeSelect.innerHTML = "";
    return;
  }
  authModeSelect.innerHTML = provider.meta.authModes
    .map((mode) => `<option value="${mode}">${mode}</option>`)
    .join("");
}

function renderProfiles() {
  const wrap = $("#profiles-table");
  if (!state.profiles.length) {
    wrap.innerHTML = `<div class="provider-card">暂无授权档案。</div>`;
    syncSourceProfiles();
    syncTargetProfiles();
    syncTargetProfileInsight();
    return;
  }

  wrap.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>显示名称</th>
          <th>Provider</th>
          <th>Auth Mode</th>
          <th>账号默认风控</th>
          <th>Status</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        ${state.profiles
          .map(
            (profile) => {
              const profileRisk = renderRiskProfileCompact(parseProfileRiskDefaultsFromExtra(profile.extra));
              const profileSource = parseProfileRiskDefaultsSourceFromExtra(profile.extra || {});
              const profileAdvice = renderProfileRiskDefaultSourceAdvice(profileSource);
              return `
              <tr class="${profile.id === state.focusedProfileId ? "active" : ""}" data-profile-row="${profile.id}">
                <td>${profile.displayName}</td>
                <td>${profile.providerKey}</td>
                <td>${profile.authMode}</td>
                <td>
                  <div>${escapeHTML(profileRisk)}</div>
                  <div class="muted">账号默认来源: ${escapeHTML(renderRiskDefaultsSourceBadge(profileSource))}</div>
                  <div class="muted">账号默认建议: ${escapeHTML(profileAdvice)}</div>
                </td>
                <td>${profile.status}</td>
                <td>
                  <div class="actions compact">
                    <button type="button" class="ghost" data-profile-edit="${profile.id}">Edit</button>
                    <button type="button" class="ghost" data-profile-validate="${profile.id}">Validate</button>
                    <button type="button" class="ghost" data-profile-delete="${profile.id}">Delete</button>
                  </div>
                </td>
              </tr>
            `;
            },
          )
          .join("")}
      </tbody>
    </table>
  `;

  wrap.querySelectorAll("[data-profile-edit]").forEach((button) => {
    button.addEventListener("click", () => {
      const profile = state.profiles.find((item) => item.id === button.dataset.profileEdit);
      if (!profile) {
        showFlash("未找到要编辑的授权档案", true);
        return;
      }
      setProfileFormEditing(profile);
      focusProfile(profile.id);
      showFlash("已载入授权档案编辑表单");
    });
  });

  wrap.querySelectorAll("[data-profile-validate]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        const result = await api(`/api/auth/profiles/${button.dataset.profileValidate}/validate`, { method: "POST" });
        showFlash(`Validate 完成：${result.status}`);
        await loadProfiles();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });

  wrap.querySelectorAll("[data-profile-delete]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await api(`/api/auth/profiles/${button.dataset.profileDelete}`, { method: "DELETE" });
        showFlash("授权档案已删除");
        await loadProfiles();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });

  syncSourceProfiles();
  syncTargetProfiles();
}

function syncTargetProfiles() {
  const targetProvider = $("#plan-target-provider").value;
  const profiles = state.profiles.filter((item) => item.providerKey === targetProvider);
  const select = $("#plan-target-profile");
  const current = select.value || "";
  select.innerHTML = `<option value="">无</option>${profiles
    .map((profile) => `<option value="${profile.id}">${profile.displayName}</option>`)
    .join("")}`;
  if ([...select.options].some((option) => option.value === current)) {
    select.value = current;
  }
  syncTargetProviderInsight();
  syncTargetProfileInsight();
}

function syncSourceProfiles() {
  const sourceProvider = $("#plan-source-provider").value;
  const profiles = state.profiles.filter((item) => item.providerKey === sourceProvider);
  const select = $("#plan-source-profile");
  const current = select.value || "";
  select.innerHTML = `<option value="">无</option>${profiles
    .map((profile) => `<option value="${profile.id}">${profile.displayName}</option>`)
    .join("")}`;
  if ([...select.options].some((option) => option.value === current)) {
    select.value = current;
  }
}

function syncAutoRecoverProviders() {
  const select = $("#auto-recover-provider");
  if (!select) {
    return;
  }
  const current = state.autoRecoverFilters.providerKey || select.value || "";
  const providerKeys = Array.from(new Set((state.providers || []).map((item) => item?.meta?.key).filter(Boolean))).sort();
  select.innerHTML = `<option value="">全部 provider</option>${providerKeys
    .map((key) => `<option value="${key}">${key}</option>`)
    .join("")}`;
  setSelectValueIfPresent("#auto-recover-provider", current);
}

function syncAutoRecoverProtocolGroups() {
  const select = $("#auto-recover-protocol-group");
  if (!select) {
    return;
  }
  const current = state.autoRecoverFilters.protocolGroup || select.value || "";
  const groups = new Set();
  (state.evidence?.autoRecoverPool || []).forEach((item) => {
    const protocolGroups = Array.isArray(item?.protocolGroups) ? item.protocolGroups : [];
    protocolGroups.forEach((group) => {
      const normalized = String(group || "").trim();
      if (normalized) {
        groups.add(normalized);
      }
    });
    if (!protocolGroups.length) {
      const sampleGroup = String(item?.sampleProtocolGroup || "").trim();
      if (sampleGroup) {
        groups.add(sampleGroup);
      }
    }
  });
  const values = Array.from(groups).sort((left, right) => left.localeCompare(right, "zh-CN"));
  select.innerHTML = `<option value="">全部协议族</option>${values
    .map((value) => `<option value="${escapeHTML(value)}">${escapeHTML(value)}</option>`)
    .join("")}`;
  setFilterControlValue("#auto-recover-protocol-group", values.includes(current) ? current : "");
}

function syncAutoRecoverProfiles() {
  const select = $("#auto-recover-profile");
  if (!select) {
    return;
  }
  const current = state.autoRecoverFilters.profileId || select.value || "";
  const options = new Map();
  (state.evidence?.autoRecoverPool || []).forEach((item) => {
    (item?.profileIds || []).forEach((profileId) => {
      const normalized = String(profileId || "").trim();
      if (!normalized || options.has(normalized)) {
        return;
      }
      const profile = (state.profiles || []).find((entry) => entry?.id === normalized);
      const label = profile ? `${profile.displayName || normalized} (${profile.providerKey || "unknown"})` : normalized;
      options.set(normalized, label);
    });
  });
  const sorted = [...options.entries()].sort((left, right) => left[1].localeCompare(right[1], "zh-CN"));
  select.innerHTML = `<option value="">全部授权档案</option>${sorted
    .map(([value, label]) => `<option value="${escapeHTML(value)}">${escapeHTML(label)}</option>`)
    .join("")}`;
  setFilterControlValue(
    "#auto-recover-profile",
    sorted.some(([value]) => value === current) ? current : "",
  );
}

function syncAutoRecoverBlockedActions() {
  const select = $("#auto-recover-blocked-action");
  if (!select) {
    return;
  }
  const current = state.autoRecoverFilters.blockedAction || select.value || "";
  const actions = new Set();
  (state.evidence?.blockedActions || []).forEach((item) => {
    const action = String(item?.action || "").trim();
    if (action) {
      actions.add(action);
    }
  });
  (state.evidence?.autoRecoverPool || []).forEach((item) => {
    (item?.blockedActions || []).forEach((action) => {
      const normalized = String(action || "").trim();
      if (normalized) {
        actions.add(normalized);
      }
    });
  });
  const values = Array.from(actions).sort();
  select.innerHTML = `<option value="">全部阻塞动作</option>${values
    .map((value) => `<option value="${value}">${value}</option>`)
    .join("")}`;
  setFilterControlValue("#auto-recover-blocked-action", values.includes(current) ? current : "");
}

function syncExecutionModeHint() {
  const mode = $("#plan-execution-mode").value;
  const hint = $("#plan-execution-hint");
  if (mode === "pre_scan_flat") {
    hint.textContent = "pre_scan_flat 适合目录较小、希望先拿到完整扫描结果再执行的场景。";
    return;
  }
  hint.textContent = "leaf_first_lazy 是默认优先推荐模式，会按顶层目录顺序逐棵子树推进，只扫描当前真正需要传的目录。";
}

function updateExecutionRecommendationAction(metadata = {}) {
  const card = $("#plan-recommendation-action");
  const executionButton = $("#apply-recommended-execution");
  const riskButton = $("#apply-recommended-risk");
  const recommendedExecution = metadata.recommendedExecutionMode || "";
  const recommendedRisk = metadata.recommendedRiskMode || "";
  const selectedExecution = $("#plan-execution-mode").value;
  const selectedRisk = $("#plan-risk-mode").value;
  if (!recommendedExecution && !recommendedRisk) {
    card.classList.add("hidden");
    executionButton.disabled = true;
    riskButton.disabled = true;
    return;
  }

  const executionReason = stringifyValue(metadata.recommendedExecutionModeReason, "暂无执行模式推荐原因");
  const riskReason = stringifyValue(metadata.recommendedRiskModeReason, "暂无风控推荐原因");
  const aggressiveWarning = stringifyValue(metadata.aggressiveRiskWarning, "-");
  card.classList.remove("hidden");

  const titleParts = [];
  if (recommendedExecution) {
    titleParts.push(
      recommendedExecution === selectedExecution
        ? `执行模式已采用推荐值：${recommendedExecution}`
        : `建议执行模式：${recommendedExecution}`,
    );
  }
  if (recommendedRisk) {
    titleParts.push(
      recommendedRisk === selectedRisk
        ? `风控档位已采用推荐值：${recommendedRisk}`
        : `建议风控档位：${recommendedRisk}`,
    );
  }
  $("#plan-recommendation-title").textContent = titleParts.join(" / ");

  const reasonParts = [];
  if (recommendedExecution) {
    reasonParts.push(`执行模式：${executionReason}`);
  }
  if (recommendedRisk) {
    reasonParts.push(`风控档位：${riskReason}`);
  }
  if (aggressiveWarning && aggressiveWarning !== "-") {
    reasonParts.push(`提示：${aggressiveWarning}`);
  }
  $("#plan-recommendation-reason").textContent = reasonParts.join(" | ");

  executionButton.disabled = !recommendedExecution || recommendedExecution === selectedExecution;
  riskButton.disabled = !recommendedRisk || recommendedRisk === selectedRisk;
}

function renderTasks() {
  const wrap = $("#tasks-list");
  if (!state.tasks.length) {
    wrap.innerHTML = `<div class="task-item">暂无任务。</div>`;
    wireTaskQuickActions(null);
    $("#task-summary").innerHTML = `
      <div class="insight-card">
        <strong>执行模式</strong>
        <span>选择任务后显示</span>
      </div>
    `;
    $("#task-runtime").innerHTML = `
      <div class="insight-card">
        <strong>运行检查点</strong>
        <span>选择任务后显示</span>
      </div>
    `;
    updateTaskRetryQueue(null);
    $("#task-resolution-guide").innerHTML = `<div class="directory-empty">选择任务后显示处理建议。</div>`;
    updateTaskTreePanels(null);
    $("#task-detail").textContent = "选择一条任务查看详情...";
    return;
  }

  wrap.innerHTML = state.tasks
    .map(
      (detail) => `
        <article class="task-item ${detail.task.id === state.selectedTaskId ? "active" : ""}" data-task-id="${detail.task.id}">
          <h3>${detail.task.sourceProvider} -> ${detail.task.targetProvider}</h3>
          <div class="meta-row">
            <span class="pill">${detail.task.state}</span>
            <span class="pill">${detail.task.completionKind || "n/a"}</span>
            <span class="pill">${detail.plan.items.length} items</span>
            <span class="pill">${stringifyValue(detail.plan?.metadata?.executionMode, "mode:n/a")}</span>
          </div>
        </article>
      `,
    )
    .join("");

  wrap.querySelectorAll("[data-task-id]").forEach((node) => {
    node.addEventListener("click", () => {
      state.selectedTaskId = node.dataset.taskId;
      renderTasks();
      renderSelectedTask();
    });
  });

  if (!state.selectedTaskId && state.tasks[0]) {
    state.selectedTaskId = state.tasks[0].task.id;
  }
  renderSelectedTask();
}

function renderSelectedTask() {
  const detail = currentSelectedTaskDetail();
  if (!detail) {
    syncTaskActionButtons();
    wireTaskQuickActions(null);
    $("#task-summary").innerHTML = `
      <div class="insight-card">
        <strong>执行模式</strong>
        <span>选择任务后显示</span>
      </div>
    `;
    $("#task-runtime").innerHTML = `
      <div class="insight-card">
        <strong>运行检查点</strong>
        <span>选择任务后显示</span>
      </div>
    `;
    updateTaskRetryQueue(null);
    updateTaskTreePanels(null);
    $("#task-resolution-guide").innerHTML = `<div class="directory-empty">选择任务后显示处理建议。</div>`;
    $("#task-detail").textContent = "选择一条任务查看详情...";
    return;
  }
  const metadata = detail.plan?.metadata || {};
  const runtime = detail.runtime || metadata.runtime || {};
  syncTaskActionButtons();
  wireTaskQuickActions(detail);
  $("#task-summary").innerHTML = `
    <div class="insight-card">
      <strong>执行模式</strong>
      <span>${stringifyValue(metadata.executionMode)}</span>
    </div>
    ${renderRuntimePathChips("Selected Roots", metadata.selectedRoots || [], "task", "roots")}
    ${renderRuntimePathChips("Scan Trace", metadata.scanTrace || [], "task", "scan")}
    <div class="insight-card">
      <strong>推荐模式</strong>
      <span>${stringifyValue(metadata.recommendedExecutionMode)}</span>
    </div>
    <div class="insight-card">
      <strong>推荐原因</strong>
      <span>${stringifyValue(metadata.recommendedExecutionModeReason)}</span>
    </div>
    <div class="insight-card">
      <strong>扫描方式</strong>
      <span>${stringifyValue(metadata.scanMode, "尚未运行或无需扫描")}</span>
    </div>
    <div class="insight-card">
      <strong>风险节流</strong>
      <span>${stringifyValue(metadata.riskProfile?.requestIntervalMs, "0")}ms / dir ${stringifyValue(metadata.riskProfile?.directoryIntervalMs, "0")}ms / retry ${stringifyValue(metadata.riskProfile?.retryLimit, "0")} / conc ${stringifyValue(metadata.riskProfile?.maxConcurrent, "0")}</span>
    </div>
    ${renderRiskResolutionMetaCards(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>推荐风控</strong>
      <span>${stringifyValue(metadata.recommendedRiskMode, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>推荐风控原因</strong>
      <span>${stringifyValue(metadata.recommendedRiskModeReason, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>激进风险提示</strong>
      <span>${stringifyValue(metadata.aggressiveRiskWarning, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>源端删除策略</strong>
      <span>${renderSourceDeletePolicy(metadata.sourceDeletePolicy)}</span>
    </div>
    <div class="insight-card">
      <strong>风险模板解释</strong>
      <span>${escapeHTML(renderRiskResolutionSummary(metadata.riskProfileResolution))}</span>
      ${renderRiskResolutionDetail(metadata.riskProfileResolution)}
    </div>
    ${renderRiskResolutionFlow(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>自动补传时间窗</strong>
      <span>${escapeHTML(renderRiskWindow(metadata.riskProfile))}</span>
    </div>
    <div class="insight-card">
      <strong>重试范围</strong>
      <span>${metadata.retryPendingOnly ? `pending_only (${Array.isArray(metadata.retryPendingPaths) ? metadata.retryPendingPaths.length : 0} items)` : "full_task"}</span>
    </div>
    <div class="insight-card">
      <strong>重试模式</strong>
      <span>${stringifyValue(metadata.retryMode, "default")}</span>
    </div>
    <div class="insight-card">
      <strong>重试来源</strong>
      <span>${stringifyValue(metadata.retryScope, metadata.retrySelectedPaths ? "selected_subset" : "-")}</span>
    </div>
    <div class="insight-card">
      <strong>重试选中路径</strong>
      <span>${Array.isArray(metadata.retrySelectedPaths) && metadata.retrySelectedPaths.length ? summarizePathList(metadata.retrySelectedPaths, 4) : "-"}</span>
    </div>
    <div class="insight-card">
      <strong>重试路径数</strong>
      <span>${stringifyValue(metadata.retrySelectedPathCount, Array.isArray(metadata.retrySelectedPaths) ? metadata.retrySelectedPaths.length : 0)}</span>
    </div>
    <div class="insight-card">
      <strong>重试 checkpoint 数</strong>
      <span>${stringifyValue(metadata.retryUploadCheckpointCount, "0")}</span>
    </div>
    <div class="insight-card">
      <strong>重试摘要</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedReason || (metadata.retrySummary?.shouldBlock ? "blocked" : "ready"), "-")}</span>
    </div>
    <div class="insight-card">
      <strong>下一步摘要</strong>
      <span>${escapeHTML(
        renderBlockedSummary(
          runtime.blockedAction || metadata.retrySummary?.blockedAction,
          runtime.blockedAdvice || metadata.retrySummary?.blockedAdvice,
          runtime.nextRetryAt || metadata.retrySummary?.nextRetryAt,
          metadata.retrySummary?.autoRecoverAdvice,
        ),
      )}</span>
    </div>
    <div class="insight-card">
      <strong>源端删除记录</strong>
      <span>${stringifyValue(runtime.sourceDeletionCount || metadata.deletedEntryCount, "0")}</span>
    </div>
    <div class="insight-card">
      <strong>建议动作</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedAction, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>后台补传候选</strong>
      <span>${renderAutoRecoverMode(metadata.retrySummary)}</span>
    </div>
    <div class="insight-card">
      <strong>队列拆分</strong>
      <span>${renderRetrySummaryBreakdown(metadata.retrySummary)}</span>
    </div>
  `;
  $("#task-runtime").innerHTML = renderRuntimeCheckpoint(runtime, metadata, "task");
  wireRuntimePathFocus("task", "#task-summary");
  wireRuntimePathFocus("task");
  wireSourceDeletionSummary("task", "#task-runtime");
  updateTaskRetryQueue(detail);
  $("#task-resolution-guide").innerHTML = renderTaskResolutionGuide(detail);
  updateTaskTreePanels(detail);
  $("#task-detail").textContent = detail ? formatJSON(detail) : "选择一条任务查看详情...";
  wireTaskResolutionGuide(detail);
}

function renderPreview() {
  if (!state.preview) {
    updateExecutionRecommendationAction();
    $("#plan-preview-meta").innerHTML = `
      <div class="insight-card">
        <strong>当前模式</strong>
        <span>等待预览</span>
      </div>
    `;
    $("#plan-preview").textContent = "等待预览...";
    return;
  }
  const metadata = state.preview.metadata || {};
  const deletedEntryCount = Number(metadata.deletedEntryCount || 0);
  const activeEntryCount = Number(metadata.activeEntryCount || 0);
  const hasDeletedRecords = deletedEntryCount > 0;
  const deletionOnlyPreview = hasDeletedRecords && activeEntryCount === 0;
  updateExecutionRecommendationAction(metadata);
  $("#plan-preview-meta").innerHTML = `
    <div class="insight-card">
      <strong>当前模式</strong>
      <span>${stringifyValue(metadata.executionMode)}</span>
    </div>
    <div class="insight-card">
      <strong>Selected Roots</strong>
      <span><code>${escapeHTML(summarizePathList(metadata.selectedRoots || []))}</code></span>
    </div>
    <div class="insight-card">
      <strong>推荐模式</strong>
      <span>${stringifyValue(metadata.recommendedExecutionMode)}</span>
    </div>
    <div class="insight-card">
      <strong>推荐原因</strong>
      <span>${stringifyValue(metadata.recommendedExecutionModeReason)}</span>
    </div>
    <div class="insight-card">
      <strong>执行顺序</strong>
      <span>${stringifyValue(metadata.executionOrder)}</span>
    </div>
    <div class="insight-card">
      <strong>风险档位</strong>
      <span>${stringifyValue(metadata.riskProfile?.mode, "balanced")}</span>
    </div>
    <div class="insight-card">
      <strong>风险节流</strong>
      <span>${stringifyValue(metadata.riskProfile?.requestIntervalMs, "0")}ms / dir ${stringifyValue(metadata.riskProfile?.directoryIntervalMs, "0")}ms / retry ${stringifyValue(metadata.riskProfile?.retryLimit, "0")} / conc ${stringifyValue(metadata.riskProfile?.maxConcurrent, "0")}</span>
    </div>
    ${renderRiskResolutionMetaCards(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>推荐风控</strong>
      <span>${stringifyValue(metadata.recommendedRiskMode, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>推荐风控原因</strong>
      <span>${stringifyValue(metadata.recommendedRiskModeReason, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>激进风险提示</strong>
      <span>${stringifyValue(metadata.aggressiveRiskWarning, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>风险模板解释</strong>
      <span>${escapeHTML(renderRiskResolutionSummary(metadata.riskProfileResolution))}</span>
      ${renderRiskResolutionDetail(metadata.riskProfileResolution)}
    </div>
    ${renderRiskResolutionFlow(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>自动补传时间窗</strong>
      <span>${escapeHTML(renderRiskWindow(metadata.riskProfile))}</span>
    </div>
    <div class="insight-card">
      <strong>源端删除策略</strong>
      <span>${renderSourceDeletePolicy(metadata.sourceDeletePolicy)}</span>
    </div>
    <div class="insight-card">
      <strong>有效条目 / 删除记录</strong>
      <span>${stringifyValue(metadata.activeEntryCount, "0")} / ${stringifyValue(metadata.deletedEntryCount, "0")}</span>
    </div>
    ${hasDeletedRecords ? `
      <div class="insight-card checkpoint-card">
        <strong>删除记录仅用于定位</strong>
        <div>${deletionOnlyPreview ? "当前预览只剩删除记录，没有可执行条目；请先恢复源文件并重新预览。" : "当前预览包含删除记录，它们只会用于定位，不会生成可执行条目。"}</div>
      </div>
    ` : ""}
    ${renderSourceDeletionSummary(metadata.sourceDeletionRecords || [], metadata.deletedEntryCount || 0, "preview", "preview")}
  `;
  wireSourceDeletionSummary("preview", "#plan-preview-meta");
  $("#plan-preview").textContent = formatJSON(state.preview);
}

function renderStatus() {
  const evidence = state.evidence || {
    totalTasks: 0,
    completedTasks: 0,
    blockedTasks: 0,
    autoRecoverTasks: 0,
    autoRecoverRunnableTasks: 0,
    autoRecoverWaitingCooldownTasks: 0,
    autoRecoverWaitingRetryWindowTasks: 0,
    autoRecoverWaitingAuthRefreshTasks: 0,
    autoRecoverWaitingLocalRestoreTasks: 0,
    autoRecoverWaitingProviderSessionTasks: 0,
    autoRecoverWaitingManualTasks: 0,
    autoRecoverWaitingRetryLimitTasks: 0,
    autoRecoverWaitingOtherTasks: 0,
    failedResultCount: 0,
    doneResultCount: 0,
    skippedResultCount: 0,
    pendingResultCount: 0,
    sourceDeletionCount: 0,
    riskHitCount: 0,
    blockedActions: [],
    autoRecoverPool: [],
    protocolCoverage: [],
    recentResults: [],
    recentProbes: [],
  };
  const protocolCoverage = Array.isArray(evidence.protocolCoverage) ? evidence.protocolCoverage : [];
  const protocolCoverageWithSamples = protocolCoverage.filter((item) => item?.hasRealSuccessSample).length;
  const autoRetryPolicy = evidence.autoRetryPolicy && typeof evidence.autoRetryPolicy === "object" ? evidence.autoRetryPolicy : {};
  const providerSmokeMatrix = Array.isArray(state.providerSmokeMatrix) ? state.providerSmokeMatrix : [];
  const acceptedSmokeGroups = Number.isFinite(evidence.acceptedSmokeGroups) ? evidence.acceptedSmokeGroups : providerSmokeMatrix.filter((item) => item?.accepted).length;
  const inProgressSmokeGroups = Number.isFinite(evidence.inProgressSmokeGroups) ? evidence.inProgressSmokeGroups : providerSmokeMatrix.filter((item) => item?.acceptanceStatus === "in_progress").length;
  const pendingSmokeGroups = Number.isFinite(evidence.pendingSmokeGroups) ? evidence.pendingSmokeGroups : providerSmokeMatrix.filter((item) => item?.acceptanceStatus === "pending").length;
  const uploadSuccessSmokeGroups = Number.isFinite(evidence.uploadSuccessGroups) ? evidence.uploadSuccessGroups : providerSmokeMatrix.filter((item) => item?.hasUploadSuccessSample).length;
  const acceptanceActionCounts = evidence.acceptanceActionCounts && typeof evidence.acceptanceActionCounts === "object" ? evidence.acceptanceActionCounts : {};
  const acceptanceActionSummary = Object.entries(acceptanceActionCounts)
    .sort((left, right) => Number(right[1] || 0) - Number(left[1] || 0) || String(left[0]).localeCompare(String(right[0])))
    .map(([label, count]) => `${label} x${count}`)
    .join(" / ");
  syncAutoRecoverProtocolGroups();
  syncAutoRecoverProfiles();
  syncAutoRecoverBlockedActions();
  $("#evidence-summary").innerHTML = `
    <div class="metric"><span>Total Tasks</span><strong>${evidence.totalTasks}</strong></div>
    <div class="metric"><span>Completed</span><strong>${evidence.completedTasks}</strong></div>
    <div class="metric"><span>Blocked Tasks</span><strong>${evidence.blockedTasks}</strong></div>
    <div class="metric"><span>Execution Mode</span><strong>${stringifyValue(evidence.executionMode, "-")}</strong></div>
    <div class="metric"><span>Scan Mode</span><strong>${stringifyValue(evidence.scanMode, "-")}</strong></div>
    <div class="metric"><span>Source Delete</span><strong>${renderSourceDeletePolicy(evidence.sourceDeletePolicy)}</strong></div>
    <div class="metric"><span>Auto Recover</span><strong>${stringifyValue(evidence.autoRecoverTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Runnable</span><strong>${stringifyValue(evidence.autoRecoverRunnableTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Cooldown</span><strong>${stringifyValue(evidence.autoRecoverWaitingCooldownTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Window</span><strong>${stringifyValue(evidence.autoRecoverWaitingRetryWindowTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Auth</span><strong>${stringifyValue(evidence.autoRecoverWaitingAuthRefreshTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Local</span><strong>${stringifyValue(evidence.autoRecoverWaitingLocalRestoreTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Session</span><strong>${stringifyValue(evidence.autoRecoverWaitingProviderSessionTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Manual</span><strong>${stringifyValue(evidence.autoRecoverWaitingManualTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Limit</span><strong>${stringifyValue(evidence.autoRecoverWaitingRetryLimitTasks, "0")}</strong></div>
    <div class="metric"><span>Recover Other</span><strong>${stringifyValue(evidence.autoRecoverWaitingOtherTasks, "0")}</strong></div>
    <div class="metric"><span>Done Results</span><strong>${evidence.doneResultCount}</strong></div>
    <div class="metric"><span>Skipped Results</span><strong>${evidence.skippedResultCount}</strong></div>
    <div class="metric"><span>Pending Manual</span><strong>${evidence.pendingResultCount}</strong></div>
    <div class="metric"><span>Source Deletes</span><strong>${stringifyValue(evidence.sourceDeletionCount, "0")}</strong></div>
    <div class="metric"><span>Failed Results</span><strong>${evidence.failedResultCount}</strong></div>
    <div class="metric"><span>Risk Hits</span><strong>${evidence.riskHitCount}</strong></div>
    <div class="metric"><span>Auto Tick</span><strong>${escapeHTML(stringifyValue(autoRetryPolicy.tick, "-"))}</strong></div>
    <div class="metric"><span>Auto Batch</span><strong>${stringifyValue(autoRetryPolicy.batchLimit, "-")}</strong></div>
    <div class="metric"><span>Auto Lane Limit</span><strong>${stringifyValue(autoRetryPolicy.limitPerLane, "-")}</strong></div>
    <div class="metric"><span>Protocol Groups</span><strong>${protocolCoverage.length}</strong></div>
    <div class="metric"><span>Sampled Groups</span><strong>${protocolCoverageWithSamples}</strong></div>
    <div class="metric"><span>Accepted Groups</span><strong>${acceptedSmokeGroups}</strong></div>
    <div class="metric"><span>In Progress</span><strong>${inProgressSmokeGroups}</strong></div>
    <div class="metric"><span>Pending Groups</span><strong>${pendingSmokeGroups}</strong></div>
    <div class="metric"><span>Upload Success Groups</span><strong>${stringifyValue(evidence.uploadSuccessGroups, String(uploadSuccessSmokeGroups))}</strong></div>
    <div class="metric"><span>Upload Success Samples</span><strong>${stringifyValue(evidence.uploadSuccessSamples, "0")}</strong></div>
    <div class="metric"><span>Checkpoint Ready</span><strong>${escapeHTML(renderUploadCheckpointReadiness(evidence))}</strong></div>
    <div class="metric"><span>Checkpoint Priority</span><strong>${escapeHTML(renderUploadCheckpointPriorityAction(evidence))}</strong></div>
    <div class="metric"><span>Recover Priority</span><strong>${escapeHTML(renderAutoRecoverPriorityAction(evidence))}</strong></div>
    <div class="metric"><span>Recover Ready</span><strong>${escapeHTML(renderAutoRecoverReadiness(evidence))}</strong></div>
    <div class="metric"><span>Fairness Ready</span><strong>${escapeHTML(renderAutoRecoverFairnessReadiness(evidence))}</strong></div>
    <div class="metric"><span>Fairness Missing</span><strong>${escapeHTML(renderAutoRecoverFairnessMissing(evidence))}</strong></div>
    <div class="metric"><span>Fairness Priority</span><strong>${escapeHTML(renderAutoRecoverFairnessPriorityAction(evidence))}</strong></div>
    <div class="metric"><span>Acceptance Actions</span><strong>${escapeHTML(acceptanceActionSummary || "-")}</strong></div>
  `;
  $("#blocked-actions-summary").innerHTML = renderBlockedActionsSummary(evidence.blockedActions || []);
  wireBlockedActionsSummary();
  const autoRetryPolicySummary = [
    `tick ${stringifyValue(autoRetryPolicy.tick, "-")}`,
    `batch ${stringifyValue(autoRetryPolicy.batchLimit, "-")}`,
    `mode ${stringifyValue(autoRetryPolicy.limitPerMode, "-")}`,
    `lane ${stringifyValue(autoRetryPolicy.limitPerLane, "-")}`,
    `group ${stringifyValue(autoRetryPolicy.limitPerProtocolGroup, "-")}`,
    `provider ${stringifyValue(autoRetryPolicy.limitPerProvider, "-")}`,
    `profile ${stringifyValue(autoRetryPolicy.limitPerProfile, "-")}`,
  ].join(" / ");
  $("#auto-retry-policy-summary").textContent = `自动补传默认调度：${autoRetryPolicySummary}`;
  $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
    filterAutoRecoverItems(evidence.autoRecoverPool || []),
    evidence.autoRecoverPool || [],
  );
  $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(autoRetryPolicy);
  $("#auto-recover-last-result-summary").textContent = renderAutoRecoverLastResultSummary();
  $("#auto-recover-last-result-detail").innerHTML = renderAutoRecoverLastResultDetail();
  wireAutoRecoverLastResultDetail();
  $("#auto-recover-summary").innerHTML = renderAutoRecoverSummary(evidence.autoRecoverPool || []);
  wireAutoRecoverSummary();
  $("#protocol-coverage-summary").innerHTML = renderProtocolCoverageSummary(protocolCoverage);
  $("#provider-smoke-summary").innerHTML = `
    ${renderProviderSmokeMatrixControls(state.providerSmokeMatrix || [])}
    ${renderProviderSmokeSummary(state.providerSmokeSummary || [])}
  `;
  const currentReport = selectedEvidenceReport();
  $("#evidence-report").innerHTML = renderEvidenceReport(currentReport);
  $("#report-history").innerHTML = renderReportHistory(state.reportHistory || []);
  hydrateReportForm(currentReport);
  $("#provider-smoke-matrix").innerHTML = renderProviderSmokeMatrix(state.providerSmokeMatrix || []);
  const smokeRecordResult = filterProviderSmokeRecords(state.providerSmokes || [], state.providerSmokeRecordFilters);
  $("#provider-smoke-records-filter-summary").textContent = renderProviderSmokeRecordSummary(smokeRecordResult);
  $("#provider-smoke-records").innerHTML = renderProviderSmokeRecords(state.providerSmokes || []);
  $("#provider-smoke-markdown").innerHTML = renderProviderSmokeMarkdown(state.selectedProviderSmokeMarkdown);

  $("#status-table").innerHTML = `
    <table>
      <thead>
        <tr>
          <th>Provider</th>
          <th>Protocol Group</th>
          <th>Profiles</th>
          <th>Tasks</th>
          <th>Completed</th>
          <th>Coverage</th>
          <th>Execution Mode</th>
          <th>Scan Mode</th>
          <th>Source Delete</th>
          <th>Risk Mode</th>
          <th>Latest Probe</th>
          <th>Last Task State</th>
          <th>Blocked</th>
          <th>Auto Recover</th>
          <th>Main Action</th>
          <th>Snapshot Summary</th>
        </tr>
      </thead>
      <tbody>
        ${state.statuses
          .map(
            (item) => `
              <tr>
                <td>${item.providerKey}</td>
                <td>${item.protocolGroup || "-"}</td>
                <td>${item.profileCount}</td>
                <td>${item.taskCount}</td>
                <td>${item.completedCount}</td>
                <td>${item.protocolCoverage ? `${stringifyValue(item.protocolCoverage.realSuccessTaskCount, "0")}/${stringifyValue(item.protocolCoverage.providerCount, "0")}` : "-"}</td>
                <td>${stringifyValue(item.snapshotSummary?.executionMode)}</td>
                <td>${stringifyValue(item.snapshotSummary?.scanMode)}</td>
                <td>${renderSourceDeletePolicy(item.snapshotSummary?.sourceDeletePolicy)}</td>
                <td>${stringifyValue(item.snapshotSummary?.riskProfile?.mode)}</td>
                <td>${item.latestProbe || "-"}</td>
                <td>${item.lastTaskState || "-"}</td>
                <td>${stringifyValue(item.blockedCount, "0")}</td>
                <td>${stringifyValue(item.autoRecoverCount, "0")}</td>
                <td>${stringifyValue(item.snapshotSummary?.blockedActions?.[0]?.action, "-")}</td>
                <td>
                  <div class="summary-block">
                    ${renderSnapshotSummary(item.snapshotSummary)}
                  </div>
                </td>
              </tr>
            `,
          )
          .join("")}
      </tbody>
    </table>
  `;

  $("#recent-results").innerHTML = renderRecentResultsTable(evidence.recentResults || []);
  $("#recent-probes").innerHTML = renderRecentProbesTable(evidence.recentProbes || []);
  const runtimePayload = recentRuntimePayload();
  $("#status-runtime-checkpoints").innerHTML = renderRuntimeCheckpoint(runtimePayload?.runtime || runtimePayload, runtimePayload, "status");
  wireRuntimePathFocus("status");
  wireSourceDeletionSummary("status", "#status-runtime-checkpoints");
  updateStatusRetryQueue(runtimePayload);
  updateStatusTreePanels(runtimePayload);
}

function renderBlockedActionsSummary(items) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="directory-empty">当前没有需要人工处理的 blocked 聚合项。</div>`;
  }
  return items
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(stringifyValue(item.action))}</strong>
            <code>${escapeHTML(stringifyValue(item.sampleProvider, "-"))}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">tasks ${stringifyValue(item.taskCount, "0")}</span>
            <span class="pill">providers ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">next ${stringifyValue(item.nextRetryAt, "-")}</span>
          </div>
          <div class="muted">${escapeHTML(stringifyValue(item.advice, "-"))}</div>
          <div class="muted">next-step: ${escapeHTML(renderBlockedSummary(item.action, item.advice, item.nextRetryAt))}</div>
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-blocked-focus-action="${escapeHTML(stringifyValue(item.action))}"
            >只看这一类阻塞</button>
            <button
              type="button"
              class="ghost"
              data-blocked-run-action="${escapeHTML(stringifyValue(item.action))}"
            >执行此阻塞动作</button>
            <button
              type="button"
              class="ghost"
              data-blocked-open-task="${escapeHTML(stringifyValue(item.sampleTaskID || item.sampleTaskId, ""))}"
            >打开样本任务</button>
          </div>
        </div>
      `,
    )
    .join("");
}

function filterAutoRecoverItems(items, filters = state.autoRecoverFilters) {
  const mode = String(filters?.mode || "").trim();
  const protocolGroup = String(filters?.protocolGroup || "").trim();
  const providerKey = String(filters?.providerKey || "").trim();
  const profileId = String(filters?.profileId || "").trim();
  const retryClass = String(filters?.retryClass || "").trim();
  const blockedAction = String(filters?.blockedAction || "").trim();
  const recoverState = String(filters?.recoverState || "").trim();
  const strategy = String(filters?.strategy || "").trim();
  const source = Array.isArray(items) ? items : [];
  return source.filter((item) => {
    if (mode && String(item?.mode || "").trim() !== mode) {
      return false;
    }
    if (protocolGroup) {
      const protocolGroups = Array.isArray(item?.protocolGroups)
        ? item.protocolGroups.map((value) => String(value || "").trim()).filter(Boolean)
        : [];
      const effectiveGroups = protocolGroups.length
        ? protocolGroups
        : [String(item?.sampleProtocolGroup || "").trim()].filter(Boolean);
      if (!effectiveGroups.includes(protocolGroup)) {
        return false;
      }
    }
    if (providerKey && String(item?.sampleProvider || "").trim() !== providerKey) {
      return false;
    }
    if (profileId) {
      const sampleProfiles = Array.isArray(item?.profileIds) ? item.profileIds.map((value) => String(value || "").trim()) : [];
      if (!sampleProfiles.includes(profileId)) {
        return false;
      }
    }
    if (retryClass) {
      const sampleClasses = Array.isArray(item?.retryClasses) ? item.retryClasses.map((value) => String(value || "").trim()) : [];
      if (!sampleClasses.includes(retryClass)) {
        return false;
      }
    }
    if (blockedAction) {
      const sampleActions = Array.isArray(item?.blockedActions) ? item.blockedActions.map((value) => String(value || "").trim()) : [];
      if (!sampleActions.includes(blockedAction)) {
        return false;
      }
    }
    if (strategy) {
      const sampleStrategies = Array.isArray(item?.strategies)
        ? item.strategies.map((value) => String(value || "").trim()).filter(Boolean)
        : [];
      const effectiveStrategies = sampleStrategies.length
        ? sampleStrategies
        : [String(item?.sampleStrategy || "").trim()].filter(Boolean);
      if (!effectiveStrategies.includes(strategy)) {
        return false;
      }
    }
    if (recoverState) {
      const stateCount =
        recoverState === "runnable_now"
          ? Number(item?.runnableTaskCount || 0)
          : recoverState === "waiting_cooldown"
            ? Number(item?.waitingCooldownTaskCount || 0)
            : recoverState === "waiting_retry_window"
              ? Number(item?.waitingRetryWindowTaskCount || 0)
              : recoverState === "waiting_auth_refresh"
                ? Number(item?.waitingAuthRefreshTaskCount || 0)
                : recoverState === "waiting_local_restore"
                  ? Number(item?.waitingLocalRestoreTaskCount || 0)
                  : recoverState === "waiting_provider_session"
                    ? Number(item?.waitingProviderSessionTaskCount || 0)
                  : recoverState === "waiting_manual_confirmation"
                    ? Number(item?.waitingManualTaskCount || 0)
                    : recoverState === "waiting_retry_limit"
                      ? Number(item?.waitingRetryLimitTaskCount || 0)
                : recoverState === "waiting_other"
                  ? Number(item?.waitingOtherTaskCount || 0)
                  : 0;
      if (stateCount <= 0) {
        return false;
      }
    }
    return true;
  });
}

function renderAutoRecoverBudgetSummary(autoRetryPolicy) {
  const applied = [];
  const limit = String(state.autoRecoverFilters.limit || "").trim();
  const limitPerMode = String(state.autoRecoverFilters.limitPerMode || "").trim();
  const limitPerLane = String(state.autoRecoverFilters.limitPerLane || "").trim();
  const limitPerProtocolGroup = String(state.autoRecoverFilters.limitPerProtocolGroup || "").trim();
  const limitPerProvider = String(state.autoRecoverFilters.limitPerProvider || "").trim();
  const limitPerProfile = String(state.autoRecoverFilters.limitPerProfile || "").trim();
  if (limit) {
    applied.push(`batch ${limit}`);
  }
  applied.push(`mode ${limitPerMode || stringifyValue(autoRetryPolicy.limitPerMode, "-")}`);
  applied.push(`lane ${limitPerLane || stringifyValue(autoRetryPolicy.limitPerLane, "-")}`);
  applied.push(`group ${limitPerProtocolGroup || stringifyValue(autoRetryPolicy.limitPerProtocolGroup, "-")}`);
  applied.push(`provider ${limitPerProvider || stringifyValue(autoRetryPolicy.limitPerProvider, "-")}`);
  applied.push(`profile ${limitPerProfile || stringifyValue(autoRetryPolicy.limitPerProfile, "-")}`);
  const source = limit || limitPerMode || limitPerLane || limitPerProtocolGroup || limitPerProvider || limitPerProfile
    ? "当前手动放行预算"
    : "当前生效预算（默认）";
  return `${source}：${applied.join(" / ")}`;
}

function autoRecoverStateLabel(recoverState) {
  switch (String(recoverState || "").trim()) {
    case "runnable_now":
      return "可立即执行";
    case "waiting_cooldown":
      return "等待冷却";
    case "waiting_retry_window":
      return "等待自动补传时间窗";
    case "waiting_auth_refresh":
      return "等待授权刷新";
    case "waiting_local_restore":
      return "等待补回本地文件";
    case "waiting_provider_session":
      return "等待重建 Provider 会话";
    case "waiting_manual_confirmation":
      return "等待人工确认";
    case "waiting_retry_limit":
      return "等待重置重试策略";
    case "waiting_other":
      return "其它等待";
    default:
      return stringifyValue(recoverState, "-");
  }
}

function autoRecoverStateAdvice(recoverState) {
  switch (String(recoverState || "").trim()) {
    case "runnable_now":
      return "当前 lane 已满足预算与时间条件，可以直接预演或执行。";
    case "waiting_cooldown":
      return "先等待冷却到期，再观察 nextRetryAt 或下次自动补传 tick。";
    case "waiting_retry_window":
      return "当前不在允许的自动补传时间窗内，需等待窗口开放或手动调整风险配置。";
    case "waiting_auth_refresh":
      return "优先刷新或重新验证授权档案，再回到状态矩阵放行。";
    case "waiting_local_restore":
      return "源文件缺失或本地路径不可读，需先补回源文件后再继续补传。";
    case "waiting_provider_session":
      return "provider 返回体缺少 uploadid / upload session 等关键恢复线索，需先补齐会话信息。";
    case "waiting_manual_confirmation":
      return "该类失败需要先人工确认，再按子集 retry 或后台补传继续处理。";
    case "waiting_retry_limit":
      return "当前任务已达到重试上限，先检查失败原因与重试策略，再决定是否重置额度。";
    case "waiting_other":
      return "当前 lane 仍有未细分等待条件，建议结合 decisions 明细和 blocked action 继续排查。";
    default:
      return "";
  }
}

function autoRecoverDecisionAdvice(decision) {
  if (!decision || typeof decision !== "object") {
    return "当前决策没有额外等待态说明。";
  }
  const directAdvice = String(decision.advice || "").trim();
  if (directAdvice) {
    return directAdvice;
  }
  const blockedReason = String(decision.blockedReason || "").trim();
  if (blockedReason === "retry_queue_waiting_for_retry_window") {
    return "当前已满足自动补传条件，但不在允许的自动补传时间窗内，等待 nextRetryAt 后系统会自动接管。";
  }
  if (blockedReason === "retry_queue_waiting_for_cooldown") {
    return "当前处于风控冷却窗口，等待 nextRetryAt 后系统会尝试自动补传。";
  }
  return autoRecoverStateAdvice(decision.recoverState) || "当前决策没有额外等待态说明。";
}

function autoRecoverOutcomeLabel(outcome) {
  switch (String(outcome || "").trim()) {
    case "recovered":
      return "已放行执行";
    case "dry_run_recoverable":
      return "预演可放行";
    case "skipped_by_limit":
      return "被批次上限挡住";
    case "skipped_by_mode_budget":
      return "被模式预算挡住";
    case "skipped_by_lane_budget":
      return "被 lane 预算挡住";
    case "skipped_by_protocol_group_budget":
      return "被协议族预算挡住";
    case "skipped_by_provider_budget":
      return "被 provider 预算挡住";
    case "skipped_by_profile_budget":
      return "被账号预算挡住";
    case "waiting_cooldown":
      return "等待冷却";
    case "waiting_retry_window":
      return "等待时间窗";
    case "blocked":
      return "仍被阻塞";
    default:
      return stringifyValue(outcome, "-");
  }
}

function renderAutoRecoverOutcomeCounts(result) {
  const counts = result && typeof result === "object" && result.outcomeCounts && typeof result.outcomeCounts === "object"
    ? result.outcomeCounts
    : null;
  if (!counts) {
    return "";
  }
  const order = [
    "dry_run_recoverable",
    "recovered",
    "skipped_limit",
    "skipped_mode_budget",
    "skipped_lane_budget",
    "skipped_protocol_group_budget",
    "skipped_provider_budget",
    "skipped_profile_budget",
    "waiting_cooldown",
    "waiting_retry_window",
    "blocked",
  ];
  const parts = order
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${autoRecoverOutcomeLabel(key)} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / outcomes ${parts.join(" / ")}` : "";
}

function retryClassSummaryLabel(retryClass) {
  const key = String(retryClass || "").trim();
  switch (key) {
    case "retry_failed":
      return "普通重试失败";
    case "rate_limited":
      return "限流冷却";
    case "pending_manual":
      return "人工确认";
    case "provider_session_missing":
      return "会话缺口";
    case "auth_expired":
      return "授权过期";
    case "local_file_missing":
      return "本地文件缺失";
    default:
      return key || "-";
  }
}

function renderAutoRecoverRetryClassCounts(result) {
  const counts = result && typeof result === "object" && result.retryClassCounts && typeof result.retryClassCounts === "object"
    ? result.retryClassCounts
    : null;
  if (!counts) {
    return "";
  }
  const order = ["retry_failed", "rate_limited", "pending_manual", "provider_session_missing", "auth_expired", "local_file_missing"];
  const parts = order
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${retryClassSummaryLabel(key)} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / classes ${parts.join(" / ")}` : "";
}

function autoRecoverStateSummaryLabel(recoverState) {
  return autoRecoverStateLabel(recoverState);
}

function renderAutoRecoverRecoverStateCounts(result) {
  const counts = result && typeof result === "object" && result.recoverStateCounts && typeof result.recoverStateCounts === "object"
    ? result.recoverStateCounts
    : null;
  if (!counts) {
    return "";
  }
  const order = [
    "runnable_now",
    "waiting_cooldown",
    "waiting_retry_window",
    "waiting_auth_refresh",
    "waiting_local_restore",
    "waiting_manual_confirmation",
    "waiting_retry_limit",
    "waiting_other",
  ];
  const parts = order
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${autoRecoverStateSummaryLabel(key)} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / states ${parts.join(" / ")}` : "";
}

function blockedActionSummaryLabel(action) {
  const key = String(action || "").trim();
  switch (key) {
    case "wait_for_cooldown":
      return "等待冷却";
    case "wait_for_retry_window":
      return "等待时间窗";
    case "refresh_auth_profile":
      return "刷新授权";
    case "restore_local_source_file":
      return "补回本地文件";
    case "manual_confirmation_required":
      return "人工确认";
    case "manual_intervention_required":
      return "人工介入";
    case "review_and_reset_retry_strategy":
      return "重置重试策略";
    default:
      return key || "-";
  }
}

function renderAutoRecoverBlockedActionCounts(result) {
  const counts = result && typeof result === "object" && result.blockedActionCounts && typeof result.blockedActionCounts === "object"
    ? result.blockedActionCounts
    : null;
  if (!counts) {
    return "";
  }
  const order = [
    "wait_for_cooldown",
    "wait_for_retry_window",
    "refresh_auth_profile",
    "restore_local_source_file",
    "manual_confirmation_required",
    "manual_intervention_required",
    "review_and_reset_retry_strategy",
  ];
  const parts = order
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${blockedActionSummaryLabel(key)} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / actions ${parts.join(" / ")}` : "";
}

function renderAutoRecoverProtocolGroupCounts(result) {
  const counts = result && typeof result === "object" && result.protocolGroupCounts && typeof result.protocolGroupCounts === "object"
    ? result.protocolGroupCounts
    : null;
  if (!counts) {
    return "";
  }
  const parts = Object.keys(counts)
    .sort()
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${key} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / groups ${parts.join(" / ")}` : "";
}

function renderAutoRecoverProviderCounts(result) {
  const counts = result && typeof result === "object" && result.providerCounts && typeof result.providerCounts === "object"
    ? result.providerCounts
    : null;
  if (!counts) {
    return "";
  }
  const parts = Object.keys(counts)
    .sort()
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${key} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / providers ${parts.join(" / ")}` : "";
}

function renderAutoRecoverStrategyCounts(result) {
  const counts = result && typeof result === "object" && result.strategyCounts && typeof result.strategyCounts === "object"
    ? result.strategyCounts
    : null;
  if (!counts) {
    return "";
  }
  const parts = Object.keys(counts)
    .sort()
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${key} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / strategies ${parts.join(" / ")}` : "";
}

function renderAutoRecoverProfileCounts(result) {
  const counts = result && typeof result === "object" && result.profileCounts && typeof result.profileCounts === "object"
    ? result.profileCounts
    : null;
  if (!counts) {
    return "";
  }
  const parts = Object.keys(counts)
    .sort()
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${key} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / profiles ${parts.join(" / ")}` : "";
}

function autoRecoverLaneSummaryLabel(lane) {
  const parts = String(lane || "").split("::");
  const mode = stringifyValue(parts[0], "-");
  const retryClass = retryClassSummaryLabel(parts[1]);
  const blockedAction = blockedActionSummaryLabel(parts[2]);
  return `${mode} + ${retryClass} + ${blockedAction}`;
}

function renderAutoRecoverLaneCounts(result) {
  const counts = result && typeof result === "object" && result.laneCounts && typeof result.laneCounts === "object"
    ? result.laneCounts
    : null;
  if (!counts) {
    return "";
  }
  const parts = Object.keys(counts)
    .sort()
    .filter((key) => Number(counts[key] || 0) > 0)
    .map((key) => `${autoRecoverLaneSummaryLabel(key)} ${stringifyValue(counts[key], "0")}`);
  return parts.length ? ` / lanes ${parts.join(" / ")}` : "";
}

function autoRecoverSuggestedBudgetValue(value, fallback) {
  const numeric = Number(value || 0);
  if (Number.isFinite(numeric) && numeric > 0) {
    return numeric;
  }
  const fallbackNumeric = Number(fallback || 0);
  return Number.isFinite(fallbackNumeric) && fallbackNumeric > 0 ? fallbackNumeric : 0;
}

function renderAutoRecoverSuggestedBudgets(result) {
  if (!result || typeof result !== "object") {
    return "";
  }
  const parts = [];
  if (Number(result.suggestedLimitPerMode || 0) > 0) {
    parts.push(`mode ${stringifyValue(result.suggestedLimitPerMode, "0")}`);
  }
  if (Number(result.suggestedLimitPerLane || 0) > 0) {
    parts.push(`lane ${stringifyValue(result.suggestedLimitPerLane, "0")}`);
  }
  if (Number(result.suggestedLimitPerProtocolGroup || 0) > 0) {
    parts.push(`group ${stringifyValue(result.suggestedLimitPerProtocolGroup, "0")}`);
  }
  if (Number(result.suggestedLimitPerProvider || 0) > 0) {
    parts.push(`provider ${stringifyValue(result.suggestedLimitPerProvider, "0")}`);
  }
  if (Number(result.suggestedLimitPerProfile || 0) > 0) {
    parts.push(`profile ${stringifyValue(result.suggestedLimitPerProfile, "0")}`);
  }
  return parts.length ? ` / suggest ${parts.join(" / ")}` : "";
}

function renderAutoRecoverLastResultSummary() {
  const result = state.autoRecoverLastResult;
  if (!result || typeof result !== "object") {
    return "尚未执行后台补传预演或实际放行。";
  }
  const label = result.dryRun ? "最近预演" : "最近执行";
  const recoveredLabel = result.dryRun ? "可放行" : "recovered";
  return `${label}：matched ${stringifyValue(result.matchedCount, "0")} / ${recoveredLabel} ${stringifyValue(result.recoveredCount, "0")} / limit ${stringifyValue(result.skippedByLimit, "0")} / modeBudget ${stringifyValue(result.skippedByModeBudget, "0")} / laneBudget ${stringifyValue(result.skippedByLaneBudget, "0")} / protocolGroupBudget ${stringifyValue(result.skippedByProtocolGroupBudget, "0")} / providerBudget ${stringifyValue(result.skippedByProviderBudget, "0")} / profileBudget ${stringifyValue(result.skippedByProfileBudget, "0")} / cooldownWait ${stringifyValue(result.skippedByCooldownWait, "0")} / retryWindowWait ${stringifyValue(result.skippedByRetryWindowWait, "0")} / blocked ${stringifyValue(result.skippedByBlockedReason, "0")}${renderAutoRecoverOutcomeCounts(result)}${renderAutoRecoverRetryClassCounts(result)}${renderAutoRecoverRecoverStateCounts(result)}${renderAutoRecoverBlockedActionCounts(result)}${renderAutoRecoverProtocolGroupCounts(result)}${renderAutoRecoverProviderCounts(result)}${renderAutoRecoverStrategyCounts(result)}${renderAutoRecoverProfileCounts(result)}${renderAutoRecoverLaneCounts(result)}${renderAutoRecoverSuggestedBudgets(result)}${result.earliestNextRetryAt ? ` / earliest ${result.earliestNextRetryAt}` : ""}`;
}

function renderAutoRecoverDecisionBudgetHint(label, currentValue, suggestedValue) {
  const currentNumeric = Number(currentValue || 0);
  const suggestedNumeric = Number(suggestedValue || 0);
  if (!Number.isFinite(currentNumeric) || currentNumeric <= 0 || !Number.isFinite(suggestedNumeric) || suggestedNumeric <= 0) {
    return "";
  }
  return `${label} current ${stringifyValue(currentNumeric, "0")} -> suggest ${stringifyValue(suggestedNumeric, "0")}`;
}

function renderAutoRecoverDecisionBudgetHints(item) {
  if (!item || typeof item !== "object") {
    return "";
  }
  const hints = [
    renderAutoRecoverDecisionBudgetHint("mode budget", item.currentModeBudget, item.suggestedModeBudget),
    renderAutoRecoverDecisionBudgetHint("lane budget", item.currentLaneBudget, item.suggestedLaneBudget),
    renderAutoRecoverDecisionBudgetHint("group budget", item.currentProtocolGroupBudget, item.suggestedProtocolGroupBudget),
    renderAutoRecoverDecisionBudgetHint("provider budget", item.currentProviderBudget, item.suggestedProviderBudget),
    renderAutoRecoverDecisionBudgetHint("profile budget", item.currentProfileBudget, item.suggestedProfileBudget),
  ].filter(Boolean);
  if (!hints.length) {
    return "";
  }
  return `预算占用：${hints.join(" / ")}`;
}

function renderAutoRecoverLastResultDetail() {
  const result = state.autoRecoverLastResult;
  const decisions = Array.isArray(result?.decisions) ? result.decisions : [];
  if (!decisions.length) {
    return `<div class="directory-empty">最近一次后台补传预演或执行暂无决策明细。</div>`;
  }
  return decisions
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(autoRecoverOutcomeLabel(item.outcome))}</strong>
            <code>${escapeHTML(stringifyValue(item.taskId, "-"))}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">provider ${escapeHTML(stringifyValue(item.providerKey, "-"))}</span>
            <span class="pill">profile ${escapeHTML(stringifyValue(item.profileId, "-"))}</span>
            <span class="pill">strategy ${escapeHTML(stringifyValue(item.strategy, "-"))}</span>
            <span class="pill">mode ${escapeHTML(stringifyValue(item.mode, "-"))}</span>
            <span class="pill">state ${escapeHTML(autoRecoverStateLabel(item.recoverState))}</span>
            <span class="pill">mode budget ${escapeHTML(stringifyValue(item.suggestedModeBudget, "-"))}</span>
            <span class="pill">lane budget ${escapeHTML(stringifyValue(item.suggestedLaneBudget, "-"))}</span>
            <span class="pill">group budget ${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, "-"))}</span>
            <span class="pill">provider budget ${escapeHTML(stringifyValue(item.suggestedProviderBudget, "-"))}</span>
            <span class="pill">profile budget ${escapeHTML(stringifyValue(item.suggestedProfileBudget, "-"))}</span>
          </div>
          <div class="muted">path: <code>${escapeHTML(stringifyValue(item.path, "-"))}</code> / protocolGroup: <code>${escapeHTML(stringifyValue(item.protocolGroup, "-"))}</code></div>
          <div class="muted">retryClass: <code>${escapeHTML(stringifyValue(item.retryClass, "-"))}</code> / blockedAction: <code>${escapeHTML(stringifyValue(item.blockedAction, "-"))}</code> / blockedReason: <code>${escapeHTML(stringifyValue(item.blockedReason, "-"))}</code> / nextRetryAt: <code>${escapeHTML(stringifyValue(item.nextRetryAt, "-"))}</code></div>
          <div class="muted">next-step: ${escapeHTML(renderBlockedSummary(item.blockedAction, item.message, item.nextRetryAt, autoRecoverDecisionAdvice(item)))}</div>
          <div class="muted">${escapeHTML(renderAutoRecoverDecisionBudgetHints(item) || "预算占用：当前决策未返回可复用的预算占用信息。")}</div>
          <div class="muted">等待态说明：${escapeHTML(autoRecoverDecisionAdvice(item))}</div>
          <div class="muted">${escapeHTML(stringifyValue(item.message, "-"))}</div>
          <div class="tree-actions">
            <button type="button" class="link-button" data-auto-recover-decision-focus-state="${escapeHTML(stringifyValue(item.recoverState, ""))}">只看该状态</button>
            <button type="button" class="link-button" data-auto-recover-decision-focus-lane-mode="${escapeHTML(stringifyValue(item.mode, ""))}" data-auto-recover-decision-focus-lane-strategy="${escapeHTML(stringifyValue(item.strategy, ""))}" data-auto-recover-decision-focus-lane-retry-class="${escapeHTML(stringifyValue(item.retryClass, ""))}" data-auto-recover-decision-focus-lane-blocked-action="${escapeHTML(stringifyValue(item.blockedAction, ""))}">只看该 lane</button>
            <button type="button" class="link-button" data-auto-recover-decision-apply-budgets="1" data-auto-recover-decision-apply-mode-budget="${escapeHTML(stringifyValue(item.suggestedModeBudget, ""))}" data-auto-recover-decision-apply-lane-budget="${escapeHTML(stringifyValue(item.suggestedLaneBudget, ""))}" data-auto-recover-decision-apply-group-budget="${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, ""))}" data-auto-recover-decision-apply-provider-budget="${escapeHTML(stringifyValue(item.suggestedProviderBudget, ""))}" data-auto-recover-decision-apply-profile-budget="${escapeHTML(stringifyValue(item.suggestedProfileBudget, ""))}">采用建议预算</button>
            <button type="button" class="link-button" data-auto-recover-decision-preview="1" data-auto-recover-decision-task-id="${escapeHTML(stringifyValue(item.taskId, ""))}" data-auto-recover-decision-mode="${escapeHTML(stringifyValue(item.mode, ""))}" data-auto-recover-decision-strategy="${escapeHTML(stringifyValue(item.strategy, ""))}" data-auto-recover-decision-protocol-group="${escapeHTML(stringifyValue(item.protocolGroup, ""))}" data-auto-recover-decision-provider="${escapeHTML(stringifyValue(item.providerKey, ""))}" data-auto-recover-decision-profile="${escapeHTML(stringifyValue(item.profileId, ""))}" data-auto-recover-decision-retry-class="${escapeHTML(stringifyValue(item.retryClass, ""))}" data-auto-recover-decision-blocked-action="${escapeHTML(stringifyValue(item.blockedAction, ""))}" data-auto-recover-decision-recover-state="${escapeHTML(stringifyValue(item.recoverState, ""))}" data-auto-recover-decision-mode-budget="${escapeHTML(stringifyValue(item.suggestedModeBudget, ""))}" data-auto-recover-decision-lane-budget="${escapeHTML(stringifyValue(item.suggestedLaneBudget, ""))}" data-auto-recover-decision-group-budget="${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, ""))}" data-auto-recover-decision-provider-budget="${escapeHTML(stringifyValue(item.suggestedProviderBudget, ""))}" data-auto-recover-decision-profile-budget="${escapeHTML(stringifyValue(item.suggestedProfileBudget, ""))}">预演该决策</button>
            <button type="button" class="link-button" data-auto-recover-decision-run="1" data-auto-recover-decision-task-id="${escapeHTML(stringifyValue(item.taskId, ""))}" data-auto-recover-decision-mode="${escapeHTML(stringifyValue(item.mode, ""))}" data-auto-recover-decision-strategy="${escapeHTML(stringifyValue(item.strategy, ""))}" data-auto-recover-decision-protocol-group="${escapeHTML(stringifyValue(item.protocolGroup, ""))}" data-auto-recover-decision-provider="${escapeHTML(stringifyValue(item.providerKey, ""))}" data-auto-recover-decision-profile="${escapeHTML(stringifyValue(item.profileId, ""))}" data-auto-recover-decision-retry-class="${escapeHTML(stringifyValue(item.retryClass, ""))}" data-auto-recover-decision-blocked-action="${escapeHTML(stringifyValue(item.blockedAction, ""))}" data-auto-recover-decision-recover-state="${escapeHTML(stringifyValue(item.recoverState, ""))}" data-auto-recover-decision-mode-budget="${escapeHTML(stringifyValue(item.suggestedModeBudget, ""))}" data-auto-recover-decision-lane-budget="${escapeHTML(stringifyValue(item.suggestedLaneBudget, ""))}" data-auto-recover-decision-group-budget="${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, ""))}" data-auto-recover-decision-provider-budget="${escapeHTML(stringifyValue(item.suggestedProviderBudget, ""))}" data-auto-recover-decision-profile-budget="${escapeHTML(stringifyValue(item.suggestedProfileBudget, ""))}">执行该决策</button>
            <button type="button" class="link-button" data-auto-recover-decision-open-task="${escapeHTML(stringifyValue(item.taskId, ""))}">打开样本任务</button>
          </div>
        </div>
      `,
    )
    .join("");
}

function currentAutoRecoverDecisionRequest(button, dryRun = false) {
  if (!button) {
    return null;
  }
  const toPositiveNumber = (value) => {
    const parsed = Number(String(value || "").trim());
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
  };
  return {
    dryRun: Boolean(dryRun),
    taskId: String(button.dataset.autoRecoverDecisionTaskId || "").trim(),
    mode: String(button.dataset.autoRecoverDecisionMode || "").trim(),
    protocolGroup: String(button.dataset.autoRecoverDecisionProtocolGroup || "").trim(),
    providerKey: String(button.dataset.autoRecoverDecisionProvider || "").trim(),
    profileId: String(button.dataset.autoRecoverDecisionProfile || "").trim(),
    strategy: String(button.dataset.autoRecoverDecisionStrategy || "").trim(),
    retryClass: String(button.dataset.autoRecoverDecisionRetryClass || "").trim(),
    blockedAction: String(button.dataset.autoRecoverDecisionBlockedAction || "").trim(),
    recoverState: String(button.dataset.autoRecoverDecisionRecoverState || "").trim(),
    limitPerMode: toPositiveNumber(button.dataset.autoRecoverDecisionModeBudget),
    limitPerLane: toPositiveNumber(button.dataset.autoRecoverDecisionLaneBudget),
    limitPerProtocolGroup: toPositiveNumber(button.dataset.autoRecoverDecisionGroupBudget),
    limitPerProvider: toPositiveNumber(button.dataset.autoRecoverDecisionProviderBudget),
    limitPerProfile: toPositiveNumber(button.dataset.autoRecoverDecisionProfileBudget),
  };
}

async function triggerAutoRecoverDecision(button, options = {}) {
  const payload = currentAutoRecoverDecisionRequest(button, Boolean(options?.dryRun));
  if (!payload) {
    throw new Error("当前决策无效，无法继续后台补传操作");
  }
  applyAutoRecoverFilters({
    mode: payload.mode,
    strategy: payload.strategy,
    protocolGroup: payload.protocolGroup,
    providerKey: payload.providerKey,
    profileId: payload.profileId,
    retryClass: payload.retryClass,
    blockedAction: payload.blockedAction,
    recoverState: payload.recoverState,
    limitPerMode: payload.limitPerMode ? String(payload.limitPerMode) : "",
    limitPerLane: payload.limitPerLane ? String(payload.limitPerLane) : "",
    limitPerProtocolGroup: payload.limitPerProtocolGroup ? String(payload.limitPerProtocolGroup) : "",
    limitPerProvider: payload.limitPerProvider ? String(payload.limitPerProvider) : "",
    limitPerProfile: payload.limitPerProfile ? String(payload.limitPerProfile) : "",
  });
  const result = await api("/api/tasks/recover", {
    method: "POST",
    body: payload,
  });
  state.autoRecoverLastResult = result;
  if (payload.dryRun) {
    $("#auto-recover-last-result-summary").textContent = renderAutoRecoverLastResultSummary();
    $("#auto-recover-last-result-detail").innerHTML = renderAutoRecoverLastResultDetail();
    wireAutoRecoverLastResultDetail();
    showFlash(`已按决策预演后台补传：${stringifyValue(payload.taskId, "-")}`);
    return result;
  }
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(`已按决策执行后台补传：${stringifyValue(payload.taskId, "-")}`);
  return result;
}

function wireAutoRecoverLastResultDetail() {
  const wrap = $("#auto-recover-last-result-detail");
  if (!wrap) {
    return;
  }
  wrap.querySelectorAll("[data-auto-recover-decision-focus-state]").forEach((button) => {
    button.addEventListener("click", () => {
      const recoverState = button.dataset.autoRecoverDecisionFocusState || "";
      applyAutoRecoverFilters({ recoverState });
      showFlash(`已按决策状态 ${recoverState} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-decision-focus-lane-mode]").forEach((button) => {
    button.addEventListener("click", () => {
      const mode = button.dataset.autoRecoverDecisionFocusLaneMode || "";
      const strategy = button.dataset.autoRecoverDecisionFocusLaneStrategy || "";
      const retryClass = button.dataset.autoRecoverDecisionFocusLaneRetryClass || "";
      const blockedAction = button.dataset.autoRecoverDecisionFocusLaneBlockedAction || "";
      applyAutoRecoverFilters({ mode, strategy, retryClass, blockedAction });
      showFlash(`已按决策 lane 收敛后台补传候选：${[mode, strategy, retryClass, blockedAction].filter(Boolean).join(" / ")}`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-decision-apply-budgets]").forEach((button) => {
    button.addEventListener("click", () => {
      const limitPerMode = button.dataset.autoRecoverDecisionApplyModeBudget || "";
      const limitPerLane = button.dataset.autoRecoverDecisionApplyLaneBudget || "";
      const limitPerProtocolGroup = button.dataset.autoRecoverDecisionApplyGroupBudget || "";
      const limitPerProvider = button.dataset.autoRecoverDecisionApplyProviderBudget || "";
      const limitPerProfile = button.dataset.autoRecoverDecisionApplyProfileBudget || "";
      applyAutoRecoverFilters({
        limitPerMode,
        limitPerLane,
        limitPerProtocolGroup,
        limitPerProvider,
        limitPerProfile,
      });
      showFlash(`已按决策采用建议预算：mode ${limitPerMode || "-"} / lane ${limitPerLane || "-"} / group ${limitPerProtocolGroup || "-"} / provider ${limitPerProvider || "-"} / profile ${limitPerProfile || "-"}`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-decision-preview]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await triggerAutoRecoverDecision(button, { dryRun: true });
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-decision-run]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await triggerAutoRecoverDecision(button, { dryRun: false });
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-decision-open-task]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await openTaskByID(button.dataset.autoRecoverDecisionOpenTask || "");
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
}

function renderAutoRecoverFilterSummary(visibleItems, allItems) {
  const total = Array.isArray(allItems) ? allItems.length : 0;
  const current = Array.isArray(visibleItems) ? visibleItems.length : 0;
  const mode = String(state.autoRecoverFilters.mode || "").trim();
  const protocolGroup = String(state.autoRecoverFilters.protocolGroup || "").trim();
  const providerKey = String(state.autoRecoverFilters.providerKey || "").trim();
  const profileId = String(state.autoRecoverFilters.profileId || "").trim();
  const retryClass = String(state.autoRecoverFilters.retryClass || "").trim();
  const blockedAction = String(state.autoRecoverFilters.blockedAction || "").trim();
  const strategy = String(state.autoRecoverFilters.strategy || "").trim();
  const recoverState = String(state.autoRecoverFilters.recoverState || "").trim();
  const limit = String(state.autoRecoverFilters.limit || "").trim();
  const limitPerMode = String(state.autoRecoverFilters.limitPerMode || "").trim();
  const limitPerLane = String(state.autoRecoverFilters.limitPerLane || "").trim();
  const limitPerProtocolGroup = String(state.autoRecoverFilters.limitPerProtocolGroup || "").trim();
  const limitPerProvider = String(state.autoRecoverFilters.limitPerProvider || "").trim();
  const limitPerProfile = String(state.autoRecoverFilters.limitPerProfile || "").trim();
  const parts = [];
  if (mode) {
    parts.push(`mode=${mode}`);
  }
  if (protocolGroup) {
    parts.push(`protocolGroup=${protocolGroup}`);
  }
  if (providerKey) {
    parts.push(`provider=${providerKey}`);
  }
  if (profileId) {
    parts.push(`profileId=${profileId}`);
  }
  if (retryClass) {
    parts.push(`retryClass=${retryClass}`);
  }
  if (blockedAction) {
    parts.push(`blockedAction=${blockedAction}`);
  }
  if (strategy) {
    parts.push(`strategy=${strategy}`);
  }
  if (recoverState) {
    parts.push(`recoverState=${recoverState}`);
  }
  if (limit) {
    parts.push(`limit=${limit}`);
  }
  if (limitPerMode) {
    parts.push(`limitPerMode=${limitPerMode}`);
  }
  if (limitPerLane) {
    parts.push(`limitPerLane=${limitPerLane}`);
  }
  if (limitPerProtocolGroup) {
    parts.push(`limitPerProtocolGroup=${limitPerProtocolGroup}`);
  }
  if (limitPerProvider) {
    parts.push(`limitPerProvider=${limitPerProvider}`);
  }
  if (limitPerProfile) {
    parts.push(`limitPerProfile=${limitPerProfile}`);
  }
  const visibleSummary = summarizeAutoRecoverVisibleItems(visibleItems);
  const stateSummary = [
    `可立即执行 ${visibleSummary.runnable}`,
    `等冷却 ${visibleSummary.waitingCooldown}`,
    `等时间窗 ${visibleSummary.waitingRetryWindow}`,
    `等授权 ${visibleSummary.waitingAuthRefresh}`,
    `等本地恢复 ${visibleSummary.waitingLocalRestore}`,
    `等人工确认 ${visibleSummary.waitingManual}`,
    `重试耗尽 ${visibleSummary.waitingRetryLimit}`,
    `其它等待 ${visibleSummary.waitingOther}`,
  ].join(" / ");
  if (!parts.length) {
    return total
      ? `当前显示全部 ${current}/${total} 条后台补传候选，涉及任务 ${visibleSummary.tasks} 个；${stateSummary}。`
      : `显示全部后台补传候选；${stateSummary}。`;
  }
  return `当前显示 ${current}/${total} 条后台补传候选，涉及任务 ${visibleSummary.tasks} 个；${stateSummary}；筛选条件：${parts.join(" / ")}。`;
}

function summarizeAutoRecoverVisibleItems(items) {
  const source = Array.isArray(items) ? items : [];
  return source.reduce(
    (summary, item) => {
      summary.tasks += Number(item?.taskCount || 0);
      summary.runnable += Number(item?.runnableTaskCount || 0);
      summary.waitingCooldown += Number(item?.waitingCooldownTaskCount || 0);
      summary.waitingRetryWindow += Number(item?.waitingRetryWindowTaskCount || 0);
      summary.waitingAuthRefresh += Number(item?.waitingAuthRefreshTaskCount || 0);
      summary.waitingLocalRestore += Number(item?.waitingLocalRestoreTaskCount || 0);
      summary.waitingManual += Number(item?.waitingManualTaskCount || 0);
      summary.waitingRetryLimit += Number(item?.waitingRetryLimitTaskCount || 0);
      summary.waitingOther += Number(item?.waitingOtherTaskCount || 0);
      return summary;
    },
    {
      tasks: 0,
      runnable: 0,
      waitingCooldown: 0,
      waitingRetryWindow: 0,
      waitingAuthRefresh: 0,
      waitingLocalRestore: 0,
      waitingManual: 0,
      waitingRetryLimit: 0,
      waitingOther: 0,
    },
  );
}

function renderAutoRecoverSummary(items) {
  const autoRetryPolicy = state.evidence?.autoRetryPolicy && typeof state.evidence.autoRetryPolicy === "object" ? state.evidence.autoRetryPolicy : {};
  const visibleItems = filterAutoRecoverItems(items);
  if (!visibleItems.length) {
    if (Array.isArray(items) && items.length) {
      return `<div class="directory-empty">当前筛选条件下没有命中的后台补传候选。</div>`;
    }
    return `<div class="directory-empty">当前没有进入后台补传候选池的任务。</div>`;
  }
  const aggregate = summarizeAutoRecoverVisibleItems(visibleItems);
  return `
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>当前可见候选池摘要</strong>
        <code>${escapeHTML(`${visibleItems.length} lanes / ${aggregate.tasks} tasks`)}</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">runnable ${stringifyValue(aggregate.runnable, "0")}</span>
        <span class="pill">wait cooldown ${stringifyValue(aggregate.waitingCooldown, "0")}</span>
        <span class="pill">wait window ${stringifyValue(aggregate.waitingRetryWindow, "0")}</span>
        <span class="pill">wait auth ${stringifyValue(aggregate.waitingAuthRefresh, "0")}</span>
        <span class="pill">wait local ${stringifyValue(aggregate.waitingLocalRestore, "0")}</span>
        <span class="pill">wait manual ${stringifyValue(aggregate.waitingManual, "0")}</span>
        <span class="pill">wait limit ${stringifyValue(aggregate.waitingRetryLimit, "0")}</span>
        <span class="pill">wait other ${stringifyValue(aggregate.waitingOther, "0")}</span>
      </div>
      <div class="muted">这里的等待态表示候选已经进入后台补传池，但当前还不能立即执行；现在会把授权刷新、本地恢复、人工确认和重试耗尽单独拆出来，方便直接判断下一步动作。</div>
    </div>
  ` + visibleItems
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(stringifyValue(item.mode))}</strong>
            <code>${escapeHTML(stringifyValue(item.sampleProvider, "-"))}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">tasks ${stringifyValue(item.taskCount, "0")}</span>
            <span class="pill">providers ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">profiles ${stringifyValue(item.profileCount, "0")}</span>
            <span class="pill">queue ${stringifyValue(item.queueItemCount, "0")}</span>
            <span class="pill">group budget ${stringifyValue(item.suggestedProtocolGroupBudget, "-")}</span>
            <span class="pill">provider budget ${stringifyValue(item.suggestedProviderBudget, "-")}</span>
            <span class="pill">profile budget ${stringifyValue(item.suggestedProfileBudget, "-")}</span>
            <span class="pill">ready ${stringifyValue(item.retryableNowCount, "0")}</span>
            <span class="pill">cooldown ${stringifyValue(item.cooldownCount, "0")}</span>
            <span class="pill">runnable tasks ${stringifyValue(item.runnableTaskCount, "0")}</span>
            <span class="pill">wait cooldown ${stringifyValue(item.waitingCooldownTaskCount, "0")}</span>
            <span class="pill">wait window ${stringifyValue(item.waitingRetryWindowTaskCount, "0")}</span>
            <span class="pill">wait other ${stringifyValue(item.waitingOtherTaskCount, "0")}</span>
            <span class="pill">checkpoint ${stringifyValue(item.uploadCheckpointEligible, "0")}</span>
            <span class="pill">sample group ${stringifyValue(item.sampleProtocolGroup, "-")}</span>
            <span class="pill">groups ${(item.protocolGroups || []).join(", ") || "-"}</span>
            <span class="pill">primary class ${stringifyValue(item.primaryRetryClass, "-")}</span>
            <span class="pill">primary action ${stringifyValue(item.primaryBlockedAction, "-")}</span>
            <span class="pill">sample profile ${stringifyValue(item.sampleProfileId, "-")}</span>
            <span class="pill">classes ${(item.retryClasses || []).join(", ") || "-"}</span>
            <span class="pill">actions ${(item.blockedActions || []).join(", ") || "-"}</span>
          </div>
          ${
            item.primaryRetryClass || item.primaryBlockedAction
              ? `<div class="muted">主失败口径：retryClass <code>${escapeHTML(stringifyValue(item.primaryRetryClass, "-"))}</code> / blockedAction <code>${escapeHTML(stringifyValue(item.primaryBlockedAction, "-"))}</code></div>`
              : ""
          }
          <div class="muted">可执行态 ${escapeHTML(stringifyValue(item.runnableTaskCount, "0"))} / 等冷却 ${escapeHTML(stringifyValue(item.waitingCooldownTaskCount, "0"))} / 等时间窗 ${escapeHTML(stringifyValue(item.waitingRetryWindowTaskCount, "0"))} / 等刷新授权 ${escapeHTML(stringifyValue(item.waitingAuthRefreshTaskCount, "0"))} / 等补源文件 ${escapeHTML(stringifyValue(item.waitingLocalRestoreTaskCount, "0"))} / 等人工确认 ${escapeHTML(stringifyValue(item.waitingManualTaskCount, "0"))} / 重试耗尽 ${escapeHTML(stringifyValue(item.waitingRetryLimitTaskCount, "0"))} / 其它等待 ${escapeHTML(stringifyValue(item.waitingOtherTaskCount, "0"))}。</div>
          <div class="muted">等待态建议：${escapeHTML(
            Number(item?.waitingAuthRefreshTaskCount || 0) > 0
              ? autoRecoverStateAdvice("waiting_auth_refresh")
              : Number(item?.waitingLocalRestoreTaskCount || 0) > 0
                ? autoRecoverStateAdvice("waiting_local_restore")
                : Number(item?.waitingManualTaskCount || 0) > 0
                  ? autoRecoverStateAdvice("waiting_manual_confirmation")
                  : Number(item?.waitingRetryLimitTaskCount || 0) > 0
                    ? autoRecoverStateAdvice("waiting_retry_limit")
                    : Number(item?.waitingRetryWindowTaskCount || 0) > 0
                      ? autoRecoverStateAdvice("waiting_retry_window")
                      : Number(item?.waitingCooldownTaskCount || 0) > 0
                        ? autoRecoverStateAdvice("waiting_cooldown")
                        : autoRecoverStateAdvice("runnable_now"),
          )}</div>
          <div class="muted">同档位会先按模式、lane，再按协议族、provider 到授权档案轮转；默认建议 mode 预算 <code>${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedModeBudget, autoRetryPolicy.limitPerMode), "-"))}</code> / lane 预算 <code>${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedLaneBudget, autoRetryPolicy.limitPerLane), "-"))}</code> / group 预算 <code>${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProtocolGroupBudget, autoRetryPolicy.limitPerProtocolGroup), "-"))}</code> / provider 预算 <code>${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProviderBudget, autoRetryPolicy.limitPerProvider), "-"))}</code> / profile 预算 <code>${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProfileBudget, autoRetryPolicy.limitPerProfile), "-"))}</code>。</div>
          <div class="muted">协议族：${escapeHTML((item.protocolGroups || []).join(", ") || stringifyValue(item.sampleProtocolGroup, "-"))}</div>
          <div class="muted">${escapeHTML(stringifyValue(item.advice, "-"))}</div>
          <div class="actions compact">
            <span class="pill">next ${escapeHTML(stringifyValue(item.nextRetryAt, "-"))}</span>
            <button
              type="button"
              class="ghost"
              data-auto-recover-focus-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
            >只看该模式</button>
            ${
              item.sampleProtocolGroup
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-protocol-group="${escapeHTML(stringifyValue(item.sampleProtocolGroup, ""))}"
            >${Array.isArray(item.protocolGroups) && item.protocolGroups.length <= 1 ? "只看该协议族" : "只看样本协议族"}</button>`
                : ""
            }
            ${
              Number(item?.runnableTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="runnable_now"
            >只看可执行态</button>`
                : ""
            }
            ${
              Number(item?.waitingCooldownTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_cooldown"
            >只看等冷却</button>`
                : ""
            }
            ${
              Number(item?.waitingRetryWindowTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_retry_window"
            >只看等时间窗</button>`
                : ""
            }
            ${
              Number(item?.waitingAuthRefreshTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_auth_refresh"
            >只看等刷新授权</button>`
                : ""
            }
            ${
              Number(item?.waitingLocalRestoreTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_local_restore"
            >只看等补源文件</button>`
                : ""
            }
            ${
              Number(item?.waitingManualTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_manual_confirmation"
            >只看等人工确认</button>`
                : ""
            }
            ${
              Number(item?.waitingRetryLimitTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_retry_limit"
            >只看重试耗尽</button>`
                : ""
            }
            ${
              Number(item?.waitingOtherTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-state="waiting_other"
            >只看其它等待</button>`
                : ""
            }
            ${
              item.primaryRetryClass
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-retry-class="${escapeHTML(stringifyValue(item.primaryRetryClass, ""))}"
            >只看主重试类型</button>`
                : ""
            }
            ${
              item.primaryBlockedAction
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-primary-blocked-action="${escapeHTML(stringifyValue(item.primaryBlockedAction, ""))}"
            >只看主阻塞动作</button>`
                : ""
            }
            ${
              Array.isArray(item.blockedActions) && item.blockedActions.length
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-blocked-action="${escapeHTML(stringifyValue(item.blockedActions[0], ""))}"
            >只看该阻塞动作</button>`
                : ""
            }
            ${
              item.primaryRetryClass || item.primaryBlockedAction
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-focus-lane-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
              data-auto-recover-focus-lane-strategy="${escapeHTML(stringifyValue(item.sampleStrategy, ""))}"
              data-auto-recover-focus-lane-retry-class="${escapeHTML(stringifyValue(item.primaryRetryClass, ""))}"
              data-auto-recover-focus-lane-blocked-action="${escapeHTML(stringifyValue(item.primaryBlockedAction, ""))}"
            >只看该 lane</button>`
                : ""
            }
            <button
              type="button"
              class="ghost"
              data-auto-recover-apply-budgets="1"
              data-auto-recover-apply-mode-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedModeBudget, autoRetryPolicy.limitPerMode), ""))}"
              data-auto-recover-apply-lane-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedLaneBudget, autoRetryPolicy.limitPerLane), ""))}"
              data-auto-recover-apply-group-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProtocolGroupBudget, autoRetryPolicy.limitPerProtocolGroup), ""))}"
              data-auto-recover-apply-provider-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProviderBudget, autoRetryPolicy.limitPerProvider), ""))}"
              data-auto-recover-apply-profile-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProfileBudget, autoRetryPolicy.limitPerProfile), ""))}"
            >采用建议预算</button>
            <button
              type="button"
              class="ghost"
              data-auto-recover-preview-lane-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
              data-auto-recover-preview-lane-strategy="${escapeHTML(stringifyValue(item.sampleStrategy, ""))}"
              data-auto-recover-preview-lane-retry-class="${escapeHTML(stringifyValue(item.primaryRetryClass, ""))}"
              data-auto-recover-preview-lane-blocked-action="${escapeHTML(stringifyValue(item.primaryBlockedAction, ""))}"
              data-auto-recover-preview-mode-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedModeBudget, autoRetryPolicy.limitPerMode), ""))}"
              data-auto-recover-preview-lane-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedLaneBudget, autoRetryPolicy.limitPerLane), ""))}"
              data-auto-recover-preview-group-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProtocolGroupBudget, autoRetryPolicy.limitPerProtocolGroup), ""))}"
              data-auto-recover-preview-provider-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProviderBudget, autoRetryPolicy.limitPerProvider), ""))}"
              data-auto-recover-preview-profile-budget="${escapeHTML(stringifyValue(autoRecoverSuggestedBudgetValue(item.suggestedProfileBudget, autoRetryPolicy.limitPerProfile), ""))}"
            >预演该 lane</button>
            ${
              Array.isArray(item.blockedActions) && item.blockedActions.length
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-blocked-action="${escapeHTML(stringifyValue(item.blockedActions[0], ""))}"
            >执行该阻塞动作</button>`
                : ""
            }
            ${
              item.sampleProtocolGroup
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-preview-protocol-group="${escapeHTML(stringifyValue(item.sampleProtocolGroup, ""))}"
            >${Array.isArray(item.protocolGroups) && item.protocolGroups.length <= 1 ? "预演该协议族" : "预演样本协议族"}</button>`
                : ""
            }
            ${
              item.sampleProtocolGroup
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-protocol-group="${escapeHTML(stringifyValue(item.sampleProtocolGroup, ""))}"
            >${Array.isArray(item.protocolGroups) && item.protocolGroups.length <= 1 ? "执行该协议族" : "执行样本协议族"}</button>`
                : ""
            }
            ${
              Number(item?.runnableTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="runnable_now"
            >只执行可执行态</button>`
                : ""
            }
            ${
              Number(item?.waitingCooldownTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_cooldown"
            >只执行等冷却</button>`
                : ""
            }
            ${
              Number(item?.waitingRetryWindowTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_retry_window"
            >只执行等时间窗</button>`
                : ""
            }
            ${
              Number(item?.waitingAuthRefreshTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_auth_refresh"
            >只执行等刷新授权</button>`
                : ""
            }
            ${
              Number(item?.waitingLocalRestoreTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_local_restore"
            >只执行等补源文件</button>`
                : ""
            }
            ${
              Number(item?.waitingManualTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_manual_confirmation"
            >只执行等人工确认</button>`
                : ""
            }
            ${
              Number(item?.waitingRetryLimitTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_retry_limit"
            >只执行重试耗尽</button>`
                : ""
            }
            ${
              Number(item?.waitingOtherTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_other"
            >只执行其它等待</button>`
                : ""
            }
            ${
              item.primaryRetryClass
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-retry-class="${escapeHTML(stringifyValue(item.primaryRetryClass, ""))}"
            >执行主重试类型</button>`
                : ""
            }
            ${
              item.primaryBlockedAction
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-primary-blocked-action="${escapeHTML(stringifyValue(item.primaryBlockedAction, ""))}"
            >执行主阻塞动作</button>`
                : ""
            }
            ${
              item.primaryRetryClass || item.primaryBlockedAction
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-lane-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
              data-auto-recover-run-lane-strategy="${escapeHTML(stringifyValue(item.sampleStrategy, ""))}"
              data-auto-recover-run-lane-retry-class="${escapeHTML(stringifyValue(item.primaryRetryClass, ""))}"
              data-auto-recover-run-lane-blocked-action="${escapeHTML(stringifyValue(item.primaryBlockedAction, ""))}"
            >执行该 lane</button>`
                : ""
            }
            <button
              type="button"
              class="ghost"
              data-auto-recover-run-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
            >执行该模式</button>
            <button
              type="button"
              class="ghost"
              data-auto-recover-open-task="${escapeHTML(stringifyValue(item.sampleTaskId || item.sampleTaskID, ""))}"
            >打开样本任务</button>
          </div>
        </div>
      `,
    )
    .join("");
}

function applyAutoRecoverFilters(nextFilters, options = {}) {
  const filters = nextFilters && typeof nextFilters === "object" ? nextFilters : {};
  if (Object.prototype.hasOwnProperty.call(filters, "mode")) {
    state.autoRecoverFilters.mode = String(filters.mode || "");
    setFilterControlValue("#auto-recover-mode", state.autoRecoverFilters.mode);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "protocolGroup")) {
    state.autoRecoverFilters.protocolGroup = String(filters.protocolGroup || "");
    setFilterControlValue("#auto-recover-protocol-group", state.autoRecoverFilters.protocolGroup);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "providerKey")) {
    state.autoRecoverFilters.providerKey = String(filters.providerKey || "");
    setFilterControlValue("#auto-recover-provider", state.autoRecoverFilters.providerKey);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "profileId")) {
    state.autoRecoverFilters.profileId = String(filters.profileId || "");
    setFilterControlValue("#auto-recover-profile", state.autoRecoverFilters.profileId);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "retryClass")) {
    state.autoRecoverFilters.retryClass = String(filters.retryClass || "");
    setFilterControlValue("#auto-recover-retry-class", state.autoRecoverFilters.retryClass);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "blockedAction")) {
    state.autoRecoverFilters.blockedAction = String(filters.blockedAction || "");
    setFilterControlValue("#auto-recover-blocked-action", state.autoRecoverFilters.blockedAction);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "recoverState")) {
    state.autoRecoverFilters.recoverState = String(filters.recoverState || "");
    setFilterControlValue("#auto-recover-state", state.autoRecoverFilters.recoverState);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "strategy")) {
    state.autoRecoverFilters.strategy = String(filters.strategy || "");
    setFilterControlValue("#auto-recover-strategy", state.autoRecoverFilters.strategy);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "limit")) {
    state.autoRecoverFilters.limit = String(filters.limit || "");
    setInputValueIfPresent("#auto-recover-limit", state.autoRecoverFilters.limit);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "limitPerMode")) {
    const val = String(filters.limitPerMode || "");
    if (val !== "") {
      state.autoRecoverFilters.limitPerMode = val;
      setInputValueIfPresent("#auto-recover-limit-per-mode", val);
    }
  }
  if (Object.prototype.hasOwnProperty.call(filters, "limitPerLane")) {
    const val = String(filters.limitPerLane || "");
    if (val !== "") {
      state.autoRecoverFilters.limitPerLane = val;
      setInputValueIfPresent("#auto-recover-limit-per-lane", val);
    }
  }
  if (Object.prototype.hasOwnProperty.call(filters, "limitPerProtocolGroup")) {
    state.autoRecoverFilters.limitPerProtocolGroup = String(filters.limitPerProtocolGroup || "");
    setInputValueIfPresent("#auto-recover-limit-per-protocol-group", state.autoRecoverFilters.limitPerProtocolGroup);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "limitPerProvider")) {
    state.autoRecoverFilters.limitPerProvider = String(filters.limitPerProvider || "");
    setInputValueIfPresent("#auto-recover-limit-per-provider", state.autoRecoverFilters.limitPerProvider);
  }
  if (Object.prototype.hasOwnProperty.call(filters, "limitPerProfile")) {
    state.autoRecoverFilters.limitPerProfile = String(filters.limitPerProfile || "");
    setInputValueIfPresent("#auto-recover-limit-per-profile", state.autoRecoverFilters.limitPerProfile);
  }
  if (options.render !== false) {
    renderStatus();
  }
}

function renderProtocolCoverageSummary(items) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="directory-empty">当前没有协议族覆盖数据。</div>`;
  }
  return items
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(stringifyValue(item.protocolGroup))}</strong>
            <code>${escapeHTML(item.hasRealSuccessSample ? "sampled" : "pending")}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">providers ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">tasks ${stringifyValue(item.taskCount, "0")}</span>
            <span class="pill">completed ${stringifyValue(item.completedTaskCount, "0")}</span>
            <span class="pill">real ${stringifyValue(item.realSuccessTaskCount, "0")}</span>
          </div>
          <div class="muted">providers: ${escapeHTML((item.providerKeys || []).join(", ") || "-")}</div>
          <div class="muted">sample: ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleTaskId, "-"))}</div>
        </div>
      `,
    )
    .join("");
}

function renderSnapshotSummary(summary) {
  if (!summary || typeof summary !== "object") {
    return "-";
  }
  const retrySummary = summary.retrySummary;
  const blockedActions = Array.isArray(summary.blockedActions) ? summary.blockedActions : [];
  const retryPaths =
    Array.isArray(summary.retrySelectedPaths) && summary.retrySelectedPaths.length
      ? summarizePathList(summary.retrySelectedPaths, 4)
      : "-";
  if (retrySummary && typeof retrySummary === "object") {
    return `
      <div><strong>lastTaskState</strong> <code>${escapeHTML(stringifyValue(summary.lastTaskState))}</code></div>
      <div><strong>executionMode</strong> <code>${escapeHTML(stringifyValue(summary.executionMode, "-"))}</code></div>
      <div><strong>scanMode</strong> <code>${escapeHTML(stringifyValue(summary.scanMode, "-"))}</code></div>
      <div><strong>retryMode</strong> <code>${escapeHTML(stringifyValue(summary.retryMode, "-"))}</code></div>
      <div><strong>retryScope</strong> <code>${escapeHTML(stringifyValue(summary.retryScope, "-"))}</code></div>
      <div><strong>retrySelectedPathCount</strong> <code>${escapeHTML(stringifyValue(summary.retrySelectedPathCount, Array.isArray(summary.retrySelectedPaths) ? summary.retrySelectedPaths.length : 0))}</code></div>
      <div><strong>retryUploadCheckpointCount</strong> <code>${escapeHTML(stringifyValue(summary.retryUploadCheckpointCount, "0"))}</code></div>
      <div><strong>retrySelectedPaths</strong> <code>${escapeHTML(retryPaths)}</code></div>
      <div><strong>riskProfileResolution</strong> <code>${escapeHTML(renderRiskResolutionSummary(summary.riskProfileResolution))}</code></div>
      <div><strong>profileDefaultKindBias</strong> <code>${escapeHTML(`${stringifyValue(summary.riskProfileResolution?.profileDefaultSourceKind, "-")} / ${stringifyValue(summary.riskProfileResolution?.profileDefaultBias, "same_as_provider")}`)}</code></div>
      <div><strong>recoverBudgetReason</strong> <code>${escapeHTML(stringifyValue(summary.riskProfileResolution?.recoverBudget?.reason, "-"))}</code></div>
      <div>${renderRiskResolutionDetail(summary.riskProfileResolution)}</div>
      <div><strong>blockedCount</strong> <code>${escapeHTML(stringifyValue(summary.blockedCount, "0"))}</code></div>
      <div><strong>autoRecoverCount</strong> <code>${escapeHTML(stringifyValue(summary.autoRecoverCount, "0"))}</code></div>
      <div><strong>retryBlocked</strong> <code>${escapeHTML(stringifyValue(retrySummary.blockedReason, "-"))}</code></div>
      <div><strong>blockedAction</strong> <code>${escapeHTML(stringifyValue(retrySummary.blockedAction, "-"))}</code></div>
      <div><strong>blockedTop</strong> <code>${escapeHTML(stringifyValue(blockedActions[0]?.action, "-"))}</code></div>
      <div><strong>blockedSummary</strong> <code>${escapeHTML(
        renderBlockedSummary(
          retrySummary.blockedAction,
          retrySummary.blockedAdvice,
          retrySummary.nextRetryAt,
          retrySummary.autoRecoverAdvice,
        ),
      )}</code></div>
      <div><strong>nextRetryAt</strong> <code>${escapeHTML(stringifyValue(retrySummary.nextRetryAt, "-"))}</code></div>
      <div><strong>queueSize</strong> <code>${escapeHTML(stringifyValue(retrySummary.queueSize, "0"))}</code></div>
      <div><strong>autoRecover</strong> <code>${escapeHTML(renderAutoRecoverMode(retrySummary))}</code></div>
      <div><strong>autoRecoverRunnableTasks</strong> <code>${escapeHTML(stringifyValue(summary.autoRecoverRunnableTasks, "0"))}</code></div>
      <div><strong>autoRecoverWaitingCooldownTasks</strong> <code>${escapeHTML(stringifyValue(summary.autoRecoverWaitingCooldownTasks, "0"))}</code></div>
      <div><strong>autoRecoverWaitingRetryWindowTasks</strong> <code>${escapeHTML(stringifyValue(summary.autoRecoverWaitingRetryWindowTasks, "0"))}</code></div>
      <div><strong>autoRecoverWaitingOtherTasks</strong> <code>${escapeHTML(stringifyValue(summary.autoRecoverWaitingOtherTasks, "0"))}</code></div>
      <div><strong>queueBreakdown</strong> <code>${escapeHTML(renderRetrySummaryBreakdown(retrySummary))}</code></div>
      <div><strong>sourceDeletePolicy</strong> <code>${escapeHTML(renderSourceDeletePolicy(summary.sourceDeletePolicy))}</code></div>
      <div><strong>sourceDeletes</strong> <code>${escapeHTML(stringifyValue(summary.sourceDeletionCount, "0"))}</code></div>
      <div><strong>autoRecoverPool</strong> <code>${escapeHTML(stringifyValue((summary.autoRecoverPool || []).map((item) => item.mode).join(", "), "-"))}</code></div>
      <div><strong>protocolCoverage</strong> <code>${escapeHTML(stringifyValue(summary.protocolCoverage?.protocolGroup, "-"))} / ${escapeHTML(stringifyValue(summary.protocolCoverage?.realSuccessTaskCount, "0"))}</code></div>
    `;
  }
  return Object.entries(summary)
    .map(([key, value]) => `<div><strong>${key}</strong> <code>${escapeHTML(stringifyValue(value))}</code></div>`)
    .join("");
}

function renderRecentResultsTable(items) {
  if (!items.length) {
    return `<div class="provider-card">暂无结果证据。</div>`;
  }
  return `
    <table>
      <thead>
        <tr>
          <th>Status</th>
          <th>Mode</th>
          <th>Execution Mode</th>
          <th>Retry Mode</th>
          <th>Retry Scope</th>
          <th>Retry Path Count</th>
          <th>Retry Paths</th>
          <th>Source Delete</th>
          <th>Recommended</th>
          <th>Message</th>
          <th>Risk Hit</th>
          <th>Conflict</th>
          <th>Created</th>
        </tr>
      </thead>
      <tbody>
        ${items
          .map(
            (item) => `
              <tr>
                <td>${item.status}</td>
                <td>${item.mode || "-"}</td>
                <td>${stringifyValue(item.payload?.executionMode)}</td>
                <td>${stringifyValue(item.payload?.retryMode)}</td>
                <td>${stringifyValue(item.payload?.retryScope, item.payload?.retrySelectedPaths?.length ? "selected_subset" : "-")}</td>
                <td>${stringifyValue(item.payload?.retrySelectedPathCount, item.payload?.retrySelectedPaths?.length || 0)}</td>
                <td><code>${escapeHTML(summarizePathList(item.payload?.retrySelectedPaths || [], 3))}</code></td>
                <td>${renderSourceDeletePolicy(item.payload?.sourceDeletePolicy)}</td>
                <td>${stringifyValue(item.payload?.recommendedExecutionMode)}</td>
                <td>${item.message || "-"}</td>
                <td>${stringifyValue(item.payload?.riskHit?.keyword || item.payload?.riskHit?.status)}</td>
                <td>${item.conflictAction || "-"}</td>
                <td>${item.createdAt || "-"}</td>
              </tr>
            `,
          )
          .join("")}
      </tbody>
    </table>
  `;
}

function renderRecentProbesTable(items) {
  if (!items.length) {
    return `<div class="provider-card">暂无 probe 证据。</div>`;
  }
  return `
    <table>
      <thead>
        <tr>
          <th>Provider</th>
          <th>Status</th>
          <th>Profile</th>
          <th>Execution Mode</th>
          <th>Scan Mode</th>
          <th>Retry Mode</th>
          <th>Retry Scope</th>
          <th>Retry Path Count</th>
          <th>Retry Paths</th>
          <th>Source Delete</th>
          <th>Risk Hit</th>
          <th>Payload</th>
          <th>Created</th>
        </tr>
      </thead>
      <tbody>
        ${items
          .map(
            (item) => `
              <tr>
                <td>${item.providerKey}</td>
                <td>${item.status}</td>
                <td>${item.profileId || "-"}</td>
                <td>${stringifyValue(item.payload?.executionMode)}</td>
                <td>${stringifyValue(item.payload?.scanMode)}</td>
                <td>${stringifyValue(item.payload?.retryMode)}</td>
                <td>${stringifyValue(item.payload?.retryScope, item.payload?.retrySelectedPaths?.length ? "selected_subset" : "-")}</td>
                <td>${stringifyValue(item.payload?.retrySelectedPathCount, item.payload?.retrySelectedPaths?.length || 0)}</td>
                <td><code>${escapeHTML(summarizePathList(item.payload?.retrySelectedPaths || [], 3))}</code></td>
                <td>${renderSourceDeletePolicy(item.payload?.sourceDeletePolicy)}</td>
                <td>${stringifyValue(item.payload?.lastRiskStatus || item.payload?.riskHitCount)}</td>
                <td><code>${JSON.stringify(item.payload || {})}</code></td>
                <td>${item.createdAt || "-"}</td>
              </tr>
            `,
          )
          .join("")}
      </tbody>
    </table>
  `;
}

async function loadProviders() {
  const data = await api("/api/providers");
  state.providers = data.items || [];
  renderProviders();
  if (state.selectedProviderCapabilityKey) {
    await loadProviderCapabilityDetail(state.selectedProviderCapabilityKey);
  }
}

async function loadProfiles() {
  const data = await api("/api/auth/profiles");
  state.profiles = data.items || [];
  renderProfiles();
}

async function loadTasks() {
  const data = await api("/api/tasks");
  state.tasks = data.items || [];
  renderTasks();
}

async function loadStatus() {
  const [evidence, statuses, report, history, smokes, smokeSummary, smokeMatrix] = await Promise.all([
    api("/api/evidence/runtime"),
    api("/api/status/providers"),
    api("/api/evidence/report"),
    api("/api/evidence/reports"),
    api("/api/provider-smokes"),
    api("/api/provider-smokes/summary"),
    api("/api/provider-smokes/matrix"),
  ]);
  state.evidence = evidence;
  state.statuses = statuses.items || [];
  state.report = report;
  state.reportHistory = history.items || [];
  state.providerSmokes = smokes.items || [];
  state.providerSmokeSummary = smokeSummary.items || [];
  state.providerSmokeMatrix = smokeMatrix.items || [];
  if (state.selectedProviderSmokeId && !state.providerSmokes.some((item) => item.id === state.selectedProviderSmokeId)) {
    state.selectedProviderSmokeId = "";
    state.selectedProviderSmokeMarkdown = "";
  }
  if (state.selectedReportId && !state.reportHistory.some((item) => item.id === state.selectedReportId)) {
    state.selectedReportId = "";
  }
  renderStatus();
}

function selectedEvidenceReport() {
  if (state.selectedReportId) {
    const record = state.reportHistory.find((item) => item.id === state.selectedReportId);
    if (record && record.markdown) {
      return record;
    }
  }
  return state.report;
}

function renderReportHistory(items) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="directory-empty">暂无持久化报告记录。</div>`;
  }
  return items
    .map(
      (item) => `
        <div class="directory-row tree-node ${item.id === state.selectedReportId ? "active" : ""}">
          <div class="directory-row-header">
            <strong>${escapeHTML(item.title || item.generatedAt || "-")}</strong>
            <code>${escapeHTML(item.id || "-")}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">time ${escapeHTML(stringifyValue(item.generatedAt || "-", "-"))}</span>
            <span class="pill">tasks ${stringifyValue(item.summary?.totalTasks, "0")}</span>
            <span class="pill">blocked ${stringifyValue(item.summary?.blockedTasks, "0")}</span>
            <span class="pill">providers ${stringifyValue(item.statuses?.length, "0")}</span>
            <span class="pill">samples ${stringifyValue(item.samples?.length, "0")}</span>
          </div>
          ${item.note ? `<div class="muted">${escapeHTML(item.note)}</div>` : ""}
          <div class="actions compact">
            <button type="button" class="ghost" data-report-view="${escapeHTML(item.id || "")}">查看</button>
            <button type="button" class="ghost" data-report-download="${escapeHTML(item.id || "")}">下载</button>
          </div>
        </div>
      `, 
    )
    .join("");
}

function providerSmokeProviderCounts(items) {
  const counts = {
    total: 0,
    ready: 0,
    partial: 0,
    pending: 0,
    missingBasic: 0,
    missingUpload: 0,
    missingAnomaly: 0,
    missingRepresentative: 0,
  };
  for (const item of Array.isArray(items) ? items : []) {
    counts.total += 1;
    if (!item?.hasBasicSuccessSample) {
      counts.missingBasic += 1;
    }
    if (!item?.hasUploadSuccessSample) {
      counts.missingUpload += 1;
    }
    if (Array.isArray(item?.anomalyMissing) && item.anomalyMissing.length) {
      counts.missingAnomaly += 1;
    }
    if (Array.isArray(item?.representativeMissing) && item.representativeMissing.length) {
      counts.missingRepresentative += 1;
    }
    const readiness = String(item?.readiness || "").trim().toLowerCase();
    if (readiness === "ready") {
      counts.ready += 1;
      continue;
    }
    if (readiness === "partial") {
      counts.partial += 1;
      continue;
    }
    counts.pending += 1;
  }
  return counts;
}

function renderProviderSmokeProviderReadinessLabel(value) {
  const readiness = String(value || "").trim().toLowerCase();
  if (readiness === "ready") {
    return "ready（基础、上传、异常、代表性样本齐）";
  }
  if (readiness === "partial") {
    return "partial（已有样本，仍缺验收项）";
  }
  return "pending（待补 provider 真实样本）";
}

function renderEvidenceProviderSmokeProviders(report) {
  const items = Array.isArray(report?.providerSmokeProviders) ? report.providerSmokeProviders : [];
  if (!items.length) {
    return `
      <div class="insight-card">
        <strong>Provider 级真实样本验收</strong>
        <span>暂无 providerSmokeProviders 数据，请先刷新或保存新版验收报告。</span>
      </div>
    `;
  }
  const counts = providerSmokeProviderCounts(items);
  const summary = report?.summary && typeof report.summary === "object" ? report.summary : {};
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderTotalCount")) {
    counts.total = Number(summary.providerSmokeProviderTotalCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderReadyCount")) {
    counts.ready = Number(summary.providerSmokeProviderReadyCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderPartialCount")) {
    counts.partial = Number(summary.providerSmokeProviderPartialCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderPendingCount")) {
    counts.pending = Number(summary.providerSmokeProviderPendingCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderMissingBasicCount")) {
    counts.missingBasic = Number(summary.providerSmokeProviderMissingBasicCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderMissingUploadCount")) {
    counts.missingUpload = Number(summary.providerSmokeProviderMissingUploadCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderMissingAnomalyCount")) {
    counts.missingAnomaly = Number(summary.providerSmokeProviderMissingAnomalyCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerSmokeProviderMissingRepresentativeCount")) {
    counts.missingRepresentative = Number(summary.providerSmokeProviderMissingRepresentativeCount || 0);
  }
  const focusItems = items
    .filter((item) => String(item?.readiness || "").toLowerCase() !== "ready")
    .slice(0, 6);
  return `
    <div class="insight-card">
      <strong>Provider 级真实样本验收</strong>
      <span>Provider Ready ${counts.ready} / ${counts.total}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>Provider 验收缺口速览</strong>
        <code>providerSmokeProviders</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">ready ${counts.ready}</span>
        <span class="pill">partial ${counts.partial}</span>
        <span class="pill">pending ${counts.pending}</span>
        <span class="pill">missing basic ${counts.missingBasic}</span>
        <span class="pill">missing upload ${counts.missingUpload}</span>
        <span class="pill">missing anomaly ${counts.missingAnomaly}</span>
        <span class="pill">missing representative ${counts.missingRepresentative}</span>
      </div>
      ${
        focusItems.length
          ? focusItems
              .map(
                (item) => `
                  <div class="muted">
                    ${escapeHTML(stringifyValue(item.providerKey, "-"))}:
                    ${escapeHTML(renderProviderSmokeProviderReadinessLabel(item.readiness))}
                    / basic ${item.hasBasicSuccessSample ? "ready" : "pending"}
                    / upload ${item.hasUploadSuccessSample ? "ready" : "pending"}
                    / anomaly ${escapeHTML(stringifyValue(item.anomalyCompletedCount, "0"))}/${escapeHTML(stringifyValue(item.anomalyTargetCount, "0"))}
                    / representative ${escapeHTML(stringifyValue(item.representativeCompletedCount, "0"))}/${escapeHTML(stringifyValue(item.representativeTargetCount, "0"))}
                    / priority ${escapeHTML(stringifyValue(item.priorityAction, "complete"))}
                  </div>
                  <div class="muted">provider preferred sample: ${escapeHTML(stringifyValue(item.preferredSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredSamplePriority, "-"))}</div>
                  <div class="muted">provider preferred upload: ${escapeHTML(stringifyValue(item.preferredUploadSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredUploadPriority, "-"))}</div>
                  <div class="muted">provider preferred anomaly: ${escapeHTML(stringifyValue(item.preferredAnomalySampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredAnomalyPriority, "-"))}</div>
                  <div class="muted">provider preferred representative: ${escapeHTML(stringifyValue(item.preferredRepresentativeSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredRepresentativePriority, "-"))}</div>
                  ${Array.isArray(item.representativeMissing) && item.representativeMissing.length ? `<div class="muted">provider representative missing: ${escapeHTML(item.representativeMissing.join(", "))}</div>` : ""}
                  ${Array.isArray(item.representativeActions) && item.representativeActions.length ? `<div class="muted">provider representative actions: ${escapeHTML(item.representativeActions.join("；"))}</div>` : ""}
                  ${item.representativeAdvice ? `<div class="muted">provider representative advice: ${escapeHTML(item.representativeAdvice)}</div>` : ""}
                `,
              )
              .join("")
          : `<div class="muted">provider priority action: complete</div>`
      }
    </div>
  `;
}

function renderEvidenceUploadCheckpointSummary(report) {
  const summary = report?.summary && typeof report.summary === "object" ? report.summary : {};
  const resumeCount = Number(summary.uploadCheckpointResumeTaskCount || 0);
  const checkpointCount = Number(summary.uploadCheckpointTaskCount || 0);
  const paths = Array.isArray(summary.uploadCheckpointResumeSamplePaths)
    ? summary.uploadCheckpointResumeSamplePaths.filter(Boolean)
    : [];
  const readiness = renderUploadCheckpointReadiness(summary);
  const priorityAction = renderUploadCheckpointPriorityAction(summary);
  return `
    <div class="insight-card">
      <strong>Upload checkpoint 默认恢复验收</strong>
      <span>Checkpoint Resume Ready: ${escapeHTML(readiness)}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>大文件/长链路恢复摘要</strong>
        <code>uploadCheckpointResume</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">checkpoint ${checkpointCount}</span>
        <span class="pill">auto-resume ${resumeCount}</span>
        <span class="pill">readiness ${escapeHTML(readiness)}</span>
      </div>
      <div class="muted">priority action: ${escapeHTML(priorityAction)}</div>
      <div class="muted">sample context: provider ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleProvider, "-"))} / group ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleProtocol, "-"))} / task ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleTaskId, "-"))} / profile ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleProfileId, "-"))}</div>
      <div class="muted">resume detail: upload ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleUploadId, "-"))} / next part ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleNextPart, "0"))} / uploaded ${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSampleUploaded, "0"))}/${escapeHTML(stringifyValue(summary.uploadCheckpointResumeSamplePartCount, "0"))}</div>
      <div class="muted">sample path: ${escapeHTML(paths.length ? paths.join(" -> ") : "-")}</div>
      <div class="muted">recover priority action: ${escapeHTML(priorityAction)}</div>
    </div>
  `;
}

function providerRiskCalibrationCounts(items) {
  const counts = {
    total: 0,
    ready: 0,
    partial: 0,
    pending: 0,
  };
  for (const item of Array.isArray(items) ? items : []) {
    counts.total += 1;
    const readiness = String(item?.defaultRiskTemplate?.calibrationReadiness || "").trim().toLowerCase();
    if (readiness === "ready") {
      counts.ready += 1;
      continue;
    }
    if (readiness === "partial") {
      counts.partial += 1;
      continue;
    }
    counts.pending += 1;
  }
  return counts;
}

function renderEvidenceRiskCalibrationSummary(report) {
  const statusKeys = new Set(
    (Array.isArray(report?.statuses) ? report.statuses : [])
      .map((item) => String(item?.providerKey || "").trim())
      .filter(Boolean),
  );
  const items = (state.providers || []).filter((entry) => statusKeys.has(String(entry?.meta?.key || "").trim()));
  if (!items.length) {
    return `
      <div class="insight-card">
        <strong>Provider 默认风控校准</strong>
        <span>暂无 provider 默认模板校准数据，请先刷新 provider 列表。</span>
      </div>
    `;
  }
  const counts = providerRiskCalibrationCounts(items);
  const focusItems = items
    .filter((item) => String(item?.meta?.defaultRiskTemplate?.calibrationReadiness || "").toLowerCase() !== "ready")
    .slice(0, 6);
  return `
    <div class="insight-card">
      <strong>Provider 默认风控校准</strong>
      <span>Calibration Ready ${counts.ready} / ${counts.total}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>默认模板缺口速览</strong>
        <code>defaultRiskTemplate</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">ready ${counts.ready}</span>
        <span class="pill">partial ${counts.partial}</span>
        <span class="pill">pending ${counts.pending}</span>
      </div>
      ${
        focusItems.length
          ? focusItems
              .map((item) => {
                const template = item?.meta?.defaultRiskTemplate || {};
                return `
                  <div class="muted">
                    ${escapeHTML(stringifyValue(item?.meta?.key, "-"))}:
                    calibration readiness ${escapeHTML(stringifyValue(template.calibrationReadiness, "pending"))}
                    / recommended ${escapeHTML(stringifyValue(template.recommendedMode, "-"))}
                    / priority calibration ${escapeHTML(stringifyValue(template.calibrationPriorityAction, "-"))}
                  </div>
                  <div class="muted">
                    window source ${escapeHTML(renderAutoRetryWindowSource(template.autoRetryWindowSource))}
                    / window advice ${escapeHTML(stringifyValue(template.autoRetryWindowAdvice, "-"))}
                  </div>
                  <div class="muted">
                    calibration missing ${escapeHTML((template.calibrationMissing || []).join(", ") || "-")}
                    / calibration coverage ${escapeHTML(stringifyValue(template.calibrationCoverage, "-"))}
                    / covered ${escapeHTML(stringifyValue(template.calibrationCoveredCount, "0"))}/${escapeHTML(stringifyValue(template.calibrationTargetCount, "0"))}
                  </div>
                  <div class="muted">
                    covered fields ${escapeHTML((template.calibrationCoveredFields || []).join(", ") || "-")}
                  </div>
                `;
              })
              .join("")
          : `<div class="muted">priority calibration: complete</div>`
      }
    </div>
  `;
}

function renderEvidenceAutoRecoverSummary(report) {
  const summary = report?.summary && typeof report.summary === "object" ? report.summary : {};
  const fairnessPool = Array.isArray(summary.autoRecoverPool) ? summary.autoRecoverPool : [];
  const recoveryReadiness = renderAutoRecoverReadiness(summary);
  const fairnessReadiness = renderAutoRecoverFairnessReadiness(summary);
  const recoveryPriorityAction = renderAutoRecoverPriorityAction(summary);
  const fairnessPriorityAction = renderAutoRecoverFairnessPriorityAction(summary);
  const focusLanes = fairnessPool.slice(0, 4);
  return `
    <div class="insight-card">
      <strong>自动补传验收</strong>
      <span>Recover Ready ${escapeHTML(recoveryReadiness)} / Fairness Ready ${escapeHTML(fairnessReadiness)}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>自动补传恢复与公平性摘要</strong>
        <code>autoRecoverPool</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">tasks ${escapeHTML(stringifyValue(summary.autoRecoverTasks, "0"))}</span>
        <span class="pill">runnable ${escapeHTML(stringifyValue(summary.autoRecoverRunnableTasks, "0"))}</span>
        <span class="pill">recover ${escapeHTML(recoveryReadiness)}</span>
        <span class="pill">fairness ${escapeHTML(fairnessReadiness)}</span>
      </div>
      <div class="muted">recover priority action: ${escapeHTML(recoveryPriorityAction)}</div>
      <div class="muted">fairness missing: ${escapeHTML(renderAutoRecoverFairnessMissing(summary))}</div>
      <div class="muted">fairness priority action: ${escapeHTML(fairnessPriorityAction)}</div>
      <div class="muted">waiting: cooldown ${escapeHTML(stringifyValue(summary.autoRecoverWaitingCooldownTasks, "0"))} / window ${escapeHTML(stringifyValue(summary.autoRecoverWaitingRetryWindowTasks, "0"))} / auth ${escapeHTML(stringifyValue(summary.autoRecoverWaitingAuthRefreshTasks, "0"))} / local ${escapeHTML(stringifyValue(summary.autoRecoverWaitingLocalRestoreTasks, "0"))} / manual ${escapeHTML(stringifyValue(summary.autoRecoverWaitingManualTasks, "0"))}</div>
      ${
        focusLanes.length
          ? focusLanes
              .map(
                (item) => `
                  <div class="muted">
                    lane ${escapeHTML(stringifyValue(item.mode, "-"))}:
                    provider ${escapeHTML(stringifyValue(item.sampleProvider, "-"))}
                    / group ${escapeHTML(stringifyValue(item.sampleProtocolGroup, "-"))}
                    / profile ${escapeHTML(stringifyValue(item.sampleProfileId, "-"))}
                    / providers ${escapeHTML(stringifyValue(item.providerCount, "0"))}
                    / profiles ${escapeHTML(stringifyValue(item.profileCount, "0"))}
                  </div>
                `,
              )
              .join("")
          : `<div class="muted">当前没有自动补传候选池样本。</div>`
      }
    </div>
  `;
}

function renderEvidenceReport(report) {
  if (!report || typeof report !== "object") {
    return `<div class="directory-empty">暂无验收报告，请先刷新或保存一份报告。</div>`;
  }
  return `
    <div class="insight-card">
      <strong>报告标题</strong>
      <span>${escapeHTML(stringifyValue(report.title, "-"))}</span>
    </div>
    <div class="insight-card">
      <strong>生成时间</strong>
      <span>${escapeHTML(stringifyValue(report.generatedAt, "-"))}</span>
    </div>
    ${report.note ? `
      <div class="insight-card">
        <strong>报告备注</strong>
        <span>${escapeHTML(report.note)}</span>
      </div>
    ` : ""}
    ${renderEvidenceAutoRecoverSummary(report)}
    ${renderEvidenceUploadCheckpointSummary(report)}
    ${renderEvidenceProviderSmokeProviders(report)}
    ${renderEvidenceRiskCalibrationSummary(report)}
    <pre class="result-box">${escapeHTML(report.markdown || "")}</pre>
  `;
}

function wireRuntimePathFocus(scope, selector = null) {
  const wrap = selector ? $(selector) : scope === "task" ? $("#task-runtime") : $("#status-runtime-checkpoints");
  if (!wrap) {
    return;
  }
  wrap.querySelectorAll("[data-runtime-focus-path]").forEach((button) => {
    button.addEventListener("click", () => {
      const path = button.dataset.runtimeFocusPath || "";
      const focusScope = button.dataset.runtimeFocusScope || scope;
      focusRuntimeTreeByPath(focusScope, path, button.dataset.runtimeFocusKind || "roots");
    });
  });
}

function wireAutoRecoverSummary() {
  const wrap = $("#auto-recover-summary");
  if (!wrap) {
    return;
  }
  wrap.querySelectorAll("[data-auto-recover-focus-mode]").forEach((button) => {
    button.addEventListener("click", () => {
      const mode = button.dataset.autoRecoverFocusMode || "";
      applyAutoRecoverFilters({ mode });
      showFlash(`已按 ${mode} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-protocol-group]").forEach((button) => {
    button.addEventListener("click", () => {
      const protocolGroup = button.dataset.autoRecoverFocusProtocolGroup || "";
      applyAutoRecoverFilters({ protocolGroup });
      showFlash(`已按协议族 ${protocolGroup} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-state]").forEach((button) => {
    button.addEventListener("click", () => {
      const recoverState = button.dataset.autoRecoverFocusState || "";
      applyAutoRecoverFilters({ recoverState });
      showFlash(`已按执行状态 ${recoverState} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-retry-class]").forEach((button) => {
    button.addEventListener("click", () => {
      const retryClass = button.dataset.autoRecoverFocusRetryClass || "";
      applyAutoRecoverFilters({ retryClass });
      showFlash(`已按主失败类型 ${retryClass} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-primary-blocked-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const blockedAction = button.dataset.autoRecoverFocusPrimaryBlockedAction || "";
      applyAutoRecoverFilters({ blockedAction });
      showFlash(`已按主阻塞动作 ${blockedAction} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-blocked-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const blockedAction = button.dataset.autoRecoverFocusBlockedAction || "";
      applyAutoRecoverFilters({ blockedAction });
      showFlash(`已按 ${blockedAction} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-lane-mode]").forEach((button) => {
    button.addEventListener("click", () => {
      const mode = button.dataset.autoRecoverFocusLaneMode || "";
      const strategy = button.dataset.autoRecoverFocusLaneStrategy || "";
      const retryClass = button.dataset.autoRecoverFocusLaneRetryClass || "";
      const blockedAction = button.dataset.autoRecoverFocusLaneBlockedAction || "";
      applyAutoRecoverFilters({ mode, strategy, retryClass, blockedAction });
      showFlash(`已按 lane 收敛后台补传候选：${[mode, strategy, retryClass, blockedAction].filter(Boolean).join(" / ")}`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-apply-budgets]").forEach((button) => {
    button.addEventListener("click", () => {
      const limitPerMode = button.dataset.autoRecoverApplyModeBudget || "";
      const limitPerLane = button.dataset.autoRecoverApplyLaneBudget || "";
      const limitPerProtocolGroup = button.dataset.autoRecoverApplyGroupBudget || "";
      const limitPerProvider = button.dataset.autoRecoverApplyProviderBudget || "";
      const limitPerProfile = button.dataset.autoRecoverApplyProfileBudget || "";
      applyAutoRecoverFilters({
        limitPerMode,
        limitPerLane,
        limitPerProtocolGroup,
        limitPerProvider,
        limitPerProfile,
      });
      showFlash(`已采用建议预算：mode ${limitPerMode || "-"} / lane ${limitPerLane || "-"} / group ${limitPerProtocolGroup || "-"} / provider ${limitPerProvider || "-"} / profile ${limitPerProfile || "-"}`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-preview-lane-mode]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        // Preview lane 时只过滤 mode/retryClass/blockedAction，不把 mode/lane 预算写入过滤器，避免因预算限制导致无决策返回。
        applyAutoRecoverFilters({
          mode: button.dataset.autoRecoverPreviewLaneMode || "",
          strategy: button.dataset.autoRecoverPreviewLaneStrategy || "",
          retryClass: button.dataset.autoRecoverPreviewLaneRetryClass || "",
          blockedAction: button.dataset.autoRecoverPreviewLaneBlockedAction || "",
          limitPerProtocolGroup: button.dataset.autoRecoverPreviewGroupBudget || "",
          limitPerProvider: button.dataset.autoRecoverPreviewProviderBudget || "",
          limitPerProfile: button.dataset.autoRecoverPreviewProfileBudget || "",
        });
        await triggerAutoRecover({ dryRun: true });
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-mode]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ mode: button.dataset.autoRecoverRunMode || "" }, { render: false });
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-preview-protocol-group]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ protocolGroup: button.dataset.autoRecoverPreviewProtocolGroup || "" });
        await triggerAutoRecover({ dryRun: true });
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-protocol-group]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ protocolGroup: button.dataset.autoRecoverRunProtocolGroup || "" }, { render: false });
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-state]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ recoverState: button.dataset.autoRecoverRunState || "" }, { render: false });
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-retry-class]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ retryClass: button.dataset.autoRecoverRunRetryClass || "" }, { render: false });
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-primary-blocked-action]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ blockedAction: button.dataset.autoRecoverRunPrimaryBlockedAction || "" }, { render: false });
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-blocked-action]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters({ blockedAction: button.dataset.autoRecoverRunBlockedAction || "" }, { render: false });
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-lane-mode]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        applyAutoRecoverFilters(
          {
            mode: button.dataset.autoRecoverRunLaneMode || "",
            strategy: button.dataset.autoRecoverRunLaneStrategy || "",
            retryClass: button.dataset.autoRecoverRunLaneRetryClass || "",
            blockedAction: button.dataset.autoRecoverRunLaneBlockedAction || "",
          },
          { render: false },
        );
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-open-task]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        await openTaskByID(button.dataset.autoRecoverOpenTask);
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
}

function autoRecoverBlockedActionFromRecoverState(recoverState) {
  switch (String(recoverState || "").trim()) {
    case "waiting_cooldown":
      return "wait_for_cooldown";
    case "waiting_retry_window":
      return "wait_for_retry_window";
    case "waiting_auth_refresh":
      return "refresh_auth_profile";
    case "waiting_local_restore":
      return "restore_local_source_file";
    case "waiting_provider_session":
      return "manual_intervention_required";
    case "waiting_manual_confirmation":
      return "manual_confirmation_required";
    case "waiting_retry_limit":
      return "review_and_reset_retry_strategy";
    default:
      return "";
  }
}

function autoRecoverScopeFromPanel(panel) {
  return panel === "pending" ? "selected_pending_subset" : "selected_directory_subset";
}

function currentAutoRecoverRequest(dryRun = false) {
  const filters = state.autoRecoverFilters || {};
  const readFilterValue = (selector, key) => {
    const stateValue = String(filters[key] || "").trim();
    if (stateValue) {
      return stateValue;
    }
    return String($(selector)?.value || "").trim();
  };
  const selectedRecoverState = readFilterValue("#auto-recover-state", "recoverState");
  const limitText = readFilterValue("#auto-recover-limit", "limit");
  const limitPerModeText = readFilterValue("#auto-recover-limit-per-mode", "limitPerMode");
  const limitPerLaneText = readFilterValue("#auto-recover-limit-per-lane", "limitPerLane");
  const limitPerProtocolGroupText = readFilterValue("#auto-recover-limit-per-protocol-group", "limitPerProtocolGroup");
  const limitPerProviderText = readFilterValue("#auto-recover-limit-per-provider", "limitPerProvider");
  const limitPerProfileText = readFilterValue("#auto-recover-limit-per-profile", "limitPerProfile");
  const limit = limitText ? Number(limitText) : 0;
  const limitPerMode = limitPerModeText ? Number(limitPerModeText) : 0;
  const limitPerLane = limitPerLaneText ? Number(limitPerLaneText) : 0;
  const limitPerProtocolGroup = limitPerProtocolGroupText ? Number(limitPerProtocolGroupText) : 0;
  const limitPerProvider = limitPerProviderText ? Number(limitPerProviderText) : 0;
  const limitPerProfile = limitPerProfileText ? Number(limitPerProfileText) : 0;
  const selectedBlockedAction = readFilterValue("#auto-recover-blocked-action", "blockedAction");
  return {
    dryRun: Boolean(dryRun),
    mode: readFilterValue("#auto-recover-mode", "mode"),
    strategy: readFilterValue("#auto-recover-strategy", "strategy"),
    protocolGroup: readFilterValue("#auto-recover-protocol-group", "protocolGroup"),
    providerKey: readFilterValue("#auto-recover-provider", "providerKey"),
    profileId: readFilterValue("#auto-recover-profile", "profileId"),
    retryClass: readFilterValue("#auto-recover-retry-class", "retryClass"),
    blockedAction: selectedBlockedAction || autoRecoverBlockedActionFromRecoverState(selectedRecoverState),
    recoverState: selectedRecoverState,
    limit: Number.isFinite(limit) && limit > 0 ? limit : 0,
    limitPerMode: Number.isFinite(limitPerMode) && limitPerMode > 0 ? limitPerMode : 0,
    limitPerLane: Number.isFinite(limitPerLane) && limitPerLane > 0 ? limitPerLane : 0,
    limitPerProtocolGroup: Number.isFinite(limitPerProtocolGroup) && limitPerProtocolGroup > 0 ? limitPerProtocolGroup : 0,
    limitPerProvider: Number.isFinite(limitPerProvider) && limitPerProvider > 0 ? limitPerProvider : 0,
    limitPerProfile: Number.isFinite(limitPerProfile) && limitPerProfile > 0 ? limitPerProfile : 0,
  };
}
function currentAutoRecoverScopedRequest(overrides = {}, options = {}) {
  const payload = currentAutoRecoverRequest(Boolean(options?.dryRun));
  const merged = {
    ...payload,
    ...(overrides && typeof overrides === "object" ? overrides : {}),
  };
  if (!Array.isArray(merged.paths)) {
    delete merged.paths;
  }
  if (!String(merged.path || "").trim()) {
    delete merged.path;
  }
  if (!String(merged.scope || "").trim()) {
    delete merged.scope;
  }
  if (!String(merged.taskId || "").trim()) {
    delete merged.taskId;
  }
  if (!String(merged.providerKey || "").trim()) {
    delete merged.providerKey;
  }
  return merged;
}

async function triggerAutoRecover(options = {}) {
  const dryRun = Boolean(options?.dryRun);
  const payload = currentAutoRecoverRequest(dryRun);
  const selectedRecoverState = String($("#auto-recover-state")?.value || "").trim();
  state.autoRecoverFilters.mode = payload.mode;
  state.autoRecoverFilters.strategy = payload.strategy;
  state.autoRecoverFilters.protocolGroup = payload.protocolGroup;
  state.autoRecoverFilters.providerKey = payload.providerKey;
  state.autoRecoverFilters.profileId = payload.profileId;
  state.autoRecoverFilters.retryClass = payload.retryClass;
  state.autoRecoverFilters.blockedAction = String($("#auto-recover-blocked-action")?.value || "").trim();
  state.autoRecoverFilters.recoverState = selectedRecoverState;
  state.autoRecoverFilters.limit = payload.limit ? String(payload.limit) : "";
  state.autoRecoverFilters.limitPerMode = payload.limitPerMode ? String(payload.limitPerMode) : "";
  state.autoRecoverFilters.limitPerLane = payload.limitPerLane ? String(payload.limitPerLane) : "";
  state.autoRecoverFilters.limitPerProtocolGroup = payload.limitPerProtocolGroup ? String(payload.limitPerProtocolGroup) : "";
  state.autoRecoverFilters.limitPerProvider = payload.limitPerProvider ? String(payload.limitPerProvider) : "";
  state.autoRecoverFilters.limitPerProfile = payload.limitPerProfile ? String(payload.limitPerProfile) : "";
  const result = await api("/api/tasks/recover", {
    method: "POST",
    body: payload,
  });
  state.autoRecoverLastResult = result;
  if (dryRun) {
    $("#auto-recover-last-result-summary").textContent = renderAutoRecoverLastResultSummary();
    $("#auto-recover-last-result-detail").innerHTML = renderAutoRecoverLastResultDetail();
    wireAutoRecoverLastResultDetail();
    showFlash(
      `后台补传预演完成：matched ${stringifyValue(result.matchedCount, "0")} / 可放行 ${stringifyValue(result.recoveredCount, "0")} / providerBudget ${stringifyValue(result.skippedByProviderBudget, "0")} / profileBudget ${stringifyValue(result.skippedByProfileBudget, "0")}`,
    );
    return result;
  }
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(
    `后台补传已执行：matched ${stringifyValue(result.matchedCount, "0")} / recovered ${stringifyValue(result.recoveredCount, "0")} / limit ${stringifyValue(result.skippedByLimit, "0")} / modeBudget ${stringifyValue(result.skippedByModeBudget, "0")} / laneBudget ${stringifyValue(result.skippedByLaneBudget, "0")} / protocolGroupBudget ${stringifyValue(result.skippedByProtocolGroupBudget, "0")} / providerBudget ${stringifyValue(result.skippedByProviderBudget, "0")} / profileBudget ${stringifyValue(result.skippedByProfileBudget, "0")} / cooldownWait ${stringifyValue(result.skippedByCooldownWait, "0")} / retryWindowWait ${stringifyValue(result.skippedByRetryWindowWait, "0")} / blocked ${stringifyValue(result.skippedByBlockedReason, "0")}`,
  );
  return result;
}

async function autoRecoverTaskPath(scope, path, panel = "directory") {
  const context =
    scope === "task"
      ? {
          taskId: state.selectedTaskId,
          providerKey: currentSelectedTaskDetail()?.task?.targetProvider || "",
        }
      : currentStatusTaskContext();
  const normalizedPath = normalizeComparePath(path);
  if (!context || !context.taskId) {
    throw new Error(scope === "task" ? "请先选择任务" : "当前状态样本没有可用任务");
  }
  if (!normalizedPath) {
    throw new Error("缺少可恢复路径");
  }
  applyAutoRecoverFilters({}, { render: false });
  const result = await api("/api/tasks/recover", {
    method: "POST",
    body: currentAutoRecoverScopedRequest({
      taskId: context.taskId,
      providerKey: context.providerKey || "",
      path: normalizedPath,
      scope: autoRecoverScopeFromPanel(panel),
    }),
  });
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(
    `后台补传子树已执行：${normalizedPath} / ${selectionScopeLabel(autoRecoverScopeFromPanel(panel))} / matched ${stringifyValue(result.matchedCount, "0")} / recovered ${stringifyValue(result.recoveredCount, "0")} / modeBudget ${stringifyValue(result.skippedByModeBudget, "0")} / laneBudget ${stringifyValue(result.skippedByLaneBudget, "0")} / providerBudget ${stringifyValue(result.skippedByProviderBudget, "0")}`,
    result.recoveredCount <= 0,
  );
}

function resetAutoRecoverFilters() {
  state.autoRecoverFilters.mode = "";
  state.autoRecoverFilters.strategy = "";
  state.autoRecoverFilters.protocolGroup = "";
  state.autoRecoverFilters.providerKey = "";
  state.autoRecoverFilters.profileId = "";
  state.autoRecoverFilters.retryClass = "";
  state.autoRecoverFilters.blockedAction = "";
  state.autoRecoverFilters.recoverState = "";
  state.autoRecoverFilters.limit = "";
  state.autoRecoverFilters.limitPerMode = "";
  state.autoRecoverFilters.limitPerLane = "";
  state.autoRecoverFilters.limitPerProtocolGroup = "";
  state.autoRecoverFilters.limitPerProvider = "";
  state.autoRecoverFilters.limitPerProfile = "";
  setFilterControlValue("#auto-recover-mode", "");
  setFilterControlValue("#auto-recover-strategy", "");
  setFilterControlValue("#auto-recover-protocol-group", "");
  setFilterControlValue("#auto-recover-provider", "");
  setFilterControlValue("#auto-recover-profile", "");
  setFilterControlValue("#auto-recover-retry-class", "");
  setFilterControlValue("#auto-recover-blocked-action", "");
  setFilterControlValue("#auto-recover-state", "");
  setInputValueIfPresent("#auto-recover-limit", "");
  setInputValueIfPresent("#auto-recover-limit-per-mode", "");
  setInputValueIfPresent("#auto-recover-limit-per-lane", "");
  setInputValueIfPresent("#auto-recover-limit-per-protocol-group", "");
  setInputValueIfPresent("#auto-recover-limit-per-provider", "");
  setInputValueIfPresent("#auto-recover-limit-per-profile", "");
  renderStatus();
  showFlash("后台补传筛选已清空");
}

function focusRuntimeTreeByPath(scope, path, kind = "roots") {
  if (!path) {
    return;
  }
  const normalized = String(path).trim();
  if (!normalized) {
    return;
  }
  if (scope === "task") {
    if (kind === "scan") {
      state.treeFilters.taskDirectory.query = normalized;
      setFilterControlValue("#task-directory-filter-query", normalized);
      updateTaskTreePanels(currentSelectedTaskDetail());
      showFlash("已按扫描轨迹定位任务目录树");
      return;
    }
    state.treeFilters.taskDirectory.query = normalized;
    setFilterControlValue("#task-directory-filter-query", normalized);
    updateTaskTreePanels(currentSelectedTaskDetail());
    showFlash("已按选定根目录定位任务目录树");
    return;
  }
  if (kind === "scan") {
    state.treeFilters.statusDirectory.query = normalized;
    setFilterControlValue("#status-directory-filter-query", normalized);
    updateStatusTreePanels(recentRuntimePayload());
    showFlash("已按最近扫描轨迹定位状态目录树");
    return;
  }
  state.treeFilters.statusDirectory.query = normalized;
  setFilterControlValue("#status-directory-filter-query", normalized);
  updateStatusTreePanels(recentRuntimePayload());
  showFlash("已按选定根目录定位状态目录树");
}

function renderProviderSmokeRecords(items) {
  const result = filterProviderSmokeRecords(items, state.providerSmokeRecordFilters);
  if (!result.totalItems) {
    return `<div class="directory-empty">暂无真实 provider smoke 记录。</div>`;
  }
  return result.items
    .map(
      (item) => `
        <div class="directory-row tree-node ${item.id === state.selectedProviderSmokeId ? "active" : ""}">
          <div class="directory-row-header">
            <strong>${escapeHTML(item.title || item.providerKey || "-")}</strong>
            <code>${escapeHTML(item.id || "-")}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">${escapeHTML(stringifyValue(item.providerKey, "-"))}</span>
            <span class="pill">${escapeHTML(stringifyValue(item.protocolGroup, "-"))}</span>
            <span class="pill">${escapeHTML(stringifyValue(item.authMode, "-"))}</span>
            <span class="pill">${escapeHTML(stringifyValue(item.category, "-"))}</span>
            <span class="pill">${escapeHTML(stringifyValue(item.result, "-"))}</span>
          </div>
          ${item.note ? `<div class="muted">${escapeHTML(item.note)}</div>` : ""}
          <div class="muted">template: ${escapeHTML(stringifyValue(item.templateVersion, "-"))} / type ${escapeHTML(stringifyValue(item.sampleType, "-"))} / completeness ${escapeHTML(stringifyValue(item.evidenceCompleteness, "-"))}</div>
          <div class="muted">reuse: ${escapeHTML(stringifyValue(item.reuseAdvice, "-"))} / priority ${escapeHTML(stringifyValue(item.reusePriority, "-"))}</div>
          <div class="muted">regression entry: ${escapeHTML(stringifyValue(item.regressionEntry, "-"))}</div>
          <div class="muted">representative: ${escapeHTML((item.representativeLabels || []).join(" / ") || "-")} / auto recover focus: ${escapeHTML(stringifyValue(item.autoRecoverFocus, "-"))}</div>
          <div class="muted">operations: ${escapeHTML((item.operations || []).join(", ") || "-")}</div>
          <div class="muted">createdAt: <code>${escapeHTML(stringifyValue(item.createdAt, "-"))}</code></div>
          <div class="actions compact">
            <button type="button" class="ghost" data-provider-smoke-view="${escapeHTML(item.id || "")}">查看 Markdown</button>
            <button type="button" class="ghost" data-provider-smoke-download="${escapeHTML(item.id || "")}">下载 Markdown</button>
          </div>
        </div>
      `,
    )
    .join("");
}

function filterProviderSmokeRecords(items, filters = {}) {
  const records = Array.isArray(items) ? items : [];
  const query = String(filters.query || "").trim().toLowerCase();
  const protocolGroup = String(filters.protocolGroup || "").trim().toLowerCase();
  const result = String(filters.result || "").trim().toLowerCase();
  const filterActive = Boolean(query || protocolGroup || result);
  const visible = records.filter((item) => {
    const matchesQuery = includesFilterText(
      [item.title, item.providerKey, item.note, item.sampleType, item.evidenceCompleteness, item.reuseAdvice, item.reusePriority, item.regressionEntry, Array.isArray(item.representativeLabels) ? item.representativeLabels.join("/") : "", item.autoRecoverFocus, Array.isArray(item.operations) ? item.operations.join(",") : ""],
      query,
    );
    const matchesGroup = includesFilterText([item.protocolGroup], protocolGroup);
    const matchesResult = includesFilterText([item.result], result);
    return matchesQuery && matchesGroup && matchesResult;
  });
  return {
    items: visible,
    totalItems: records.length,
    visibleItems: visible.length,
    filterActive,
  };
}

function renderProviderSmokeRecordSummary(result) {
  if (!result.totalItems) {
    return "当前没有 smoke 记录。";
  }
  if (!result.filterActive) {
    return `显示全部 ${result.visibleItems} 条 smoke 记录。`;
  }
  return `当前显示 ${result.visibleItems} / ${result.totalItems} 条 smoke 记录。`;
}

function renderProviderSmokeSummary(items) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="directory-empty">暂无真实样本矩阵。</div>`;
  }
  return items
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(stringifyValue(item.protocolGroup, "-"))}</strong>
            <code>${escapeHTML(stringifyValue(item.sampleRecordId, "-"))}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">smokes ${stringifyValue(item.smokeCount, "0")}</span>
            <span class="pill">success ${stringifyValue(item.successCount, "0")}</span>
            <span class="pill">upload ${stringifyValue(item.uploadSuccessCount, "0")}</span>
            <span class="pill">failure ${stringifyValue(item.failureCount, "0")}</span>
            <span class="pill">providers ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">${item.hasRealSuccessSample ? "sampled" : "pending"}</span>
          </div>
          <div class="muted">providers: ${escapeHTML((item.providerKeys || []).join(", ") || "-")}</div>
          <div class="muted">sample: ${escapeHTML(stringifyValue(item.sampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleCategory, "-"))}</div>
          <div class="muted">preferred sample: ${escapeHTML(stringifyValue(item.preferredSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredSampleProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredSamplePriority, "-"))}</div>
          <div class="muted">preferred upload: ${escapeHTML(stringifyValue(item.preferredUploadSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredUploadProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredUploadPriority, "-"))}</div>
          <div class="muted">preferred anomaly: ${escapeHTML(stringifyValue(item.preferredAnomalySampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredAnomalyProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredAnomalyPriority, "-"))}</div>
          <div class="muted">preferred representative: ${escapeHTML(stringifyValue(item.preferredRepresentativeSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredRepresentativeProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredRepresentativePriority, "-"))}</div>
          <div class="muted">latestSmokeAt: <code>${escapeHTML(stringifyValue(item.latestSmokeAt, "-"))}</code></div>
        </div>
      `,
    )
    .join("");
}

function providerSmokeMatrixCounts(items) {
  const counts = {
    total: 0,
    accepted: 0,
    inProgress: 0,
    pending: 0,
  };
  for (const item of Array.isArray(items) ? items : []) {
    counts.total += 1;
    if (item?.accepted) {
      counts.accepted += 1;
      continue;
    }
    if (item?.acceptanceStatus === "in_progress") {
      counts.inProgress += 1;
      continue;
    }
    counts.pending += 1;
  }
  return counts;
}

function providerSmokeMatrixFilterLabel(filter) {
  switch (filter) {
    case "accepted":
      return "已验收";
    case "in_progress":
      return "进行中";
    case "pending":
      return "待补齐";
    default:
      return "全部";
  }
}

function filteredProviderSmokeMatrix(items) {
  const filter = state.providerSmokeMatrixFilter || "all";
  const matrix = Array.isArray(items) ? items : [];
  if (filter === "all") {
    return matrix;
  }
  return matrix.filter((item) => {
    if (filter === "accepted") {
      return Boolean(item?.accepted);
    }
    if (filter === "in_progress") {
      return item?.acceptanceStatus === "in_progress";
    }
    if (filter === "pending") {
      return item?.acceptanceStatus === "pending";
    }
    return true;
  });
}

function setProviderSmokeMatrixFilter(filter) {
  state.providerSmokeMatrixFilter = ["all", "accepted", "in_progress", "pending"].includes(filter) ? filter : "all";
  renderStatus();
}

function renderProviderSmokeMatrixControls(items) {
  const counts = providerSmokeMatrixCounts(items);
  const filters = [
    { key: "all", label: `全部 ${counts.total}` },
    { key: "accepted", label: `已验收 ${counts.accepted}` },
    { key: "in_progress", label: `进行中 ${counts.inProgress}` },
    { key: "pending", label: `待补齐 ${counts.pending}` },
  ];
  return `
    <div class="provider-smoke-matrix-toolbar">
      <div class="directory-row-header">
        <strong>验收矩阵视图</strong>
        <code>${escapeHTML(providerSmokeMatrixFilterLabel(state.providerSmokeMatrixFilter))}</code>
      </div>
      <div class="muted">可按验收状态快速筛选，也能直接跳到对应 smoke 样本或样本任务，方便继续补齐真实联调样本。</div>
      <div class="actions compact">
        ${filters
          .map(
            (item) => `
              <button
                type="button"
                class="ghost ${state.providerSmokeMatrixFilter === item.key ? "active" : ""}"
                data-provider-smoke-filter="${escapeHTML(item.key)}"
              >${escapeHTML(item.label)}</button>
            `,
          )
          .join("")}
      </div>
    </div>
  `;
}

function renderProviderSmokeChecklist(item) {
  return [
    `upload ${item?.hasUploadSuccessSample ? "ready" : "pending"}`,
    `coverage ${stringifyValue(item?.coverageRealSuccessTaskCount, "0")}/${stringifyValue(item?.coverageTaskCount, "0")}`,
    `anomaly ${stringifyValue(item?.anomalyCompletedCount, "0")}/${stringifyValue(item?.anomalyTargetCount, "0")}`,
    `representative ${stringifyValue(item?.representativeCompletedCount, "0")}/${stringifyValue(item?.representativeTargetCount, "0")}`,
  ].join(" / ");
}

function renderProviderSmokeGaps(item) {
  const gaps = [];
  if (!item?.hasUploadSuccessSample) {
    gaps.push("upload");
  }
  const anomalyMissing = Array.isArray(item?.anomalyMissing) ? item.anomalyMissing.filter(Boolean) : [];
  if (anomalyMissing.length) {
    gaps.push(`anomaly(${anomalyMissing.join(",")})`);
  }
  const representativeMissing = Array.isArray(item?.representativeMissing) ? item.representativeMissing.filter(Boolean) : [];
  if (representativeMissing.length) {
    gaps.push(`representative(${representativeMissing.join(",")})`);
  }
  return gaps.length ? gaps.join(" / ") : "complete";
}

function renderProviderSmokeReadiness(item) {
  const hasUpload = Boolean(item?.hasUploadSuccessSample);
  const coverageDone = Number(item?.coverageRealSuccessTaskCount || 0);
  const coverageTotal = Number(item?.coverageTaskCount || 0);
  const anomalyDone = Number(item?.anomalyCompletedCount || 0);
  const anomalyTotal = Number(item?.anomalyTargetCount || 0);
  const representativeDone = Number(item?.representativeCompletedCount || 0);
  const representativeTotal = Number(item?.representativeTargetCount || 0);
  if (hasUpload && coverageDone >= coverageTotal && anomalyDone >= anomalyTotal && representativeDone >= representativeTotal) {
    return "ready";
  }
  if (hasUpload || coverageDone > 0 || anomalyDone > 0 || representativeDone > 0) {
    return "partial";
  }
  return "pending";
}

function renderProviderSmokeNextAction(item) {
  const actions = [];
  if (!item?.hasUploadSuccessSample) {
    actions.push("补 1 条真实上传成功样本");
  }
  if (Number(item?.coverageRealSuccessTaskCount || 0) < Number(item?.coverageTaskCount || 0)) {
    actions.push("补 1 条真实任务覆盖样本");
  }
  if (Array.isArray(item?.anomalyActions) && item.anomalyActions.length) {
    actions.push(item.anomalyActions[0]);
  }
  if (Array.isArray(item?.representativeActions) && item.representativeActions.length) {
    actions.push(item.representativeActions[0]);
  }
  if (!actions.length) {
    return "complete";
  }
  return Array.from(new Set(actions.filter(Boolean))).join("；") || "complete";
}

function renderProviderSmokePriorityAction(item) {
  if (!item?.hasUploadSuccessSample) {
    return "补 1 条真实上传成功样本";
  }
  if (Number(item?.coverageRealSuccessTaskCount || 0) < Number(item?.coverageTaskCount || 0)) {
    return "补 1 条真实任务覆盖样本";
  }
  if (Array.isArray(item?.anomalyActions) && item.anomalyActions.length) {
    return item.anomalyActions[0];
  }
  if (Array.isArray(item?.representativeActions) && item.representativeActions.length) {
    return item.representativeActions[0];
  }
  return "complete";
}

function renderProviderSmokeMatrix(items) {
  const visibleItems = filteredProviderSmokeMatrix(items);
  if (!Array.isArray(visibleItems) || !visibleItems.length) {
    return `<div class="directory-empty">当前筛选 ${escapeHTML(providerSmokeMatrixFilterLabel(state.providerSmokeMatrixFilter))} 没有真实样本矩阵。</div>`;
  }
  return visibleItems
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(stringifyValue(item.protocolGroup, "-"))}</strong>
            <code>${escapeHTML(stringifyValue(item.sampleRecordId || item.coverageSampleTaskId, "-"))}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">smoke ${stringifyValue(item.smokeCount, "0")}</span>
            <span class="pill">upload-smoke ${stringifyValue(item.uploadSuccessCount, "0")} / ${item.hasUploadSuccessSample ? "ready" : "pending"}</span>
            <span class="pill">coverage ${stringifyValue(item.coverageRealSuccessTaskCount, "0")}/${stringifyValue(item.coverageTaskCount, "0")}</span>
            <span class="pill">${item.accepted ? "accepted" : item.acceptanceStatus || "pending"}</span>
          </div>
          <div class="muted">smoke sample: ${escapeHTML(stringifyValue(item.sampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleCategory, "-"))}</div>
          <div class="muted">coverage sample: ${escapeHTML(stringifyValue(item.coverageSampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.coverageSampleTaskState, "-"))} / ${escapeHTML(stringifyValue(item.coverageSampleCompletionKind, "-"))}</div>
          <div class="muted">readiness: ${escapeHTML(renderProviderSmokeReadiness(item))}</div>
          <div class="muted">checklist: ${escapeHTML(renderProviderSmokeChecklist(item))}</div>
          <div class="muted">gaps: ${escapeHTML(renderProviderSmokeGaps(item))}</div>
          <div class="muted">next action: ${escapeHTML(renderProviderSmokeNextAction(item))}</div>
          <div class="muted">priority action: ${escapeHTML(renderProviderSmokePriorityAction(item))}</div>
          <div class="muted">异常样本：auth ${item.hasAuthExpiredSample ? "ready" : "pending"} / rate ${item.hasRateLimitedSample ? "ready" : "pending"} / local ${item.hasLocalFileMissingSample ? "ready" : "pending"} / manual ${item.hasPendingManualSample ? "ready" : "pending"}</div>
          <div class="muted">代表样本：large ${item.hasLargeFileSample ? "ready" : "pending"} / nested ${item.hasNestedDirectorySample ? "ready" : "pending"} / retry ${item.hasRetryRecoverySample ? "ready" : "pending"}</div>
          ${Array.isArray(item.anomalyMissing) && item.anomalyMissing.length ? `<div class="muted">anomaly missing: ${escapeHTML(item.anomalyMissing.join(", "))}</div>` : ""}
          ${Array.isArray(item.anomalyActions) && item.anomalyActions.length ? `<div class="muted">anomaly actions: ${escapeHTML(item.anomalyActions.join("；"))}</div>` : ""}
          ${item.anomalyAdvice ? `<div class="muted">anomaly advice: ${escapeHTML(item.anomalyAdvice)}</div>` : ""}
          ${Array.isArray(item.representativeMissing) && item.representativeMissing.length ? `<div class="muted">representative missing: ${escapeHTML(item.representativeMissing.join(", "))}</div>` : ""}
          ${Array.isArray(item.representativeActions) && item.representativeActions.length ? `<div class="muted">representative actions: ${escapeHTML(item.representativeActions.join("；"))}</div>` : ""}
          ${item.representativeAdvice ? `<div class="muted">representative advice: ${escapeHTML(item.representativeAdvice)}</div>` : ""}
          ${Array.isArray(item.acceptanceMissing) && item.acceptanceMissing.length ? `<div class="muted">missing: ${escapeHTML(item.acceptanceMissing.join(", "))}</div>` : ""}
          ${Array.isArray(item.acceptanceActions) && item.acceptanceActions.length ? `<div class="muted">actions: ${escapeHTML(item.acceptanceActions.join("；"))}</div>` : ""}
          ${item.acceptanceAdvice ? `<div class="muted">advice: ${escapeHTML(item.acceptanceAdvice)}</div>` : ""}
          <div class="actions compact">
            ${item.sampleRecordId ? `<button type="button" class="ghost" data-provider-smoke-open-record="${escapeHTML(stringifyValue(item.sampleRecordId))}">打开 smoke 样本</button>` : ""}
            ${item.coverageSampleTaskId ? `<button type="button" class="ghost" data-provider-smoke-open-task="${escapeHTML(stringifyValue(item.coverageSampleTaskId))}">打开任务样本</button>` : ""}
            <button type="button" class="ghost" data-provider-smoke-draft="${escapeHTML(stringifyValue(item.protocolGroup))}">预填 smoke 表单</button>
            <button type="button" class="ghost" data-provider-smoke-draft-action="${escapeHTML(stringifyValue(item.protocolGroup))}">${escapeHTML(providerSmokeDraftActionLabel(item))}</button>
            <button type="button" class="ghost" data-provider-smoke-prefill-profile-risk="${escapeHTML(stringifyValue(item.protocolGroup))}">预填账号默认风控</button>
            <button type="button" class="ghost" data-provider-smoke-focus-group="${escapeHTML(stringifyValue(item.protocolGroup))}">只看该组记录</button>
            <button type="button" class="ghost" data-provider-smoke-filter-status="${escapeHTML(item.accepted ? "accepted" : item.acceptanceStatus || "pending")}">只看此类</button>
          </div>
          <div class="muted">latest smoke: <code>${escapeHTML(stringifyValue(item.latestSmokeAt, "-"))}</code> / coverage observed: <code>${escapeHTML(stringifyValue(item.coverageLastObservedAt, "-"))}</code></div>
        </div>
      `,
    )
    .join("");
}

function renderProviderSmokeMarkdown(markdown) {
  if (!markdown) {
    return `<div class="directory-empty">请选择一条 smoke 记录查看 Markdown。</div>`;
  }
  return `<pre class="result-box">${escapeHTML(markdown)}</pre>`;
}

function hydrateReportForm(report) {
  if (!report || typeof report !== "object") {
    return;
  }
  const titleInput = $("#report-title");
  const noteInput = $("#report-note");
  if (titleInput) {
    titleInput.value = report.title || "";
  }
  if (noteInput) {
    noteInput.value = report.note || "";
  }
}

function hydrateProviderSmokeForm(record) {
  if (!record || typeof record !== "object") {
    return;
  }
  const providerKeyInput = $("#provider-smoke-provider-key");
  const protocolGroupInput = $("#provider-smoke-protocol-group");
  const authModeInput = $("#provider-smoke-auth-mode");
  const categoryInput = $("#provider-smoke-category");
  const resultInput = $("#provider-smoke-result");
  const titleInput = $("#provider-smoke-title");
  const noteInput = $("#provider-smoke-note");
  const operationsInput = $("#provider-smoke-operations");
  if (providerKeyInput) providerKeyInput.value = record.providerKey || "";
  if (protocolGroupInput) protocolGroupInput.value = record.protocolGroup || "";
  if (authModeInput) authModeInput.value = record.authMode || "";
  if (categoryInput) categoryInput.value = record.category || "";
  if (resultInput) resultInput.value = record.result || "success";
  if (titleInput) titleInput.value = record.title || "";
  if (noteInput) noteInput.value = record.note || "";
  if (operationsInput) operationsInput.value = Array.isArray(record.operations) ? record.operations.join(",") : "";
}

function providerSmokeMissingReasons(item) {
  return Array.isArray(item?.acceptanceMissing) ? item.acceptanceMissing.filter(Boolean) : [];
}

function providerSmokeAnomalyMissingReasons(item) {
  return Array.isArray(item?.anomalyMissing) ? item.anomalyMissing.filter(Boolean) : [];
}

function providerSmokeDraftSpecFromAnomaly(item) {
  const missing = providerSmokeAnomalyMissingReasons(item);
  if (missing.includes("auth_expired_sample_missing")) {
    return {
      label: "补授权失效样本",
      category: "failed",
      result: "failure",
      operations: ["ValidateAuth", "Upload"],
      focusResult: "failure",
      note: "目标异常：auth_expired",
    };
  }
  if (missing.includes("rate_limited_sample_missing")) {
    return {
      label: "补限流样本",
      category: "partial_blocked",
      result: "failure",
      operations: ["ValidateAuth", "List", "Metadata", "Upload"],
      focusResult: "failure",
      note: "目标异常：rate_limited / risk_control",
    };
  }
  if (missing.includes("local_file_missing_sample_missing")) {
    return {
      label: "补本地文件缺失样本",
      category: "failed",
      result: "failure",
      operations: ["ValidateAuth", "Upload"],
      focusResult: "failure",
      note: "目标异常：local_file_missing",
    };
  }
  if (missing.includes("pending_manual_sample_missing")) {
    return {
      label: "补人工确认样本",
      category: "partial_blocked",
      result: "failure",
      operations: ["ValidateAuth", "List", "Metadata", "Upload"],
      focusResult: "failure",
      note: "目标异常：pending_manual / overwrite downgrade",
    };
  }
  return null;
}

function providerSmokeDraftSpecFromRepresentative(item) {
  const missing = Array.isArray(item?.representativeMissing) ? item.representativeMissing : [];
  if (missing.includes("large_file_sample_missing")) {
    return {
      label: "补大文件样本",
      category: "binary_upload_success",
      result: "success",
      operations: ["ValidateAuth", "List", "Metadata", "CreateDir", "Upload", "checkpoint"],
      focusResult: "success",
      note: "目标代表样本：large_file / multipart / 大文件上传恢复",
    };
  }
  if (missing.includes("nested_directory_sample_missing")) {
    return {
      label: "补多层目录样本",
      category: "browse_only",
      result: "success",
      operations: ["ValidateAuth", "List", "Metadata", "CreateDir", "leaf_first"],
      focusResult: "success",
      note: "目标代表样本：nested_directory / 多层目录 / subtree",
    };
  }
  if (missing.includes("retry_recovery_sample_missing")) {
    return {
      label: "补重试恢复样本",
      category: "binary_upload_success",
      result: "success",
      operations: ["ValidateAuth", "List", "Metadata", "Upload", "checkpoint"],
      focusResult: "success",
      note: "目标代表样本：retry_recovery / checkpoint / resume / 续传",
    };
  }
  return null;
}

function draftProviderSmokeFromMatrix(item) {
  const protocolGroup = String(item?.protocolGroup || "").trim();
  const providerKey = firstNonEmpty(
    String(item?.sampleProviderKey || "").trim(),
    String(item?.coverageSampleProviderKey || "").trim(),
    Array.isArray(item?.providerKeys) ? String(item.providerKeys[0] || "").trim() : "",
    Array.isArray(item?.coverageProviderKeys) ? String(item.coverageProviderKeys[0] || "").trim() : "",
  );
  const status = item?.accepted ? "accepted" : String(item?.acceptanceStatus || "pending");
  const missing = providerSmokeMissingReasons(item);
  const actions = Array.isArray(item?.acceptanceActions) ? item.acceptanceActions.filter(Boolean) : [];
  const operations = [];
  let category = "partial_blocked";
  if (missing.includes("upload_smoke_success_missing")) {
    category = "binary_upload_success";
    operations.push("ValidateAuth", "List", "Metadata", "CreateDir", "Upload");
  } else if (missing.includes("real_smoke_success_missing")) {
    category = "browse_only";
    operations.push("ValidateAuth", "List", "Metadata");
  } else if (missing.includes("task_coverage_missing")) {
    category = "binary_upload_success";
    operations.push("ValidateAuth", "List", "Metadata", "Upload");
  } else if (item?.hasUploadSuccessSample) {
    category = "binary_upload_success";
    operations.push("ValidateAuth", "List", "Metadata", "CreateDir", "Upload");
  } else if (item?.hasRealSuccessSample) {
    category = "browse_only";
    operations.push("ValidateAuth", "List", "Metadata");
  }
  const noteParts = [
    protocolGroup ? `协议组：${protocolGroup}` : "",
    providerKey ? `建议 provider：${providerKey}` : "",
    `验收状态：${status}`,
    missing.length ? `缺口：${missing.join(", ")}` : "",
    actions.length ? `动作：${actions.join("；")}` : "",
    item?.acceptanceAdvice ? `建议：${item.acceptanceAdvice}` : "",
  ].filter(Boolean);
  return {
    providerKey,
    protocolGroup,
    authMode: "",
    category,
    result: "success",
    title: protocolGroup ? `${protocolGroup} ${status} smoke` : "provider smoke",
    note: noteParts.join("；"),
    operations,
  };
}

function draftProviderSmokeActionFromMatrix(item) {
  const draft = draftProviderSmokeFromMatrix(item);
  const anomalySpec = providerSmokeDraftSpecFromAnomaly(item);
  const representativeSpec = providerSmokeDraftSpecFromRepresentative(item);
  const actions = Array.isArray(item?.acceptanceActions) ? item.acceptanceActions.filter(Boolean) : [];
  if (anomalySpec) {
    draft.category = anomalySpec.category;
    draft.result = anomalySpec.result;
    draft.operations = Array.isArray(anomalySpec.operations) ? anomalySpec.operations.slice() : draft.operations;
    draft.title = draft.protocolGroup ? `${draft.protocolGroup} ${anomalySpec.label}` : anomalySpec.label;
    draft.note = [draft.note, anomalySpec.note, item?.anomalyAdvice || ""].filter(Boolean).join("；");
    draft.focusResult = anomalySpec.focusResult || draft.result;
    return draft;
  }
  if (representativeSpec) {
    draft.category = representativeSpec.category;
    draft.result = representativeSpec.result;
    draft.operations = Array.isArray(representativeSpec.operations) ? representativeSpec.operations.slice() : draft.operations;
    draft.title = draft.protocolGroup ? `${draft.protocolGroup} ${representativeSpec.label}` : representativeSpec.label;
    draft.note = [draft.note, representativeSpec.note, item?.representativeAdvice || ""].filter(Boolean).join("；");
    draft.focusResult = representativeSpec.focusResult || draft.result;
    return draft;
  }
  if (actions.length) {
    draft.title = `${draft.title} action`;
    draft.note = [draft.note, `本次目标：${actions.join("；")}`].filter(Boolean).join("；");
  }
  if (!draft.operations.length) {
    draft.operations = ["ValidateAuth"];
  }
  draft.focusResult = draft.result;
  return draft;
}

function providerSmokeDraftActionLabel(item) {
  const anomalySpec = providerSmokeDraftSpecFromAnomaly(item);
  if (anomalySpec) {
    return anomalySpec.label;
  }
  const representativeSpec = providerSmokeDraftSpecFromRepresentative(item);
  if (representativeSpec) {
    return representativeSpec.label;
  }
  const missing = providerSmokeMissingReasons(item);
  if (missing.includes("upload_smoke_success_missing")) {
    return "补上传 smoke";
  }
  if (missing.includes("task_coverage_missing")) {
    return "补任务覆盖样本";
  }
  if (missing.includes("real_smoke_success_missing")) {
    return "补 smoke 成功样本";
  }
  return "按缺口预填动作";
}

function focusProviderSmokeRecordsByResult(result) {
  const normalized = String(result || "").trim().toLowerCase();
  state.providerSmokeRecordFilters.result = normalized;
  setFilterControlValue("#provider-smoke-records-filter-result", normalized);
  renderStatus();
  showFlash(normalized ? `已按 ${normalized} 收敛 smoke 记录` : "已清空 smoke 记录结果筛选");
}

function focusProviderSmokeMatrixByStatus(status) {
  setProviderSmokeMatrixFilter(status || "all");
  showFlash(`已按 ${status || "all"} 收敛验收矩阵`);
}

function openProviderSmokeRecordInMatrix(id) {
  const record = (state.providerSmokes || []).find((item) => item?.id === id) || null;
  if (record) {
    hydrateProviderSmokeForm(record);
    state.selectedProviderSmokeId = id;
    focusProviderSmokeRecordsByGroup(record.protocolGroup || "");
  }
  return loadProviderSmokeMarkdown(id).then(() => {
    showFlash("已打开 smoke 样本并回填表单");
  });
}

function draftProviderSmokeFromGap(item) {
  const draft = draftProviderSmokeActionFromMatrix(item);
  const label = providerSmokeDraftActionLabel(item);
  if (!String(draft.title || "").trim()) {
    if (draft.protocolGroup) {
      draft.title = `${draft.protocolGroup} ${label}`;
    } else {
      draft.title = label;
    }
  }
  return draft;
}

function draftProviderSmokeAndFocus(item, { fromGap = false } = {}) {
  const draft = fromGap ? draftProviderSmokeFromGap(item) : draftProviderSmokeFromMatrix(item);
  hydrateProviderSmokeForm(draft);
  focusProviderSmokeRecordsByGroup(draft.protocolGroup || "");
  if (String(draft.focusResult || draft.result || "").trim()) {
    focusProviderSmokeRecordsByResult(draft.focusResult || draft.result || "");
  }
  if (fromGap) {
    showFlash("已按验收缺口预填 smoke 动作");
    return;
  }
  showFlash("已按验收矩阵预填 smoke 表单");
}

function buildProviderSmokeDraftByProtocolGroup(protocolGroup, { fromGap = false } = {}) {
  const row = (state.providerSmokeMatrix || []).find((item) => item.protocolGroup === protocolGroup);
  if (!row) {
    showFlash("未找到对应协议组的验收矩阵项", true);
    return false;
  }
  draftProviderSmokeAndFocus(row, { fromGap });
  return true;
}

function draftProviderSmokeAndOpenTask(taskID) {
  if (!taskID) {
    showFlash("当前验收矩阵项没有可打开的任务样本", true);
    return;
  }
  openTaskByID(taskID).catch((error) => {
    showFlash(error.message, true);
  });
}

function profileRiskDefaultsFromSmokeMatrix(item) {
  const missing = providerSmokeMissingReasons(item);
  const profile = {};
  if (missing.includes("upload_smoke_success_missing")) {
    profile.requestIntervalMs = 1600;
    profile.directoryIntervalMs = 2200;
    profile.retryLimit = 3;
    profile.maxConcurrent = 1;
    profile.riskKeywords = ["upload_sample", "profile_calibrated"];
    return profile;
  }
  if (missing.includes("task_coverage_missing")) {
    profile.requestIntervalMs = 1400;
    profile.directoryIntervalMs = 2000;
    profile.retryLimit = 3;
    profile.riskKeywords = ["task_coverage", "profile_calibrated"];
    return profile;
  }
  if (missing.includes("real_smoke_success_missing")) {
    profile.requestIntervalMs = 1200;
    profile.directoryIntervalMs = 1800;
    profile.retryLimit = 2;
    profile.riskKeywords = ["browse_sample", "profile_calibrated"];
    return profile;
  }
  profile.requestIntervalMs = 1800;
  profile.directoryIntervalMs = 2600;
  profile.retryLimit = 3;
  profile.maxConcurrent = 1;
  profile.riskKeywords = ["accepted_group", "edge_sample"];
  return profile;
}

function prefillProfileRiskDefaultsFromMatrix(item) {
  const draft = draftProviderSmokeFromMatrix(item);
  const defaults = profileRiskDefaultsFromSmokeMatrix(item);
  const status = item?.accepted ? "accepted" : String(item?.acceptanceStatus || "pending").trim() || "pending";
  const sourceDisplay = draft.protocolGroup ? `Smoke Matrix ${draft.protocolGroup} (${status})` : `Smoke Matrix (${status})`;
  activateTab("providers");
  setSelectValueIfPresent("#profile-provider", draft.providerKey || "");
  setInputValueIfPresent("#profile-display-name", draft.protocolGroup ? `${draft.protocolGroup} 风控模板` : "真实样本风控模板");
  hydrateRiskProfileForm("profile-risk", defaults);
  try {
    const extra = parseJSONInput($("#profile-extra").value, {});
    const merged = mergeProfileRiskDefaultsIntoExtra(extra, defaults);
    merged.riskDefaultsSource = draft.protocolGroup ? `smoke_matrix:${draft.protocolGroup}:${status}` : `smoke_matrix:unknown:${status}`;
    merged.riskDefaultsSourceDisplay = sourceDisplay;
    $("#profile-extra").value = Object.keys(merged).length ? JSON.stringify(merged, null, 2) : "";
  } catch (error) {
    $("#profile-extra").value = JSON.stringify(
      {
        ...mergeProfileRiskDefaultsIntoExtra({}, defaults),
        riskDefaultsSource: draft.protocolGroup ? `smoke_matrix:${draft.protocolGroup}:${status}` : `smoke_matrix:unknown:${status}`,
        riskDefaultsSourceDisplay: sourceDisplay,
      },
      null,
      2,
    );
  }
  showFlash("已按真实样本预填账号默认风控");
}

function draftProviderSmokeFromAccepted(item) {
  const draft = draftProviderSmokeFromMatrix(item);
  draft.title = draft.protocolGroup ? `${draft.protocolGroup} 边界样本 smoke` : "边界样本 smoke";
  draft.note = [draft.note, "当前协议组已验收，建议继续补充边界场景样本。"].filter(Boolean).join("；");
  if (!draft.operations.length) {
    draft.operations = ["ValidateAuth", "List", "Metadata"];
  };
  return draft;
}

function focusProviderSmokeRecordsByGroup(protocolGroup) {
  const normalized = String(protocolGroup || "").trim();
  state.providerSmokeRecordFilters.protocolGroup = normalized;
  setFilterControlValue("#provider-smoke-records-filter-group", normalized);
  renderStatus();
  showFlash(normalized ? `已按 ${normalized} 收敛 smoke 记录` : "已清空 smoke 记录协议组筛选");
}

async function loadProviderSmokeMarkdown(id) {
  if (!id) {
    return;
  }
  const response = await fetch(`/api/provider-smokes/${encodeURIComponent(id)}?format=markdown`, {
    headers: {
      "Accept": "text/plain",
    },
  });
  if (!response.ok) {
    throw new Error(`加载 smoke Markdown 失败：${response.status}`);
  }
  state.selectedProviderSmokeId = id;
  state.selectedProviderSmokeMarkdown = await response.text();
  renderStatus();
}

async function bootstrapData() {
  await Promise.all([loadProviders(), loadProfiles(), loadTasks(), loadStatus()]);
}

function wireLogin() {
  $("#login-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    const password = $("#login-password").value.trim();
    try {
      const result = await api("/api/session/login", {
        method: "POST",
        body: { password },
      });
      localStorage.setItem("cloudpan_console_session", "ok");
      state.authenticated = true;
      syncSessionState();
      $("#login-result").textContent = formatJSON(result);
      showFlash("登录验证成功");
      await bootstrapData();
    } catch (error) {
      $("#login-result").textContent = error.message;
      showFlash(error.message, true);
    }
  });

  $("#logout-button").addEventListener("click", () => {
    localStorage.removeItem("cloudpan_console_session");
    state.authenticated = false;
    syncSessionState();
    showFlash("本地控制台会话已清理");
  });
}

function wireProfiles() {
  $("#profile-provider").addEventListener("change", syncAuthModes);
  $("#plan-source-provider").addEventListener("change", syncSourceProfiles);
  $("#plan-target-profile").addEventListener("change", syncTargetProfileInsight);
  $("#plan-target-provider").addEventListener("change", async () => {
    syncTargetProfiles();
    const providerKey = $("#plan-target-provider").value;
    if (!providerKey) {
      return;
    }
    try {
      await loadProviderCapabilityDetail(providerKey);
    } catch (error) {
      showFlash(`加载 provider 能力详情失败：${error.message}`, true);
    }
  });
  $("#plan-execution-mode").addEventListener("change", () => {
    syncExecutionModeHint();
    updateExecutionRecommendationAction(state.preview?.metadata || {});
  });
  $("#providers-grid").addEventListener("click", async (event) => {
    const button = event.target.closest("[data-provider-detail-open]");
    if (!button) {
      return;
    }
    try {
      await loadProviderCapabilityDetail(button.dataset.providerDetailOpen || "");
      showFlash(`已加载 ${button.dataset.providerDetailOpen} provider 能力详情`);
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#refresh-providers").addEventListener("click", async () => {
    try {
      await loadProviders();
      showFlash("Provider 列表已刷新");
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#refresh-profiles").addEventListener("click", async () => {
    try {
      await loadProfiles();
      showFlash("授权档案已刷新");
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#sync-profile-risk-defaults").addEventListener("click", () => {
    try {
      const extra = parseJSONInput($("#profile-extra").value, {});
      const merged = mergeProfileRiskDefaultsIntoExtra(extra, collectRiskProfileFromForm("profile-risk"));
      $("#profile-extra").value = Object.keys(merged).length ? JSON.stringify(merged, null, 2) : "";
      showFlash("账号默认风控已同步到 Extra JSON");
    } catch (error) {
      showFlash(`Extra JSON 无法解析：${error.message}`, true);
    }
  });

  $("#clear-profile-risk-defaults").addEventListener("click", () => {
    try {
      const extra = parseJSONInput($("#profile-extra").value, {});
      const merged = mergeProfileRiskDefaultsIntoExtra(extra, null);
      $("#profile-extra").value = Object.keys(merged).length ? JSON.stringify(merged, null, 2) : "";
      hydrateRiskProfileForm("profile-risk", null);
      showFlash("账号默认风控已清空");
    } catch (error) {
      showFlash(`Extra JSON 无法解析：${error.message}`, true);
    }
  });

  $("#profile-cancel-edit").addEventListener("click", () => {
    resetProfileForm();
    focusProfile("");
    showFlash("已退出授权档案编辑");
  });

  $("#profile-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const profileID = $("#profile-id").value.trim();
      const extra = parseJSONInput($("#profile-extra").value, {});
      const payload = {
        providerKey: $("#profile-provider").value,
        authMode: $("#profile-auth-mode").value,
        displayName: $("#profile-display-name").value.trim(),
        token: $("#profile-token").value.trim(),
        cookie: $("#profile-cookie").value.trim(),
        extra: mergeProfileRiskDefaultsIntoExtra(extra, collectRiskProfileFromForm("profile-risk")),
      };
      await api(profileID ? `/api/auth/profiles/${profileID}` : "/api/auth/profiles", {
        method: profileID ? "PATCH" : "POST",
        body: payload,
      });
      resetProfileForm();
      await loadProfiles();
      showFlash(profileID ? "授权档案已更新" : "授权档案已创建");
    } catch (error) {
      showFlash(error.message, true);
    }
  });
}

function buildPlanPayload() {
  const riskOverride = parseJSONInput($("#plan-risk-override").value, collectRiskOverrideFromForm());
  return {
    sourceProvider: $("#plan-source-provider").value,
    sourceProfileId: $("#plan-source-profile").value,
    targetProvider: $("#plan-target-provider").value,
    targetProfileId: $("#plan-target-profile").value,
    thresholdMB: Number($("#plan-threshold").value || "0"),
    riskMode: $("#plan-risk-mode").value,
    riskOverride,
    executionMode: $("#plan-execution-mode").value,
    sourceDeletePolicy: $("#plan-source-delete-policy").value || "record_only",
    conflictPolicy: $("#plan-conflict-policy").value,
    selectedRoots: parseJSONInput($("#plan-selected-roots").value, []),
    entries: parseJSONInput($("#plan-entries").value, []),
  };
}

function payloadHasOnlyDeletedEntries(payload) {
  const entries = Array.isArray(payload?.entries) ? payload.entries : [];
  return entries.length > 0 && entries.every((entry) => Boolean(entry?.deleted));
}

function wirePlanner() {
  ["#risk-request-interval", "#risk-directory-interval", "#risk-page-size", "#risk-cooldown-seconds", "#risk-retry-limit", "#risk-max-concurrent", "#risk-auto-retry-start-hour", "#risk-auto-retry-end-hour", "#risk-keywords"].forEach(
    (selector) => {
      $(selector).addEventListener("input", syncRiskOverrideJSON);
    },
  );
  $("#sync-risk-override").addEventListener("click", () => {
    syncRiskOverrideJSON();
    showFlash("风控覆盖已同步到 JSON");
  });
  $("#clear-risk-override").addEventListener("click", () => {
    hydrateRiskOverrideForm(null);
    $("#plan-risk-override").value = "";
    showFlash("风控覆盖已清空，将使用默认档位和 provider 校准");
  });
  $("#plan-risk-override").addEventListener("blur", () => {
    try {
      hydrateRiskOverrideForm(parseJSONInput($("#plan-risk-override").value, null));
    } catch (error) {
      showFlash(`Risk Override JSON 无法解析：${error.message}`, true);
    }
  });
  $("#apply-recommended-execution").addEventListener("click", () => {
    const recommended = state.preview?.metadata?.recommendedExecutionMode;
    if (!recommended) {
      showFlash("请先生成计划预览", true);
      return;
    }
    setSelectValueIfPresent("#plan-execution-mode", recommended);
    showFlash(`已采用推荐执行模式：${recommended}`);
  });
  $("#apply-recommended-risk").addEventListener("click", () => {
    const recommended = state.preview?.metadata?.recommendedRiskMode;
    if (!recommended) {
      showFlash("请先生成计划预览", true);
      return;
    }
    setSelectValueIfPresent("#plan-risk-mode", recommended);
    showFlash(`已采用推荐风控档位：${recommended}`);
  });

  $("#preview-plan").addEventListener("click", async () => {
    try {
      const payload = buildPlanPayload();
      state.preview = await api("/api/plans/preview", {
        method: "POST",
        body: {
          sourceProvider: payload.sourceProvider,
          targetProvider: payload.targetProvider,
          thresholdMB: payload.thresholdMB,
          riskMode: payload.riskMode,
          riskOverride: payload.riskOverride,
          executionMode: payload.executionMode,
          sourceDeletePolicy: payload.sourceDeletePolicy,
          conflictPolicy: payload.conflictPolicy,
          selectedRoots: payload.selectedRoots,
          entries: payload.entries,
        },
      });
      renderPreview();
      showFlash("计划预览已生成");
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#plan-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const payload = buildPlanPayload();
      if (payloadHasOnlyDeletedEntries(payload)) {
        showFlash("当前只有删除记录，没有可执行条目；请先恢复源文件并重新预览", true);
        return;
      }
      const detail = await api("/api/tasks", {
        method: "POST",
        body: payload,
      });
      state.selectedTaskId = detail.task.id;
      showFlash("任务已创建");
      await loadTasks();
      document.querySelector('[data-view="tasks"]').click();
    } catch (error) {
      showFlash(error.message, true);
    }
  });
}

async function runTaskAction(action, body = undefined) {
  if (!state.selectedTaskId) {
    showFlash("请先选择任务", true);
    return false;
  }
  return runTaskActionForTask(state.selectedTaskId, action, body);
}

async function runTaskActionForTask(taskId, action, body = undefined) {
  const normalizedTaskId = String(taskId || "").trim();
  if (!normalizedTaskId) {
    showFlash("缺少任务标识", true);
    return false;
  }
  if (state.taskActionPending) {
    showFlash("任务操作执行中，请稍候", true);
    return false;
  }
  state.taskActionPending = true;
  syncTaskActionButtons();
  try {
    await api(`/api/tasks/${normalizedTaskId}/${action}`, { method: "POST", body });
    showFlash(`任务 ${action} 已执行`);
    await Promise.all([loadTasks(), loadStatus()]);
    return true;
  } catch (error) {
    showFlash(error.message, true);
    return false;
  } finally {
    state.taskActionPending = false;
    syncTaskActionButtons();
  }
}

function visibleSelectionPaths(scope, source) {
  if (source === "directory_tree" || source === "selected_directory_subset") {
    return visibleTreePaths(scope, "directory");
  }
  if (source === "pending_tree" || source === "selected_pending_subset") {
    return visibleTreePaths(scope, "pending");
  }
  return visibleRetryPaths(scope);
}

function recoverScopeFromSource(source) {
  if (source === "directory_tree" || source === "selected_directory_subset") {
    return "selected_directory_subset";
  }
  return source === "pending_tree" || source === "selected_pending_subset" ? "selected_pending_subset" : "selected_retry_subset";
}
function selectionSourceLabel(source) {
  if (source === "directory_tree" || source === "selected_directory_subset") {
    return "目录子集";
  }
  if (source === "pending_tree" || source === "selected_pending_subset") {
    return "待补传子集";
  }
  return "重试队列子集";
}

function selectionScopeLabel(scope) {
  const normalized = String(scope || "").trim();
  if (normalized === "selected_directory_subset") {
    return "selected_directory_subset";
  }
  if (normalized === "selected_pending_subset") {
    return "selected_pending_subset";
  }
  return normalized || "selected_retry_subset";
}

function taskContextByScope(scope) {
  return scope === "task"
    ? {
        taskId: state.selectedTaskId,
        providerKey: currentSelectedTaskDetail()?.task?.targetProvider || "",
        detail: currentSelectedTaskDetail(),
      }
    : currentStatusTaskContext();
}

function prefillWizardFromVisibleSelection(scope, source) {
  const context = taskContextByScope(scope);
  if (!context?.detail) {
    showFlash(scope === "task" ? "请先选择任务" : "当前状态样本没有可用任务", true);
    return;
  }
  const paths = visibleSelectionPaths(scope, source);
  if (!paths.length) {
    showFlash("当前筛选结果里没有可用于重建向导的路径", true);
    return;
  }
  prefillWizardFromTaskPaths(context.detail, paths, `${selectionSourceLabel(source)} / ${selectionScopeLabel(recoverScopeFromSource(source))}`);
}

async function retryVisibleSelection(scope, source) {
  const context = taskContextByScope(scope);
  if (!context?.taskId) {
    showFlash(scope === "task" ? "请先选择任务" : "当前状态样本没有可用任务", true);
    return;
  }
  const paths = visibleSelectionPaths(scope, source);
  if (!paths.length) {
    showFlash("当前筛选结果里没有可重试的路径", true);
    return;
  }
  const ok = await runTaskActionForTask(context.taskId, "retry", { paths, scope: recoverScopeFromSource(source) });
  if (ok) {
    showFlash(`已按${selectionSourceLabel(source)} (${selectionScopeLabel(recoverScopeFromSource(source))}) 重建 ${paths.length} 条路径的 retry 范围`);
  }
}

async function autoRecoverVisibleSelection(scope, source) {
  const context = taskContextByScope(scope);
  if (!context || !context.taskId) {
    showFlash(scope === "task" ? "请先选择任务" : "当前状态样本没有可用任务", true);
    return;
  }
  const paths = visibleSelectionPaths(scope, source);
  if (!paths.length) {
    showFlash("当前筛选结果里没有可后台补传的路径", true);
    return;
  }
  applyAutoRecoverFilters({}, { render: false });
  const result = await api("/api/tasks/recover", {
    method: "POST",
    body: currentAutoRecoverScopedRequest({
      taskId: context.taskId,
      providerKey: context.providerKey || "",
      paths,
      scope: recoverScopeFromSource(source),
    }),
  });
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(
    `后台补传筛选已执行：${selectionSourceLabel(source)} / ${selectionScopeLabel(recoverScopeFromSource(source))} / paths ${paths.length} / matched ${stringifyValue(result.matchedCount, "0")} / recovered ${stringifyValue(result.recoveredCount, "0")} / modeBudget ${stringifyValue(result.skippedByModeBudget, "0")} / laneBudget ${stringifyValue(result.skippedByLaneBudget, "0")} / providerBudget ${stringifyValue(result.skippedByProviderBudget, "0")}`,
    result.recoveredCount <= 0,
  );
}

async function retryTaskPath(scope, path) {
  const normalizedPath = normalizeComparePath(path);
  const taskId = scope === "task" ? state.selectedTaskId : currentStatusTaskContext()?.taskId;
  if (!taskId) {
    showFlash(scope === "task" ? "请先选择任务" : "当前状态样本没有可用任务", true);
    return false;
  }
  if (!normalizedPath) {
    showFlash("当前路径无效，无法重试", true);
    return false;
  }
  const ok = await runTaskActionForTask(taskId, "retry", {
    paths: [normalizedPath],
    scope: "selected_pending_subset",
  });
  if (ok) {
    showFlash(`已按 ${normalizedPath} (${selectionScopeLabel("selected_pending_subset")}) 重建 retry 范围`);
  }
  return ok;
}

function wireTasks() {
  $("#refresh-tasks").addEventListener("click", async () => {
    try {
      await loadTasks();
      showFlash("任务列表已刷新");
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#task-run").addEventListener("click", () => runTaskAction("run"));
  $("#task-pause").addEventListener("click", () => runTaskAction("pause"));
  $("#task-resume").addEventListener("click", () => runTaskAction("resume"));
  $("#task-retry").addEventListener("click", () => runTaskAction("retry"));
  $("#task-retry-visible-queue").addEventListener("click", () => retryVisibleSelection("task", "selected_retry_subset"));
  $("#task-directory-prefill-visible").addEventListener("click", () => prefillWizardFromVisibleSelection("task", "directory_tree"));
  $("#task-retry-visible-directory").addEventListener("click", () => retryVisibleSelection("task", "selected_directory_subset"));
  $("#task-pending-prefill-visible").addEventListener("click", () => prefillWizardFromVisibleSelection("task", "pending_tree"));
  $("#task-retry-visible-pending").addEventListener("click", () => retryVisibleSelection("task", "selected_pending_subset"));
  $("#task-auto-recover-visible-queue").addEventListener("click", () => autoRecoverVisibleSelection("task", "retry_queue"));
  $("#task-auto-recover-visible-pending").addEventListener("click", () => autoRecoverVisibleSelection("task", "pending_tree"));
}

function wireStatus() {
  $("#refresh-status").addEventListener("click", async () => {
    try {
      await loadStatus();
      showFlash("状态矩阵已刷新");
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#auto-recover-mode").addEventListener("change", () => {
    state.autoRecoverFilters.mode = $("#auto-recover-mode").value;
    renderStatus();
  });
  $("#auto-recover-strategy").addEventListener("change", () => {
    state.autoRecoverFilters.strategy = $("#auto-recover-strategy").value;
    renderStatus();
  });
  $("#auto-recover-protocol-group").addEventListener("change", () => {
    state.autoRecoverFilters.protocolGroup = $("#auto-recover-protocol-group").value;
    renderStatus();
  });
  $("#auto-recover-provider").addEventListener("change", () => {
    state.autoRecoverFilters.providerKey = $("#auto-recover-provider").value;
    renderStatus();
  });
  $("#auto-recover-profile").addEventListener("change", () => {
    state.autoRecoverFilters.profileId = $("#auto-recover-profile").value;
    renderStatus();
  });
  $("#auto-recover-retry-class").addEventListener("change", () => {
    state.autoRecoverFilters.retryClass = $("#auto-recover-retry-class").value;
    renderStatus();
  });
  $("#auto-recover-blocked-action").addEventListener("change", () => {
    state.autoRecoverFilters.blockedAction = $("#auto-recover-blocked-action").value;
    renderStatus();
  });
  $("#auto-recover-state").addEventListener("change", () => {
    state.autoRecoverFilters.recoverState = $("#auto-recover-state").value;
    renderStatus();
  });
  $("#auto-recover-limit").addEventListener("input", () => {
    state.autoRecoverFilters.limit = $("#auto-recover-limit").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
    $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(state.evidence?.autoRetryPolicy || {});
  });
  $("#auto-recover-limit-per-mode").addEventListener("input", () => {
    state.autoRecoverFilters.limitPerMode = $("#auto-recover-limit-per-mode").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
    $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(state.evidence?.autoRetryPolicy || {});
  });
  $("#auto-recover-limit-per-lane").addEventListener("input", () => {
    state.autoRecoverFilters.limitPerLane = $("#auto-recover-limit-per-lane").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
    $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(state.evidence?.autoRetryPolicy || {});
  });
  $("#auto-recover-limit-per-protocol-group").addEventListener("input", () => {
    state.autoRecoverFilters.limitPerProtocolGroup = $("#auto-recover-limit-per-protocol-group").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
    $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(state.evidence?.autoRetryPolicy || {});
  });
  $("#auto-recover-limit-per-provider").addEventListener("input", () => {
    state.autoRecoverFilters.limitPerProvider = $("#auto-recover-limit-per-provider").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
    $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(state.evidence?.autoRetryPolicy || {});
  });
  $("#auto-recover-limit-per-profile").addEventListener("input", () => {
    state.autoRecoverFilters.limitPerProfile = $("#auto-recover-limit-per-profile").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
    $("#auto-recover-budget-summary").textContent = renderAutoRecoverBudgetSummary(state.evidence?.autoRetryPolicy || {});
  });
  $("#auto-recover-preview").addEventListener("click", async () => {
    try {
      await triggerAutoRecover({ dryRun: true });
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#auto-recover-run").addEventListener("click", async () => {
    try {
      await triggerAutoRecover();
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#auto-recover-reset").addEventListener("click", () => {
    resetAutoRecoverFilters();
  });
  $("#status-retry-visible-queue").addEventListener("click", () => retryVisibleSelection("status", "selected_retry_subset"));
  $("#status-directory-prefill-visible").addEventListener("click", () => prefillWizardFromVisibleSelection("status", "directory_tree"));
  $("#status-retry-visible-directory").addEventListener("click", () => retryVisibleSelection("status", "selected_directory_subset"));
  $("#status-auto-recover-visible-queue").addEventListener("click", () => autoRecoverVisibleSelection("status", "retry_queue"));
  $("#status-pending-prefill-visible").addEventListener("click", () => prefillWizardFromVisibleSelection("status", "pending_tree"));
  $("#status-retry-visible-pending").addEventListener("click", () => retryVisibleSelection("status", "selected_pending_subset"));
  $("#status-auto-recover-visible-pending").addEventListener("click", () => autoRecoverVisibleSelection("status", "pending_tree"));
  $("#save-provider-smoke").addEventListener("click", async () => {
    try {
      const payload = {
        providerKey: $("#provider-smoke-provider-key").value.trim(),
        protocolGroup: $("#provider-smoke-protocol-group").value.trim(),
        authMode: $("#provider-smoke-auth-mode").value.trim(),
        category: $("#provider-smoke-category").value.trim(),
        result: $("#provider-smoke-result").value,
        title: $("#provider-smoke-title").value.trim(),
        note: $("#provider-smoke-note").value.trim(),
        operations: $("#provider-smoke-operations")
          .value.split(",")
          .map((item) => item.trim())
          .filter(Boolean),
        environment: {
          os: navigator.platform || "",
          userAgent: navigator.userAgent || "",
        },
      };
      const record = await api("/api/provider-smokes", {
        method: "POST",
        body: payload,
      });
      hydrateProviderSmokeForm(record);
      state.selectedProviderSmokeId = record.id || "";
      state.selectedProviderSmokeMarkdown = record.markdown || "";
      showFlash("Provider smoke 记录已保存");
      await loadStatus();
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#refresh-report").addEventListener("click", async () => {
    try {
      await loadStatus();
      showFlash("验收报告已刷新");
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#save-report").addEventListener("click", async () => {
    try {
      const payload = {
        title: $("#report-title").value.trim(),
        note: $("#report-note").value.trim(),
      };
      const record = await api("/api/evidence/report", {
        method: "POST",
        body: payload,
      });
      state.selectedReportId = record.id || "";
      showFlash("验收报告已保存");
      await loadStatus();
      if (state.selectedReportId) {
        $("#report-history").querySelector(`[data-report-view="${state.selectedReportId}"]`)?.focus?.();
      }
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#download-report").addEventListener("click", async () => {
    try {
      const report = selectedEvidenceReport();
      if (!report || !report.markdown) {
        showFlash("暂无可下载的验收报告", true);
        return;
      }
      const blob = new Blob([report.markdown], { type: "text/plain;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = `${String(report.title || "cloudpan-sync-report").trim().replace(/\s+/g, "-")}.md`;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      showFlash("验收报告已下载");
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#report-history").addEventListener("click", async (event) => {
    const viewButton = event.target.closest("[data-report-view]");
    if (viewButton) {
      state.selectedReportId = viewButton.dataset.reportView || "";
      renderStatus();
      showFlash("已切换验收报告");
      return;
    }
    const downloadButton = event.target.closest("[data-report-download]");
    if (downloadButton) {
      state.selectedReportId = downloadButton.dataset.reportDownload || "";
      $("#download-report").click();
    }
  });
  $("#provider-smoke-records").addEventListener("click", async (event) => {
    const viewButton = event.target.closest("[data-provider-smoke-view]");
    if (viewButton) {
      try {
        await loadProviderSmokeMarkdown(viewButton.dataset.providerSmokeView || "");
        showFlash("已切换 smoke Markdown");
      } catch (error) {
        showFlash(error.message, true);
      }
      return;
    }
    const downloadButton = event.target.closest("[data-provider-smoke-download]");
    if (downloadButton) {
      try {
        const id = downloadButton.dataset.providerSmokeDownload || "";
        if (!id) {
          return;
        }
        const response = await fetch(`/api/provider-smokes/${encodeURIComponent(id)}?format=markdown`, {
          headers: { Accept: "text/plain" },
        });
        if (!response.ok) {
          throw new Error(`下载 smoke Markdown 失败：${response.status}`);
        }
        const markdown = await response.text();
        const record = state.providerSmokes.find((item) => item.id === id);
        const title = String(record?.title || record?.providerKey || "provider-smoke").trim().replace(/\s+/g, "-");
        const blob = new Blob([markdown], { type: "text/plain;charset=utf-8" });
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement("a");
        anchor.href = url;
        anchor.download = `${title}.md`;
        document.body.appendChild(anchor);
        anchor.click();
        anchor.remove();
        URL.revokeObjectURL(url);
        showFlash("smoke Markdown 已下载");
      } catch (error) {
        showFlash(error.message, true);
      }
    }
  });
  $("#provider-smoke-summary").addEventListener("click", (event) => {
    const filterButton = event.target.closest("[data-provider-smoke-filter]");
    if (!filterButton) {
      return;
    }
    setProviderSmokeMatrixFilter(filterButton.dataset.providerSmokeFilter || "all");
  });
  $("#provider-smoke-matrix").addEventListener("click", async (event) => {
    const openRecordButton = event.target.closest("[data-provider-smoke-open-record]");
    if (openRecordButton) {
      try {
        await openProviderSmokeRecordInMatrix(openRecordButton.dataset.providerSmokeOpenRecord || "");
      } catch (error) {
        showFlash(error.message, true);
      }
      return;
    }
    const openTaskButton = event.target.closest("[data-provider-smoke-open-task]");
    if (openTaskButton) {
      try {
        await openTaskByID(openTaskButton.dataset.providerSmokeOpenTask || "");
      } catch (error) {
        showFlash(error.message, true);
      }
      return;
    }
    const draftButton = event.target.closest("[data-provider-smoke-draft]");
    if (draftButton) {
      buildProviderSmokeDraftByProtocolGroup(draftButton.dataset.providerSmokeDraft || "");
      return;
    }
    const draftActionButton = event.target.closest("[data-provider-smoke-draft-action]");
    if (draftActionButton) {
      buildProviderSmokeDraftByProtocolGroup(draftActionButton.dataset.providerSmokeDraftAction || "", { fromGap: true });
      return;
    }
    const prefillProfileRiskButton = event.target.closest("[data-provider-smoke-prefill-profile-risk]");
    if (prefillProfileRiskButton) {
      const protocolGroup = prefillProfileRiskButton.dataset.providerSmokePrefillProfileRisk || "";
      const row = (state.providerSmokeMatrix || []).find((item) => item.protocolGroup === protocolGroup);
      if (!row) {
        showFlash("未找到对应协议组的验收矩阵项", true);
        return;
      }
      prefillProfileRiskDefaultsFromMatrix(row);
      return;
    }
    const focusGroupButton = event.target.closest("[data-provider-smoke-focus-group]");
    if (focusGroupButton) {
      focusProviderSmokeRecordsByGroup(focusGroupButton.dataset.providerSmokeFocusGroup || "");
      return;
    }
    const filterStatusButton = event.target.closest("[data-provider-smoke-filter-status]");
    if (filterStatusButton) {
      focusProviderSmokeMatrixByStatus(filterStatusButton.dataset.providerSmokeFilterStatus || "all");
    }
  });
}

function wireTreeFilters() {
  const bindTextFilter = (selector, section, key, rerender) => {
    $(selector).addEventListener("input", (event) => {
      state.treeFilters[section][key] = event.target.value;
      rerender();
    });
  };
  const bindCheckboxFilter = (selector, section, key, rerender) => {
    $(selector).addEventListener("change", (event) => {
      state.treeFilters[section][key] = event.target.checked;
      rerender();
    });
  };

  const rerenderTask = () => updateTaskTreePanels(currentSelectedTaskDetail());
  const rerenderStatus = () => updateStatusTreePanels(recentRuntimePayload());

  bindTextFilter("#task-directory-filter-query", "taskDirectory", "query", rerenderTask);
  bindTextFilter("#task-directory-filter-status", "taskDirectory", "status", rerenderTask);
  bindCheckboxFilter("#task-directory-filter-leaf-only", "taskDirectory", "leafOnly", rerenderTask);
  bindCheckboxFilter("#task-directory-filter-problem-only", "taskDirectory", "problemOnly", rerenderTask);
  bindTextFilter("#task-pending-filter-query", "taskPending", "query", rerenderTask);
  bindTextFilter("#task-pending-filter-reason", "taskPending", "reason", rerenderTask);
  bindCheckboxFilter("#task-pending-filter-leaf-only", "taskPending", "leafOnly", rerenderTask);
  bindTextFilter("#task-retry-filter-query", "taskRetry", "query", () => updateTaskRetryQueue(currentSelectedTaskDetail()));
  bindTextFilter("#task-retry-filter-class", "taskRetry", "retryClass", () => updateTaskRetryQueue(currentSelectedTaskDetail()));
  bindTextFilter("#task-retry-filter-state", "taskRetry", "retryState", () => updateTaskRetryQueue(currentSelectedTaskDetail()));

  bindTextFilter("#status-directory-filter-query", "statusDirectory", "query", rerenderStatus);
  bindTextFilter("#status-directory-filter-status", "statusDirectory", "status", rerenderStatus);
  bindCheckboxFilter("#status-directory-filter-leaf-only", "statusDirectory", "leafOnly", rerenderStatus);
  bindCheckboxFilter("#status-directory-filter-problem-only", "statusDirectory", "problemOnly", rerenderStatus);
  bindTextFilter("#status-pending-filter-query", "statusPending", "query", rerenderStatus);
  bindTextFilter("#status-pending-filter-reason", "statusPending", "reason", rerenderStatus);
  bindCheckboxFilter("#status-pending-filter-leaf-only", "statusPending", "leafOnly", rerenderStatus);
  bindTextFilter("#status-retry-filter-query", "statusRetry", "query", () => updateStatusRetryQueue(recentRuntimePayload()));
  bindTextFilter("#status-retry-filter-class", "statusRetry", "retryClass", () => updateStatusRetryQueue(recentRuntimePayload()));
  bindTextFilter("#status-retry-filter-state", "statusRetry", "retryState", () => updateStatusRetryQueue(recentRuntimePayload()));

  $("#task-directory-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("taskDirectory");
    setFilterControlValue("#task-directory-filter-query", "");
    setFilterControlValue("#task-directory-filter-status", "");
    setFilterControlValue("#task-directory-filter-leaf-only", false);
    setFilterControlValue("#task-directory-filter-problem-only", false);
    rerenderTask();
    showFlash("已清空任务目录树筛选");
  });
  $("#task-pending-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("taskPending");
    setFilterControlValue("#task-pending-filter-query", "");
    setFilterControlValue("#task-pending-filter-reason", "");
    setFilterControlValue("#task-pending-filter-leaf-only", false);
    rerenderTask();
    showFlash("已清空任务待补传筛选");
  });
  $("#status-directory-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("statusDirectory");
    setFilterControlValue("#status-directory-filter-query", "");
    setFilterControlValue("#status-directory-filter-status", "");
    setFilterControlValue("#status-directory-filter-leaf-only", false);
    setFilterControlValue("#status-directory-filter-problem-only", false);
    rerenderStatus();
    showFlash("已清空状态目录树筛选");
  });
  $("#status-pending-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("statusPending");
    setFilterControlValue("#status-pending-filter-query", "");
    setFilterControlValue("#status-pending-filter-reason", "");
    setFilterControlValue("#status-pending-filter-leaf-only", false);
    rerenderStatus();
    showFlash("已清空状态待补传筛选");
  });
  $("#provider-smoke-records-filter-query").addEventListener("input", (event) => {
    state.providerSmokeRecordFilters.query = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-group").addEventListener("input", (event) => {
    state.providerSmokeRecordFilters.protocolGroup = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-result").addEventListener("change", (event) => {
    state.providerSmokeRecordFilters.result = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-clear").addEventListener("click", () => {
    state.providerSmokeRecordFilters.query = "";
    state.providerSmokeRecordFilters.protocolGroup = "";
    state.providerSmokeRecordFilters.result = "";
    setFilterControlValue("#provider-smoke-records-filter-query", "");
    setFilterControlValue("#provider-smoke-records-filter-group", "");
    setFilterControlValue("#provider-smoke-records-filter-result", "");
    renderStatus();
    showFlash("已清空 smoke 记录筛选");
  });
  $("#task-directory-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("task", "directory");
    } catch (error) {
      showFlash(`复制目录路径失败：${error.message}`, true);
    }
  });
  $("#task-pending-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("task", "pending");
    } catch (error) {
      showFlash(`复制待补传路径失败：${error.message}`, true);
    }
  });
  $("#status-directory-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("status", "directory");
    } catch (error) {
      showFlash(`复制目录路径失败：${error.message}`, true);
    }
  });
  $("#status-pending-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("status", "pending");
    } catch (error) {
      showFlash(`复制待补传路径失败：${error.message}`, true);
    }
  });
}

async function init() {
  setupTabs();
  wireLogin();
  wireProfiles();
  wirePlanner();
  wireTasks();
  wireStatus();
  wireTreeFilters();
  syncSessionState();
  syncExecutionModeHint();
  renderPreview();
  renderStatus();
  window.__cloudpanTestHooks = {
    openTaskByID,
    renderTasks,
    renderSelectedTask,
    focusBlockedActionSummary,
    state,
  };
  if (state.authenticated) {
    try {
      await bootstrapData();
    } catch (error) {
      showFlash(error.message, true);
    }
  }
}
window.addEventListener("DOMContentLoaded", init);
