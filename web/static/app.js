const state = {
  authenticated: localStorage.getItem("cloudpan_console_session") === "ok",
  providers: [],
  profiles: [],
  tasks: [],
  preview: null,
  selectedTaskId: null,
  evidence: null,
  statuses: [],
};

function $(selector) {
  return document.querySelector(selector);
}

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

function renderRuntimeCheckpoint(runtime) {
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
  `;
}

function renderTreeNodes(nodes, options = {}) {
  const {
    mode = "directory",
    emptyMessage = "暂无数据。",
  } = options;
  const normalized = mode === "pending" ? normalizePendingTree(nodes) : buildDirectoryStateTree(nodes);
  if (!normalized.length) {
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

    return `
      <div class="directory-row tree-node">
        <div class="directory-row-header">
          <strong>${escapeHTML(node.name || node.path)}</strong>
          <code>${escapeHTML(node.path)}</code>
        </div>
        ${metrics}
        ${children}
      </div>
    `;
  };

  return normalized
    .map(
      (root) => `
        <section class="directory-group">
          <h4>Root <code>${escapeHTML(root.rootPath || root.path)}</code></h4>
          ${renderNode(root)}
        </section>
      `,
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
        { label: "打开授权面板", view: "providers", providerKey, profileId },
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
        { label: "打开任务向导", view: "wizard", providerKey },
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
        { label: "打开任务向导", view: "wizard", providerKey },
        { label: "查看状态矩阵", view: "status" },
      ],
    },
  };

  const config = stepsByAction[action] || {
    title: "人工处理建议",
    steps: [advice || "请根据 blocked 原因检查授权、源文件和 provider 返回状态。"] ,
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
      activateTab(view);
      if (view === "providers") {
        setSelectValueIfPresent("#profile-provider", button.dataset.taskGuideProvider);
      }
      if (view === "wizard") {
        setSelectValueIfPresent("#plan-target-provider", detail?.task?.targetProvider || button.dataset.taskGuideProvider);
        setSelectValueIfPresent("#plan-target-profile", detail?.targetProfileId || button.dataset.taskGuideProfile);
        setSelectValueIfPresent("#plan-source-provider", detail?.task?.sourceProvider);
      }
      if (view === "tasks") {
        state.selectedTaskId = detail?.task?.id || state.selectedTaskId;
        renderTasks();
        renderSelectedTask();
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
  syncExecutionModeHint();
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
              <tr>
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

function syncExecutionModeHint() {
  const mode = $("#plan-execution-mode").value;
  const hint = $("#plan-execution-hint");
  if (mode === "pre_scan_flat") {
    hint.textContent = "pre_scan_flat 适合目录较小、希望先拿到完整扫描结果再执行的场景。";
    return;
  }
  hint.textContent = "leaf_first_lazy 是默认优先推荐模式，会按顶层目录顺序逐棵子树推进，只扫描当前真正需要传的目录。";
}

function renderTasks() {
  const wrap = $("#tasks-list");
  if (!state.tasks.length) {
    wrap.innerHTML = `<div class="task-item">暂无任务。</div>`;
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
    $("#task-resolution-guide").innerHTML = `<div class="directory-empty">选择任务后显示处理建议。</div>`;
    $("#task-directory-states").innerHTML = `<div class="directory-empty">暂无目录状态。</div>`;
    $("#task-pending-tree").innerHTML = `<div class="directory-empty">暂无待补传项。</div>`;
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
  const detail = state.tasks.find((item) => item.task.id === state.selectedTaskId);
  if (!detail) {
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
    $("#task-directory-states").innerHTML = `<div class="directory-empty">暂无目录状态。</div>`;
    $("#task-pending-tree").innerHTML = `<div class="directory-empty">暂无待补传项。</div>`;
    $("#task-resolution-guide").innerHTML = `<div class="directory-empty">选择任务后显示处理建议。</div>`;
    $("#task-detail").textContent = "选择一条任务查看详情...";
    return;
  }
  const metadata = detail.plan?.metadata || {};
  const runtime = detail.runtime || metadata.runtime || {};
  $("#task-summary").innerHTML = `
    <div class="insight-card">
      <strong>执行模式</strong>
      <span>${stringifyValue(metadata.executionMode)}</span>
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
      <strong>扫描方式</strong>
      <span>${stringifyValue(metadata.scanMode, "尚未运行或无需扫描")}</span>
    </div>
    <div class="insight-card">
      <strong>风险节流</strong>
      <span>${stringifyValue(metadata.riskProfile?.requestIntervalMs, "0")}ms / dir ${stringifyValue(metadata.riskProfile?.directoryIntervalMs, "0")}ms / retry ${stringifyValue(metadata.riskProfile?.retryLimit, "0")}</span>
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
      <strong>重试摘要</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedReason || (metadata.retrySummary?.shouldBlock ? "blocked" : "ready"), "-")}</span>
    </div>
    <div class="insight-card">
      <strong>建议动作</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedAction, "-")}</span>
    </div>
  `;
  $("#task-runtime").innerHTML = renderRuntimeCheckpoint(runtime);
  $("#task-resolution-guide").innerHTML = renderTaskResolutionGuide(detail);
  $("#task-directory-states").innerHTML = renderDirectoryStates(runtime.directoryStates);
  $("#task-pending-tree").innerHTML = renderPendingTree(runtime.pendingTree);
  $("#task-detail").textContent = detail ? formatJSON(detail) : "选择一条任务查看详情...";
  wireTaskResolutionGuide(detail);
}

function renderPreview() {
  if (!state.preview) {
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
  $("#plan-preview-meta").innerHTML = `
    <div class="insight-card">
      <strong>当前模式</strong>
      <span>${stringifyValue(metadata.executionMode)}</span>
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
      <span>${stringifyValue(metadata.riskProfile?.requestIntervalMs, "0")}ms / dir ${stringifyValue(metadata.riskProfile?.directoryIntervalMs, "0")}ms / retry ${stringifyValue(metadata.riskProfile?.retryLimit, "0")}</span>
    </div>
  `;
  $("#plan-preview").textContent = formatJSON(state.preview);
}

function renderStatus() {
  const evidence = state.evidence || {
    totalTasks: 0,
    completedTasks: 0,
    blockedTasks: 0,
    failedResultCount: 0,
    doneResultCount: 0,
    skippedResultCount: 0,
    pendingResultCount: 0,
    riskHitCount: 0,
    blockedActions: [],
    recentResults: [],
    recentProbes: [],
  };
  $("#evidence-summary").innerHTML = `
    <div class="metric"><span>Total Tasks</span><strong>${evidence.totalTasks}</strong></div>
    <div class="metric"><span>Completed</span><strong>${evidence.completedTasks}</strong></div>
    <div class="metric"><span>Blocked Tasks</span><strong>${evidence.blockedTasks}</strong></div>
    <div class="metric"><span>Done Results</span><strong>${evidence.doneResultCount}</strong></div>
    <div class="metric"><span>Skipped Results</span><strong>${evidence.skippedResultCount}</strong></div>
    <div class="metric"><span>Pending Manual</span><strong>${evidence.pendingResultCount}</strong></div>
    <div class="metric"><span>Failed Results</span><strong>${evidence.failedResultCount}</strong></div>
    <div class="metric"><span>Risk Hits</span><strong>${evidence.riskHitCount}</strong></div>
  `;
  $("#blocked-actions-summary").innerHTML = renderBlockedActionsSummary(evidence.blockedActions || []);

  $("#status-table").innerHTML = `
    <table>
      <thead>
        <tr>
          <th>Provider</th>
          <th>Profiles</th>
          <th>Tasks</th>
          <th>Completed</th>
          <th>Execution Mode</th>
          <th>Scan Mode</th>
          <th>Risk Mode</th>
          <th>Latest Probe</th>
          <th>Last Task State</th>
          <th>Blocked</th>
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
                <td>${item.profileCount}</td>
                <td>${item.taskCount}</td>
                <td>${item.completedCount}</td>
                <td>${stringifyValue(item.snapshotSummary?.executionMode)}</td>
                <td>${stringifyValue(item.snapshotSummary?.scanMode)}</td>
                <td>${stringifyValue(item.snapshotSummary?.riskProfile?.mode)}</td>
                <td>${item.latestProbe || "-"}</td>
                <td>${item.lastTaskState || "-"}</td>
                <td>${stringifyValue(item.blockedCount, "0")}</td>
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
  $("#status-runtime-checkpoints").innerHTML = renderRuntimeCheckpoint(runtimePayload?.runtime || runtimePayload);
  $("#status-directory-states").innerHTML = renderDirectoryStates(runtimePayload?.runtime?.directoryStates || runtimePayload?.directoryStates);
  $("#status-pending-tree").innerHTML = renderPendingTree(runtimePayload?.runtime?.pendingTree || runtimePayload?.pendingTree);
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
      <div><strong>retryBlocked</strong> <code>${escapeHTML(stringifyValue(retrySummary.blockedReason, "-"))}</code></div>
      <div><strong>blockedAction</strong> <code>${escapeHTML(stringifyValue(retrySummary.blockedAction, "-"))}</code></div>
      <div><strong>blockedTop</strong> <code>${escapeHTML(stringifyValue(blockedActions[0]?.action, "-"))}</code></div>
      <div><strong>nextRetryAt</strong> <code>${escapeHTML(stringifyValue(retrySummary.nextRetryAt, "-"))}</code></div>
      <div><strong>queueSize</strong> <code>${escapeHTML(stringifyValue(retrySummary.queueSize, "0"))}</code></div>
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
  const evidence = await api("/api/evidence/runtime");
  const statuses = await api("/api/status/providers");
  state.evidence = evidence;
  state.statuses = statuses.items || [];
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
  $("#plan-execution-mode").addEventListener("change", syncExecutionModeHint);

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
  return {
    sourceProvider: $("#plan-source-provider").value,
    sourceProfileId: $("#plan-source-profile").value,
    targetProvider: $("#plan-target-provider").value,
    targetProfileId: $("#plan-target-profile").value,
    thresholdMB: Number($("#plan-threshold").value || "0"),
    riskMode: $("#plan-risk-mode").value,
    riskOverride: parseJSONInput($("#plan-risk-override").value, null),
    executionMode: $("#plan-execution-mode").value,
    conflictPolicy: $("#plan-conflict-policy").value,
    selectedRoots: parseJSONInput($("#plan-selected-roots").value, []),
    entries: parseJSONInput($("#plan-entries").value, []),
  };
}

function wirePlanner() {
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

async function runTaskAction(action) {
  if (!state.selectedTaskId) {
    showFlash("请先选择任务", true);
    return;
  }
  try {
    await api(`/api/tasks/${state.selectedTaskId}/${action}`, { method: "POST" });
    showFlash(`任务 ${action} 已执行`);
    await Promise.all([loadTasks(), loadStatus()]);
  } catch (error) {
    showFlash(error.message, true);
  }
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
}

async function init() {
  setupTabs();
  wireLogin();
  wireProfiles();
  wirePlanner();
  wireTasks();
  wireStatus();
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
