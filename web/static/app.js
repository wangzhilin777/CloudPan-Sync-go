const state = {
  authenticated: localStorage.getItem("cloudpan_console_session") === "ok",
  providers: [],
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
    providerKey: "",
    retryClass: "",
    blockedAction: "",
    limit: "",
  },
  treeGroupsCollapsed: {},
  treeFilters: {
    taskDirectory: { query: "", status: "" },
    taskPending: { query: "", reason: "", leafOnly: false },
    taskRetry: { query: "", retryClass: "", retryState: "" },
    statusDirectory: { query: "", status: "" },
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
  const reasons = Array.isArray(resolution.calibrationReasons) ? resolution.calibrationReasons.filter(Boolean) : [];
  const overrideFields = Array.isArray(resolution.overrideFields) ? resolution.overrideFields.filter(Boolean) : [];
  const parts = [`provider ${providerKey}`];
  if (reasons.length) {
    parts.push(`校准 ${reasons.join(" / ")}`);
  }
  if (overrideFields.length) {
    parts.push(`override ${overrideFields.join(", ")}`);
  }
  return parts.join(" | ");
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
  const filterActive = Boolean(query || status);
  const prune = (nodes) =>
    nodes.flatMap((node) => {
      const children = prune(node.children || []);
      const selfMatch =
        includesFilterText([node.path, node.name, node.rootPath, node.lastItemPath], query) &&
        includesFilterText([node.status], status);
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

function renderTreeFilterSummary(result, label) {
  if (!result.totalNodes) {
    return `暂无${label}。`;
  }
  if (!result.filterActive) {
    return `显示全部 ${result.visibleNodes} 个${label}。`;
  }
  return `当前显示 ${result.visibleNodes} / ${result.totalNodes} 个${label}。`;
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
        `
        : ""
    }
    ${renderUploadCheckpoint(runtime.uploadCheckpoint)}
  `;
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
      <strong>上传分片进度</strong>
      <span>${uploadedPartCount} / ${partCount}，证据 ${uploadedPartsLen} 段</span>
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
    const metrics =
      mode === "pending"
        ? `
            <div class="directory-metrics">
              <span class="pill">${escapeHTML(node.nodeType)}</span>
              <span class="pill">pending ${node.itemCount}</span>
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
            </div>
            <div class="muted">last item: <code>${escapeHTML(stringifyValue(node.lastItemPath, "-"))}</code></div>
          `;

    const children = node.children.length
      ? `<div class="directory-children">${node.children.map((child) => renderNode(child)).join("")}</div>`
      : "";
    const syncTargetPanel = panel === "directory" ? "pending" : "directory";
    const syncLabel = panel === "directory" ? "待补传树" : "目录树";

    return `
      <div class="directory-row tree-node">
        <div class="directory-row-header">
          <strong>${escapeHTML(node.name || node.path)}</strong>
          <code>${escapeHTML(node.path)}</code>
        </div>
        ${metrics}
        <div class="actions compact">
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
            scope === "task" && panel === "pending"
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
                  scope === "task" && panel === "pending"
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
    ? renderTreeFilterSummary(directoryResult, "目录节点")
    : "等待任务数据...";
  $("#task-pending-filter-summary").textContent = detail
    ? renderTreeFilterSummary(pendingResult, "待补传节点")
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
  $("#status-directory-filter-summary").textContent = renderTreeFilterSummary(directoryResult, "目录节点");
  $("#status-pending-filter-summary").textContent = renderTreeFilterSummary(pendingResult, "待补传节点");
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

function visibleTreePaths(scope, panel) {
  const detail = currentSelectedTaskDetail();
  const runtimePayload = recentRuntimePayload();
  if (scope === "task") {
    const runtime = detail?.runtime || detail?.plan?.metadata?.runtime || {};
    if (panel === "pending") {
      const result = filterPendingTree(runtime.pendingTree || [], state.treeFilters.taskPending);
      return flattenVisibleTreePaths(result.nodes, "pending");
    }
    const result = filterDirectoryTree(runtime.directoryStates || [], state.treeFilters.taskDirectory);
    return flattenVisibleTreePaths(result.nodes, "directory");
  }
  const runtime = runtimePayload?.runtime || runtimePayload || {};
  if (panel === "pending") {
    const result = filterPendingTree(runtime.pendingTree || [], state.treeFilters.statusPending);
    return flattenVisibleTreePaths(result.nodes, "pending");
  }
  const result = filterDirectoryTree(runtime.directoryStates || [], state.treeFilters.statusDirectory);
  return flattenVisibleTreePaths(result.nodes, "directory");
}

function visibleRetryPaths(scope) {
  const detail = currentSelectedTaskDetail();
  const runtimePayload = recentRuntimePayload();
  const runtime = scope === "task" ? detail?.runtime || detail?.plan?.metadata?.runtime || {} : runtimePayload?.runtime || runtimePayload || {};
  const filters = scope === "task" ? state.treeFilters.taskRetry : state.treeFilters.statusRetry;
  const result = filterRetryQueue(runtime.retryQueue || [], filters);
  return Array.from(new Set((result.items || []).map((item) => item.path).filter(Boolean)));
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
        await retryTaskPath(button.dataset.treeRetryPath || "");
      } catch (error) {
        showFlash(error.message, true);
      }
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

function collectRiskOverrideFromForm() {
  const override = {};
  const numberFields = [
    ["#risk-request-interval", "requestIntervalMs"],
    ["#risk-directory-interval", "directoryIntervalMs"],
    ["#risk-page-size", "pageSize"],
    ["#risk-cooldown-seconds", "cooldownSeconds"],
    ["#risk-retry-limit", "retryLimit"],
    ["#risk-max-concurrent", "maxConcurrent"],
    ["#risk-auto-retry-start-hour", "autoRetryStartHour"],
    ["#risk-auto-retry-end-hour", "autoRetryEndHour"],
  ];
  numberFields.forEach(([selector, key]) => {
    const value = optionalNumberValue(selector);
    if (value !== null && Number.isFinite(value)) {
      override[key] = value;
    }
  });
  const keywords = $("#risk-keywords").value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
  if (keywords.length > 0) {
    override.riskKeywords = keywords;
  }
  return Object.keys(override).length > 0 ? override : null;
}

function hydrateRiskOverrideForm(value) {
  const override = value && typeof value === "object" ? value : {};
  $("#risk-request-interval").value = override.requestIntervalMs ?? "";
  $("#risk-directory-interval").value = override.directoryIntervalMs ?? "";
  $("#risk-page-size").value = override.pageSize ?? "";
  $("#risk-cooldown-seconds").value = override.cooldownSeconds ?? "";
  $("#risk-retry-limit").value = override.retryLimit ?? "";
  $("#risk-max-concurrent").value = override.maxConcurrent ?? "";
  $("#risk-auto-retry-start-hour").value = override.autoRetryStartHour ?? "";
  $("#risk-auto-retry-end-hour").value = override.autoRetryEndHour ?? "";
  $("#risk-keywords").value = Array.isArray(override.riskKeywords) ? override.riskKeywords.join(",") : "";
}

function syncRiskOverrideJSON() {
  const override = collectRiskOverrideFromForm();
  $("#plan-risk-override").value = override ? JSON.stringify(override, null, 2) : "";
  return override;
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

function buildCreatePayloadFromTaskPath(detail, path) {
  const payload = buildCreatePayloadFromTask(detail);
  const normalizedPath = normalizeComparePath(path);
  if (!payload || !normalizedPath) {
    return payload;
  }
  const narrowedEntries = (payload.entries || []).filter((entry) => pathMatchesSubtree(entry?.path, normalizedPath));
  const narrowedRoots = (payload.selectedRoots || []).filter((root) => pathMatchesSubtree(normalizedPath, root) || pathMatchesSubtree(root, normalizedPath));
  payload.selectedRoots = narrowedRoots.length ? narrowedRoots.filter((root) => pathMatchesSubtree(normalizedPath, root)) : [normalizedPath];
  payload.entries = narrowedEntries.length ? narrowedEntries : payload.entries;
  return payload;
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

function renderTaskResolutionGuide(detail) {
  const metadata = detail?.plan?.metadata || {};
  const runtime = detail?.runtime || {};
  const retrySummary = metadata.retrySummary || {};
  const action = runtime.blockedAction || retrySummary.blockedAction || "";
  const advice = runtime.blockedAdvice || retrySummary.blockedAdvice || "";
  if (!action) {
    return `
      <div class="insight-card">
        <strong>下一步处理</strong>
        <span>当前任务没有 blocked 人工处理动作，可直接继续运行或观察状态矩阵。</span>
      </div>
    `;
  }

  const providerKey = detail.task?.targetProvider || "";
  const profileId = detail.targetProfileId || "";
  const nextRetryAt = runtime.nextRetryAt || retrySummary.nextRetryAt || "";
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
        { label: "打开状态矩阵", view: "status" },
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
        { label: "打开状态矩阵", view: "status" },
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
        { label: "查看状态矩阵", view: "status" },
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
        { label: "查看状态矩阵", view: "status" },
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
        { label: "查看状态矩阵", view: "status" },
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

  const options = state.providers
    .map((entry) => `<option value="${entry.meta.key}">${entry.meta.displayName}</option>`)
    .join("");
  providerSelect.innerHTML = options;
  sourceSelect.innerHTML = options;
  targetSelect.innerHTML = options;

  providerCards.innerHTML = state.providers
    .map(
      (entry) => `
        <article class="provider-card">
          <h3>${entry.meta.displayName}</h3>
          <div class="meta-row">
            <span class="pill">${entry.meta.key}</span>
            <span class="pill">${entry.meta.protocolGroup}</span>
            <span class="pill">${entry.meta.status}</span>
          </div>
          <div class="meta-row">
            ${entry.meta.authModes.map((mode) => `<span class="pill">${mode}</span>`).join("")}
          </div>
        </article>
      `,
    )
    .join("");

  syncAuthModes();
  syncSourceProfiles();
  syncTargetProfiles();
  syncAutoRecoverProviders();
  syncAutoRecoverBlockedActions();
  syncExecutionModeHint();
}

function prefillWizardFromTaskPath(detail, path) {
  const payload = buildCreatePayloadFromTaskPath(detail, path);
  if (!payload) {
    showFlash("请先选择任务", true);
    return;
  }
  activateTab("wizard");
  setSelectValueIfPresent("#plan-source-provider", payload.sourceProvider);
  syncSourceProfiles();
  setSelectValueIfPresent("#plan-source-profile", payload.sourceProfileId || "");
  setSelectValueIfPresent("#plan-target-provider", payload.targetProvider);
  syncTargetProfiles();
  setSelectValueIfPresent("#plan-target-profile", payload.targetProfileId || "");
  setSelectValueIfPresent("#plan-execution-mode", payload.executionMode || "leaf_first_lazy");
  setSelectValueIfPresent("#plan-risk-mode", payload.riskMode || "balanced");
  setSelectValueIfPresent("#plan-conflict-policy", payload.conflictPolicy || "auto_rename_new");
  setInputValueIfPresent("#plan-threshold", payload.thresholdMB || 0);
  hydrateRiskOverrideForm(payload.riskOverride || null);
  $("#plan-risk-override").value = payload.riskOverride ? JSON.stringify(payload.riskOverride, null, 2) : "";
  $("#plan-selected-roots").value = JSON.stringify(payload.selectedRoots || ["/"], null, 2);
  $("#plan-entries").value = JSON.stringify(payload.entries || [], null, 2);
  syncExecutionModeHint();
  showFlash(`已按 ${path} 重建向导范围`);
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
    return;
  }

  wrap.innerHTML = `
    <table>
      <thead>
        <tr>
          <th>显示名称</th>
          <th>Provider</th>
          <th>Auth Mode</th>
          <th>Status</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        ${state.profiles
          .map(
            (profile) => `
              <tr class="${profile.id === state.focusedProfileId ? "active" : ""}" data-profile-row="${profile.id}">
                <td>${profile.displayName}</td>
                <td>${profile.providerKey}</td>
                <td>${profile.authMode}</td>
                <td>${profile.status}</td>
                <td>
                  <div class="actions compact">
                    <button type="button" class="ghost" data-profile-validate="${profile.id}">Validate</button>
                    <button type="button" class="ghost" data-profile-delete="${profile.id}">Delete</button>
                  </div>
                </td>
              </tr>
            `,
          )
          .join("")}
      </tbody>
    </table>
  `;

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
  select.innerHTML = `<option value="">无</option>${profiles
    .map((profile) => `<option value="${profile.id}">${profile.displayName}</option>`)
    .join("")}`;
}

function syncSourceProfiles() {
  const sourceProvider = $("#plan-source-provider").value;
  const profiles = state.profiles.filter((item) => item.providerKey === sourceProvider);
  const select = $("#plan-source-profile");
  select.innerHTML = `<option value="">无</option>${profiles
    .map((profile) => `<option value="${profile.id}">${profile.displayName}</option>`)
    .join("")}`;
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
  setSelectValueIfPresent("#auto-recover-blocked-action", current);
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
  const button = $("#apply-recommended-execution");
  const recommended = metadata.recommendedExecutionMode || "";
  const selected = $("#plan-execution-mode").value;
  if (!recommended) {
    card.classList.add("hidden");
    button.disabled = true;
    return;
  }
  const reason = stringifyValue(metadata.recommendedExecutionModeReason, "暂无推荐原因");
  card.classList.remove("hidden");
  $("#plan-recommendation-title").textContent =
    recommended === selected ? `当前已采用推荐模式：${recommended}` : `建议切换到：${recommended}`;
  $("#plan-recommendation-reason").textContent = reason;
  button.disabled = recommended === selected;
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
    <div class="insight-card">
      <strong>风险模板解释</strong>
      <span>${escapeHTML(renderRiskResolutionSummary(metadata.riskProfileResolution))}</span>
    </div>
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
      <strong>重试摘要</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedReason || (metadata.retrySummary?.shouldBlock ? "blocked" : "ready"), "-")}</span>
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
    <div class="insight-card">
      <strong>风险模板解释</strong>
      <span>${escapeHTML(renderRiskResolutionSummary(metadata.riskProfileResolution))}</span>
    </div>
    <div class="insight-card">
      <strong>自动补传时间窗</strong>
      <span>${escapeHTML(renderRiskWindow(metadata.riskProfile))}</span>
    </div>
  `;
  $("#plan-preview").textContent = formatJSON(state.preview);
}

function renderStatus() {
  const evidence = state.evidence || {
    totalTasks: 0,
    completedTasks: 0,
    blockedTasks: 0,
    autoRecoverTasks: 0,
    failedResultCount: 0,
    doneResultCount: 0,
    skippedResultCount: 0,
    pendingResultCount: 0,
    riskHitCount: 0,
    blockedActions: [],
    autoRecoverPool: [],
    protocolCoverage: [],
    recentResults: [],
    recentProbes: [],
  };
  const protocolCoverage = Array.isArray(evidence.protocolCoverage) ? evidence.protocolCoverage : [];
  const protocolCoverageWithSamples = protocolCoverage.filter((item) => item?.hasRealSuccessSample).length;
  const providerSmokeMatrix = Array.isArray(state.providerSmokeMatrix) ? state.providerSmokeMatrix : [];
  const acceptedSmokeGroups = providerSmokeMatrix.filter((item) => item?.accepted).length;
  const inProgressSmokeGroups = providerSmokeMatrix.filter((item) => item?.acceptanceStatus === "in_progress").length;
  const pendingSmokeGroups = providerSmokeMatrix.filter((item) => item?.acceptanceStatus === "pending").length;
  syncAutoRecoverBlockedActions();
  $("#evidence-summary").innerHTML = `
    <div class="metric"><span>Total Tasks</span><strong>${evidence.totalTasks}</strong></div>
    <div class="metric"><span>Completed</span><strong>${evidence.completedTasks}</strong></div>
    <div class="metric"><span>Blocked Tasks</span><strong>${evidence.blockedTasks}</strong></div>
    <div class="metric"><span>Auto Recover</span><strong>${stringifyValue(evidence.autoRecoverTasks, "0")}</strong></div>
    <div class="metric"><span>Done Results</span><strong>${evidence.doneResultCount}</strong></div>
    <div class="metric"><span>Skipped Results</span><strong>${evidence.skippedResultCount}</strong></div>
    <div class="metric"><span>Pending Manual</span><strong>${evidence.pendingResultCount}</strong></div>
    <div class="metric"><span>Failed Results</span><strong>${evidence.failedResultCount}</strong></div>
    <div class="metric"><span>Risk Hits</span><strong>${evidence.riskHitCount}</strong></div>
    <div class="metric"><span>Protocol Groups</span><strong>${protocolCoverage.length}</strong></div>
    <div class="metric"><span>Sampled Groups</span><strong>${protocolCoverageWithSamples}</strong></div>
    <div class="metric"><span>Accepted Groups</span><strong>${acceptedSmokeGroups}</strong></div>
    <div class="metric"><span>In Progress</span><strong>${inProgressSmokeGroups}</strong></div>
    <div class="metric"><span>Pending Groups</span><strong>${pendingSmokeGroups}</strong></div>
  `;
  $("#blocked-actions-summary").innerHTML = renderBlockedActionsSummary(evidence.blockedActions || []);
  wireBlockedActionsSummary();
  $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
    filterAutoRecoverItems(evidence.autoRecoverPool || []),
    evidence.autoRecoverPool || [],
  );
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
  $("#status-runtime-checkpoints").innerHTML = renderRuntimeCheckpoint(runtimePayload?.runtime || runtimePayload, runtimePayload?.runtime || runtimePayload, "status");
  wireRuntimePathFocus("status");
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
  const providerKey = String(filters?.providerKey || "").trim();
  const retryClass = String(filters?.retryClass || "").trim();
  const blockedAction = String(filters?.blockedAction || "").trim();
  const source = Array.isArray(items) ? items : [];
  return source.filter((item) => {
    if (mode && String(item?.mode || "").trim() !== mode) {
      return false;
    }
    if (providerKey && String(item?.sampleProvider || "").trim() !== providerKey) {
      return false;
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
    return true;
  });
}

function renderAutoRecoverFilterSummary(visibleItems, allItems) {
  const total = Array.isArray(allItems) ? allItems.length : 0;
  const current = Array.isArray(visibleItems) ? visibleItems.length : 0;
  const mode = String(state.autoRecoverFilters.mode || "").trim();
  const providerKey = String(state.autoRecoverFilters.providerKey || "").trim();
  const retryClass = String(state.autoRecoverFilters.retryClass || "").trim();
  const blockedAction = String(state.autoRecoverFilters.blockedAction || "").trim();
  const limit = String(state.autoRecoverFilters.limit || "").trim();
  const parts = [];
  if (mode) {
    parts.push(`mode=${mode}`);
  }
  if (providerKey) {
    parts.push(`provider=${providerKey}`);
  }
  if (retryClass) {
    parts.push(`retryClass=${retryClass}`);
  }
  if (blockedAction) {
    parts.push(`blockedAction=${blockedAction}`);
  }
  if (limit) {
    parts.push(`limit=${limit}`);
  }
  if (!parts.length) {
    return total ? `当前显示全部 ${current}/${total} 条后台补传候选。` : "显示全部后台补传候选。";
  }
  return `当前显示 ${current}/${total} 条后台补传候选，筛选条件：${parts.join(" / ")}。`;
}

function renderAutoRecoverSummary(items) {
  const visibleItems = filterAutoRecoverItems(items);
  if (!visibleItems.length) {
    if (Array.isArray(items) && items.length) {
      return `<div class="directory-empty">当前筛选条件下没有命中的后台补传候选。</div>`;
    }
    return `<div class="directory-empty">当前没有进入后台补传候选池的任务。</div>`;
  }
  return visibleItems
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
            <span class="pill">queue ${stringifyValue(item.queueItemCount, "0")}</span>
            <span class="pill">ready ${stringifyValue(item.retryableNowCount, "0")}</span>
            <span class="pill">cooldown ${stringifyValue(item.cooldownCount, "0")}</span>
            <span class="pill">checkpoint ${stringifyValue(item.uploadCheckpointEligible, "0")}</span>
            <span class="pill">classes ${(item.retryClasses || []).join(", ") || "-"}</span>
            <span class="pill">actions ${(item.blockedActions || []).join(", ") || "-"}</span>
          </div>
          <div class="muted">${escapeHTML(stringifyValue(item.advice, "-"))}</div>
          <div class="actions compact">
            <span class="pill">next ${escapeHTML(stringifyValue(item.nextRetryAt, "-"))}</span>
            <button
              type="button"
              class="ghost"
              data-auto-recover-focus-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
            >只看该模式</button>
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
              Array.isArray(item.blockedActions) && item.blockedActions.length
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-blocked-action="${escapeHTML(stringifyValue(item.blockedActions[0], ""))}"
            >执行该阻塞动作</button>`
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
  if (retrySummary && typeof retrySummary === "object") {
    return `
      <div><strong>lastTaskState</strong> <code>${escapeHTML(stringifyValue(summary.lastTaskState))}</code></div>
      <div><strong>blockedCount</strong> <code>${escapeHTML(stringifyValue(summary.blockedCount, "0"))}</code></div>
      <div><strong>autoRecoverCount</strong> <code>${escapeHTML(stringifyValue(summary.autoRecoverCount, "0"))}</code></div>
      <div><strong>retryBlocked</strong> <code>${escapeHTML(stringifyValue(retrySummary.blockedReason, "-"))}</code></div>
      <div><strong>blockedAction</strong> <code>${escapeHTML(stringifyValue(retrySummary.blockedAction, "-"))}</code></div>
      <div><strong>blockedTop</strong> <code>${escapeHTML(stringifyValue(blockedActions[0]?.action, "-"))}</code></div>
      <div><strong>nextRetryAt</strong> <code>${escapeHTML(stringifyValue(retrySummary.nextRetryAt, "-"))}</code></div>
      <div><strong>queueSize</strong> <code>${escapeHTML(stringifyValue(retrySummary.queueSize, "0"))}</code></div>
      <div><strong>autoRecover</strong> <code>${escapeHTML(renderAutoRecoverMode(retrySummary))}</code></div>
      <div><strong>queueBreakdown</strong> <code>${escapeHTML(renderRetrySummaryBreakdown(retrySummary))}</code></div>
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
      state.autoRecoverFilters.mode = mode;
      setFilterControlValue("#auto-recover-mode", mode);
      renderStatus();
      showFlash(`已按 ${mode} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-blocked-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const blockedAction = button.dataset.autoRecoverFocusBlockedAction || "";
      state.autoRecoverFilters.blockedAction = blockedAction;
      setFilterControlValue("#auto-recover-blocked-action", blockedAction);
      renderStatus();
      showFlash(`已按 ${blockedAction} 收敛后台补传候选`);
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-mode]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        state.autoRecoverFilters.mode = button.dataset.autoRecoverRunMode || "";
        setFilterControlValue("#auto-recover-mode", state.autoRecoverFilters.mode);
        await triggerAutoRecover();
      } catch (error) {
        showFlash(error.message, true);
      }
    });
  });
  wrap.querySelectorAll("[data-auto-recover-run-blocked-action]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        state.autoRecoverFilters.blockedAction = button.dataset.autoRecoverRunBlockedAction || "";
        setFilterControlValue("#auto-recover-blocked-action", state.autoRecoverFilters.blockedAction);
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

function currentAutoRecoverRequest() {
  const limitText = String($("#auto-recover-limit")?.value || "").trim();
  const limit = limitText ? Number(limitText) : 0;
  return {
    mode: String($("#auto-recover-mode")?.value || "").trim(),
    providerKey: String($("#auto-recover-provider")?.value || "").trim(),
    retryClass: String($("#auto-recover-retry-class")?.value || "").trim(),
    blockedAction: String($("#auto-recover-blocked-action")?.value || "").trim(),
    limit: Number.isFinite(limit) && limit > 0 ? limit : 0,
  };
}

async function triggerAutoRecover() {
  const payload = currentAutoRecoverRequest();
  state.autoRecoverFilters.mode = payload.mode;
  state.autoRecoverFilters.providerKey = payload.providerKey;
  state.autoRecoverFilters.retryClass = payload.retryClass;
  state.autoRecoverFilters.blockedAction = payload.blockedAction;
  state.autoRecoverFilters.limit = payload.limit ? String(payload.limit) : "";
  const result = await api("/api/tasks/recover", {
    method: "POST",
    body: payload,
  });
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(
    `后台补传已执行：matched ${stringifyValue(result.matchedCount, "0")} / recovered ${stringifyValue(result.recoveredCount, "0")} / skipped ${stringifyValue(result.skippedByLimit, "0")}`,
  );
}

function resetAutoRecoverFilters() {
  state.autoRecoverFilters.mode = "";
  state.autoRecoverFilters.providerKey = "";
  state.autoRecoverFilters.retryClass = "";
  state.autoRecoverFilters.blockedAction = "";
  state.autoRecoverFilters.limit = "";
  setFilterControlValue("#auto-recover-mode", "");
  setFilterControlValue("#auto-recover-provider", "");
  setFilterControlValue("#auto-recover-retry-class", "");
  setFilterControlValue("#auto-recover-blocked-action", "");
  setInputValueIfPresent("#auto-recover-limit", "");
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
      [item.title, item.providerKey, item.note, Array.isArray(item.operations) ? item.operations.join(",") : ""],
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
            <span class="pill">failure ${stringifyValue(item.failureCount, "0")}</span>
            <span class="pill">providers ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">${item.hasRealSuccessSample ? "sampled" : "pending"}</span>
          </div>
          <div class="muted">providers: ${escapeHTML((item.providerKeys || []).join(", ") || "-")}</div>
          <div class="muted">sample: ${escapeHTML(stringifyValue(item.sampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleCategory, "-"))}</div>
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
            <span class="pill">coverage ${stringifyValue(item.coverageRealSuccessTaskCount, "0")}/${stringifyValue(item.coverageTaskCount, "0")}</span>
            <span class="pill">${item.accepted ? "accepted" : item.acceptanceStatus || "pending"}</span>
          </div>
          <div class="muted">smoke sample: ${escapeHTML(stringifyValue(item.sampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleCategory, "-"))}</div>
          <div class="muted">coverage sample: ${escapeHTML(stringifyValue(item.coverageSampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.coverageSampleTaskState, "-"))} / ${escapeHTML(stringifyValue(item.coverageSampleCompletionKind, "-"))}</div>
          ${Array.isArray(item.acceptanceMissing) && item.acceptanceMissing.length ? `<div class="muted">missing: ${escapeHTML(item.acceptanceMissing.join(", "))}</div>` : ""}
          ${item.acceptanceAdvice ? `<div class="muted">advice: ${escapeHTML(item.acceptanceAdvice)}</div>` : ""}
          <div class="actions compact">
            ${item.sampleRecordId ? `<button type="button" class="ghost" data-provider-smoke-open-record="${escapeHTML(stringifyValue(item.sampleRecordId))}">打开 smoke 样本</button>` : ""}
            ${item.coverageSampleTaskId ? `<button type="button" class="ghost" data-provider-smoke-open-task="${escapeHTML(stringifyValue(item.coverageSampleTaskId))}">打开任务样本</button>` : ""}
            <button type="button" class="ghost" data-provider-smoke-draft="${escapeHTML(stringifyValue(item.protocolGroup))}">预填 smoke 表单</button>
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

function draftProviderSmokeFromMatrix(item) {
  const protocolGroup = String(item?.protocolGroup || "").trim();
  const providerKey = firstNonEmpty(
    String(item?.sampleProviderKey || "").trim(),
    String(item?.coverageSampleProviderKey || "").trim(),
    Array.isArray(item?.providerKeys) ? String(item.providerKeys[0] || "").trim() : "",
    Array.isArray(item?.coverageProviderKeys) ? String(item.coverageProviderKeys[0] || "").trim() : "",
  );
  const status = item?.accepted ? "accepted" : String(item?.acceptanceStatus || "pending");
  const missing = Array.isArray(item?.acceptanceMissing) ? item.acceptanceMissing.filter(Boolean) : [];
  const noteParts = [
    protocolGroup ? `协议组：${protocolGroup}` : "",
    providerKey ? `建议 provider：${providerKey}` : "",
    missing.length ? `缺口：${missing.join(", ")}` : "",
    item?.acceptanceAdvice ? `建议：${item.acceptanceAdvice}` : "",
  ].filter(Boolean);
  return {
    providerKey,
    protocolGroup,
    authMode: "",
    category: "",
    result: "success",
    title: protocolGroup ? `${protocolGroup} ${status} smoke` : "provider smoke",
    note: noteParts.join("；"),
    operations: [],
  };
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
  $("#plan-target-provider").addEventListener("change", syncTargetProfiles);
  $("#plan-execution-mode").addEventListener("change", () => {
    syncExecutionModeHint();
    updateExecutionRecommendationAction(state.preview?.metadata || {});
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

  $("#profile-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const payload = {
        providerKey: $("#profile-provider").value,
        authMode: $("#profile-auth-mode").value,
        displayName: $("#profile-display-name").value.trim(),
        token: $("#profile-token").value.trim(),
        cookie: $("#profile-cookie").value.trim(),
        extra: parseJSONInput($("#profile-extra").value, {}),
      };
      await api("/api/auth/profiles", {
        method: "POST",
        body: payload,
      });
      event.target.reset();
      $("#profile-extra").value = "";
      await loadProfiles();
      showFlash("授权档案已创建");
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
    conflictPolicy: $("#plan-conflict-policy").value,
    selectedRoots: parseJSONInput($("#plan-selected-roots").value, []),
    entries: parseJSONInput($("#plan-entries").value, []),
  };
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
  try {
    await api(`/api/tasks/${state.selectedTaskId}/${action}`, { method: "POST", body });
    showFlash(`任务 ${action} 已执行`);
    await Promise.all([loadTasks(), loadStatus()]);
    return true;
  } catch (error) {
    showFlash(error.message, true);
    return false;
  }
}

async function retryVisibleSelection(source) {
  if (!state.selectedTaskId) {
    showFlash("请先选择任务", true);
    return;
  }
  const paths = source === "pending_tree" ? visibleTreePaths("task", "pending") : visibleRetryPaths("task");
  if (!paths.length) {
    showFlash("当前筛选结果里没有可重试的路径", true);
    return;
  }
  const ok = await runTaskAction("retry", { paths, scope: source });
  if (ok) {
    showFlash(`已按当前筛选重建 ${paths.length} 条路径的 retry 范围`);
  }
}

async function retryTaskPath(path) {
  const normalizedPath = normalizeComparePath(path);
  if (!normalizedPath) {
    showFlash("当前路径无效，无法重试", true);
    return false;
  }
  const ok = await runTaskAction("retry", {
    paths: [normalizedPath],
    scope: "selected_pending_subset",
  });
  if (ok) {
    showFlash(`已按 ${normalizedPath} 重建 retry 范围`);
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
  $("#task-retry-visible-queue").addEventListener("click", () => retryVisibleSelection("selected_retry_subset"));
  $("#task-retry-visible-pending").addEventListener("click", () => retryVisibleSelection("selected_pending_subset"));
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
  $("#auto-recover-provider").addEventListener("change", () => {
    state.autoRecoverFilters.providerKey = $("#auto-recover-provider").value;
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
  $("#auto-recover-limit").addEventListener("input", () => {
    state.autoRecoverFilters.limit = $("#auto-recover-limit").value.trim();
    $("#auto-recover-filter-summary").textContent = renderAutoRecoverFilterSummary(
      filterAutoRecoverItems(state.evidence?.autoRecoverPool || []),
      state.evidence?.autoRecoverPool || [],
    );
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
        await loadProviderSmokeMarkdown(openRecordButton.dataset.providerSmokeOpenRecord || "");
        showFlash("已打开 smoke 样本");
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
      const protocolGroup = draftButton.dataset.providerSmokeDraft || "";
      const row = (state.providerSmokeMatrix || []).find((item) => item.protocolGroup === protocolGroup);
      if (!row) {
        showFlash("未找到对应协议组的验收矩阵项", true);
        return;
      }
      hydrateProviderSmokeForm(draftProviderSmokeFromMatrix(row));
      focusProviderSmokeRecordsByGroup(protocolGroup);
      showFlash("已按验收矩阵预填 smoke 表单");
      return;
    }
    const focusGroupButton = event.target.closest("[data-provider-smoke-focus-group]");
    if (focusGroupButton) {
      focusProviderSmokeRecordsByGroup(focusGroupButton.dataset.providerSmokeFocusGroup || "");
      return;
    }
    const filterStatusButton = event.target.closest("[data-provider-smoke-filter-status]");
    if (filterStatusButton) {
      setProviderSmokeMatrixFilter(filterStatusButton.dataset.providerSmokeFilterStatus || "all");
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
  bindTextFilter("#task-pending-filter-query", "taskPending", "query", rerenderTask);
  bindTextFilter("#task-pending-filter-reason", "taskPending", "reason", rerenderTask);
  bindCheckboxFilter("#task-pending-filter-leaf-only", "taskPending", "leafOnly", rerenderTask);
  bindTextFilter("#task-retry-filter-query", "taskRetry", "query", () => updateTaskRetryQueue(currentSelectedTaskDetail()));
  bindTextFilter("#task-retry-filter-class", "taskRetry", "retryClass", () => updateTaskRetryQueue(currentSelectedTaskDetail()));
  bindTextFilter("#task-retry-filter-state", "taskRetry", "retryState", () => updateTaskRetryQueue(currentSelectedTaskDetail()));

  bindTextFilter("#status-directory-filter-query", "statusDirectory", "query", rerenderStatus);
  bindTextFilter("#status-directory-filter-status", "statusDirectory", "status", rerenderStatus);
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
  if (state.authenticated) {
    try {
      await bootstrapData();
    } catch (error) {
      showFlash(error.message, true);
    }
  }
}

window.addEventListener("DOMContentLoaded", init);
