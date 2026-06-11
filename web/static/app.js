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
    sampleType: "",
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
  directoryBrowsers: {
    source: { currentPath: "/", items: [], loading: false, lastLoadedPath: "", selectedPath: "" },
    target: { currentPath: "/", items: [], loading: false, lastLoadedPath: "", selectedPath: "" },
  },
  authAssistDiscovery: null,
};

const treeGroupsStorageKey = "cloudpan_console_tree_groups_collapsed";
const authAssistStorageKey = "cloudpan_console_auth_assist";
const languageStorageKey = "cloudpan_console_language";

const translations = {
  "zh-CN": {
    page_title: "CloudPan Sync Go 控制台",
    session_logged_in: "已登录",
    session_logged_out: "未登录",
    flash_login_success: "登录验证成功",
    flash_logout_success: "本地控制台会话已清理",
    hero: {
      eyebrow: "CloudPan Sync Go",
      title: "多网盘互传控制台，围绕任务、状态和证据做日常操作。",
      lede1: "当前页面覆盖登录、网盘授权、任务向导、任务列表、状态矩阵与运行证据，适合统一管理多网盘之间的同步任务。",
      lede2: "当前控制台会直接展示叶子目录优先、后台补传，以及“源端删除仅记录、不默认删除目标端”等核心同步语义，方便查看任务策略和处理结果。"
    },
    badge: { local_console: "本地控制台" },
    language: { label: "界面语言" },
    tabs: {
      login: "登录",
      providers: "网盘源 / 授权",
      wizard: "任务向导",
      tasks: "任务列表详情",
      status: "状态矩阵 / 证据"
    },
    login: {
      title: "登录验证",
      desc: "当前登录只调用 `POST /api/session/login` 做控制台访问确认，不依赖旧 Python 会话模型。",
      password_label: "管理员密码",
      password_placeholder: "默认是 admin",
      submit: "验证登录",
      logout: "退出本地会话",
      constraints_title: "当前约束",
      constraint_api: "前端只走 Go API",
      constraint_python: "不兼容 Python 旧页面",
      constraint_entry: "适合作为后续真实联调入口",
      waiting: "等待登录验证..."
    },
    common: {
      refresh: "刷新"
    },
    tasks: {
      list_title: "任务列表",
      detail_title: "任务详情",
      run: "运行",
      pause: "暂停",
      resume: "继续",
      retry: "重试",
      execution_mode: "执行模式",
      waiting_selected: "选择任务后显示",
      prefill_wizard: "按当前任务重建向导",
      copy_payload: "复制任务创建参数",
      runtime_checkpoint: "运行检查点",
      flash_auto_recover_filters_cleared: "后台补传筛选已清空",
      retry_queue: "重试队列",
      retry_queue_desc: "按失败分类查看可重试、阻塞中、已耗尽的队列项。",
      retry_current_filter: "重试当前筛选",
      auto_recover_current_filter: "后台补传当前筛选",
      retry_filter_query_placeholder: "筛选路径 / 原因 / 网盘源状态",
      all_classes: "全部分类",
      all_states: "全部状态",
      retry_class_pending_manual: "待人工确认（pending_manual）",
      retry_class_provider_session_missing: "缺少会话字段（provider_session_missing）",
      retry_class_rate_limited: "触发限流（rate_limited）",
      retry_class_auth_expired: "授权已失效（auth_expired）",
      retry_class_local_file_missing: "本地文件缺失（local_file_missing）",
      retry_state_retryable: "可立即重试（retryable）",
      retry_state_blocked: "阻塞中（blocked）",
      retry_state_exhausted: "已耗尽（exhausted）",
      waiting_task_data: "等待任务数据...",
      next_steps: "下一步处理",
      next_steps_desc: "根据阻塞动作给出处理步骤。",
      waiting_task_resolution: "选择任务后显示处理建议。",
      detail_waiting: "选择一条任务查看详情...",
      selected_roots: "选定根目录",
      target_root: "目标根目录",
      recommended_mode: "推荐模式",
      recommended_reason: "推荐原因",
      scan_mode: "扫描方式",
      risk_throttle: "风险节流",
      recommended_risk: "推荐风控",
      recommended_risk_reason: "推荐风控原因",
      aggressive_warning: "激进风险提示",
      source_delete_policy: "源端删除策略",
      risk_resolution: "风险模板解释",
      retry_window: "自动补传时间窗",
      retry_scope: "重试范围",
      retry_mode: "重试模式",
      retry_source: "重试来源",
      retry_selected_paths: "重试选中路径",
      retry_selected_path_count: "重试路径数",
      retry_checkpoint_count: "重试断点数（checkpoint）",
      retry_summary: "重试摘要",
      next_summary: "下一步摘要",
      source_deletion_count: "源端删除记录",
      suggested_action: "建议动作",
      auto_recover_candidate: "后台补传候选",
      queue_breakdown: "队列拆分",
      scan_trace: "扫描轨迹",
      pending_only_with_count: "仅待处理项 ({count} 项)",
      full_task: "整个任务",
      runtime_empty: "暂无运行时信息",
      runtime_state: "执行状态",
      pause_request: "暂停请求",
      requested_at: "请求时间",
      current_root: "当前根目录",
      current_directory: "当前目录",
      last_completed: "上次完成",
      progress: "处理进度",
      result_count: "结果计数",
      risk_hits: "风控命中",
      retry_queue_label: "重试队列",
      blocked_reason: "阻塞原因",
      handling_action: "处理动作",
      handling_advice: "处理建议",
      blocked_summary: "阻塞摘要",
      next_auto_recover: "下次自动补传",
      auto_recover_advice: "自动补传提示",
      wait_auth_refresh: "恢复等待 - 刷新授权",
      wait_local_restore: "恢复等待 - 补回本地文件",
      wait_provider_session: "恢复等待 - 网盘会话",
      wait_manual_confirmation: "恢复等待 - 手动确认",
      wait_retry_limit: "恢复等待 - 重试上限",
      wait_retry_window: "恢复等待 - 时间窗",
      task_next_steps_idle: "当前任务没有需要人工处理的阻塞动作（blocked），可直接继续运行或观察状态矩阵。",
      task_next_steps_title: "下一步处理",
      auto_recover_takeover_title: "等待后台自动补传接管",
      upload_checkpoint_resume_title: "等待上传会话自动续跑",
      auto_recover_takeover_step_1: "当前队列满足后台自动补传条件，系统随后会自动尝试继续执行。",
      auto_recover_takeover_step_2: "先到状态矩阵查看后台补传候选池，确认该任务是否已经进入重试队列自动补传通道（retry_queue_auto_retry）。",
      auto_recover_takeover_step_3: "如果长时间未自动推进，再检查重试摘要（retrySummary）、网盘返回状态和风险时间窗是否把它留在等待态。",
      upload_checkpoint_resume_step_1: "当前失败队列携带可恢复的上传断点（upload checkpoint），单机执行进程会优先尝试恢复上传会话。",
      upload_checkpoint_resume_step_2: "先到状态矩阵查看后台补传候选池，确认该任务是否已经进入上传断点自动续跑通道（upload checkpoint auto-resume）。",
      upload_checkpoint_resume_step_3: "如果长时间未自动推进，再检查 providerData、uploadId、nextPartNumber 等恢复线索是否完整。",
      auto_recover_takeover_default_advice: "当前失败队列都带可恢复的上传断点（upload checkpoint），单机执行进程会优先尝试恢复上传会话。",
      focus_auto_recover_only: "只看自动补传候选",
      focus_checkpoint_only: "只看自动续跑候选",
      open_status_matrix: "打开状态矩阵"
      ,
      deletion_summary: "删除记录摘要",
      prefill_from_deletion: "用此删除记录重建向导",
      prefill_all_deletions: "按全部删除记录重建向导",
      deletion_summary_desc: "{count} 条，默认只记录，不会自动删除目标端真实文件。",
      no_deletion_samples: "暂无可展开样本。",
      more_deletion_samples: "还有 {count} 条未展开。",
      reason_label: "原因",
      deleted_at_label: "删除时间",
      directory_states: "目录状态",
      pending_tree: "待补传树",
      collapse_all: "收起全部",
      expand_all: "展开全部",
      prefill_current_filter: "按当前筛选重建向导",
      copy_current_paths: "复制当前路径",
      directory_filter_placeholder: "筛选路径 / last item",
      pending_filter_placeholder: "筛选路径 / 文件名",
      pending_reason_placeholder: "筛选原因 / 网盘状态",
      all_statuses: "全部状态",
      status_pending: "待处理（pending）",
      status_running: "执行中（running）",
      status_blocked: "阻塞中（blocked）",
      status_completed: "已完成（completed）",
      leaf_only: "仅叶子节点",
      problem_only: "仅问题节点",
      clear_filters: "清空筛选",
      waiting_directory_data: "等待任务数据...",
      waiting_status_directory_data: "等待状态数据...",
      no_directory_states: "暂无目录状态。",
      no_pending_items: "暂无待补传项。",
      no_pending_nodes: "暂无待补传节点。",
      rebuild_from_current_path: "按当前路径重建向导",
      retry_current_path: "重试当前路径",
      auto_recover_current_path: "后台补传当前路径",
      copy_current_subtree: "复制当前子树",
      focus_parent: "只看父级",
      focus_current_path: "只看当前路径",
      sync_to_tree: "同步到{label}",
      sync_to_pending_tree: "待补传树",
      sync_to_directory_tree: "目录树",
      expand_subtree: "展开子树",
      collapse_subtree: "收起子树",
      rebuild_from_root: "按当前 root 重建向导",
      retry_current_root: "重试当前 root",
      auto_recover_current_root: "后台补传当前 root",
      focus_root: "只看 root",
      sync_other_tree: "同步另一棵树",
      expand: "展开",
      collapse: "收起",
      recent_directory_states: "最近目录状态",
      recent_pending_tree: "最近待补传树"
      ,
      no_tree_nodes: "暂无{label}。",
      showing_all_tree_nodes: "显示全部 {visible} 个{label}。{suffix}",
      showing_filtered_tree_nodes: "当前显示 {visible} / {total} 个{label}。{suffix}",
      retry_queue_empty: "当前没有需要后续重试的队列项。",
      retry_queue_none: "当前没有重试队列项。",
      retry_queue_current: "当前显示",
      retry_queue_retryable_now: "可立即重试",
      retry_queue_blocked: "阻塞",
      retry_queue_exhausted: "耗尽",
      retry_queue_filtered_empty: "筛选后没有命中的重试队列项。",
      focus_pending_tree: "定位待补传树",
      flash_task_tree_focused_by_scan: "已按扫描轨迹定位任务目录树",
      flash_task_tree_focused_by_root: "已按选定根目录定位任务目录树",
      flash_status_tree_focused_by_scan: "已按最近扫描轨迹定位状态目录树",
      flash_status_tree_focused_by_root: "已按选定根目录定位状态目录树",
      flash_task_directory_filters_cleared: "已清空任务目录树筛选",
      flash_task_pending_filters_cleared: "已清空任务待补传筛选",
      flash_status_directory_filters_cleared: "已清空状态目录树筛选",
      flash_status_pending_filters_cleared: "已清空状态待补传筛选",
      flash_no_visible_tree_paths: "当前{label}没有可复制的路径",
      flash_copied_tree_paths: "已复制 {count} 条{label}路径",
      flash_tree_subtree_not_found: "当前筛选结果里未找到对应子树",
      flash_tree_subtree_empty: "当前子树没有可复制的路径",
      flash_copied_tree_subtree_paths: "已复制 {count} 条{label}子树路径",
      flash_tree_already_top: "当前已经是最上层路径",
      flash_tree_focused_by_parent: "已按父级路径 {path} 收敛{label}",
      flash_tree_focused_by_path: "已按 {path} 收敛{label}",
      flash_task_pending_focused_by_retry: "已按当前重试项定位待补传树",
      flash_status_pending_focused_by_retry: "已按当前重试项定位最近待补传树",
      flash_task_retry_class_focused: "已收敛到当前同类重试队列",
      flash_status_retry_class_focused: "已收敛到最近同类重试队列",
      flash_task_retry_blocked_action_focused: "已按当前阻塞动作收敛任务重试队列",
      flash_task_pending_not_found: "当前任务没有可定位的待补传路径",
      flash_task_pending_focused: "已按当前任务定位待补传树",
      flash_blocked_task_missing: "当前阻塞摘要里没有可用样本任务",
      flash_blocked_task_not_found: "未找到对应样本任务，可能已被清理",
      flash_blocked_task_opened: "已打开阻塞摘要对应的样本任务",
      flash_status_retry_blocked_action_focused: "已按 blocked action 收敛最近重试队列",
      flash_retry_selection_rebuilt: "已按 {source}（{scope}）重建 {count} 条路径的重试范围",
      flash_retry_single_path_rebuilt: "已按 {path}（{scope}）重建重试范围",
      flash_tasks_refreshed: "任务列表已刷新",
      flash_status_refreshed: "状态矩阵已刷新",
      flash_provider_smoke_saved: "网盘样本记录已保存",
      flash_report_refreshed: "验收报告已刷新",
      flash_report_saved: "验收报告已保存",
      flash_report_downloaded: "验收报告已下载",
      flash_report_switched: "已切换验收报告",
      flash_smoke_markdown_switched: "已切换样本文档",
      flash_smoke_markdown_downloaded: "样本文档已下载",
      flash_smoke_markdown_download_failed: "下载样本文档失败：{status}",
      flash_auto_recover_decision_previewed: "已按决策预演后台补传：{taskId}",
      flash_auto_recover_decision_executed: "已按决策执行后台补传：{taskId}",
      flash_auto_recover_decision_state_focused: "已按决策状态 {state} 收敛后台补传候选",
      flash_auto_recover_decision_lane_focused: "已按决策通道（lane）收敛后台补传候选：{lane}",
      flash_auto_recover_decision_budgets_applied: "已按决策采用建议预算：模式 {mode} / 通道 {lane} / 协议族 {group} / 网盘源 {provider} / 授权档案 {profile}",
      flash_auto_recover_mode_focused: "已按 {mode} 收敛后台补传候选",
      flash_auto_recover_protocol_group_focused: "已按协议族 {protocolGroup} 收敛后台补传候选",
      flash_auto_recover_state_focused: "已按执行状态 {state} 收敛后台补传候选",
      flash_auto_recover_retry_class_focused: "已按主失败类型 {retryClass} 收敛后台补传候选",
      flash_auto_recover_primary_blocked_action_focused: "已按主阻塞动作 {blockedAction} 收敛后台补传候选",
      flash_auto_recover_blocked_action_focused: "已按 {blockedAction} 收敛后台补传候选",
      flash_auto_recover_lane_focused: "已按通道（lane）收敛后台补传候选：{lane}",
      flash_auto_recover_budgets_applied: "已采用建议预算：模式 {mode} / 通道 {lane} / 协议族 {group} / 网盘源 {provider} / 授权档案 {profile}",
      focus_same_retry_class: "只看同类队列",
      result_count_compact: "完成 {done} / 跳过 {skipped} / 失败 {failed}",
      retry_queue_compact: "可重试 {retryable} / 阻塞 {blocked}",
      action_refresh_auth_profile: "刷新授权后继续",
      action_restore_local_source_file: "补回本地文件后继续",
      action_manual_intervention_required: "修复 provider 会话后继续",
      action_wait_for_cooldown: "等待冷却到 {time}",
      action_wait_for_cooldown_fallback: "等待冷却结束后继续",
      action_wait_for_retry_window: "等待时间窗到 {time}",
      action_wait_for_retry_window_fallback: "等待时间窗开放后继续",
      action_manual_confirmation_required: "人工确认后继续",
      action_review_and_reset_retry_strategy: "调整重试策略后继续",
      guide_refresh_auth_title: "刷新授权档案",
      guide_refresh_auth_step_1: "切到“网盘源 / 授权”面板，定位当前目标端授权档案。",
      guide_refresh_auth_step_2: "更新 token / cookie 后先执行授权验证，确认授权恢复正常。",
      guide_refresh_auth_step_3: "回到任务详情页，再执行继续或重试。",
      guide_restore_local_title: "补回本地回退文件",
      guide_restore_local_step_1: "先补回源文件或校正本地兜底路径，确保 localPath 对应文件真实存在。",
      guide_restore_local_step_2: "如果路径配置有误，建议回到任务向导核对 entries / selectedRoots。",
      guide_restore_local_step_3: "补齐后返回任务详情页重新重试。",
      guide_manual_intervention_title: "修复 provider 会话缺口",
      guide_manual_intervention_step_1: "当前重试分类（retryClass）是 provider_session_missing，说明网盘返回体缺少 uploadid、upload session 这类关键会话字段。",
      guide_manual_intervention_step_2: "先核对网盘返回体、上传会话构建逻辑和目标端授权档案，确认是否需要重新生成会话或刷新授权。",
      guide_manual_intervention_step_3: "修复后回到状态矩阵，确认这类阻塞项已经收敛，再执行重试。",
      guide_wait_cooldown_title: "等待冷却窗口结束",
      guide_wait_cooldown_step_1: "当前最早自动补传时间是 {time}。",
      guide_wait_cooldown_step_1_fallback: "当前处于风控冷却窗口。",
      guide_wait_cooldown_step_2: "冷却期间无需手动重试，系统会在窗口结束后自动尝试补传。",
      guide_wait_cooldown_step_3: "如果想确认整体阻塞分布，可切到状态矩阵查看阻塞聚合看板。",
      guide_wait_window_title: "等待自动补传时间窗",
      guide_wait_window_step_1: "当前下一次允许自动补传的时间是 {time}。",
      guide_wait_window_step_1_fallback: "当前不在允许的自动补传时间窗内。",
      guide_wait_window_step_2: "这类任务仍会留在自动补传候选池里，但在时间窗开始前不会被后台执行进程实际执行。",
      guide_wait_window_step_3: "如果需要排查影响范围，可切到状态矩阵按阻塞动作（blocked action）或通道（lane）直接聚焦。",
      guide_manual_confirmation_title: "等待人工确认",
      guide_manual_confirmation_step_1: "当前任务存在 pending_manual 项，说明网盘仍需要人工确认，或要等待后续兜底运行能力补齐。",
      guide_manual_confirmation_step_2: "先在状态矩阵和待补传树里确认影响范围，再决定是否拆分任务或等待后续链路补齐。",
      guide_manual_confirmation_step_3: "确认后再回到任务详情执行重试。",
      guide_review_retry_title: "调整重试策略",
      guide_review_retry_step_1: "当前任务已经达到重试上限（retryLimit），继续原样重试也不会再推进。",
      guide_review_retry_step_2: "建议回到任务向导调整 riskOverride / retryLimit / 执行策略，必要时拆成更小任务。",
      guide_review_retry_step_3: "创建新任务后，用状态矩阵对比新的阻塞分布是否收敛。",
      guide_open_providers: "打开授权面板",
      guide_focus_auth_queue: "只看授权失效队列",
      guide_open_status_blocked: "按当前阻塞打开状态矩阵",
      guide_focus_local_missing: "只看本地缺失队列",
      guide_open_wizard: "打开任务向导",
      guide_focus_session_missing: "只看会话缺口队列",
      guide_focus_cooldown: "只看冷却队列",
      guide_focus_window: "只看时间窗等待态",
      guide_focus_current_retry: "只看当前任务重试队列",
      guide_focus_manual_pending: "只看待确认队列",
      guide_stay_task_detail: "留在任务详情",
      guide_focus_exhausted: "只看已耗尽队列",
      guide_manual_fallback_title: "人工处理建议",
      guide_manual_fallback_step: "请根据阻塞原因检查授权、源文件和网盘返回状态。"
    },
    status: {
      summary_title: "运行证据摘要",
      waiting_retry_policy: "等待自动补传调度策略...",
      auto_recover_pool: "自动补传候选池",
      auto_recover_pool_desc: "按后台补传模式聚合当前可自动接管或等待窗口的任务",
      all_modes: "全部模式",
      all_strategies: "全部策略",
      all_protocol_groups: "全部协议族",
      all_providers: "全部网盘源",
      all_profiles: "全部授权档案",
      all_retry_classes: "全部失败类型",
      all_blocked_actions: "全部阻塞动作",
      all_execution_states: "全部执行状态",
      auto_recover_mode_upload_checkpoint_auto_resume: "上传会话自动续跑（upload_checkpoint_auto_resume）",
      auto_recover_mode_retry_queue_auto_retry: "重试队列自动补传（retry_queue_auto_retry）",
      auto_recover_mode_retry_window_waiting_auto_retry: "时间窗等待后自动补传（retry_window_waiting_auto_retry）",
      auto_recover_mode_cooldown_elapsed_auto_retry: "冷却结束后自动补传（cooldown_elapsed_auto_retry）",
      strategy_fast_upload: "快速上传（fast_upload）",
      strategy_download_upload: "下载后上传（download_upload）",
      strategy_pending_manual: "待人工确认（pending_manual）",
      retry_class_rate_limited: "触发限流（rate_limited）",
      retry_class_pending_manual: "待人工确认（pending_manual）",
      retry_class_provider_session_missing: "缺少会话字段（provider_session_missing）",
      retry_class_auth_expired: "授权已失效（auth_expired）",
      retry_class_local_file_missing: "本地文件缺失（local_file_missing）",
      retry_class_retry_failed: "重试仍失败（retry_failed）",
      state_runnable_now: "可直接放行（runnable_now）",
      state_waiting_cooldown: "等待冷却（waiting_cooldown）",
      state_waiting_retry_window: "等待时间窗（waiting_retry_window）",
      state_waiting_auth_refresh: "等待授权刷新（waiting_auth_refresh）",
      state_waiting_local_restore: "等待本地恢复（waiting_local_restore）",
      state_waiting_provider_session: "等待会话恢复（waiting_provider_session）",
      state_waiting_manual_confirmation: "等待人工确认（waiting_manual_confirmation）",
      state_waiting_retry_limit: "等待重试上限（waiting_retry_limit）",
      state_waiting_other: "其它等待（waiting_other）",
      auto_recover_limit_placeholder: "本轮上限，例如 3",
      auto_recover_limit_per_mode_placeholder: "模式预算，例如 1",
      auto_recover_limit_per_lane_placeholder: "通道预算，例如 1",
      auto_recover_limit_per_protocol_group_placeholder: "协议族预算，例如 1",
      auto_recover_limit_per_provider_placeholder: "网盘源预算，例如 1",
      auto_recover_limit_per_profile_placeholder: "账号预算，例如 1",
      preview_filtered: "预演当前筛选",
      run_filtered: "执行当前筛选",
      clear_filters: "清空筛选",
      auto_recover_all: "显示全部后台补传候选。",
      waiting_budget_summary: "等待当前补传预算摘要...",
      no_auto_recover_result: "尚未执行后台补传预演或实际放行。",
      protocol_coverage: "协议族覆盖",
      protocol_coverage_desc: "按协议族聚合真实成功样本",
      report_title: "验收报告",
      report_title_placeholder: "报告标题，默认使用标准标题",
      report_note_placeholder: "报告备注，写交接要点或里程碑说明",
      waiting_report_data: "等待报告数据...",
      save: "保存",
      download_markdown: "下载样本文档",
      provider_smoke_title: "真实网盘样本",
      provider_smoke_summary_desc: "按协议族聚合真实样本",
      provider_smoke_matrix_desc: "真实样本矩阵，合并样本记录与协议族覆盖情况。",
      provider_smoke_filter_query_placeholder: "筛选记录标题 / 网盘源 / 备注",
      provider_smoke_filter_group_placeholder: "筛选协议族",
      provider_smoke_filter_sample_placeholder: "筛选样本类型 / 复用优先级",
      all_results: "全部结果",
      waiting_smoke_records: "等待样本记录...",
      provider_smoke_provider_key_placeholder: "网盘源标识，例如 123_open",
      provider_smoke_protocol_group_placeholder: "协议族，例如 aliyun_123_open",
      provider_smoke_auth_mode_placeholder: "授权方式，例如 manual_token",
      provider_smoke_title_placeholder: "记录标题，默认使用网盘源名称",
      provider_smoke_note_placeholder: "备注，例如 已验证 ValidateAuth / List / Metadata",
      provider_smoke_operations_placeholder: "操作清单，用逗号分隔，例如 ValidateAuth, List, Upload",
      smoke_category_auto: "自动分类",
      smoke_category_auth_only: "仅授权验证（auth_only）",
      smoke_category_browse_only: "仅浏览验证（browse_only）",
      smoke_category_fast_upload_success: "快速上传成功（fast_upload_success）",
      smoke_category_binary_upload_success: "二进制上传成功（binary_upload_success）",
      smoke_category_partial_blocked: "部分阻塞（partial_blocked）",
      smoke_category_failed: "失败（failed）",
      result_success: "成功（success）",
      result_failure: "失败（failure）",
      provider_matrix: "网盘源状态矩阵",
      provider_matrix_title: "网盘源状态矩阵",
      runtime_checkpoints: "运行检查点概览",
      runtime_overview_title: "运行检查点概览",
      from_recent_probe: "来自最近探测 / 快照",
      runtime_overview_desc: "来自最近探测 / 快照",
      recent_retry_queue: "最近重试队列",
      retry_queue_title: "最近重试队列",
      retry_filtered: "重试当前筛选",
      auto_recover_filtered: "后台补传当前筛选",
      waiting_status_data: "等待状态数据...",
      recent_probe: "最近探测",
      recent_probe_title: "最近探测",
      recent_results: "最近结果",
      recent_results_title: "最近结果",
      total_tasks: "任务总数",
      completed_tasks: "已完成任务",
      blocked_tasks: "阻塞任务",
      execution_mode: "执行模式",
      scan_mode: "扫描模式",
      source_delete_policy: "源端删除策略",
      auto_recover_tasks: "自动补传任务",
      runnable_now: "可直接放行",
      waiting_cooldown: "等待冷却",
      waiting_retry_window: "等待时间窗",
      waiting_auth_refresh: "等待授权刷新",
      waiting_local_restore: "等待本地恢复",
      waiting_provider_session: "等待会话恢复",
      waiting_manual_confirmation: "等待人工确认",
      waiting_retry_limit: "等待重试上限",
      waiting_other: "其它等待",
      auto_retry_policy_summary_prefix: "自动补传默认调度",
      blocked_empty: "当前没有需要人工处理的阻塞汇总项。",
      auto_recover_last_preview: "最近预演",
      auto_recover_last_run: "最近执行",
      auto_recover_last_recoverable: "预演可放行",
      auto_recover_last_recovered: "已恢复",
      auto_recover_decision_empty: "最近一次后台补传预演或执行暂无决策明细。",
      auto_recover_budget_hint_empty: "预算占用：当前决策未返回可复用的预算占用信息。",
      auto_recover_budget_hint_prefix: "预算占用",
      auto_recover_waiting_advice: "等待态说明",
      focus_current_state: "只看该状态",
      focus_current_lane: "只看该通道",
      apply_suggested_budgets: "采用建议预算",
      preview_current_decision: "预演该决策",
      run_current_decision: "执行该决策",
      open_sample_task: "打开样本任务",
      acceptance_report_empty: "暂无验收报告，请先刷新或保存一份报告。",
      report_title_label: "报告标题",
      report_generated_at: "生成时间",
      report_note_label: "报告备注",
      recent_results_empty: "暂无结果证据。",
      recent_results_col_status: "状态",
      recent_results_col_mode: "模式",
      recent_results_col_execution_mode: "执行模式",
      recent_results_col_retry_mode: "重试模式",
      recent_results_col_retry_scope: "重试范围",
      recent_results_col_retry_path_count: "重试路径数",
      recent_results_col_retry_paths: "重试路径",
      recent_results_col_source_delete_policy: "源端删除策略",
      recent_results_col_recommendation: "推荐结果",
      recent_results_col_message: "消息",
      recent_results_col_risk_hit: "风控命中",
      recent_results_col_conflict: "冲突处理",
      recent_results_col_created_at: "创建时间",
      recent_probes_empty: "暂无探测证据。",
      recent_probes_col_provider: "网盘源",
      recent_probes_col_status: "状态",
      recent_probes_col_profile: "授权档案",
      recent_probes_col_execution_mode: "执行模式",
      recent_probes_col_scan_mode: "扫描模式",
      recent_probes_col_retry_mode: "重试模式",
      recent_probes_col_retry_scope: "重试范围",
      recent_probes_col_retry_path_count: "重试路径数",
      recent_probes_col_retry_paths: "重试路径",
      recent_probes_col_source_delete_policy: "源端删除策略",
      recent_probes_col_risk_hit: "风控命中",
      recent_probes_col_payload: "载荷",
      recent_probes_col_created_at: "创建时间",
      auto_recover_acceptance: "自动补传验收",
      auto_recover_fairness_summary: "自动补传恢复与公平性摘要",
      recovery_priority_action: "恢复优先动作",
      recovery_priority_action_counts: "恢复优先动作统计:",
      fairness_gap: "公平性缺口",
      fairness_priority_action: "公平性优先动作",
      waiting_reason_summary: "等待原因",
      no_auto_recover_pool_samples: "当前没有自动补传候选池样本。",
      auto_recover_filtered_empty: "当前筛选条件下没有命中的后台补传候选。",
      auto_recover_pool_empty: "当前没有进入后台补传候选池的任务。",
      lane_summary_prefix: "通道",
      sample_record_empty: "暂无真实网盘样本记录。",
      smoke_record_empty: "当前没有样本记录。",
      smoke_record_showing_all: "显示全部 {visible} 条样本记录。",
      smoke_record_showing_filtered: "当前显示 {visible} / {total} 条样本记录。",
      sample_matrix_empty: "暂无真实样本矩阵。",
      smoke_view_markdown: "查看样本文档",
      smoke_download_markdown: "下载样本文档",
      acceptance_matrix_view: "验收矩阵视图",
      acceptance_matrix_hint: "可按验收状态快速筛选，也能直接跳到对应样本记录或任务样本，方便继续补齐真实联调样本。",
      protocol_sampled: "已取样",
      protocol_pending_sample: "待取样",
      protocol_related_providers: "涉及网盘源",
      protocol_sample_context: "样本上下文",
      risk_calibration_title: "网盘源默认风控校准",
      risk_calibration_empty: "暂无网盘源默认模板校准数据，请先刷新网盘源列表。",
      risk_calibration_ready: "校准就绪",
      risk_calibration_gap_summary: "默认模板缺口速览",
      risk_calibration_missing_counts: "校准缺失字段统计",
      risk_calibration_priority_counts: "校准优先动作统计",
      risk_calibration_readiness: "校准就绪度",
      risk_calibration_priority_action: "优先校准",
      risk_calibration_window_source: "时间窗来源",
      risk_calibration_window_advice: "时间窗建议",
      risk_calibration_missing: "校准缺失",
      risk_calibration_coverage: "校准覆盖",
      risk_calibration_covered_fields: "已覆盖字段",
      risk_calibration_sample_advice: "校准样本建议",
      risk_calibration_default_action: "优先校准动作",
      provider_smoke_acceptance_title: "网盘源级真实样本验收",
      provider_smoke_acceptance_empty: "暂无网盘源级验收数据（providerSmokeProviders），请先刷新或保存新版验收报告。",
      provider_smoke_ready: "网盘就绪",
      provider_smoke_gap_summary: "网盘源验收缺口速览",
      provider_smoke_missing_basic_providers: "缺基础样本网盘源",
      provider_smoke_missing_upload_providers: "缺上传样本网盘源",
      provider_smoke_missing_anomaly_providers: "缺异常样本网盘源",
      provider_smoke_missing_representative_providers: "缺代表样本网盘源",
      provider_smoke_priority_counts: "网盘源优先动作统计",
      provider_smoke_basic_sample: "优先基础样本",
      provider_smoke_upload_sample: "优先上传样本",
      provider_smoke_anomaly_sample: "优先异常样本",
      provider_smoke_representative_sample: "优先代表样本",
      provider_smoke_representative_missing: "代表样本缺口",
      provider_smoke_representative_actions: "代表样本动作建议",
      provider_smoke_representative_advice: "代表样本补齐建议",
      provider_smoke_default_action: "网盘源优先动作",
      provider_smoke_default_action_complete: "已补齐",
      provider_smoke_readiness_ready: "已就绪（基础、上传、异常、代表样本齐）",
      provider_smoke_readiness_partial: "进行中（已有样本，仍缺验收项）",
      provider_smoke_readiness_pending: "待补齐（待补真实网盘样本）",
      provider_smoke_basic_ready: "基础",
      provider_smoke_upload_ready: "上传",
      provider_smoke_anomaly_ready: "异常",
      provider_smoke_representative_ready: "代表",
      blocked_metric_tasks: "任务",
      blocked_metric_providers: "网盘源",
      blocked_metric_next: "下次重试",
      blocked_next_step: "下一步：{value}",
      blocked_missing_path: "缺少可恢复路径",
      blocked_focus_action: "只看这一类阻塞",
      blocked_run_action: "执行此阻塞动作",
      auto_recover_run_wait_auth: "只执行等刷新授权",
      auto_recover_run_wait_local: "只执行等补源文件",
      auto_recover_run_wait_manual: "只执行等人工确认",
      auto_recover_run_wait_limit: "只执行重试耗尽",
      auto_recover_run_wait_other: "只执行其它等待",
      auto_recover_run_primary_retry: "执行主重试类型",
      auto_recover_run_primary_blocked: "执行主阻塞动作",
      auto_recover_run_lane: "执行该通道",
      auto_recover_run_mode: "执行该模式",
      matrix_open_smoke_record: "打开样本记录",
      matrix_open_task_sample: "打开任务样本",
      matrix_prefill_smoke_form: "预填样本表单",
      matrix_prefill_profile_risk: "预填账号默认风控",
      matrix_focus_group_records: "只看该组记录",
      matrix_focus_acceptance_type: "只看此类",
      selection_source_directory: "目录子集",
      selection_source_pending: "待补传子集",
      selection_source_retry: "重试队列子集",
      selection_scope_directory: "目录子集",
      selection_scope_pending: "待补传子集",
      selection_scope_retry: "重试队列子集",
      result_success: "成功（success）",
      result_failure: "失败（failure）",
      smoke_matrix_source_display: "Smoke Matrix {protocolGroup} ({status})",
      smoke_matrix_source_display_fallback: "Smoke Matrix ({status})",
      smoke_matrix_risk_profile_title: "{protocolGroup} 风控模板",
      smoke_matrix_risk_profile_title_fallback: "真实样本风控模板",
      smoke_summary_smokes: "样本数",
      smoke_summary_success: "成功",
      smoke_summary_upload: "上传成功",
      smoke_summary_failure: "失败",
      smoke_summary_providers: "网盘源",
      smoke_summary_sample: "样本记录",
      smoke_summary_providers_label: "涉及网盘源：",
      smoke_summary_sample_label: "样本：",
      smoke_summary_preferred_sample: "优先基础样本：",
      smoke_summary_preferred_upload: "优先上传样本：",
      smoke_summary_preferred_anomaly: "优先异常样本：",
      smoke_summary_preferred_representative: "优先代表样本：",
      smoke_summary_latest_smoke_at: "最近样本时间：",
      matrix_filter_all: "全部",
      matrix_filter_accepted: "已验收",
      matrix_filter_in_progress: "进行中",
      matrix_filter_pending: "待补齐",
      matrix_filter_empty: "当前筛选 {filter} 没有真实样本矩阵。",
      flash_smoke_records_filtered: "已按 {label} 收敛样本记录",
      flash_smoke_records_result_cleared: "已清空样本记录结果筛选",
      flash_smoke_records_group_cleared: "已清空样本记录协议组筛选",
      flash_smoke_records_cleared: "已清空样本记录筛选",
      flash_smoke_matrix_filtered: "已按 {label} 收敛验收矩阵",
      flash_smoke_record_opened: "已打开样本记录并回填表单",
      flash_smoke_gap_prefilled: "已按验收缺口预填样本动作",
      flash_smoke_matrix_prefilled: "已按验收矩阵预填样本表单",
      flash_smoke_profile_risk_prefilled: "已按真实样本预填账号默认风控",
      smoke_note_protocol_group: "协议族：{value}",
      smoke_note_provider: "建议网盘源：{value}",
      smoke_note_status: "验收状态：{value}",
      smoke_note_missing: "缺口：{value}",
      smoke_note_action: "动作：{value}",
      smoke_note_advice: "建议：{value}",
      smoke_note_goal: "本次目标：{value}",
      smoke_note_boundary_sample: "当前协议组已验收，建议继续补充边界场景样本。",
      smoke_markdown_load_failed: "加载样本文档失败：{status}",
      smoke_action_fill_upload_success: "补 1 条真实上传成功样本",
      smoke_action_fill_task_coverage: "补 1 条真实任务覆盖样本",
      smoke_action_fill_upload_smoke: "补上传成功样本",
      smoke_draft_title_provider: "网盘样本",
      smoke_draft_title_status: "{protocolGroup} {status} 样本",
      auto_recover_summary_title: "当前可见候选池摘要",
      auto_recover_queue_count: "队列项 {value}",
      auto_recover_ready_count: "可立即重试 {value}",
      auto_recover_cooldown_count: "冷却中 {value}",
      auto_recover_profile_counts: "授权档案 {value}",
      auto_recover_summary_compact: "{lanes} 个通道 / {tasks} 个任务",
      smoke_action_fill_task_sample: "补任务覆盖样本",
      smoke_action_fill_smoke_success: "补成功样本",
      smoke_action_prefill_by_gap: "按缺口预填动作",
      smoke_draft_label_auth_expired: "补授权失效样本",
      smoke_draft_label_rate_limited: "补限流样本",
      smoke_draft_label_local_file_missing: "补本地文件缺失样本",
      smoke_draft_label_pending_manual: "补人工确认样本",
      smoke_draft_label_large_file: "补大文件样本",
      smoke_draft_label_nested_directory: "补多层目录样本",
      smoke_draft_label_retry_recovery: "补重试恢复样本",
      smoke_draft_note_auth_expired: "目标异常：auth_expired",
      smoke_draft_note_rate_limited: "目标异常：rate_limited / risk_control",
      smoke_draft_note_local_file_missing: "目标异常：local_file_missing",
      smoke_draft_note_pending_manual: "目标异常：pending_manual / overwrite downgrade",
      smoke_draft_note_large_file: "目标代表样本：large_file / multipart / 大文件上传恢复",
      smoke_draft_note_nested_directory: "目标代表样本：nested_directory / 多层目录 / subtree",
      smoke_draft_note_retry_recovery: "目标代表样本：retry_recovery / checkpoint / resume / 续传",
      smoke_title_boundary_sample: "边界场景样本",
      protocol_coverage_empty: "当前没有协议族覆盖数据。",
      upload_checkpoint_acceptance_title: "上传断点续传默认恢复验收",
      upload_checkpoint_readiness: "断点续传就绪度：{value}",
      upload_checkpoint_resume_summary: "大文件/长链路恢复摘要",
      upload_checkpoint_tasks: "断点任务",
      upload_checkpoint_resume_tasks: "自动续传",
      upload_checkpoint_readiness_compact: "就绪度",
      upload_checkpoint_priority_action: "优先恢复动作：{value}",
      upload_checkpoint_sample_context: "样本上下文：网盘源 {provider} / 协议组 {protocolGroup} / 任务 {taskId} / 授权档案 {profileId}",
      upload_checkpoint_resume_detail: "续传详情：上传 {uploadId} / 下一分片 {nextPart} / 已上传 {uploaded}/{partCount}",
      upload_checkpoint_sample_paths: "样本路径：{value}",
      upload_checkpoint_priority_action_label: "恢复优先动作：{value}",
      waiting_reason_summary_detail: "冷却 {cooldown} / 时间窗 {retryWindow} / 授权 {authRefresh} / 本地文件 {localRestore} / 人工处理 {manual}",
      lane_summary_detail: "网盘源 {provider} / 协议组 {protocolGroup} / 授权档案 {profileId} / 网盘源数 {providerCount} / 授权档案数 {profileCount}",
      sample_context_provider: "网盘源 {value}",
      sample_context_protocol_group: "协议组 {value}",
      sample_context_task: "任务 {value}",
      sample_context_profile: "授权档案 {value}",
      matrix_smoke_count: "样本",
      matrix_upload_smoke: "上传样本",
      matrix_coverage: "任务覆盖",
      matrix_smoke_sample: "样本记录：",
      matrix_coverage_sample: "任务样本：",
      matrix_readiness: "就绪度：",
      matrix_checklist: "验收清单：",
      matrix_gaps: "缺口：",
      matrix_next_action: "下一步动作：",
      matrix_priority_action: "验收优先动作：",
      matrix_anomaly_summary: "异常样本：",
      matrix_representative_summary: "代表样本：",
      matrix_anomaly_missing: "异常样本缺口：",
      matrix_anomaly_actions: "异常样本动作：",
      matrix_anomaly_advice: "异常样本建议：",
      matrix_representative_missing: "代表样本缺口：",
      matrix_representative_actions: "代表样本动作：",
      matrix_representative_advice: "代表样本建议：",
      matrix_state_ready: "已就绪",
      matrix_state_pending: "待补齐",
      matrix_state_partial: "进行中",
      matrix_state_complete: "已完成",
      matrix_state_accepted: "已验收",
      matrix_checklist_upload: "上传",
      matrix_checklist_coverage: "覆盖",
      matrix_checklist_anomaly: "异常",
      matrix_checklist_representative: "代表",
      matrix_gap_upload: "上传",
      matrix_gap_anomaly: "异常({value})",
      matrix_gap_representative: "代表({value})",
      matrix_gap_complete: "已补齐",
      matrix_missing: "验收缺口：",
      matrix_actions: "验收动作：",
      matrix_advice: "验收建议：",
      matrix_latest_observed: "最近样本 / 覆盖观察："
    },
    providers: {
      profile_title: "授权档案",
      assist_title: "授权引导入口",
      assist_desc: "默认优先通过 OpenList 获取登录态和存储信息，失败后再切到 Alist，最后才使用手动高级模式。",
      openlist_url_label: "OpenList 地址",
      openlist_token_label: "OpenList 访问令牌",
      alist_url_label: "Alist 地址",
      alist_token_label: "Alist 访问令牌",
      url_placeholder: "例如：http://127.0.0.1:5244",
      token_placeholder: "可留空，登录后再补",
      assist_use_openlist: "优先使用 OpenList",
      assist_use_alist: "切到 Alist 兜底",
      assist_use_manual: "使用手动高级模式",
      assist_discover_openlist: "检测 OpenList",
      assist_discover_alist: "检测 Alist",
      assist_open_openlist: "打开 OpenList",
      assist_open_alist: "打开 Alist",
      assist_clear: "清空引导配置",
      assist_summary_default: "当前将优先尝试 OpenList 引导；如未配置地址或令牌，会自动提示切到 Alist 或手动模式。",
      assist_discovery_default: "检测结果会在这里显示；如果能列出可见存储，说明当前 OpenList / Alist 地址和令牌基本可用。",
      assist_discovery_empty: "{kind} 已连通，但当前没有返回可见存储。可继续确认令牌权限，或直接回到底部手动模式。",
      assist_discovery_summary: "{kind} 已连通，当前可见存储：{summary}{suffix}。",
      assist_discovery_hidden_suffix: "；其余 {count} 项未展开",
      assist_discovery_default_html: "<div>检测结果会在这里显示；如果能列出可见存储，说明当前 OpenList / Alist 地址和令牌基本可用。</div>",
      assist_discovery_name_fallback: "未命名存储",
      assist_discovery_driver_fallback: "未知驱动",
      assist_discovery_status_fallback: "状态未知",
      assist_discovery_select: "选用 {name}",
      assist_discovery_more: "仅展示前 12 项，其余 {count} 项请缩小权限范围后重试。",
      assist_discovery_choose_storage: "{kind} 已连通，请先从下方选择一个可见存储，再继续补当前网盘源需要的 Token、Cookie 或 Extra JSON。",
      provider_label: "网盘源",
      auth_mode_label: "授权方式",
      display_name_label: "显示名称",
      display_name_placeholder: "例如：189Cloud 主账号",
      token_label: "令牌 Token",
      cookie_label: "Cookie",
      extra_label: "附加配置(JSON)",
      optional_placeholder: "可留空",
      extra_placeholder: "例如：{\"pwdId\":\"abcd\"}",
      auth_guide_default: "选择网盘源和授权方式后，这里会提示当前常见必填项、可留空项和 Extra JSON 示例。",
      risk_title: "账号默认风控模板",
      risk_desc: "可选。保存到 `extra.riskDefaults`，作为该授权档案默认风控。",
      risk_keywords_placeholder: "rate_limit,captcha,profile_limit",
      risk_sync: "同步到 Extra JSON",
      risk_clear: "清空账号默认风控",
      submit_create: "创建授权档案",
      submit_update: "更新授权档案",
      cancel_edit: "取消编辑",
      catalog_title: "网盘能力概览",
      catalog_detail_default: "点击任一网盘卡片，查看能力说明、默认风控模板和恢复预算。",
      saved_profiles_title: "已保存授权档案",
      saved_profiles_hint: "支持验证与删除",
      flash_use_openlist: "已切换为 OpenList 优先引导",
      flash_use_alist: "已切换为 Alist 兜底引导",
      flash_use_manual: "已切换为手动高级模式",
      flash_detect_openlist: "已检测 OpenList，可见存储 {count} 项",
      flash_detect_alist: "已检测 Alist，可见存储 {count} 项",
      flash_reset_assist: "授权引导配置已清空，已恢复 OpenList 优先",
      flash_refresh_catalog: "网盘列表已刷新",
      flash_refresh_profiles: "授权档案已刷新",
      flash_sync_risk: "账号默认风控已同步到 Extra JSON",
      flash_clear_risk: "账号默认风控已清空",
      flash_cancel_edit: "已退出授权档案编辑",
      flash_profile_created: "授权档案已创建",
      flash_profile_updated: "授权档案已更新",
      assist_discovery_not_found: "没有找到对应的发现结果，请重新检测一次",
      assist_discovery_applied: "已从 {kind} 回填存储“{name}”，可继续补当前网盘源需要的授权字段",
      assist_discovery_apply_failed: "回填发现结果失败：{error}",
      assist_open_url_required: "请先填写 {display} 地址",
      assist_open_login_page: "已打开 {display} 登录页",
      auth_guide_provider_prefix: "当前网盘源：{provider}。",
      auth_guide_bridge_default: "当前授权入口：OpenList 优先，Alist 兜底，手动模式作为最后兜底。",
      auth_guide_bridge_openlist_ready: "当前授权入口：优先通过 OpenList 辅助获取登录态、存储和目录信息。若 OpenList 不可用，再切到 Alist 或手动模式。",
      auth_guide_bridge_openlist_missing: "当前授权入口：已选 OpenList 优先，但还没填写 OpenList 地址；可先补地址，或临时切到 Alist / 手动模式。",
      auth_guide_bridge_alist_ready: "当前授权入口：当前已切到 Alist 兜底；如果 Alist 能看到目标存储，可先在 Alist 登录后再回填下方授权字段。",
      auth_guide_bridge_alist_missing: "当前授权入口：已切到 Alist 兜底，但还没填写 Alist 地址；可先补地址，或改回 OpenList / 手动模式。",
      auth_guide_bridge_manual: "当前授权入口：已切到手动高级模式，建议只在 OpenList / Alist 都不可用时使用。",
      assist_summary_openlist_ready: "当前优先走 OpenList：{url}。建议先在 OpenList 登录并确认能看到对应存储；失败后再切到 Alist 或手动模式。",
      assist_summary_openlist_missing: "当前优先走 OpenList，但还没填写 OpenList 地址。可先填写地址并在新窗口登录；若暂时没有 OpenList，再切到 Alist 兜底。",
      assist_summary_alist_ready: "当前已切到 Alist 兜底：{url}。建议先在 Alist 登录并确认能看到对应存储；如果仍拿不到字段，再回到底部手动模式。",
      assist_summary_alist_missing: "当前已切到 Alist 兜底，但还没填写 Alist 地址。可先补 Alist 地址；如果也没有 Alist，再使用手动高级模式。",
      assist_summary_manual: "当前已切到手动高级模式。请直接填写下方 Token、Cookie 和附加配置；如果后面补上 OpenList 或 Alist，也可以再切回引导模式。",
      provider_detail_load_failed: "加载网盘能力详情失败：{error}",
      provider_detail_loaded: "已加载 {provider} 网盘能力详情",
      risk_override_synced_note: "风控覆盖已同步到 JSON",
      risk_override_cleared_note: "风控覆盖已清空，将使用默认档位和网盘源校准",
      browser_source_refreshed: "源目录已刷新：{path}",
      browser_target_refreshed: "目标目录已刷新：{path}",
      browser_source_up: "已返回上级源目录：{path}",
      browser_target_up: "已返回上级目标目录：{path}",
      browser_source_opened: "已打开源目录：{path}",
      browser_target_opened: "已打开目标目录：{path}",
      browser_source_jumped: "已跳转到源目录：{path}",
      browser_target_jumped: "已跳转到目标目录：{path}"
    },
    wizard: {
      title: "任务预览 / 创建",
      source_provider: "源网盘源",
      target_provider: "目标网盘源",
      target_provider_insight_default: "选择目标网盘源后，这里会显示默认风控模板、推荐档位和恢复预算。",
      source_profile: "源授权档案",
      target_profile: "目标授权档案",
      target_profile_insight_default: "选择目标授权档案后，这里会显示账号默认风控模板，并支持一键写入任务覆盖。",
      source_directory: "源目录选择",
      target_directory: "目标目录选择",
      browser_up: "返回上级",
      browser_refresh: "刷新目录",
      browser_select_current: "使用当前目录",
      browser_current_path: "当前目录",
      browser_selection_result: "回填结果",
      browser_level_root: "当前层级：根目录",
      source_browser_selection_default: "将回填到选定根目录(JSON)",
      target_browser_selection_default: "将回填到目标根目录",
      source_browser_hint_default: "选择源网盘源和源授权档案后，这里会显示目录浏览兼容提示。",
      target_browser_hint_default: "选择目标网盘源和目标授权档案后，这里会显示目录浏览兼容提示。",
      source_browser_empty_default: "选择源网盘源和源授权档案后可浏览目录。",
      target_browser_empty_default: "选择目标网盘源和目标授权档案后可浏览目录。",
      target_browser_create_placeholder: "新建子目录名称，例如 archive-2026",
      target_browser_create: "在当前目录新建文件夹",
      risk_mode: "风控档位",
      risk_mode_balanced: "均衡（balanced）",
      risk_mode_safe: "保守（safe）",
      risk_mode_fast: "快速（fast）",
      risk_mode_custom: "自定义（custom）",
      risk_override_title: "任务级风控覆盖",
      risk_override_desc: "可选。留空时使用当前档位 + 网盘源默认校准。",
      risk_request_interval: "请求间隔(ms)",
      risk_directory_interval: "目录间隔(ms)",
      risk_page_size: "分页大小",
      risk_cooldown_seconds: "冷却时间(s)",
      risk_retry_limit: "重试次数",
      risk_max_concurrent: "最大并发",
      risk_auto_retry_start_hour: "自动补传开始时段",
      risk_auto_retry_end_hour: "自动补传结束时段",
      risk_keywords: "风险关键词",
      number_example_1200: "例如 1200",
      number_example_2000: "例如 2000",
      number_example_100: "例如 100",
      number_example_600: "例如 600",
      number_example_2: "例如 2",
      number_example_1: "例如 1",
      number_example_7: "例如 7",
      risk_keywords_placeholder: "rate_limit,captcha,forbidden",
      sync_risk_override: "同步到 JSON",
      clear_risk_override: "清空覆盖",
      risk_override_json: "风控覆盖(JSON)",
      execution_mode: "执行模式",
      execution_mode_leaf_first_lazy: "按目录逐棵推进（leaf_first_lazy）",
      execution_mode_pre_scan_flat: "先完整扫描再执行（pre_scan_flat）",
      source_delete_policy: "源端删除策略",
      source_delete_policy_record_only: "仅记录，不删目标端（record_only）",
      execution_hint_default: "默认优先推荐 `leaf_first_lazy`，适合大目录、风控敏感网盘源和按需扫描场景。",
      delete_policy_hint: "当前首版仅支持 `record_only`，用于显式记录源端删除事件，不会默认删除目标端已有文件。",
      recommendation_title_default: "等待预览推荐",
      recommendation_reason_default: "生成预览后会显示推荐模式、推荐风控与当前选择是否一致。",
      apply_recommended_execution: "采用推荐模式",
      apply_recommended_risk: "采用推荐风控",
      selected_roots: "选定根目录(JSON)",
      selected_roots_hint_default: "源目录支持手动填写。格式示例：`[\"/电影\",\"/相册/旅行\"]`；如果只同步整个根目录，可填写 `[\"/\"]`。",
      target_root: "目标根目录",
      target_root_hint_default: "目标目录支持手动填写。示例：`/归档/2026`；如果希望直接写入目标根目录，可保留 `/`。",
      target_path_mapping_hint: "上传目标路径会按“目标根目录 + 源端相对路径”生成；留空时默认写到目标端根目录。",
      threshold_mb: "阈值(MB)",
      conflict_policy: "冲突策略",
      conflict_policy_auto_rename_new: "自动重命名新文件（auto_rename_new）",
      conflict_policy_overwrite_existing: "覆盖已有文件（overwrite_existing）",
      entries_json: "条目(JSON)",
      preview_plan: "预览计划",
      create_task: "创建任务",
      preview_result: "预览结果",
      preview_current_mode: "当前模式",
      preview_waiting: "等待预览",
      preview_waiting_text: "等待预览...",
      provider_recommended_risk: "推荐风控档位",
      provider_capability_summary: "能力摘要",
      provider_default_template: "网盘源默认模板",
      risk_chain_title: "风控链路",
      risk_base_template_title: "基础模板",
      risk_calibrated_result_title: "校准结果",
      risk_profile_source_title: "账号默认来源",
      risk_source_kind_title: "来源类型",
      risk_bias_title: "偏向策略",
      risk_kind_bias_title: "来源类型 / 偏向",
      risk_profile_fields_title: "账号默认字段",
      risk_override_fields_title: "任务覆盖字段",
      risk_provider_baseline_title: "网盘源基线",
      risk_provider_calibrated_title: "网盘源校准后",
      risk_profile_applied_title: "账号默认注入",
      risk_task_override_title: "任务覆盖",
      risk_applied_title: "最终生效",
      risk_profile_source_compact_title: "来源",
      risk_profile_write_hint: "可直接写入本次任务覆盖，便于在此基础上再细调。",
      risk_profile_source_builtin: "授权档案内置账号默认风控",
      risk_profile_source_provider_default: "未配置，使用网盘源默认模板",
      risk_auto_retry_window_title: "自动补传时段",
      risk_budget_title: "恢复预算",
      risk_budget_advice_title: "预算建议",
      risk_reason_title: "校准依据",
      risk_hints_title: "风控提示",
      risk_traits_title: "风险特征",
      risk_missing_title: "校准缺失",
      risk_priority_action_title: "优先校准动作",
      risk_recommend_reason_title: "推荐说明",
      risk_summary_provider: "网盘源 {value}",
      risk_summary_profile: "账号默认 {value}",
      risk_summary_source_kind: "来源类型 {value}",
      risk_summary_bias: "偏向 {value}",
      risk_summary_profile_fields: "账号字段 {value}",
      risk_summary_override: "任务覆盖 {value}",
      risk_summary_mode: "最终档位 {value}",
      risk_sensitive_providers_title: "敏感网盘源",
      provider_default_window_source: "网盘源默认模板",
      provider_empty_window_source: "默认留空，等待账号或任务覆盖",
      recover_budget_no_advice: "未返回账号级预算建议。",
      recover_budget_sensitive: "高风险网盘源建议单账号串行推进：{budget}{reason}",
      recover_budget_profile_rotation: "建议按账号轮转控制补传并发：{budget}{reason}",
      recover_budget_default: "建议恢复预算：{budget}{reason}",
      provider_apply_default_risk: "采用网盘源推荐风控",
      provider_open_capability: "查看网盘源能力详情",
      profile_default_risk: "账号默认风控",
      profile_source: "来源",
      profile_extra_keys: "附加配置项",
      profile_enabled_fields: "已启用字段",
      profile_recover_budget: "账号恢复预算建议",
      profile_apply_default_risk: "应用账号默认到任务覆盖",
      profile_clear_default_risk: "改回账号默认",
      execution_mode_prescan: "先完整扫描再执行（pre_scan_flat）适合目录较小、希望先拿到完整扫描结果后再统一执行的场景。",
      execution_mode_leaf_first: "按目录逐棵推进（leaf_first_lazy）是默认优先推荐模式，会按顶层目录顺序逐棵子树推进，只扫描当前真正需要传的目录。",
      recommendation_no_execution_reason: "暂无执行模式推荐原因",
      recommendation_no_risk_reason: "暂无风控推荐原因",
      recommendation_execution_applied: "执行模式已采用推荐值：{mode}",
      recommendation_execution_suggested: "建议执行模式：{mode}",
      recommendation_risk_applied: "风控档位已采用推荐值：{mode}",
      recommendation_risk_suggested: "建议风控档位：{mode}",
      recommendation_execution_reason: "执行模式：{reason}",
      recommendation_risk_reason: "风控档位：{reason}",
      recommendation_warning: "提示：{warning}",
      preview_meta_current_mode: "当前模式",
      preview_meta_selected_roots: "选定根目录",
      preview_meta_target_root: "目标根目录",
      preview_meta_recommended_mode: "推荐模式",
      preview_meta_recommended_reason: "推荐原因",
      preview_meta_execution_order: "执行顺序",
      preview_meta_risk_mode: "风险档位",
      preview_meta_risk_throttle: "风险节流",
      preview_meta_recommended_risk: "推荐风控",
      preview_meta_recommended_risk_reason: "推荐风控原因",
      preview_meta_aggressive_warning: "激进风险提示",
      preview_meta_risk_resolution: "风险模板解释",
      preview_meta_retry_window: "自动补传时间窗",
      preview_meta_source_delete_policy: "源端删除策略",
      preview_meta_entry_counts: "有效条目 / 删除记录",
      preview_meta_delete_only: "删除记录仅用于定位",
      preview_meta_delete_only_hint: "当前预览只剩删除记录，没有可执行条目；请先恢复源文件并重新预览。",
      preview_meta_delete_mix_hint: "当前预览包含删除记录，它们只会用于定位，不会生成可执行条目。",
      browser_root: "根目录",
      browser_level_named: "当前层级：第 {level} 层目录（{name}）",
      browser_target_manual_fill: "将回填到目标根目录：{path}",
      browser_source_manual_fill: "将回填到选定根目录(JSON)：{path}",
      browser_choose_provider_first: "请先选择网盘源，再继续浏览目录。",
      browser_no_list_target: "{provider} 当前未声明目录浏览能力，请直接在“目标根目录”输入框手动填写目标路径。",
      browser_no_list_source: "{provider} 当前未声明目录浏览能力，请直接在“选定根目录(JSON)”输入框手动填写源目录路径。",
      browser_missing_ids: "{provider} 当前目录未返回稳定 fileId / parentId，子目录创建或部分跳转可能受限。",
      browser_load_fail_target: "{provider} 目录加载失败时，可先验证授权档案、切回根目录，或直接在“目标根目录”输入框手动填写路径。",
      browser_load_fail_source: "{provider} 目录加载失败时，可先验证授权档案、切回根目录，或直接在“选定根目录(JSON)”输入框手动填写路径。",
      browser_ready: "{provider} 已启用目录浏览，可直接点选目录回填到任务向导。",
      selected_roots_manual_supported: "也可手动填写，例如 [\"/电影\",\"/相册/旅行\"]；只同步整个根目录时填写 [\"/\"]。",
      selected_roots_manual_required: "当前源网盘源可能不支持目录浏览，建议直接手动填写 JSON 路径数组，例如 [\"/电影\",\"/相册/旅行\"]；只同步根目录时填写 [\"/\"]。",
      target_root_manual_supported: "也可手动填写目标目录，例如 /归档/2026；如果直接写入目标根目录，可保留 /。",
      target_root_manual_required: "当前目标网盘源可能不支持目录浏览，建议直接手动填写目标目录路径，例如 /归档/2026；如果直接写入目标根目录，可保留 /。",
      browser_loading: "正在加载目录列表...",
      browser_error_target: "目录加载失败：{error}。建议先刷新目录、切回根目录；如果仍失败，可直接改填“目标根目录”。",
      browser_error_source: "目录加载失败：{error}。建议先刷新目录、切回根目录；如果仍失败，可直接改填“选定根目录(JSON)”。",
      browser_empty_target: "当前目录下没有可继续浏览的子目录。你可以直接使用当前目录，或在这里新建子目录。",
      browser_empty_source: "当前目录下没有可继续浏览的子目录。你可以直接使用当前目录作为源目录。",
      browser_pill_current: "当前目录",
      browser_pill_selected: "已选目录",
      browser_open: "打开目录",
      browser_use_this: "使用此目录",
      flash_provider_no_recommended_risk: "当前网盘源没有可用的推荐风控档位",
      flash_provider_risk_applied: "已采用网盘源推荐风控：{mode}",
      flash_provider_opened: "已打开 {provider} 网盘源能力详情",
      flash_profile_no_default_risk: "当前授权档案没有账号默认风控可写入",
      flash_profile_risk_applied: "已将账号默认风控写入任务覆盖，可继续按任务单独微调",
      flash_profile_risk_cleared: "已清空任务覆盖，将回到账号默认 / 网盘源默认链路",
      flash_extra_json_parse_error: "Extra JSON 无法解析：{error}",
      flash_risk_override_synced: "风控覆盖已同步到 JSON",
      flash_risk_override_cleared: "风控覆盖已清空，将使用默认档位和网盘源校准",
      flash_source_browser_refreshed: "源目录已刷新：{path}",
      flash_target_browser_refreshed: "目标目录已刷新：{path}",
      flash_source_browser_up: "已返回上级源目录：{path}",
      flash_target_browser_up: "已返回上级目标目录：{path}",
      flash_source_browser_opened: "已打开源目录：{path}",
      flash_target_browser_opened: "已打开目标目录：{path}",
      flash_source_browser_jumped: "已跳转到源目录：{path}",
      flash_target_browser_jumped: "已跳转到目标目录：{path}",
      flash_target_root_selected: "已将 {path} 回填到“目标根目录”",
      flash_source_root_selected: "已将 {path} 回填到“选定根目录(JSON)”",
      flash_choose_target_profile_first: "请先选择目标网盘源和目标授权档案",
      flash_enter_new_dir_name: "请先输入新建目录名称",
      flash_browser_missing_parent_id: "当前目录缺少 parentId / fileId，暂时无法创建子目录",
      flash_target_dir_created: "已在 {path} 下创建目录 {name}，并自动进入该目录回填到“目标根目录”",
      flash_risk_override_parse_error: "风控覆盖 JSON 无法解析：{error}",
      flash_preview_required_execution: "请先生成计划预览",
      flash_execution_applied: "已采用推荐执行模式：{mode}",
      flash_risk_applied: "已采用推荐风控档位：{mode}",
      flash_preview_generated: "计划预览已生成",
      flash_delete_only_blocked: "当前只有删除记录，没有可执行条目；请先恢复源文件并重新预览",
      flash_task_created: "任务已创建",
      flash_task_required: "请先选择任务",
      flash_task_id_missing: "缺少任务标识",
      flash_task_action_pending: "任务操作执行中，请稍候",
      flash_task_action_executed: "任务{action}已执行",
      flash_profile_missing_edit: "未找到要编辑的授权档案",
      flash_profile_edit_loaded: "已载入授权档案编辑表单",
      flash_profile_validate_done: "授权校验完成：{status}",
      flash_profile_deleted: "授权档案已删除",
      flash_prefill_task_required: "请先选择任务",
      flash_prefill_status_task_missing: "当前状态样本没有可用任务",
      flash_prefill_task_loaded: "已按当前任务重建向导参数",
      flash_prefill_visible_path_missing: "当前筛选结果里没有可用于重建向导的路径",
      flash_retry_visible_path_missing: "当前筛选结果里没有可重试的路径",
      flash_auto_recover_visible_path_missing: "当前筛选结果里没有可后台补传的路径",
      flash_retry_invalid_path: "当前路径无效，无法重试",
      flash_copy_task_required: "请先选择任务",
      flash_copy_task_payload_done: "任务创建参数已复制到剪贴板",
      flash_copy_failed: "复制失败：{error}",
      flash_smoke_matrix_group_missing: "未找到对应协议组的验收矩阵项",
      flash_smoke_matrix_task_missing: "当前验收矩阵项没有可打开的任务样本",
      flash_copy_directory_path_failed: "复制目录路径失败：{error}",
      flash_copy_pending_path_failed: "复制待补传路径失败：{error}",
      flash_focus_profile: "已定位到当前授权档案",
      flash_focus_auto_recover_mode: "已按 {mode} 收敛后台补传候选",
      flash_open_status: "已打开状态矩阵",
      flash_prefill_task_from_detail: "已按当前任务预填任务向导参数"
    }
  },
  "en-US": {
    page_title: "CloudPan Sync Go Console",
    session_logged_in: "Signed In",
    session_logged_out: "Signed Out",
    flash_login_success: "Console login verified",
    flash_logout_success: "Local console session cleared",
    hero: {
      eyebrow: "CloudPan Sync Go",
      title: "A multi-cloud transfer console focused on tasks, status, and evidence.",
      lede1: "This page covers sign-in, provider authorization, task planning, task details, status matrix, and runtime evidence for managing sync jobs across multiple cloud drives.",
      lede2: "The console highlights core sync semantics such as leaf-first execution, background retry, and record-only source deletions so task strategy and outcomes are easier to understand."
    },
    badge: { local_console: "Local Console" },
    language: { label: "Language" },
    tabs: {
      login: "Login",
      providers: "Providers / Auth",
      wizard: "Task Wizard",
      tasks: "Task Details",
      status: "Status / Evidence"
    },
    login: {
      title: "Login Verification",
      desc: "The current sign-in flow only calls `POST /api/session/login` to confirm console access and does not depend on the legacy Python session model.",
      password_label: "Admin Password",
      password_placeholder: "Default is admin",
      submit: "Sign In",
      logout: "Clear Local Session",
      constraints_title: "Current Constraints",
      constraint_api: "Frontend only calls the Go API",
      constraint_python: "Legacy Python pages are not supported",
      constraint_entry: "Suitable as the primary entry for real integration work",
      waiting: "Waiting for login verification..."
    },
    common: {
      refresh: "Refresh"
    },
    tasks: {
      list_title: "Task List",
      detail_title: "Task Details",
      run: "Run",
      pause: "Pause",
      resume: "Resume",
      retry: "Retry",
      execution_mode: "Execution Mode",
      waiting_selected: "Shown after selecting a task",
      prefill_wizard: "Rebuild Wizard from Task",
      copy_payload: "Copy Task Payload",
      runtime_checkpoint: "Runtime Checkpoints",
      retry_queue: "Retry Queue",
      retry_queue_desc: "Review retryable / blocked / exhausted items by failure class",
      retry_current_filter: "Retry Current Filter",
      auto_recover_current_filter: "Auto Recover Current Filter",
      retry_filter_query_placeholder: "Filter by path / reason / provider status",
      all_classes: "All Classes",
      all_states: "All States",
      retry_class_pending_manual: "Pending Manual (pending_manual)",
      retry_class_provider_session_missing: "Provider Session Missing (provider_session_missing)",
      retry_class_rate_limited: "Rate Limited (rate_limited)",
      retry_class_auth_expired: "Auth Expired (auth_expired)",
      retry_class_local_file_missing: "Local File Missing (local_file_missing)",
      retry_state_retryable: "Retryable (retryable)",
      retry_state_blocked: "Blocked (blocked)",
      retry_state_exhausted: "Exhausted (exhausted)",
      waiting_task_data: "Waiting for task data...",
      next_steps: "Next Steps",
      next_steps_desc: "Suggested handling steps grouped by blocked action",
      waiting_task_resolution: "Suggestions appear after selecting a task.",
      detail_waiting: "Select a task to inspect details...",
      selected_roots: "Selected Roots",
      target_root: "Target Root",
      recommended_mode: "Recommended Mode",
      recommended_reason: "Recommendation Reason",
      scan_mode: "Scan Mode",
      risk_throttle: "Risk Throttle",
      recommended_risk: "Recommended Risk",
      recommended_risk_reason: "Recommended Risk Reason",
      aggressive_warning: "Aggressive Risk Warning",
      source_delete_policy: "Source Delete Policy",
      risk_resolution: "Risk Template Notes",
      retry_window: "Auto-Recovery Window",
      retry_scope: "Retry Scope",
      retry_mode: "Retry Mode",
      retry_source: "Retry Source",
      retry_selected_paths: "Selected Retry Paths",
      retry_selected_path_count: "Retry Path Count",
      retry_checkpoint_count: "Retry Checkpoint Count",
      retry_summary: "Retry Summary",
      next_summary: "Next Step Summary",
      source_deletion_count: "Source Deletion Records",
      suggested_action: "Suggested Action",
      auto_recover_candidate: "Auto-Recovery Candidate",
      queue_breakdown: "Queue Breakdown",
      scan_trace: "Scan Trace",
      pending_only_with_count: "Pending only ({count} items)",
      full_task: "Full task",
      runtime_empty: "No runtime information yet",
      runtime_state: "Execution State",
      pause_request: "Pause Request",
      requested_at: "Requested At",
      current_root: "Current Root",
      current_directory: "Current Directory",
      last_completed: "Last Completed",
      progress: "Progress",
      result_count: "Result Counts",
      risk_hits: "Risk Hits",
      retry_queue_label: "Retry Queue",
      blocked_reason: "Blocked Reason",
      handling_action: "Handling Action",
      handling_advice: "Handling Advice",
      blocked_summary: "Blocked Summary",
      next_auto_recover: "Next Auto-Recovery",
      auto_recover_advice: "Auto-Recovery Advice",
      wait_auth_refresh: "Recovery Wait - Auth Refresh",
      wait_local_restore: "Recovery Wait - Local Restore",
      wait_provider_session: "Recovery Wait - Provider Session",
      wait_manual_confirmation: "Recovery Wait - Manual Confirmation",
      wait_retry_limit: "Recovery Wait - Retry Limit",
      wait_retry_window: "Recovery Wait - Window",
      task_next_steps_idle: "This task has no blocked action requiring manual handling. You can continue running it or inspect the status matrix.",
      task_next_steps_title: "Next Steps",
      auto_recover_takeover_title: "Waiting for Auto-Recovery Takeover",
      upload_checkpoint_resume_title: "Waiting for Upload Session Resume",
      auto_recover_takeover_step_1: "The current queue already meets auto-recovery conditions, and the system will try to continue it on a later tick.",
      auto_recover_takeover_step_2: "Open the status matrix and confirm that this task has entered the retry_queue_auto_retry lane.",
      auto_recover_takeover_step_3: "If it still does not progress for a long time, inspect retrySummary, provider status, and the recovery window to see why it remains waiting.",
      upload_checkpoint_resume_step_1: "The current failure queue contains recoverable upload checkpoints, so the worker will try to resume the upload session first.",
      upload_checkpoint_resume_step_2: "Open the status matrix and confirm that this task has entered the upload-checkpoint auto-resume lane.",
      upload_checkpoint_resume_step_3: "If it still does not progress for a long time, inspect providerData / uploadId / nextPartNumber and other recovery clues.",
      auto_recover_takeover_default_advice: "The current failed queue already contains recoverable upload checkpoints, so the worker will try to resume the upload session first.",
      focus_auto_recover_only: "Show Auto-Recovery Only",
      focus_checkpoint_only: "Show Auto-Resume Only",
      open_status_matrix: "Open Status Matrix"
      ,
      deletion_summary: "Deletion Summary",
      prefill_from_deletion: "Rebuild Wizard from This Deletion",
      prefill_all_deletions: "Rebuild Wizard from All Deletions",
      deletion_summary_desc: "{count} records. They are recorded only and will not delete real files on the target side automatically.",
      no_deletion_samples: "No sample records to expand.",
      more_deletion_samples: "{count} more records are not expanded yet.",
      reason_label: "reason",
      deleted_at_label: "deletedAt",
      directory_states: "Directory States",
      pending_tree: "Pending Retry Tree",
      collapse_all: "Collapse All",
      expand_all: "Expand All",
      prefill_current_filter: "Rebuild Wizard from Current Filter",
      copy_current_paths: "Copy Current Paths",
      directory_filter_placeholder: "Filter by path / last item",
      pending_filter_placeholder: "Filter by path / file name",
      pending_reason_placeholder: "Filter by reason / provider status",
      all_statuses: "All Statuses",
      status_pending: "Pending (pending)",
      status_running: "Running (running)",
      status_blocked: "Blocked (blocked)",
      status_completed: "Completed (completed)",
      leaf_only: "Leaf Only",
      problem_only: "Problem Only",
      clear_filters: "Clear Filters",
      waiting_directory_data: "Waiting for task data...",
      waiting_status_directory_data: "Waiting for status data...",
      no_directory_states: "No directory states yet.",
      no_pending_items: "No pending retry items yet.",
      no_pending_nodes: "No pending retry nodes yet.",
      rebuild_from_current_path: "Rebuild Wizard from This Path",
      retry_current_path: "Retry This Path",
      auto_recover_current_path: "Auto Recover This Path",
      copy_current_subtree: "Copy This Subtree",
      focus_parent: "Focus Parent",
      focus_current_path: "Focus This Path",
      sync_to_tree: "Sync to {label}",
      sync_to_pending_tree: "Pending Retry Tree",
      sync_to_directory_tree: "Directory Tree",
      expand_subtree: "Expand Subtree",
      collapse_subtree: "Collapse Subtree",
      rebuild_from_root: "Rebuild Wizard from This Root",
      retry_current_root: "Retry This Root",
      auto_recover_current_root: "Auto Recover This Root",
      focus_root: "Focus Root",
      sync_other_tree: "Sync the Other Tree",
      expand: "Expand",
      collapse: "Collapse",
      recent_directory_states: "Recent Directory States",
      recent_pending_tree: "Recent Pending Retry Tree"
      ,
      no_tree_nodes: "No {label} yet.",
      showing_all_tree_nodes: "Showing all {visible} {label}. {suffix}",
      showing_filtered_tree_nodes: "Showing {visible} / {total} {label}. {suffix}",
      retry_queue_empty: "There are no retry queue items that need follow-up.",
      retry_queue_none: "There are no retry queue items.",
      retry_queue_current: "Currently Visible",
      retry_queue_retryable_now: "Retryable Now",
      retry_queue_blocked: "Blocked",
      retry_queue_exhausted: "Exhausted",
      retry_queue_filtered_empty: "No retry queue items match the current filters.",
      focus_pending_tree: "Focus Pending Retry Tree",
      flash_no_visible_tree_paths: "There are no visible paths to copy in {label}",
      flash_copied_tree_paths: "Copied {count} {label} paths",
      flash_tree_subtree_not_found: "The current filtered result does not contain the selected subtree",
      flash_tree_subtree_empty: "The current subtree has no paths to copy",
      flash_copied_tree_subtree_paths: "Copied {count} {label} subtree paths",
      flash_tree_already_top: "The current path is already at the top level",
      flash_tree_focused_by_parent: "Focused {label} by parent path {path}",
      flash_tree_focused_by_path: "Focused {label} by {path}",
      flash_task_pending_focused_by_retry: "Focused the task pending tree by the current retry item",
      flash_status_pending_focused_by_retry: "Focused the recent pending tree by the current retry item",
      flash_task_retry_class_focused: "Focused the current similar retry queue",
      flash_status_retry_class_focused: "Focused the recent similar retry queue",
      flash_task_retry_blocked_action_focused: "Focused the task retry queue by the current blocked action",
      flash_task_pending_not_found: "The current task has no pending path to focus",
      flash_task_pending_focused: "Focused the task pending tree by the current task",
      flash_blocked_task_missing: "The current blocked summary does not contain a usable sample task",
      flash_blocked_task_not_found: "The sample task was not found and may already have been cleaned up",
      flash_blocked_task_opened: "Opened the sample task referenced by the blocked summary",
      flash_status_retry_blocked_action_focused: "Focused the recent retry queue by blocked action",
      flash_retry_selection_rebuilt: "Rebuilt the retry scope for {count} paths from {source} ({scope})",
      flash_retry_single_path_rebuilt: "Rebuilt the retry scope for {path} ({scope})",
      flash_tasks_refreshed: "Task list refreshed",
      flash_status_refreshed: "Status matrix refreshed",
      flash_provider_smoke_saved: "Provider smoke record saved",
      flash_report_refreshed: "Acceptance report refreshed",
      flash_report_saved: "Acceptance report saved",
      flash_report_downloaded: "Acceptance report downloaded",
      flash_report_switched: "Acceptance report switched",
      flash_smoke_markdown_switched: "Smoke Markdown switched",
      flash_smoke_markdown_downloaded: "Smoke Markdown downloaded",
      flash_smoke_markdown_download_failed: "Failed to download smoke Markdown: {status}",
      flash_auto_recover_decision_previewed: "Auto-recovery decision previewed for {taskId}",
      flash_auto_recover_decision_executed: "Auto-recovery decision executed for {taskId}",
      flash_auto_recover_decision_state_focused: "Auto-recovery candidates filtered by decision state {state}",
      flash_auto_recover_decision_lane_focused: "Auto-recovery candidates filtered by decision lane: {lane}",
      flash_auto_recover_decision_budgets_applied: "Suggested decision budgets applied: mode {mode} / lane {lane} / group {group} / provider {provider} / profile {profile}",
      flash_auto_recover_mode_focused: "Auto-recovery candidates filtered by {mode}",
      flash_auto_recover_protocol_group_focused: "Auto-recovery candidates filtered by protocol group {protocolGroup}",
      flash_auto_recover_state_focused: "Auto-recovery candidates filtered by execution state {state}",
      flash_auto_recover_retry_class_focused: "Auto-recovery candidates filtered by primary failure type {retryClass}",
      flash_auto_recover_primary_blocked_action_focused: "Auto-recovery candidates filtered by primary blocked action {blockedAction}",
      flash_auto_recover_blocked_action_focused: "Auto-recovery candidates filtered by {blockedAction}",
      flash_auto_recover_lane_focused: "Auto-recovery candidates filtered by lane: {lane}",
      flash_auto_recover_budgets_applied: "Suggested budgets applied: mode {mode} / lane {lane} / group {group} / provider {provider} / profile {profile}",
      focus_same_retry_class: "Show Similar Queue",
      result_count_compact: "完成 {done} / 跳过 {skipped} / 失败 {failed}",
      retry_queue_compact: "retryable {retryable} / blocked {blocked}",
      action_refresh_auth_profile: "Continue after refreshing auth",
      action_restore_local_source_file: "Continue after restoring local files",
      action_manual_intervention_required: "Continue after fixing the provider session",
      action_wait_for_cooldown: "Wait for cooldown until {time}",
      action_wait_for_cooldown_fallback: "Continue after the cooldown ends",
      action_wait_for_retry_window: "Wait for the retry window until {time}",
      action_wait_for_retry_window_fallback: "Continue after the retry window opens",
      action_manual_confirmation_required: "Continue after manual confirmation",
      action_review_and_reset_retry_strategy: "Continue after adjusting the retry strategy",
      guide_refresh_auth_title: "Refresh Auth Profile",
      guide_refresh_auth_step_1: "Open the Providers / Auth panel and locate the current target auth profile.",
      guide_refresh_auth_step_2: "Update the token/cookie and run Validate first to confirm the auth is healthy again.",
      guide_refresh_auth_step_3: "Return to task details, then Resume or Retry.",
      guide_restore_local_title: "Restore Local Fallback Files",
      guide_restore_local_step_1: "Restore the source file or correct the local fallback path so the localPath file really exists.",
      guide_restore_local_step_2: "If the path configuration is wrong, return to the task wizard and verify entries / selectedRoots.",
      guide_restore_local_step_3: "After fixing it, return to task details and Retry again.",
      guide_manual_intervention_title: "Fix the Provider Session Gap",
      guide_manual_intervention_step_1: "The current retryClass is provider_session_missing, which means the provider response is missing key session fields such as uploadid / upload session.",
      guide_manual_intervention_step_2: "Check the provider response, upload-session construction logic, and target auth profile to confirm whether a new session or auth refresh is needed.",
      guide_manual_intervention_step_3: "After fixing it, return to the status matrix, confirm the blocked items have converged, and then Retry.",
      guide_wait_cooldown_title: "Wait for the Cooldown Window",
      guide_wait_cooldown_step_1: "The earliest auto-recovery time is {time}.",
      guide_wait_cooldown_step_1_fallback: "The task is currently inside the risk cooldown window.",
      guide_wait_cooldown_step_2: "You do not need to retry manually during cooldown. The system will try again automatically after the window ends.",
      guide_wait_cooldown_step_3: "If you want to confirm the overall blocked distribution, open the status matrix and inspect the blocked summary board.",
      guide_wait_window_title: "Wait for the Auto-Recovery Window",
      guide_wait_window_step_1: "The next allowed auto-recovery time is {time}.",
      guide_wait_window_step_1_fallback: "The task is currently outside the allowed auto-recovery window.",
      guide_wait_window_step_2: "These tasks remain in the auto-recovery pool, but the worker will not execute them before the window begins.",
      guide_wait_window_step_3: "If you need to inspect the impact range, open the status matrix and focus by blocked action or lane.",
      guide_manual_confirmation_title: "Wait for Manual Confirmation",
      guide_manual_confirmation_step_1: "The current task contains pending_manual items, which means the provider still requires manual confirmation or later fallback runtime capability.",
      guide_manual_confirmation_step_2: "First inspect the impact range in the status matrix and pending-retry tree, then decide whether to split the task or wait for later runtime support.",
      guide_manual_confirmation_step_3: "After confirming, return to task details and Retry.",
      guide_review_retry_title: "Adjust the Retry Strategy",
      guide_review_retry_step_1: "The current task has reached retryLimit, so retrying it as-is will not make progress.",
      guide_review_retry_step_2: "Return to the task wizard to adjust riskOverride / retryLimit / execution strategy, and split it into smaller tasks if needed.",
      guide_review_retry_step_3: "After creating the new task, compare the new blocked distribution in the status matrix to confirm it has converged.",
      guide_open_providers: "Open Providers",
      guide_focus_auth_queue: "Show Expired Auth Queue",
      guide_open_status_blocked: "Open Status Matrix for This Block",
      guide_focus_local_missing: "Show Local-Missing Queue",
      guide_open_wizard: "Open Task Wizard",
      guide_focus_session_missing: "Show Session-Gap Queue",
      guide_focus_cooldown: "Show Cooldown Queue",
      guide_focus_window: "Show Waiting-Window State",
      guide_focus_current_retry: "Show Current Task Retry Queue",
      guide_focus_manual_pending: "Show Pending-Manual Queue",
      guide_stay_task_detail: "Stay on Task Details",
      guide_focus_exhausted: "Show Exhausted Queue",
      guide_manual_fallback_title: "Manual Handling Suggestion",
      guide_manual_fallback_step: "Check auth, source files, and provider status according to the blocked reason."
    },
    status: {
      summary_title: "Runtime Evidence Summary",
      waiting_retry_policy: "Waiting for the auto-recovery scheduling policy...",
      auto_recover_pool: "Auto-Recovery Pool",
      auto_recover_pool_desc: "Group tasks that can be auto-taken-over or are waiting for a recovery window",
      all_modes: "All Modes",
      all_strategies: "All Strategies",
      all_protocol_groups: "All Protocol Groups",
      all_providers: "All Providers",
      all_profiles: "All Auth Profiles",
      all_retry_classes: "All Failure Types",
      all_blocked_actions: "All Blocked Actions",
      all_execution_states: "All Execution States",
      auto_recover_mode_upload_checkpoint_auto_resume: "Upload checkpoint auto resume (upload_checkpoint_auto_resume)",
      auto_recover_mode_retry_queue_auto_retry: "Retry queue auto retry (retry_queue_auto_retry)",
      auto_recover_mode_retry_window_waiting_auto_retry: "Retry after window opens (retry_window_waiting_auto_retry)",
      auto_recover_mode_cooldown_elapsed_auto_retry: "Retry after cooldown (cooldown_elapsed_auto_retry)",
      strategy_fast_upload: "Fast upload (fast_upload)",
      strategy_download_upload: "Download then upload (download_upload)",
      strategy_pending_manual: "Pending manual (pending_manual)",
      retry_class_rate_limited: "Rate limited (rate_limited)",
      retry_class_pending_manual: "Pending manual (pending_manual)",
      retry_class_provider_session_missing: "Provider session missing (provider_session_missing)",
      retry_class_auth_expired: "Auth expired (auth_expired)",
      retry_class_local_file_missing: "Local file missing (local_file_missing)",
      retry_class_retry_failed: "Retry failed (retry_failed)",
      state_runnable_now: "Runnable now (runnable_now)",
      state_waiting_cooldown: "Waiting cooldown (waiting_cooldown)",
      state_waiting_retry_window: "Waiting retry window (waiting_retry_window)",
      state_waiting_auth_refresh: "Waiting auth refresh (waiting_auth_refresh)",
      state_waiting_local_restore: "Waiting local restore (waiting_local_restore)",
      state_waiting_provider_session: "Waiting provider session (waiting_provider_session)",
      state_waiting_manual_confirmation: "Waiting manual confirmation (waiting_manual_confirmation)",
      state_waiting_retry_limit: "Waiting retry limit (waiting_retry_limit)",
      state_waiting_other: "Other waiting (waiting_other)",
      auto_recover_limit_placeholder: "Run limit, for example 3",
      auto_recover_limit_per_mode_placeholder: "Mode budget, for example 1",
      auto_recover_limit_per_lane_placeholder: "Lane budget, for example 1",
      auto_recover_limit_per_protocol_group_placeholder: "Protocol-group budget, for example 1",
      auto_recover_limit_per_provider_placeholder: "Provider budget, for example 1",
      auto_recover_limit_per_profile_placeholder: "Profile budget, for example 1",
      preview_filtered: "Preview Current Filter",
      run_filtered: "Run Current Filter",
      clear_filters: "Clear Filters",
      auto_recover_all: "Showing every auto-recovery candidate.",
      waiting_budget_summary: "Waiting for the current recovery budget summary...",
      no_auto_recover_result: "No auto-recovery preview or execution has been run yet.",
      protocol_coverage: "Protocol Coverage",
      protocol_coverage_desc: "Aggregate real success samples by protocol group",
      report_title: "Acceptance Report",
      report_title_placeholder: "Report title, default to the standard title",
      report_note_placeholder: "Report note, handoff details or milestone summary",
      waiting_report_data: "Waiting for report data...",
      save: "Save",
      download_markdown: "Download Markdown",
      provider_smoke_title: "Real Provider Smoke",
      provider_smoke_summary_desc: "Aggregate real samples by protocol group",
      provider_smoke_matrix_desc: "Real sample matrix merged from smoke records and protocol coverage",
      provider_smoke_filter_query_placeholder: "Filter by title / provider / note",
      provider_smoke_filter_group_placeholder: "Filter by protocolGroup",
      provider_smoke_filter_sample_placeholder: "Filter by sampleType / reusePriority",
      all_results: "All Results",
      waiting_smoke_records: "Waiting for smoke records...",
      provider_smoke_provider_key_placeholder: "providerKey, for example 123_open",
      provider_smoke_protocol_group_placeholder: "protocolGroup, for example aliyun_123_open",
      provider_smoke_auth_mode_placeholder: "authMode, for example manual_token",
      provider_smoke_title_placeholder: "Record title, default to the provider name",
      provider_smoke_note_placeholder: "Note, for example validated ValidateAuth/List/Metadata",
      provider_smoke_operations_placeholder: "Operations, comma separated, for example ValidateAuth,List,Upload",
      smoke_category_auto: "Auto Category",
      smoke_category_auth_only: "Auth Only (auth_only)",
      smoke_category_browse_only: "Browse Only (browse_only)",
      smoke_category_fast_upload_success: "Fast Upload Success (fast_upload_success)",
      smoke_category_binary_upload_success: "Binary Upload Success (binary_upload_success)",
      smoke_category_partial_blocked: "Partially Blocked (partial_blocked)",
      smoke_category_failed: "Failed (failed)",
      result_success: "Success (success)",
      result_failure: "Failure (failure)",
      provider_matrix: "Provider Status Matrix",
      provider_matrix_title: "Provider Status Matrix",
      flash_auto_recover_filters_cleared: "Auto-recovery filters cleared",
      runtime_checkpoints: "Runtime Checkpoint Overview",
      runtime_overview_title: "Runtime Checkpoint Overview",
      from_recent_probe: "From the latest probe / snapshot",
      runtime_overview_desc: "From the latest probe / snapshot",
      recent_retry_queue: "Recent Retry Queue",
      retry_queue_title: "Recent Retry Queue",
      retry_filtered: "Retry Current Filter",
      auto_recover_filtered: "Auto Recover Current Filter",
      waiting_status_data: "Waiting for status data...",
      recent_probe: "Recent Probe",
      recent_probe_title: "Recent Probe",
      recent_results: "Recent Results",
      recent_results_title: "Recent Results",
      total_tasks: "Total Tasks",
      completed_tasks: "Completed Tasks",
      blocked_tasks: "Blocked Tasks",
      execution_mode: "Execution Mode",
      scan_mode: "Scan Mode",
      source_delete_policy: "Source Delete Policy",
      auto_recover_tasks: "Auto-Recovery Tasks",
      runnable_now: "Runnable Now",
      waiting_cooldown: "Waiting Cooldown",
      waiting_retry_window: "Waiting Window",
      waiting_auth_refresh: "Waiting Auth Refresh",
      waiting_local_restore: "Waiting Local Restore",
      waiting_provider_session: "Waiting Session Recovery",
      waiting_manual_confirmation: "Waiting Manual Confirmation",
      waiting_retry_limit: "Waiting Retry Limit",
      waiting_other: "Other Waiting",
      auto_retry_policy_summary_prefix: "Default auto-recovery scheduling",
      blocked_empty: "There are no blocked aggregates that need manual handling right now.",
      auto_recover_last_preview: "Latest Preview",
      auto_recover_last_run: "Latest Run",
      auto_recover_last_recoverable: "Recoverable",
      auto_recover_last_recovered: "Recovered",
      auto_recover_decision_empty: "The latest auto-recovery preview or run does not contain decision details.",
      auto_recover_budget_hint_empty: "Budget usage: the current decision did not return reusable budget hints.",
      auto_recover_budget_hint_prefix: "Budget usage",
      auto_recover_waiting_advice: "Waiting-state note",
      focus_current_state: "Show This State",
      focus_current_lane: "Show This Lane",
      apply_suggested_budgets: "Apply Suggested Budgets",
      preview_current_decision: "Preview This Decision",
      run_current_decision: "Run This Decision",
      open_sample_task: "Open Sample Task",
      acceptance_report_empty: "No acceptance report is available yet. Refresh or save one first.",
      report_title_label: "Report Title",
      report_generated_at: "Generated At",
      report_note_label: "Report Note",
      recent_results_empty: "There are no result records yet.",
      recent_results_col_status: "Status",
      recent_results_col_mode: "Mode",
      recent_results_col_execution_mode: "Execution Mode",
      recent_results_col_retry_mode: "Retry Mode",
      recent_results_col_retry_scope: "Retry Scope",
      recent_results_col_retry_path_count: "Retry Path Count",
      recent_results_col_retry_paths: "Retry Paths",
      recent_results_col_source_delete_policy: "Source Delete Policy",
      recent_results_col_recommendation: "Recommendation",
      recent_results_col_message: "Message",
      recent_results_col_risk_hit: "Risk Hit",
      recent_results_col_conflict: "Conflict Handling",
      recent_results_col_created_at: "Created At",
      recent_probes_empty: "There are no probe records yet.",
      recent_probes_col_provider: "Provider",
      recent_probes_col_status: "Status",
      recent_probes_col_profile: "Auth Profile",
      recent_probes_col_execution_mode: "Execution Mode",
      recent_probes_col_scan_mode: "Scan Mode",
      recent_probes_col_retry_mode: "Retry Mode",
      recent_probes_col_retry_scope: "Retry Scope",
      recent_probes_col_retry_path_count: "Retry Path Count",
      recent_probes_col_retry_paths: "Retry Paths",
      recent_probes_col_source_delete_policy: "Source Delete Policy",
      recent_probes_col_risk_hit: "Risk Hit",
      recent_probes_col_payload: "Payload",
      recent_probes_col_created_at: "Created At",
      auto_recover_acceptance: "Auto-Recovery Acceptance",
      auto_recover_fairness_summary: "Auto-Recovery and Fairness Summary",
      recovery_priority_action: "Recovery Priority Action",
      recovery_priority_action_counts: "Recovery Priority Action Counts",
      fairness_gap: "Fairness Gap",
      fairness_priority_action: "Fairness Priority Action",
      flash_task_tree_focused_by_scan: "Task directory tree focused by the latest scan trail",
      flash_task_tree_focused_by_root: "Task directory tree focused by the selected root",
      flash_status_tree_focused_by_scan: "Status directory tree focused by the latest scan trail",
      flash_status_tree_focused_by_root: "Status directory tree focused by the selected root",
      flash_task_directory_filters_cleared: "Task directory-tree filters cleared",
      flash_task_pending_filters_cleared: "Task pending-tree filters cleared",
      flash_status_directory_filters_cleared: "Status directory-tree filters cleared",
      flash_status_pending_filters_cleared: "Status pending-tree filters cleared",
      waiting_reason_summary: "Waiting Reasons",
      no_auto_recover_pool_samples: "There are no auto-recovery pool samples right now.",
      auto_recover_filtered_empty: "No auto-recovery candidates match the current filters.",
      auto_recover_pool_empty: "There are no tasks in the auto-recovery candidate pool right now.",
      lane_summary_prefix: "Lane",
      sample_record_empty: "There are no real provider smoke records yet.",
      smoke_record_empty: "There are no smoke records.",
      smoke_record_showing_all: "Showing all {visible} smoke records.",
      smoke_record_showing_filtered: "Showing {visible} / {total} smoke records.",
      sample_matrix_empty: "There is no real sample matrix yet.",
      smoke_view_markdown: "View Markdown",
      smoke_download_markdown: "Download Markdown",
      acceptance_matrix_view: "Acceptance Matrix View",
      acceptance_matrix_hint: "You can filter quickly by acceptance state and jump to the related smoke sample or sample task to keep filling real integration evidence.",
      protocol_sampled: "Sampled",
      protocol_pending_sample: "Pending Sample",
      protocol_related_providers: "Related Providers",
      protocol_sample_context: "Sample Context",
      risk_calibration_title: "Provider Default Risk Calibration",
      risk_calibration_empty: "No provider default-template calibration data is available yet. Refresh the provider list first.",
      risk_calibration_ready: "Calibration Ready",
      risk_calibration_gap_summary: "Default Template Gap Summary",
      risk_calibration_missing_counts: "Calibration Missing Field Counts",
      risk_calibration_priority_counts: "Calibration Priority Counts",
      risk_calibration_readiness: "Calibration Readiness",
      risk_calibration_priority_action: "Priority Calibration Action",
      risk_calibration_window_source: "Window Source",
      risk_calibration_window_advice: "Window Advice",
      risk_calibration_missing: "Calibration Missing",
      risk_calibration_coverage: "Calibration Coverage",
      risk_calibration_covered_fields: "Covered Fields",
      risk_calibration_sample_advice: "Calibration Sample Advice",
      risk_calibration_default_action: "Priority Calibration Action",
      provider_smoke_acceptance_title: "Provider-Level Real Sample Acceptance",
      provider_smoke_acceptance_empty: "No providerSmokeProviders data is available yet. Refresh or save a new acceptance report first.",
      provider_smoke_ready: "Providers Ready",
      provider_smoke_gap_summary: "Provider Acceptance Gap Summary",
      provider_smoke_missing_basic_providers: "Providers Missing Basic Samples",
      provider_smoke_missing_upload_providers: "Providers Missing Upload Samples",
      provider_smoke_missing_anomaly_providers: "Providers Missing Anomaly Samples",
      provider_smoke_missing_representative_providers: "Providers Missing Representative Samples",
      provider_smoke_priority_counts: "Provider Priority Action Counts",
      provider_smoke_basic_sample: "Preferred Basic Sample",
      provider_smoke_upload_sample: "Preferred Upload Sample",
      provider_smoke_anomaly_sample: "Preferred Anomaly Sample",
      provider_smoke_representative_sample: "Preferred Representative Sample",
      provider_smoke_representative_missing: "Representative Gaps",
      provider_smoke_representative_actions: "Representative Actions",
      provider_smoke_representative_advice: "Representative Advice",
      provider_smoke_default_action: "Provider Priority Action",
      provider_smoke_default_action_complete: "Complete",
      provider_smoke_readiness_ready: "Ready (basic, upload, anomaly, and representative samples are complete)",
      provider_smoke_readiness_partial: "In Progress (some samples exist, but acceptance is still incomplete)",
      provider_smoke_readiness_pending: "Pending (real provider samples are still missing)",
      provider_smoke_basic_ready: "Basic",
      provider_smoke_upload_ready: "Upload",
      provider_smoke_anomaly_ready: "Anomaly",
      provider_smoke_representative_ready: "Representative",
      blocked_metric_tasks: "Tasks",
      blocked_metric_providers: "Providers",
      blocked_metric_next: "Next Retry",
      blocked_next_step: "Next Step: {value}",
      blocked_missing_path: "Missing recoverable path",
      blocked_focus_action: "Show This Blocked Type",
      blocked_run_action: "Run This Blocked Action",
      auto_recover_run_wait_auth: "Run Waiting Auth Refresh",
      auto_recover_run_wait_local: "Run Waiting Local Restore",
      auto_recover_run_wait_manual: "Run Waiting Manual Confirmation",
      auto_recover_run_wait_limit: "Run Retry-Limit Waiting",
      auto_recover_run_wait_other: "Run Other Waiting",
      auto_recover_run_primary_retry: "Run Primary Retry Class",
      auto_recover_run_primary_blocked: "Run Primary Blocked Action",
      auto_recover_run_lane: "Run This Lane",
      auto_recover_run_mode: "Run This Mode",
      matrix_open_smoke_record: "Open Smoke Sample",
      matrix_open_task_sample: "Open Task Sample",
      matrix_prefill_smoke_form: "Prefill Smoke Form",
      matrix_prefill_profile_risk: "Prefill Profile Risk Defaults",
      matrix_focus_group_records: "Show Group Records",
      matrix_focus_acceptance_type: "Show This Type",
      selection_source_directory: "Directory Subset",
      selection_source_pending: "Pending Subset",
      selection_source_retry: "Retry-Queue Subset",
      selection_scope_directory: "Directory Subset",
      selection_scope_pending: "Pending Subset",
      selection_scope_retry: "Retry-Queue Subset",
      result_success: "Success (success)",
      result_failure: "Failure (failure)",
      smoke_matrix_source_display: "Smoke Matrix {protocolGroup} ({status})",
      smoke_matrix_source_display_fallback: "Smoke Matrix ({status})",
      smoke_matrix_risk_profile_title: "{protocolGroup} Risk Template",
      smoke_matrix_risk_profile_title_fallback: "Real-Sample Risk Template",
      smoke_summary_smokes: "Smokes",
      smoke_summary_success: "Success",
      smoke_summary_upload: "Upload Success",
      smoke_summary_failure: "Failure",
      smoke_summary_providers: "Providers",
      smoke_summary_sample: "Sample Status",
      smoke_summary_providers_label: "Providers:",
      smoke_summary_sample_label: "Sample:",
      smoke_summary_preferred_sample: "Preferred Sample:",
      smoke_summary_preferred_upload: "Preferred Upload:",
      smoke_summary_preferred_anomaly: "Preferred Anomaly:",
      smoke_summary_preferred_representative: "Preferred Representative:",
      smoke_summary_latest_smoke_at: "Latest Smoke At:",
      matrix_filter_all: "All",
      matrix_filter_accepted: "Accepted",
      matrix_filter_in_progress: "In Progress",
      matrix_filter_pending: "Pending",
      matrix_filter_empty: "The current filter {filter} has no real sample matrix.",
      flash_smoke_records_filtered: "Smoke records filtered by {label}",
      flash_smoke_records_result_cleared: "Smoke record result filter cleared",
      flash_smoke_records_group_cleared: "Smoke record protocol-group filter cleared",
      flash_smoke_records_cleared: "Smoke record filters cleared",
      flash_smoke_matrix_filtered: "Acceptance matrix filtered by {label}",
      flash_smoke_record_opened: "Smoke sample opened and the form has been prefilled",
      flash_smoke_gap_prefilled: "Smoke action prefilled from the acceptance gap",
      flash_smoke_matrix_prefilled: "Smoke form prefilled from the acceptance matrix",
      flash_smoke_profile_risk_prefilled: "Account default risk prefilled from the real sample",
      smoke_note_protocol_group: "Protocol Group: {value}",
      smoke_note_provider: "Suggested Provider: {value}",
      smoke_note_status: "Acceptance Status: {value}",
      smoke_note_missing: "Gaps: {value}",
      smoke_note_action: "Actions: {value}",
      smoke_note_advice: "Advice: {value}",
      smoke_note_goal: "Goal: {value}",
      smoke_note_boundary_sample: "This protocol group is already accepted. Continue by adding boundary-case samples.",
      smoke_markdown_load_failed: "Failed to load smoke Markdown: {status}",
      smoke_action_fill_upload_success: "Add 1 real upload-success sample",
      smoke_action_fill_task_coverage: "Add 1 real task-coverage sample",
      smoke_action_fill_upload_smoke: "Add upload smoke",
      smoke_draft_title_provider: "Provider Smoke",
      smoke_draft_title_status: "{protocolGroup} {status} 样本",
      auto_recover_summary_title: "Visible Candidate Summary",
      auto_recover_queue_count: "queue {value}",
      auto_recover_ready_count: "ready {value}",
      auto_recover_cooldown_count: "cooldown {value}",
      auto_recover_profile_counts: "profiles {value}",
      auto_recover_summary_compact: "{lanes} lanes / {tasks} tasks",
      smoke_action_fill_task_sample: "Add task-coverage sample",
      smoke_action_fill_smoke_success: "Add smoke success sample",
      smoke_action_prefill_by_gap: "Prefill action from gaps",
      smoke_draft_label_auth_expired: "Add auth-expired sample",
      smoke_draft_label_rate_limited: "Add rate-limit sample",
      smoke_draft_label_local_file_missing: "Add local-file-missing sample",
      smoke_draft_label_pending_manual: "Add manual-confirmation sample",
      smoke_draft_label_large_file: "Add large-file sample",
      smoke_draft_label_nested_directory: "Add nested-directory sample",
      smoke_draft_label_retry_recovery: "Add retry-recovery sample",
      smoke_draft_note_auth_expired: "Target anomaly: auth_expired",
      smoke_draft_note_rate_limited: "Target anomaly: rate_limited / risk_control",
      smoke_draft_note_local_file_missing: "Target anomaly: local_file_missing",
      smoke_draft_note_pending_manual: "Target anomaly: pending_manual / overwrite downgrade",
      smoke_draft_note_large_file: "Target representative sample: large_file / multipart / large-upload recovery",
      smoke_draft_note_nested_directory: "Target representative sample: nested_directory / multi-level directory / subtree",
      smoke_draft_note_retry_recovery: "Target representative sample: retry_recovery / checkpoint / resume",
      smoke_title_boundary_sample: "Boundary Sample Smoke",
      protocol_coverage_empty: "There is no protocol-group coverage data right now.",
      upload_checkpoint_acceptance_title: "Upload Checkpoint Default Recovery Acceptance",
      upload_checkpoint_readiness: "Checkpoint resume readiness: {value}",
      upload_checkpoint_resume_summary: "Large-file / Long-path Recovery Summary",
      upload_checkpoint_tasks: "Checkpoint Tasks",
      upload_checkpoint_resume_tasks: "Auto Resume",
      upload_checkpoint_readiness_compact: "Readiness",
      upload_checkpoint_priority_action: "Priority recovery action: {value}",
      upload_checkpoint_sample_context: "Sample context: provider {provider} / protocol group {protocolGroup} / task {taskId} / auth profile {profileId}",
      upload_checkpoint_resume_detail: "Resume details: upload {uploadId} / next part {nextPart} / uploaded {uploaded}/{partCount}",
      upload_checkpoint_sample_paths: "Sample paths: {value}",
      upload_checkpoint_priority_action_label: "Recovery priority action: {value}",
      waiting_reason_summary_detail: "cooldown {cooldown} / retry window {retryWindow} / auth {authRefresh} / local file {localRestore} / manual {manual}",
      lane_summary_detail: "provider {provider} / protocol group {protocolGroup} / auth profile {profileId} / providers {providerCount} / profiles {profileCount}",
      sample_context_provider: "Provider {value}",
      sample_context_protocol_group: "Protocol Group {value}",
      sample_context_task: "Task {value}",
      sample_context_profile: "Auth Profile {value}",
      matrix_smoke_count: "Smoke",
      matrix_upload_smoke: "Upload Smoke",
      matrix_coverage: "Coverage",
      matrix_smoke_sample: "Smoke Sample:",
      matrix_coverage_sample: "Coverage Sample:",
      matrix_readiness: "Readiness:",
      matrix_checklist: "Checklist:",
      matrix_gaps: "Gaps:",
      matrix_next_action: "Next Action:",
      matrix_priority_action: "Priority Action:",
      matrix_anomaly_summary: "Anomaly Samples:",
      matrix_representative_summary: "Representative Samples:",
      matrix_anomaly_missing: "Anomaly Gaps:",
      matrix_anomaly_actions: "Anomaly Actions:",
      matrix_anomaly_advice: "Anomaly Advice:",
      matrix_representative_missing: "Representative Gaps:",
      matrix_representative_actions: "Representative Actions:",
      matrix_representative_advice: "Representative Advice:",
      matrix_state_ready: "Ready",
      matrix_state_pending: "Pending",
      matrix_state_partial: "In Progress",
      matrix_state_complete: "Complete",
      matrix_state_accepted: "Accepted",
      matrix_checklist_upload: "Upload",
      matrix_checklist_coverage: "Coverage",
      matrix_checklist_anomaly: "Anomaly",
      matrix_checklist_representative: "Representative",
      matrix_gap_upload: "Upload",
      matrix_gap_anomaly: "Anomaly ({value})",
      matrix_gap_representative: "Representative ({value})",
      matrix_gap_complete: "Complete",
      matrix_missing: "Acceptance Missing:",
      matrix_actions: "Acceptance Actions:",
      matrix_advice: "Acceptance Advice:",
      matrix_latest_observed: "Latest Smoke / Coverage Observed:"
    },
    providers: {
      profile_title: "Auth Profiles",
      assist_title: "Authorization Guide",
      assist_desc: "OpenList is preferred for session and storage discovery, Alist is the fallback, and manual advanced mode remains the last resort.",
      openlist_url_label: "OpenList URL",
      openlist_token_label: "OpenList Access Token",
      alist_url_label: "Alist URL",
      alist_token_label: "Alist Access Token",
      url_placeholder: "Example: http://127.0.0.1:5244",
      token_placeholder: "Optional for now, fill it in after login",
      assist_use_openlist: "Prefer OpenList",
      assist_use_alist: "Use Alist Fallback",
      assist_use_manual: "Use Manual Advanced Mode",
      assist_discover_openlist: "Detect OpenList",
      assist_discover_alist: "Detect Alist",
      assist_open_openlist: "Open OpenList",
      assist_open_alist: "Open Alist",
      assist_clear: "Clear Guide Settings",
      assist_summary_default: "OpenList guidance is preferred by default. If the URL or token is missing, the console will prompt you to switch to Alist or manual mode.",
      assist_discovery_default: "Detection results appear here. If visible storages are listed, the current OpenList / Alist URL and token are basically usable.",
      assist_discovery_empty: "{kind} is reachable, but no visible storage was returned. Confirm the token permissions, or switch back to manual mode below.",
      assist_discovery_summary: "{kind} is reachable. Visible storages: {summary}{suffix}.",
      assist_discovery_hidden_suffix: "; {count} more items are hidden",
      assist_discovery_default_html: "<div>Detection results appear here. If visible storages are listed, the current OpenList / Alist URL and token are basically usable.</div>",
      assist_discovery_name_fallback: "Unnamed Storage",
      assist_discovery_driver_fallback: "Unknown Driver",
      assist_discovery_status_fallback: "Unknown Status",
      assist_discovery_select: "Use {name}",
      assist_discovery_more: "Only the first 12 items are shown. Narrow the permission scope and retry for the remaining {count} items.",
      assist_discovery_choose_storage: "{kind} is reachable. Choose a visible storage below, then continue filling the Token, Cookie, or Extra JSON required by the current provider.",
      provider_label: "Provider",
      auth_mode_label: "Auth Mode",
      display_name_label: "Display Name",
      display_name_placeholder: "Example: 189Cloud Primary Account",
      token_label: "Token",
      cookie_label: "Cookie",
      extra_label: "Extra JSON",
      optional_placeholder: "Optional",
      extra_placeholder: "Example: {\"pwdId\":\"abcd\"}",
      auth_guide_default: "After you choose a provider and auth mode, this panel shows common required fields, optional fields, and Extra JSON examples.",
      risk_title: "Default Account Risk Template",
      risk_desc: "Optional. Saved to `extra.riskDefaults` as the default risk template for this auth profile.",
      risk_keywords_placeholder: "rate_limit,captcha,profile_limit",
      risk_sync: "Sync to Extra JSON",
      risk_clear: "Clear Default Risk Template",
      submit_create: "Create Auth Profile",
      submit_update: "Update Auth Profile",
      cancel_edit: "Cancel Edit",
      catalog_title: "Provider Capability Overview",
      catalog_detail_default: "Click any provider card to inspect capabilities, default risk templates, and recovery budgets.",
      saved_profiles_title: "Saved Auth Profiles",
      saved_profiles_hint: "Supports validate and delete",
      flash_use_openlist: "Switched to OpenList-first guidance",
      flash_use_alist: "Switched to Alist fallback guidance",
      flash_use_manual: "Switched to manual advanced mode",
      flash_detect_openlist: "OpenList detected with {count} visible storages",
      flash_detect_alist: "Alist detected with {count} visible storages",
      flash_reset_assist: "Authorization guide settings cleared. OpenList-first guidance has been restored",
      flash_refresh_catalog: "Provider list refreshed",
      flash_refresh_profiles: "Auth profiles refreshed",
      flash_sync_risk: "Default account risk settings synced to Extra JSON",
      flash_clear_risk: "Default account risk settings cleared",
      flash_cancel_edit: "Auth profile edit cancelled",
      flash_profile_created: "Auth profile created",
      flash_profile_updated: "Auth profile updated",
      assist_discovery_not_found: "No matching discovery result was found. Please detect again.",
      assist_discovery_applied: "Applied storage “{name}” from {kind}; you can continue filling the provider auth fields.",
      assist_discovery_apply_failed: "Failed to apply the discovery result: {error}",
      assist_open_url_required: "Please fill the {display} URL first",
      assist_discovery_invalid_address: "Please fill the {display} URL first",
      assist_open_login_page: "Opened the {display} login page",
      auth_guide_provider_prefix: "Current provider: {provider}.",
      auth_guide_bridge_default: "Current authorization path: OpenList first, Alist as fallback, manual mode as the last resort.",
      auth_guide_bridge_openlist_ready: "Current authorization path: OpenList is preferred for session, storage, and directory discovery. If OpenList is unavailable, switch to Alist or manual mode.",
      auth_guide_bridge_openlist_missing: "Current authorization path: OpenList is preferred, but the OpenList URL is still missing. Fill it in first, or temporarily switch to Alist / manual mode.",
      auth_guide_bridge_alist_ready: "Current authorization path: Alist fallback is active. If Alist can already see the target storage, sign in there first and then fill the auth fields below.",
      auth_guide_bridge_alist_missing: "Current authorization path: Alist fallback is active, but the Alist URL is still missing. Fill in the address first, or switch back to OpenList / manual mode.",
      auth_guide_bridge_manual: "Current authorization path: manual advanced mode is active. Use it only when both OpenList and Alist are unavailable.",
      assist_summary_openlist_ready: "OpenList is currently preferred: {url}. Sign in with OpenList first and confirm the target storage is visible. If it fails, switch to Alist or manual mode.",
      assist_summary_openlist_missing: "OpenList is currently preferred, but the OpenList URL is still missing. Fill in the address and sign in in a new window first. If OpenList is unavailable, switch to Alist fallback.",
      assist_summary_alist_ready: "Alist fallback is active: {url}. Sign in with Alist first and confirm the target storage is visible. If required fields are still missing, switch to manual mode below.",
      assist_summary_alist_missing: "Alist fallback is active, but the Alist URL is still missing. Fill in the Alist address first. If Alist is also unavailable, use manual advanced mode.",
      assist_summary_manual: "Manual advanced mode is active. Fill in the Token, Cookie, and Extra fields directly below. If OpenList or Alist becomes available later, you can switch back to guided mode.",
      provider_detail_load_failed: "Failed to load provider capabilities: {error}",
      provider_detail_loaded: "Loaded {provider} provider capabilities",
      provider_detail_empty: "Click any provider card to inspect capabilities, default risk templates, and recovery budgets.",
      provider_loading_detail: "Loading capability details...",
      provider_capability_summary_title: "Capability Summary",
      provider_auth_modes_title: "Auth Modes",
      provider_conflict_policies_title: "Conflict Policies",
      provider_fallback_modes_title: "Fallback Modes",
      provider_default_risk_title: "Default Risk Template",
      risk_override_synced_note: "Risk override synced to JSON",
      risk_override_cleared_note: "Risk override cleared. The current mode plus provider calibration will be used again.",
      browser_source_refreshed: "Source directory refreshed: {path}",
      browser_target_refreshed: "Target directory refreshed: {path}",
      browser_source_up: "Moved to the parent source directory: {path}",
      browser_target_up: "Moved to the parent target directory: {path}",
      browser_source_opened: "Opened source directory: {path}",
      browser_target_opened: "Opened target directory: {path}",
      browser_source_jumped: "Jumped to source directory: {path}",
      browser_target_jumped: "Jumped to target directory: {path}"
    },
    wizard: {
      title: "Task Preview / Create",
      source_provider: "Source Provider",
      target_provider: "Target Provider",
      target_provider_insight_default: "After you choose the target provider, this panel shows the default risk template, recommended mode, and recovery budget.",
      source_profile: "Source Auth Profile",
      target_profile: "Target Auth Profile",
      target_profile_insight_default: "After you choose the target auth profile, this panel shows account-level default risk settings and lets you apply them to the task override.",
      source_directory: "Source Directory",
      target_directory: "Target Directory",
      browser_up: "Up One Level",
      browser_refresh: "Refresh Directories",
      browser_select_current: "Use Current Directory",
      browser_current_path: "Current Directory",
      browser_selection_result: "Selection Result",
      browser_level_root: "Current Level: Root Directory",
      source_browser_selection_default: "Will fill Selected Roots (JSON)",
      target_browser_selection_default: "Will fill Target Root",
      source_browser_hint_default: "Choose the source provider and source auth profile to see directory-browser compatibility hints here.",
      target_browser_hint_default: "Choose the target provider and target auth profile to see directory-browser compatibility hints here.",
      source_browser_empty_default: "Choose the source provider and source auth profile to browse directories.",
      target_browser_empty_default: "Choose the target provider and target auth profile to browse directories.",
      target_browser_create_placeholder: "New subdirectory name, for example archive-2026",
      target_browser_create: "Create Folder Here",
      risk_mode: "Risk Mode",
      risk_mode_balanced: "Balanced (balanced)",
      risk_mode_safe: "Safe (safe)",
      risk_mode_fast: "Fast (fast)",
      risk_mode_custom: "Custom (custom)",
      risk_override_title: "Task-Level Risk Override",
      risk_override_desc: "Optional. Leave it empty to use the current mode plus provider-default calibration.",
      risk_request_interval: "Request Interval (ms)",
      risk_directory_interval: "Directory Interval (ms)",
      risk_page_size: "Page Size",
      risk_cooldown_seconds: "Cooldown (s)",
      risk_retry_limit: "Retry Limit",
      risk_max_concurrent: "Max Concurrency",
      risk_auto_retry_start_hour: "Auto-Recovery Start Hour",
      risk_auto_retry_end_hour: "Auto-Recovery End Hour",
      risk_keywords: "Risk Keywords",
      number_example_1200: "Example: 1200",
      number_example_2000: "Example: 2000",
      number_example_100: "Example: 100",
      number_example_600: "Example: 600",
      number_example_2: "Example: 2",
      number_example_1: "Example: 1",
      number_example_7: "Example: 7",
      risk_keywords_placeholder: "rate_limit,captcha,forbidden",
      sync_risk_override: "Sync to JSON",
      clear_risk_override: "Clear Override",
      risk_override_json: "Risk Override (JSON)",
      execution_mode: "Execution Mode",
      execution_mode_leaf_first_lazy: "Leaf-first lazy scan (leaf_first_lazy)",
      execution_mode_pre_scan_flat: "Pre-scan before execution (pre_scan_flat)",
      source_delete_policy: "Source Delete Policy",
      source_delete_policy_record_only: "Record only, keep target files (record_only)",
      execution_hint_default: "The default recommendation is `leaf_first_lazy`, which fits large directories, risk-sensitive providers, and on-demand scanning.",
      delete_policy_hint: "This release only supports `record_only`: source deletions are recorded explicitly and will not delete existing target files by default.",
      recommendation_title_default: "Waiting for preview recommendations",
      recommendation_reason_default: "After preview generation, this panel shows the recommended execution mode, recommended risk mode, and whether they match your current choices.",
      apply_recommended_execution: "Use Recommended Mode",
      apply_recommended_risk: "Use Recommended Risk",
      selected_roots: "Selected Roots (JSON)",
      selected_roots_hint_default: "You can also enter paths manually, for example `[\"/Movies\",\"/Albums/Trips\"]`. Use `[\"/\"]` to sync the entire root.",
      target_root: "Target Root",
      target_root_hint_default: "You can also enter the target directory manually, for example `/Archive/2026`. Keep `/` to write directly to the target root.",
      target_path_mapping_hint: "Upload targets are generated as “target root + source relative path”. Leave it empty to write directly under the target root.",
      threshold_mb: "Threshold (MB)",
      conflict_policy: "Conflict Policy",
      conflict_policy_auto_rename_new: "Auto rename new file (auto_rename_new)",
      conflict_policy_overwrite_existing: "Overwrite existing file (overwrite_existing)",
      entries_json: "Entries (JSON)",
      preview_plan: "Preview Plan",
      create_task: "Create Task",
      preview_result: "Preview Result",
      preview_current_mode: "Current Mode",
      preview_waiting: "Waiting for preview",
      preview_waiting_text: "Waiting for preview...",
      provider_recommended_risk: "Recommended Risk Mode",
      provider_capability_summary: "Capability Summary",
      provider_default_template: "Provider Default Template",
      risk_chain_title: "Risk Flow",
      risk_base_template_title: "Base Template",
      risk_calibrated_result_title: "Calibrated Result",
      risk_profile_source_title: "Account Default Source",
      risk_source_kind_title: "Source Kind",
      risk_bias_title: "Bias Strategy",
      risk_kind_bias_title: "Source Kind / Bias",
      risk_profile_fields_title: "Account Default Fields",
      risk_override_fields_title: "Task Override Fields",
      risk_provider_baseline_title: "Provider Baseline",
      risk_provider_calibrated_title: "Provider Calibrated",
      risk_profile_applied_title: "Account Default Applied",
      risk_task_override_title: "Task Override",
      risk_applied_title: "Final Applied",
      risk_profile_source_compact_title: "Source",
      risk_profile_write_hint: "You can write these defaults into the task override first and fine-tune them afterward.",
      risk_profile_source_builtin: "Built into the auth profile as account defaults",
      risk_profile_source_provider_default: "Not configured. The provider default template is used.",
      risk_auto_retry_window_title: "Auto-Recovery Window",
      risk_budget_title: "Recovery Budget",
      risk_budget_advice_title: "Budget Advice",
      risk_reason_title: "Calibration Reason",
      risk_hints_title: "Risk Hints",
      risk_traits_title: "Risk Traits",
      risk_missing_title: "Calibration Missing",
      risk_priority_action_title: "Priority Calibration Action",
      risk_recommend_reason_title: "Recommendation Reason",
      risk_summary_provider: "Provider {value}",
      risk_summary_profile: "Account Default {value}",
      risk_summary_source_kind: "Source Kind {value}",
      risk_summary_bias: "Bias {value}",
      risk_summary_profile_fields: "Account Fields {value}",
      risk_summary_override: "Task Override {value}",
      risk_summary_mode: "Final Mode {value}",
      risk_sensitive_providers_title: "Sensitive Providers",
      provider_default_window_source: "Provider default template",
      provider_empty_window_source: "Left empty by default until the account or task override is provided",
      recover_budget_no_advice: "No account-level recovery budget advice was returned.",
      recover_budget_sensitive: "High-risk providers should recover serially per account: {budget}{reason}",
      recover_budget_profile_rotation: "Rotate recovery concurrency by account: {budget}{reason}",
      recover_budget_default: "Recommended recovery budget: {budget}{reason}",
      provider_apply_default_risk: "Apply Provider Recommended Risk",
      provider_open_capability: "Open Provider Capabilities",
      profile_default_risk: "Account Default Risk",
      profile_source: "Source",
      profile_extra_keys: "Extra Config Keys",
      profile_enabled_fields: "Enabled Fields",
      profile_recover_budget: "Account Recovery Budget Advice",

      risk_profile_reason_title: "Recovery Budget Reason",
      profile_apply_default_risk: "Apply Account Default Risk",
      profile_clear_default_risk: "Revert to Account Default",
      execution_mode_prescan: "Scan everything first, then execute (`pre_scan_flat`). This fits smaller directory sets when you want a complete scan result before execution starts.",
      execution_mode_leaf_first: "Process one directory tree at a time (`leaf_first_lazy`). This is the default recommendation and scans only directories that really need to transfer.",
      recommendation_no_execution_reason: "No execution-mode recommendation yet",
      recommendation_no_risk_reason: "No risk-mode recommendation yet",
      recommendation_execution_applied: "Execution mode already matches the recommendation: {mode}",
      recommendation_execution_suggested: "Recommended execution mode: {mode}",
      recommendation_risk_applied: "Risk mode already matches the recommendation: {mode}",
      recommendation_risk_suggested: "Recommended risk mode: {mode}",
      recommendation_execution_reason: "Execution mode: {reason}",
      recommendation_risk_reason: "Risk mode: {reason}",
      recommendation_warning: "Note: {warning}",
      preview_meta_current_mode: "Current Mode",
      preview_meta_selected_roots: "Selected Roots",
      preview_meta_target_root: "Target Root",
      preview_meta_recommended_mode: "Recommended Mode",
      preview_meta_recommended_reason: "Recommendation Reason",
      preview_meta_execution_order: "Execution Order",
      preview_meta_risk_mode: "Risk Mode",
      preview_meta_risk_throttle: "Risk Throttle",
      preview_meta_recommended_risk: "Recommended Risk",
      preview_meta_recommended_risk_reason: "Recommended Risk Reason",
      preview_meta_aggressive_warning: "Aggressive Risk Warning",
      preview_meta_risk_resolution: "Risk Template Resolution",
      preview_meta_retry_window: "Auto-Recovery Window",
      preview_meta_source_delete_policy: "Source Delete Policy",
      preview_meta_entry_counts: "Active Entries / Delete Records",
      preview_meta_delete_only: "Delete Records Are Only Used for Location",
      preview_meta_delete_only_hint: "This preview only contains delete records and no runnable items. Restore the source files and regenerate the preview first.",
      preview_meta_delete_mix_hint: "This preview includes delete records. They are only used for location and will not generate runnable items.",
      browser_root: "Root",
      browser_level_named: "Current Level: Depth {level} ({name})",
      browser_target_manual_fill: "Filled into Target Root: {path}",
      browser_source_manual_fill: "Filled into Selected Roots (JSON): {path}",
      browser_choose_provider_first: "Choose a provider before browsing directories.",
      browser_no_list_target: "{provider} does not currently declare directory-browsing capability. Enter the target path manually in “Target Root”.",
      browser_no_list_source: "{provider} does not currently declare directory-browsing capability. Enter the source path manually in “Selected Roots (JSON)”.",
      browser_missing_ids: "{provider} did not return a stable fileId / parentId for the current directory, so creating subdirectories or partial navigation may be limited.",
      browser_load_fail_target: "If loading directories for {provider} fails, validate the auth profile, return to root, or fill the path manually in “Target Root”.",
      browser_load_fail_source: "If loading directories for {provider} fails, validate the auth profile, return to root, or fill the path manually in “Selected Roots (JSON)”.",
      browser_ready: "{provider} supports directory browsing. You can click directories to fill the task wizard.",
      selected_roots_manual_supported: "You can also enter paths manually, for example [\"/Movies\",\"/Albums/Trips\"]. Use [\"/\"] to sync the entire root.",
      selected_roots_manual_required: "This source provider may not support directory browsing. Enter a JSON path array manually, for example [\"/Movies\",\"/Albums/Trips\"]. Use [\"/\"] for the whole root.",
      target_root_manual_supported: "You can also enter the target path manually, for example /Archive/2026. Keep / to write directly to the target root.",
      target_root_manual_required: "This target provider may not support directory browsing. Enter the target path manually, for example /Archive/2026. Keep / to write directly to the target root.",
      browser_loading: "Loading directory list...",
      browser_error_target: "Directory loading failed: {error}. Try refreshing, returning to root, or filling “Target Root” manually.",
      browser_error_source: "Directory loading failed: {error}. Try refreshing, returning to root, or filling “Selected Roots (JSON)” manually.",
      browser_empty_target: "No subdirectories are available here. You can use the current directory directly or create a new subdirectory here.",
      browser_empty_source: "No subdirectories are available here. You can use the current directory directly as the source directory.",
      browser_pill_current: "Current",
      browser_pill_selected: "Selected",
      browser_open: "Open",
      browser_use_this: "Use This Directory",
      flash_provider_no_recommended_risk: "This provider does not expose a recommended risk mode right now",
      flash_provider_risk_applied: "Applied the provider recommended risk mode: {mode}",
      flash_provider_opened: "Opened provider capabilities for {provider}",
      flash_profile_no_default_risk: "This auth profile does not contain account-level default risk settings",
      flash_profile_risk_applied: "Applied the account default risk settings to the task override. You can continue fine-tuning them for this task.",
      flash_profile_risk_cleared: "Cleared the task override and reverted to the account-default / provider-default chain",
      flash_risk_override_synced: "Risk override synced to JSON",
      flash_risk_override_cleared: "Risk override cleared. The current mode plus provider calibration will be used again.",
      flash_source_browser_refreshed: "Source directory refreshed: {path}",
      flash_target_browser_refreshed: "Target directory refreshed: {path}",
      flash_source_browser_up: "Moved to the parent source directory: {path}",
      flash_target_browser_up: "Moved to the parent target directory: {path}",
      flash_source_browser_opened: "Opened source directory: {path}",
      flash_target_browser_opened: "Opened target directory: {path}",
      flash_source_browser_jumped: "Jumped to source directory: {path}",
      flash_target_browser_jumped: "Jumped to target directory: {path}",
      flash_target_root_selected: "Filled “Target Root” with {path}",
      flash_source_root_selected: "Filled “Selected Roots (JSON)” with {path}",
      flash_choose_target_profile_first: "Choose the target provider and target auth profile first",
      flash_enter_new_dir_name: "Enter a name for the new directory first",
      flash_browser_missing_parent_id: "The current directory is missing parentId / fileId, so creating a subdirectory is not available yet",
      flash_target_dir_created: "Created directory {name} under {path}, entered it automatically, and filled it into “Target Root”",
      flash_risk_override_parse_error: "Risk override JSON cannot be parsed: {error}",
      flash_preview_required_execution: "Generate a plan preview first",
      flash_execution_applied: "Applied the recommended execution mode: {mode}",
      flash_risk_applied: "Applied the recommended risk mode: {mode}",
      flash_preview_generated: "Plan preview generated",
      flash_delete_only_blocked: "This payload only contains delete records and no runnable items. Restore the source files and regenerate the preview first.",
      flash_task_created: "Task created",
      flash_task_required: "Please choose a task first",
      flash_task_id_missing: "Missing task identifier",
      flash_task_action_pending: "A task action is already running. Please wait.",
      flash_task_action_executed: "Task action executed: {action}",
      flash_prefill_status_task_missing: "The current status sample does not have a usable task",
      flash_prefill_visible_path_missing: "The current filtered result does not contain paths that can rebuild the wizard",
      flash_retry_visible_path_missing: "The current filtered result does not contain retryable paths",
      flash_auto_recover_visible_path_missing: "The current filtered result does not contain paths that can enter auto-recovery",
      flash_retry_invalid_path: "The current path is invalid and cannot be retried",
      flash_focus_profile: "Focused the current auth profile",
      flash_focus_auto_recover_mode: "Filtered background recovery candidates by {mode}",
      flash_open_status: "Opened the status matrix",
      flash_prefill_task_from_detail: "Prefilled the task wizard from the current task",
      flash_smoke_matrix_group_missing: "The matching protocol-group acceptance matrix entry was not found",
      flash_smoke_matrix_task_missing: "The current acceptance-matrix entry does not have a task sample to open",
      flash_copy_directory_path_failed: "Copying directory paths failed: {error}",
      flash_copy_pending_path_failed: "Copying pending paths failed: {error}"
    }
  }
};


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

function defaultAuthAssistState() {
  return {
    preferred: "openlist",
    openlistURL: "",
    openlistToken: "",
    alistURL: "",
    alistToken: "",
  };
}

function loadAuthAssistState() {
  try {
    const raw = localStorage.getItem(authAssistStorageKey);
    if (!raw) {
      return defaultAuthAssistState();
    }
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return defaultAuthAssistState();
    }
    return {
      preferred: ["openlist", "alist", "manual"].includes(String(parsed.preferred || "").trim())
        ? String(parsed.preferred).trim()
        : "openlist",
      openlistURL: String(parsed.openlistURL || "").trim(),
      openlistToken: String(parsed.openlistToken || "").trim(),
      alistURL: String(parsed.alistURL || "").trim(),
      alistToken: String(parsed.alistToken || "").trim(),
    };
  } catch (error) {
    return defaultAuthAssistState();
  }
}

function saveAuthAssistState() {
  try {
    localStorage.setItem(authAssistStorageKey, JSON.stringify(state.authAssist || defaultAuthAssistState()));
  } catch (error) {
    // Ignore storage quota / privacy mode failures.
  }
}

state.authAssist = loadAuthAssistState();
state.language = loadLanguage();

function loadLanguage() {
  try {
    const raw = String(localStorage.getItem(languageStorageKey) || "zh-CN").trim();
    return translations[raw] ? raw : "zh-CN";
  } catch (error) {
    return "zh-CN";
  }
}

function saveLanguage() {
  try {
    localStorage.setItem(languageStorageKey, state.language || "zh-CN");
  } catch (error) {
    // Ignore storage failures.
  }
}

function translationValue(key, fallback = "") {
  const locale = translations[state.language] || translations["zh-CN"];
  const segments = String(key || "").split(".");
  let current = locale;
  for (const segment of segments) {
    if (!current || typeof current !== "object" || !(segment in current)) {
      return fallback || key;
    }
    current = current[segment];
  }
  return typeof current === "string" ? current : fallback || key;
}

function t(key, fallback = "") {
  return translationValue(key, fallback);
}

function tf(key, params = {}, fallback = "") {
  let text = t(key, fallback);
  Object.entries(params || {}).forEach(([name, value]) => {
    text = text.replaceAll(`{${name}}`, String(value));
  });
  return text;
}

function ti18n(key, fallback = "", params = {}) {
  return tf(key, params, fallback);
}

function applyI18n() {
  document.documentElement.lang = state.language || "zh-CN";
  document.title = t("page_title", document.title || "CloudPan Sync Go");
  document.querySelectorAll("[data-i18n-text]").forEach((node) => {
    const key = node.dataset.i18nText;
    if (!key) {
      return;
    }
    node.textContent = t(key, node.textContent || "");
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((node) => {
    const key = node.dataset.i18nPlaceholder;
    if (!key) {
      return;
    }
    node.setAttribute("placeholder", t(key, node.getAttribute("placeholder") || ""));
  });
  const select = $("#language-select");
  if (select) {
    select.value = state.language || "zh-CN";
  }
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

function localizeRecommendationReason(reason, fallback) {
  const normalized = stringifyValue(reason, fallback);
  const knownReasons = {
    "Multiple top-level roots are safer to process subtree by subtree.": "检测到多个顶层目录，按子树逐个推进会更稳妥，也更方便分批处理。",
    "Large transfer sets benefit from a balanced pacing profile.": "传输条目较多时，使用 balanced 风控档位更稳妥，能兼顾速度与稳定性。",
  };
  return knownReasons[normalized] || normalized;
}

function renderExecutionModeLabel(mode) {
  const normalized = stringifyValue(mode, "-");
  const labels = {
    pre_scan_flat: "先完整扫描再执行（pre_scan_flat）",
    leaf_first_lazy: "按目录逐棵推进（leaf_first_lazy）",
  };
  return labels[normalized] || normalized;
}

function renderRiskModeLabel(mode) {
  const normalized = stringifyValue(mode, "-");
  const labels = {
    safe: "稳妥（safe）",
    balanced: "平衡（balanced）",
    fast: "快速（fast）",
    custom: "自定义（custom）",
  };
  return labels[normalized] || normalized;
}

function renderAuthModeLabel(mode) {
  const normalized = stringifyValue(mode, "-");
  const labels = {
    manual_token: "手动令牌（manual_token）",
    manual_cookie: "手动 Cookie（manual_cookie）",
    official_oauth: "官方 OAuth（official_oauth）",
    web_login_capture: "网页登录采集（web_login_capture）",
  };
  return labels[normalized] || normalized;
}

function renderProfileAuthGuide(provider, authMode) {
  if (!provider || typeof provider !== "object") {
    return t(
      "providers.auth_guide_default",
      "选择网盘源和授权方式后，这里会提示当前常见必填项、可留空项和 Extra JSON 示例。",
    );
  }
  const displayName = stringifyValue(provider.meta?.displayName, provider.meta?.key || "当前网盘源");
  const mode = String(authMode || "").trim();
  const required = [t("providers.display_name_label", "显示名称")];
  const optional = [];
  let intro =
    state.language === "en-US"
      ? `Current provider: ${displayName}.`
      : tf("providers.auth_guide_provider_prefix", { provider: displayName }, `当前网盘源：${displayName}。`);
  let extraHint =
    state.language === "en-US"
      ? "In most cases, `Extra JSON` can stay empty at first."
      : "一般情况下 `附加配置(JSON)` 可以先留空。";

  if (mode === "manual_token") {
    required.push(t("providers.token_label", "令牌 Token"));
    optional.push(t("providers.cookie_label", "Cookie"));
    intro +=
      state.language === "en-US"
        ? " Manual token mode is active. Fill in the token first."
        : " 当前使用手动令牌模式，优先填写令牌。";
  } else if (mode === "manual_cookie") {
    required.push(t("providers.cookie_label", "Cookie"));
    optional.push(t("providers.token_label", "令牌 Token"));
    intro +=
      state.language === "en-US"
        ? " Manual cookie mode is active. Fill in the full cookie first."
        : " 当前使用手动 Cookie 模式，优先填写完整 Cookie。";
  } else if (mode === "official_oauth") {
    required.push(t("providers.token_label", "令牌 Token"));
    optional.push(t("providers.cookie_label", "Cookie"));
    intro +=
      state.language === "en-US"
        ? " Official OAuth mode is active. Usually you should fill in the access token returned by the open platform first."
        : " 当前使用官方 OAuth 模式，通常先填写开放平台返回的 access token。";
  } else if (mode === "web_login_capture") {
    required.push(t("providers.cookie_label", "Cookie"));
    optional.push(t("providers.token_label", "令牌 Token"));
    intro +=
      state.language === "en-US"
        ? " Web login capture mode is active. Usually you should extract the cookie from an existing signed-in browser session first."
        : " 当前使用网页登录采集模式，通常先从浏览器已登录会话中整理 Cookie。";
  }

  const providerKey = String(provider.meta?.key || "").trim();
  if (["aliyundrive_open", "123_open"].includes(providerKey)) {
    required.push(
      state.language === "en-US" ? "domainId inside Extra JSON" : "Extra JSON 内的 domainId",
      state.language === "en-US" ? "driveId inside Extra JSON" : "Extra JSON 内的 driveId",
    );
    extraHint =
      state.language === "en-US"
        ? 'Open API providers usually also require `domainId` and `driveId` in `Extra JSON`, for example `{"domainId":"bj1","driveId":"drive-1"}`.'
        : 'Open 接口通常还需要在 `附加配置(JSON)` 中补 `domainId` 和 `driveId`，例如 `{"domainId":"bj1","driveId":"drive-1"}`。';
  } else if (["quark", "uc"].includes(providerKey)) {
    required.push(state.language === "en-US" ? "pwdId inside Extra JSON for the share code" : "分享口令对应的 Extra JSON: pwdId");
    extraHint =
      state.language === "en-US"
        ? 'Quark / UC providers commonly also require `pwdId` in `Extra JSON`, for example `{"pwdId":"share-code"}`.'
        : 'Quark / UC 常见还需要在 `附加配置(JSON)` 中填写 `pwdId`，例如 `{"pwdId":"分享口令"}`。';
  }

  if (mode === "web_login_capture") {
    extraHint +=
      state.language === "en-US"
        ? " If you cannot access the web-login helper right now, switch to a supported manual mode for this provider first."
        : " 如果暂时拿不到网页登录辅助入口，可先切到该网盘源支持的手动模式继续。";
  }

  const assist = state.authAssist || defaultAuthAssistState();
  let bridgeHint =
    state.language === "en-US"
      ? "Current authorization path: OpenList first, Alist as fallback, manual mode as the last resort."
      : t("providers.auth_guide_bridge_default", "当前授权入口：OpenList 优先，Alist 兜底，手动模式作为最后兜底。");
  if (assist.preferred === "openlist") {
    bridgeHint = assist.openlistURL
      ? state.language === "en-US"
        ? "Current authorization path: OpenList is preferred for session, storage, and directory discovery. If OpenList is unavailable, switch to Alist or manual mode."
        : t("providers.auth_guide_bridge_openlist_ready", "当前授权入口：优先通过 OpenList 辅助获取登录态、存储和目录信息。若 OpenList 不可用，再切到 Alist 或手动模式。")
      : state.language === "en-US"
        ? "Current authorization path: OpenList is preferred, but the OpenList URL is still missing. Fill it in first, or temporarily switch to Alist / manual mode."
        : t("providers.auth_guide_bridge_openlist_missing", "当前授权入口：已选 OpenList 优先，但还没填写 OpenList 地址；可先补地址，或临时切到 Alist / 手动模式。");
  } else if (assist.preferred === "alist") {
    bridgeHint = assist.alistURL
      ? state.language === "en-US"
        ? "Current authorization path: Alist fallback is active. If Alist can already see the target storage, sign in there first and then fill the auth fields below."
        : t("providers.auth_guide_bridge_alist_ready", "当前授权入口：当前已切到 Alist 兜底；如果 Alist 能看到目标存储，可先在 Alist 登录后再回填下方授权字段。")
      : state.language === "en-US"
        ? "Current authorization path: Alist fallback is active, but the Alist URL is still missing. Fill in the address first, or switch back to OpenList / manual mode."
        : t("providers.auth_guide_bridge_alist_missing", "当前授权入口：已切到 Alist 兜底，但还没填写 Alist 地址；可先补地址，或改回 OpenList / 手动模式。");
  } else if (assist.preferred === "manual") {
    bridgeHint =
      state.language === "en-US"
        ? "Current authorization path: manual advanced mode is active. Use it only when both OpenList and Alist are unavailable."
        : t("providers.auth_guide_bridge_manual", "当前授权入口：已切到手动高级模式，建议只在 OpenList / Alist 都不可用时使用。");
  }

  if (state.language === "en-US") {
    return `${bridgeHint} ${intro} Required: ${required.join(", ")}.${optional.length ? ` Optional: ${optional.join(", ")}.` : ""} ${extraHint}`;
  }
  return `${bridgeHint} ${intro} 必填：${required.join("、")}。${optional.length ? ` 可留空：${optional.join("、")}。` : ""} ${extraHint}`;
}

function syncProfileAuthGuide() {
  const wrap = $("#profile-auth-guide");
  if (!wrap) {
    return;
  }
  const providerKey = $("#profile-provider")?.value || "";
  const authMode = $("#profile-auth-mode")?.value || "";
  const provider = (state.providers || []).find((item) => item?.meta?.key === providerKey);
  wrap.textContent = renderProfileAuthGuide(provider, authMode);
}

function normalizeAssistURL(value) {
  const normalized = String(value || "").trim();
  if (!normalized) {
    return "";
  }
  if (/^https?:\/\//i.test(normalized)) {
    return normalized;
  }
  return `http://${normalized}`;
}

function renderAuthAssistSummary() {
  const assist = state.authAssist || defaultAuthAssistState();
  const openlistReady = Boolean(assist.openlistURL);
  const alistReady = Boolean(assist.alistURL);
  if (assist.preferred === "openlist") {
    return openlistReady
      ? state.language === "en-US"
        ? `OpenList is currently preferred: ${assist.openlistURL}. Sign in with OpenList first and confirm the target storage is visible. If it fails, switch to Alist or manual mode.`
        : tf("providers.assist_summary_openlist_ready", { url: assist.openlistURL }, `当前优先走 OpenList：${assist.openlistURL}。建议先在 OpenList 登录并确认能看到对应存储；失败后再切到 Alist 或手动模式。`)
      : state.language === "en-US"
        ? "OpenList is currently preferred, but the OpenList URL is still missing. Fill in the address and sign in in a new window first. If OpenList is unavailable, switch to Alist fallback."
        : t("providers.assist_summary_openlist_missing", "当前优先走 OpenList，但还没填写 OpenList 地址。可先填写地址并在新窗口登录；若暂时没有 OpenList，再切到 Alist 兜底。");
  }
  if (assist.preferred === "alist") {
    return alistReady
      ? state.language === "en-US"
        ? `Alist fallback is active: ${assist.alistURL}. Sign in with Alist first and confirm the target storage is visible. If required fields are still missing, switch to manual mode below.`
        : tf("providers.assist_summary_alist_ready", { url: assist.alistURL }, `当前已切到 Alist 兜底：${assist.alistURL}。建议先在 Alist 登录并确认能看到对应存储；如果仍拿不到字段，再回到底部手动模式。`)
      : state.language === "en-US"
        ? "Alist fallback is active, but the Alist URL is still missing. Fill in the Alist address first. If Alist is also unavailable, use manual advanced mode."
        : t("providers.assist_summary_alist_missing", "当前已切到 Alist 兜底，但还没填写 Alist 地址。可先补 Alist 地址；如果也没有 Alist，再使用手动高级模式。");
  }
  return state.language === "en-US"
    ? "Manual advanced mode is active. Fill in the Token, Cookie, and Extra fields directly below. If OpenList or Alist becomes available later, you can switch back to guided mode."
    : t("providers.assist_summary_manual", "当前已切到手动高级模式。请直接填写下方 Token、Cookie 和附加配置；如果后面补上 OpenList 或 Alist，也可以再切回引导模式。");
}

function syncAuthAssistInputs() {
  const assist = state.authAssist || defaultAuthAssistState();
  setInputValueIfPresent("#profile-assist-openlist-url", assist.openlistURL);
  setInputValueIfPresent("#profile-assist-openlist-token", assist.openlistToken);
  setInputValueIfPresent("#profile-assist-alist-url", assist.alistURL);
  setInputValueIfPresent("#profile-assist-alist-token", assist.alistToken);
  const summary = $("#profile-assist-summary");
  if (summary) {
    summary.textContent = renderAuthAssistSummary();
  }
}

function collectAuthAssistStateFromForm() {
  return {
    preferred: state.authAssist?.preferred || "openlist",
    openlistURL: normalizeAssistURL($("#profile-assist-openlist-url")?.value || ""),
    openlistToken: ($("#profile-assist-openlist-token")?.value || "").trim(),
    alistURL: normalizeAssistURL($("#profile-assist-alist-url")?.value || ""),
    alistToken: ($("#profile-assist-alist-token")?.value || "").trim(),
  };
}

function persistAuthAssistState(nextState = {}) {
  state.authAssist = {
    ...(state.authAssist || defaultAuthAssistState()),
    ...nextState,
  };
  saveAuthAssistState();
  syncAuthAssistInputs();
  syncProfileAuthGuide();
}

function renderAuthAssistDiscovery(response) {
  if (!response || typeof response !== "object") {
    return t(
      "providers.assist_discovery_default",
      "检测结果会在这里显示；如果能列出可见存储，说明当前 OpenList / Alist 地址和令牌基本可用。",
    );
  }
  const storages = Array.isArray(response.storages) ? response.storages : [];
  if (!storages.length) {
    return tf(
      "providers.assist_discovery_empty",
      { kind: response.kind === "alist" ? "Alist" : "OpenList" },
      `${response.kind === "alist" ? "Alist" : "OpenList"} 已连通，但当前没有返回可见存储。可继续确认令牌权限，或直接回到底部手动模式。`,
    );
  }
  const summary = storages
    .slice(0, 6)
    .map((item) => {
      const name = stringifyValue(item.name, "-");
      const driver = stringifyValue(item.driver, "-");
      const mountPath = stringifyValue(item.mountPath, "-");
      return `${name}（${driver} / ${mountPath}）`;
    })
    .join("；");
  const suffix = storages.length > 6
    ? tf(
      "providers.assist_discovery_hidden_suffix",
      { count: storages.length - 6 },
      state.language === "en-US"
        ? `; ${storages.length - 6} more items are hidden`
        : `；其余 ${storages.length - 6} 项未展开`,
    )
    : "";
  return tf(
    "providers.assist_discovery_summary",
    { kind: response.kind === "alist" ? "Alist" : "OpenList", summary, suffix },
    `${response.kind === "alist" ? "Alist" : "OpenList"} 已连通，当前可见存储：${summary}${suffix}。`,
  );
}

function renderAuthAssistDiscoveryHTML(response) {
  if (!response || typeof response !== "object") {
    return t(
      "providers.assist_discovery_default_html",
      "<div>检测结果会在这里显示；如果能列出可见存储，说明当前 OpenList / Alist 地址和令牌基本可用。</div>",
    );
  }
  const kind = response.kind === "alist" ? "alist" : "openlist";
  const kindLabel = kind === "alist" ? "Alist" : "OpenList";
  const storages = Array.isArray(response.storages) ? response.storages : [];
  if (!storages.length) {
    return `<div>${escapeHTML(renderAuthAssistDiscovery(response))}</div>`;
  }
  const items = storages
    .slice(0, 12)
    .map((item, index) => {
      const name = stringifyValue(item.name, t("providers.assist_discovery_name_fallback", "未命名存储"));
      const driver = stringifyValue(item.driver, t("providers.assist_discovery_driver_fallback", "未知驱动"));
      const mountPath = stringifyValue(item.mountPath, "/");
      const status = stringifyValue(item.status, t("providers.assist_discovery_status_fallback", "状态未知"));
      return `
        <div class="auth-assist-discovery-item">
          <button type="button" class="ghost" data-assist-select-index="${index}">
            ${escapeHTML(t("providers.assist_discovery_select", "选用 {name}").replace("{name}", name))}
          </button>
          <div class="muted">${escapeHTML(driver)} / <code>${escapeHTML(mountPath)}</code> / ${escapeHTML(status)}</div>
        </div>
      `;
    })
    .join("");
  const moreNotice =
    storages.length > 12
      ? `<div class="muted">${escapeHTML(t("providers.assist_discovery_more", "仅展示前 12 项，其余 {count} 项请缩小权限范围后重试。").replace("{count}", String(storages.length - 12)))}</div>`
      : "";
  return `
    <div>${escapeHTML(t("providers.assist_discovery_choose_storage", "{kind} 已连通，请先从下方选择一个可见存储，再继续补当前网盘源需要的 Token、Cookie 或 Extra JSON。").replace("{kind}", kindLabel))}</div>
    <div class="auth-assist-discovery-list">${items}</div>
    ${moreNotice}
  `;
}

function updateProfileExtraWithAssist(extra, payload) {
  const next = extra && typeof extra === "object" ? { ...extra } : {};
  Object.entries(payload || {}).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") {
      delete next[key];
      return;
    }
    next[key] = value;
  });
  return next;
}

function applyAuthAssistDiscoverySelection(index) {
  const discovery = state.authAssistDiscovery;
  const storages = Array.isArray(discovery?.storages) ? discovery.storages : [];
  const item = storages[index];
  if (!item) {
    showFlash(t("providers.assist_discovery_not_found", "没有找到对应的发现结果，请重新检测一次"), true);
    return;
  }
  const kind = discovery.kind === "alist" ? "alist" : "openlist";
  const kindLabel = kind === "alist" ? "Alist" : "OpenList";
  const displayName = stringifyValue(item.name, "").trim();
  if (displayName) {
    $("#profile-display-name").value = displayName;
  }
  const extra = parseJSONInput($("#profile-extra").value, {});
  const merged = updateProfileExtraWithAssist(extra, {
    assistKind: kind,
    assistBaseUrl: stringifyValue(discovery.baseUrl, ""),
    assistStorageId: stringifyValue(item.id, ""),
    assistStorageDriver: stringifyValue(item.driver, ""),
    assistStorageMountPath: stringifyValue(item.mountPath, ""),
    assistStorageName: stringifyValue(item.name, ""),
    assistStorageStatus: stringifyValue(item.status, ""),
  });
  $("#profile-extra").value = Object.keys(merged).length ? JSON.stringify(merged, null, 2) : "";
  syncAuthAssistDiscovery(discovery);
  showFlash(
    t("providers.assist_discovery_applied", "已从 {kind} 回填存储“{name}”，可继续补当前网盘源需要的授权字段")
      .replace("{kind}", kindLabel)
      .replace("{name}", stringifyValue(item.name, "未命名存储")),
  );
}

function syncAuthAssistDiscovery(message) {
  const wrap = $("#profile-assist-discovery");
  if (!wrap) {
    return;
  }
  if (message && typeof message === "object") {
    state.authAssistDiscovery = message;
    wrap.innerHTML = renderAuthAssistDiscoveryHTML(message);
    return;
  }
  if (typeof message === "string" && message.trim()) {
    state.authAssistDiscovery = null;
    wrap.textContent = message;
    return;
  }
  state.authAssistDiscovery = null;
  wrap.innerHTML = renderAuthAssistDiscoveryHTML(null);
}

async function discoverAuthAssist(kind) {
  const assist = collectAuthAssistStateFromForm();
  const baseUrl = kind === "alist" ? assist.alistURL : assist.openlistURL;
  const token = kind === "alist" ? assist.alistToken : assist.openlistToken;
  const display = kind === "alist" ? "Alist" : "OpenList";
  if (!baseUrl) {
    throw new Error(t("providers.assist_discovery_invalid_address", "请先填写 {display} 地址").replace("{display}", display));
  }
  const result = await api("/api/auth/assist/discover", {
    method: "POST",
    body: {
      kind,
      baseUrl,
      token,
    },
  });
  syncAuthAssistDiscovery(result);
  return result;
}

function switchAuthAssistMode(mode) {
  const normalized = ["openlist", "alist", "manual"].includes(String(mode || "").trim()) ? String(mode).trim() : "openlist";
  persistAuthAssistState({ preferred: normalized, ...collectAuthAssistStateFromForm() });
}

function openAuthAssistURL(kind) {
  const assist = collectAuthAssistStateFromForm();
  const url = kind === "alist" ? assist.alistURL : assist.openlistURL;
  const display = kind === "alist" ? "Alist" : "OpenList";
  if (!url) {
    showFlash(
      t("providers.assist_open_url_required", "请先填写 {display} 地址").replace("{display}", display),
      true,
    );
    return;
  }
  window.open(url, "_blank", "noopener");
  showFlash(t("providers.assist_open_login_page", "已打开 {display} 登录页").replace("{display}", display));
}

function localizeAPIError(error, status) {
  const code = String(error?.code || "").trim();
  const message = String(error?.message || "").trim();
  const knownMessages = {
    provider_not_found: "未找到对应网盘源，请先确认当前网盘源是否存在。",
    auth_mode_not_supported: "当前网盘源不支持这种授权方式，请切换到列表中可选的授权方式。",
    display_name_required: "显示名称不能为空，请先给这个授权档案起一个容易识别的名字。",
    provider_key_required: "请先选择网盘源，再创建授权档案。",
    missing_access_token: "当前授权方式需要填写令牌 Token，请先补全后再保存。",
    missing_domain_or_drive_id: "当前 Open 接口还需要在附加配置(JSON)里填写 domainId 和 driveId。",
    missing_cookie: "当前授权方式需要填写 Cookie，请先补全后再保存。",
    missing_pwd_id: "当前网盘源还需要在附加配置(JSON)里填写 pwdId，例如分享口令。",
    missing_access_token_or_cookie: "当前网盘源至少需要填写 Token 或 Cookie 其中一项。",
    invalid_json: "输入内容不是合法 JSON，请检查括号、引号和逗号后重试。",
    profile_not_found: "没有找到对应授权档案，可能已被删除，请刷新后重试。",
    invalid_password: "管理员密码不正确，请确认后重新输入。",
  };
  if (code && knownMessages[code]) {
    return knownMessages[code];
  }
  return message || `请求失败：${status}`;
}

function renderProfileStatusLabel(status) {
  const normalized = stringifyValue(status, "-");
  const labels = {
    saved: "已保存（saved）",
    verified: "已校验（verified）",
    pending: "待校验（pending）",
    failed: "校验失败（failed）",
    invalid: "已失效（invalid）",
  };
  return labels[normalized] || normalized;
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
  const profileSource = stringifyValue(resolution.profileDefaultSource, t("wizard.risk_profile_source_provider_default", "未配置，使用网盘源默认模板"));
  const profileSourceKind = stringifyValue(resolution.profileDefaultSourceKind, "-");
  const profileDefaultBias = stringifyValue(resolution.profileDefaultBias, "same_as_provider");
  const profileDefaultFields = Array.isArray(resolution.profileDefaultFields)
    ? resolution.profileDefaultFields.filter(Boolean)
    : [];
  const overrideFields = Array.isArray(resolution.overrideFields) ? resolution.overrideFields.filter(Boolean) : [];
  const steps = [tf("wizard.risk_summary_provider", { value: providerKey }, `网盘源 ${providerKey}`)];
  steps.push(tf("wizard.risk_summary_profile", { value: profileSource }, `账号默认 ${profileSource}`));
  if (profileSourceKind !== "-") {
    steps.push(tf("wizard.risk_summary_source_kind", { value: profileSourceKind }, `来源类型 ${profileSourceKind}`));
  }
  if (profileDefaultBias !== "same_as_provider") {
    steps.push(tf("wizard.risk_summary_bias", { value: profileDefaultBias }, `偏向 ${profileDefaultBias}`));
  }
  if (profileDefaultFields.length) {
    steps.push(tf("wizard.risk_summary_profile_fields", { value: profileDefaultFields.join(", ") }, `账号字段 ${profileDefaultFields.join(", ")}`));
  }
  steps.push(tf("wizard.risk_summary_override", { value: overrideFields.length ? overrideFields.join(", ") : "无" }, `任务覆盖 ${overrideFields.length ? overrideFields.join(", ") : "无"}`));
  steps.push(tf("wizard.risk_summary_mode", { value: stringifyValue(resolution.applied?.mode, stringifyValue(resolution.calibrated?.mode, "balanced")) }, `最终档位 ${stringifyValue(resolution.applied?.mode, stringifyValue(resolution.calibrated?.mode, "balanced"))}`));
  return steps.join(" -> ");
}

function renderRiskResolutionFlow(resolution) {
  if (!resolution || typeof resolution !== "object") {
    return `
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.risk_chain_title", "风控链路"))}</strong>
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
      <strong>${escapeHTML(t("wizard.risk_provider_baseline_title", "网盘源基线"))}</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.base))}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_provider_calibrated_title", "网盘源校准后"))}</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.calibrated))}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_profile_applied_title", "账号默认注入"))}</strong>
      <span>${escapeHTML(renderRiskProfileCompact(resolution.profileApplied))}</span>
      <div class="muted">${escapeHTML(t("wizard.risk_profile_source_compact_title", "来源"))} ${escapeHTML(stringifyValue(resolution.profileDefaultSource, t("wizard.risk_profile_source_provider_default", "未配置，使用网盘源默认模板")))} / ${escapeHTML(t("wizard.risk_source_kind_title", "来源类型"))} ${escapeHTML(profileSourceKind)} / ${escapeHTML(t("wizard.risk_bias_title", "偏向策略"))} ${escapeHTML(profileDefaultBias)} / ${escapeHTML(t("wizard.risk_profile_fields_title", "账号默认字段"))} ${escapeHTML(profileDefaultFields.join(", ") || "-")}</div>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_task_override_title", "任务覆盖"))}</strong>
      <span>${overrideFields.length ? escapeHTML(overrideFields.join(", ")) : "无"}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_applied_title", "最终生效"))}</strong>
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
    return t("wizard.profile_recover_budget", "账号恢复预算建议") === "Account Recovery Budget Advice"
      ? "No account-level recovery budget advice was returned."
      : "未返回账号级预算建议。";
  }
  const reason = String(policy.reason || "").trim();
  if (isSensitiveRecoverBudgetTemplate(policy, providerKey)) {
    return t("wizard.recover_budget_sensitive", "高风险网盘源建议单账号串行推进：{budget}{reason}")
      .replace("{budget}", renderRecoverBudgetCompact(policy))
      .replace("{reason}", reason ? `；${reason}` : "");
  }
  if (Number(policy.profileBudget || 0) <= 1 && Number(policy.providerBudget || 0) > 0) {
    return t("wizard.recover_budget_profile_rotation", "建议按账号轮转控制补传并发：{budget}{reason}")
      .replace("{budget}", renderRecoverBudgetCompact(policy))
      .replace("{reason}", reason ? `；${reason}` : "");
  }
  return t("wizard.recover_budget_default", "建议恢复预算：{budget}{reason}")
    .replace("{budget}", renderRecoverBudgetCompact(policy))
    .replace("{reason}", reason ? `；${reason}` : "");
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
  if (capability.supportsAuthValidation) enabled.push("授权校验");
  if (capability.supportsList) enabled.push("目录浏览");
  if (capability.supportsMetadata) enabled.push("详情读取");
  if (capability.supportsCreateDir) enabled.push("创建目录");
  if (capability.supportsFastUpload) enabled.push("秒传预检");
  if (capability.supportsUpload) enabled.push("上传");
  return enabled.length ? enabled.join(", ") : "-";
}

function renderProviderLifecycleStatusLabel(status) {
  const normalized = stringifyValue(status, "-");
  const labels = {
    planned: "规划中（planned）",
    active: "可用（active）",
    beta: "测试中（beta）",
    deprecated: "已废弃（deprecated）",
  };
  return labels[normalized] || normalized;
}

function renderProviderRiskTemplateDetail(template, { title = t("providers.provider_default_risk_title", "默认风控模板"), compact = false } = {}) {
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
    `<div class="muted">${escapeHTML(t("wizard.risk_auto_retry_window_title", "自动补传时段"))} ${escapeHTML(renderRiskWindow(template.calibrated))}</div>`,
    `<div class="muted">${escapeHTML(t("status.risk_calibration_window_source", "时间窗来源"))} ${escapeHTML(renderAutoRetryWindowSource(template.autoRetryWindowSource))}</div>`,
    `<div class="muted">${escapeHTML(t("status.risk_calibration_coverage", "校准覆盖"))} ${escapeHTML(stringifyValue(template.calibrationCoverage, "-"))}</div>
    <div class="muted">校准完成 ${escapeHTML(stringifyValue(template.calibrationCoveredCount, "0"))}/${escapeHTML(stringifyValue(template.calibrationTargetCount, "0"))} / ${escapeHTML(t("status.risk_calibration_missing", "校准缺失"))} ${escapeHTML(stringifyValue(template.calibrationMissingCount, "0"))}</div>
    <div class="muted">${escapeHTML(t("status.risk_calibration_covered_fields", "已覆盖字段"))} ${escapeHTML((template.calibrationCoveredFields || []).join(", ") || "-")}</div>
    <div class="muted">${escapeHTML(t("status.risk_calibration_readiness", "校准就绪度"))} ${escapeHTML(stringifyValue(template.calibrationReadiness, "-"))}</div>
    <div class="muted">${escapeHTML(t("status.risk_calibration_sample_advice", "校准样本建议"))} ${escapeHTML(stringifyValue(template.calibrationSampleAdvice, "-"))}</div>`,
    `<div class="muted">${escapeHTML(t("wizard.provider_recommended_risk", "推荐风控档位"))} ${escapeHTML(renderRiskModeLabel(template.recommendedMode))}</div>`,
    `<div class="muted">${escapeHTML(t("wizard.risk_budget_title", "恢复预算"))} ${escapeHTML(renderRecoverBudgetCompact(template.recoverBudget))}</div>`,
    `<div class="muted">${escapeHTML(t("wizard.risk_budget_advice_title", "预算建议"))} ${escapeHTML(renderRecoverBudgetAdvice(template.recoverBudget, template.providerKey || ""))}</div>`,
  ];
  if (!compact) {
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_base_template_title", "基础模板"))} ${escapeHTML(renderRiskProfileCompact(template.base))}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_reason_title", "校准依据"))} ${escapeHTML(reasons.join(" / ") || "-")}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_hints_title", "风控提示"))} ${escapeHTML(providerHints.join(" / ") || "-")}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_traits_title", "风控特征"))} ${escapeHTML(providerTraits.join(", ") || "-")}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_missing_title", "校准缺失"))} ${escapeHTML((template.calibrationMissing || []).join(", ") || "-")}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_priority_action_title", "优先校准动作"))} ${escapeHTML(stringifyValue(template.calibrationPriorityAction, "-"))}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("status.risk_calibration_window_advice", "时间窗建议"))} ${escapeHTML(stringifyValue(template.autoRetryWindowAdvice, "-"))}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_recommend_reason_title", "推荐说明"))} ${escapeHTML(stringifyValue(template.recommendedReason, "-"))}</div>`);
    parts.push(`<div class="muted">${escapeHTML(t("wizard.risk_hints_title", "风控提示"))} ${escapeHTML(stringifyValue(template.aggressiveRiskWarning, "-"))}</div>`);
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
    wrap.innerHTML = `<div class="muted">${escapeHTML(t("providers.provider_detail_empty", "点击任一网盘源卡片，查看能力声明、默认风控模板和恢复预算。"))}</div>`;
    return;
  }
  if (!detail) {
    wrap.innerHTML = `
      <div class="insight-grid">
        <div class="insight-card">
          <strong>${escapeHTML(entry.meta.displayName)}</strong>
          <span>${escapeHTML(t("providers.provider_loading_detail", "正在加载能力详情..."))}</span>
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
        <strong>${escapeHTML(t("providers.provider_capability_summary_title", "能力摘要"))}</strong>
        <span>${escapeHTML(renderProviderCapabilityCompact(capability))}</span>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("providers.provider_auth_modes_title", "授权方式"))}</strong>
        <span>${escapeHTML((provider.authModes || []).join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("providers.provider_conflict_policies_title", "冲突策略"))}</strong>
        <span>${escapeHTML((provider.conflictPolicies || []).join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("providers.provider_fallback_modes_title", "兜底策略"))}</strong>
        <span>${escapeHTML((provider.fallbackModes || []).join(", ") || "-")}</span>
      </div>
      ${renderProviderRiskTemplateDetail({ ...(provider.defaultRiskTemplate || {}), providerKey: provider.key || providerKey }, { title: t("providers.provider_default_risk_title", "默认风控模板") })}
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
    wrap.innerHTML = `<div class="muted">${escapeHTML(t("wizard.target_provider_insight_default", "选择目标网盘源后，这里会显示默认风控模板、推荐档位和恢复预算。"))}</div>`;
    return;
  }
  wrap.innerHTML = `
    <div class="section-head">
      <h3>${escapeHTML(stringifyValue(entry.meta.displayName, providerKey))}</h3>
      <span class="muted">${escapeHTML(providerKey)} / ${escapeHTML(stringifyValue(entry.meta.protocolGroup, "-"))}</span>
    </div>
    <div class="insight-grid">
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.provider_recommended_risk", "推荐风控档位"))}</strong>
        <span>${escapeHTML(renderRiskModeLabel(entry.meta.defaultRiskTemplate?.recommendedMode))}</span>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.provider_capability_summary", "能力摘要"))}</strong>
        <span>${escapeHTML(renderProviderCapabilityCompact(entry.capability))}</span>
      </div>
      ${renderProviderRiskTemplateDetail({ ...(entry.meta.defaultRiskTemplate || {}), providerKey }, { title: t("wizard.provider_default_template", "网盘源默认模板"), compact: true })}
    </div>
    <div class="actions compact-actions">
      <button type="button" class="ghost" id="apply-provider-default-risk">${escapeHTML(t("wizard.provider_apply_default_risk", "采用网盘源推荐风控"))}</button>
      <button type="button" class="ghost" id="open-target-provider-capability">${escapeHTML(t("wizard.provider_open_capability", "查看网盘源能力详情"))}</button>
    </div>
  `;
  $("#apply-provider-default-risk").onclick = () => {
    const recommended = entry.meta.defaultRiskTemplate?.recommendedMode || "";
    if (!recommended) {
      showFlash(t("wizard.flash_provider_no_recommended_risk", "当前网盘源没有可用的推荐风控档位"), true);
      return;
    }
    setSelectValueIfPresent("#plan-risk-mode", recommended);
    showFlash(tf("wizard.flash_provider_risk_applied", { mode: recommended }, `已采用网盘源推荐风控：${recommended}`));
  };
  $("#open-target-provider-capability").onclick = async () => {
    try {
      await loadProviderCapabilityDetail(providerKey);
      activateTab("providers");
      showFlash(tf("wizard.flash_provider_opened", { provider: providerKey }, `已打开 ${providerKey} 网盘源能力详情`));
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
    wrap.innerHTML = `<div class="muted">${escapeHTML(t("wizard.target_profile_insight_default", "选择目标授权档案后，这里会显示账号默认风控模板，并支持一键写入任务覆盖。"))}</div>`;
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
      <span class="muted">${escapeHTML(stringifyValue(profile.providerKey, "-"))} / ${escapeHTML(renderAuthModeLabel(profile.authMode))}</span>
    </div>
    <div class="insight-grid">
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.profile_default_risk", "账号默认风控"))}</strong>
        <span>${escapeHTML(renderRiskProfileCompact(riskDefaults))}</span>
        <div class="muted">${escapeHTML(t("wizard.risk_profile_write_hint", "可直接写入本次任务覆盖，便于在此基础上再细调。"))}</div>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.profile_source", "来源"))}</strong>
        <span>${riskDefaults ? escapeHTML(riskDefaultSource || t("wizard.risk_profile_source_builtin", "授权档案内置账号默认风控")) : escapeHTML(t("wizard.risk_profile_source_provider_default", "未配置，使用网盘源默认模板"))}</span>
        <div class="muted">${escapeHTML(renderProfileRiskDefaultSourceAdvice(riskDefaultSource || ""))}</div>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.profile_extra_keys", "附加配置项"))}</strong>
        <span>${escapeHTML(extraKeys.join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.profile_enabled_fields", "已启用字段"))}</strong>
        <span>${escapeHTML(profileDefaultFields.join(", ") || "-")}</span>
      </div>
      <div class="insight-card">
        <strong>${escapeHTML(t("wizard.profile_recover_budget", "账号恢复预算建议"))}</strong>
        <span>${escapeHTML(renderRecoverBudgetAdvice(recoverBudget, profile.providerKey || ""))}</span>
      </div>
    </div>
    <div class="actions compact-actions">
      <button type="button" class="ghost" id="apply-profile-default-risk"${riskDefaults ? "" : " disabled"}>${escapeHTML(t("wizard.profile_apply_default_risk", "应用账号默认到任务覆盖"))}</button>
      <button type="button" class="ghost" id="clear-profile-default-risk">${escapeHTML(t("wizard.profile_clear_default_risk", "改回账号默认"))}</button>
    </div>
  `;
  const applyButton = $("#apply-profile-default-risk");
  if (applyButton) {
    applyButton.onclick = () => {
      if (!riskDefaults) {
        showFlash(t("wizard.flash_profile_no_default_risk", "当前授权档案没有账号默认风控可写入"), true);
        return;
      }
      hydrateRiskOverrideForm(riskDefaults);
      $("#plan-risk-override").value = JSON.stringify(riskDefaults, null, 2);
      setSelectValueIfPresent("#plan-risk-mode", "custom");
      showFlash(t("wizard.flash_profile_risk_applied", "已将账号默认风控写入任务覆盖，可继续按任务单独微调"));
    };
  }
  const clearButton = $("#clear-profile-default-risk");
  if (clearButton) {
    clearButton.onclick = () => {
      hydrateRiskOverrideForm(null);
      $("#plan-risk-override").value = "";
      showFlash(t("wizard.flash_profile_risk_cleared", "已清空任务覆盖，将回到账号默认 / 网盘源默认链路"));
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
    <div class="muted">${escapeHTML(t("wizard.risk_chain_title", "风控链路"))} ${escapeHTML(renderRiskResolutionSummary(resolution))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_base_template_title", "基础模板"))} ${escapeHTML(renderRiskProfileCompact(resolution.base))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_calibrated_result_title", "校准结果"))} ${escapeHTML(renderRiskProfileCompact(resolution.calibrated))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_profile_source_title", "账号默认来源"))} ${escapeHTML(stringifyValue(resolution.profileDefaultSource, "-"))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_source_kind_title", "来源类型"))} ${escapeHTML(profileSourceKind)}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_bias_title", "偏向策略"))} ${escapeHTML(profileDefaultBias)}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_profile_applied_title", "账号默认注入"))} ${escapeHTML(renderRiskProfileCompact(resolution.profileApplied))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_applied_title", "最终生效"))} ${escapeHTML(renderRiskProfileCompact(resolution.applied))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_budget_title", "恢复预算"))} ${escapeHTML(renderRecoverBudgetCompact(recoverBudget))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_profile_reason_title", "恢复预算理由"))} ${escapeHTML(stringifyValue(recoverBudget?.reason, "-"))}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_sensitive_providers_title", "敏感网盘源"))} ${escapeHTML(sensitiveProviders.join(", ") || "-")}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_hints_title", "风控提示"))} ${escapeHTML(providerHints.join(" / ") || "-")}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_traits_title", "风控特征"))} ${escapeHTML(providerTraits.join(", ") || "-")}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_reason_title", "校准依据"))} ${escapeHTML(reasons.join(" / ") || "-")}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_profile_fields_title", "账号默认字段"))} ${escapeHTML(profileDefaultFields.join(", ") || "-")}</div>
    <div class="muted">${escapeHTML(t("wizard.risk_override_fields_title", "任务覆盖字段"))} ${escapeHTML(overrideFields.join(", ") || "-")}</div>
  `;
}

function renderRiskResolutionMetaCards(resolution) {
  if (!resolution || typeof resolution !== "object") {
    return "";
  }
  const recoverBudget = resolution.recoverBudget && typeof resolution.recoverBudget === "object" ? resolution.recoverBudget : {};
  const profileSource = stringifyValue(resolution.profileDefaultSource, t("wizard.risk_profile_source_provider_default", "未配置，使用网盘源默认模板"));
  const profileSourceKind = stringifyValue(resolution.profileDefaultSourceKind, "-");
  const profileDefaultBias = stringifyValue(resolution.profileDefaultBias, "same_as_provider");
  const profileDefaultFields = Array.isArray(resolution.profileDefaultFields)
    ? resolution.profileDefaultFields.filter(Boolean)
    : [];
  return `
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_profile_source_title", "账号默认来源"))}</strong>
      <span>${escapeHTML(profileSource)}</span>
      <div class="muted">${escapeHTML(renderProfileRiskDefaultSourceAdvice(profileSource))}</div>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_kind_bias_title", "来源类型 / 偏向"))}</strong>
      <span>${escapeHTML(`${profileSourceKind} / ${profileDefaultBias}`)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_profile_fields_title", "账号默认字段"))}</strong>
      <span>${escapeHTML(profileDefaultFields.join(", ") || "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.risk_profile_reason_title", "恢复预算理由"))}</strong>
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
      return t("wizard.provider_default_window_source", "网盘源默认模板");
    case "empty_until_profile_or_override":
      return t("providers.provider_default_risk_title", "默认风控模板") === "Default Risk Template"
        ? "Left empty by default until the account or task override is provided"
        : "默认留空，等待账号或任务覆盖";
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
    return tf("tasks.no_tree_nodes", { label }, `暂无${label}。`);
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
    return tf("tasks.showing_all_tree_nodes", { visible: result.visibleNodes, label, suffix }, `显示全部 ${result.visibleNodes} 个${label}。${suffix}`);
  }
  return tf("tasks.showing_filtered_tree_nodes", { visible: result.visibleNodes, total: result.totalNodes, label, suffix }, `当前显示 ${result.visibleNodes} / ${result.totalNodes} 个${label}。${suffix}`);
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
    return `<div class="directory-empty">${escapeHTML(t("tasks.retry_queue_empty", "当前没有需要后续重试的队列项。"))}</div>`;
  }
  const retryable = result.items.filter((item) => item.retryable && !item.exhausted).length;
  const blocked = result.items.filter((item) => item.blocked && !item.exhausted).length;
  const exhausted = result.items.filter((item) => item.exhausted).length;
  return `
    <div class="retry-summary-grid">
      <div class="retry-card">
        <strong>${escapeHTML(t("tasks.retry_queue_current", "当前显示"))}</strong>
        <span>${result.visibleItems} / ${result.totalItems}</span>
      </div>
      <div class="retry-card">
        <strong>${escapeHTML(t("tasks.retry_queue_retryable_now", "可立即重试"))}</strong>
        <span>${retryable}</span>
      </div>
      <div class="retry-card">
        <strong>${escapeHTML(t("tasks.retry_queue_blocked", "阻塞"))}</strong>
        <span>${blocked}</span>
      </div>
      <div class="retry-card">
        <strong>${escapeHTML(t("tasks.retry_queue_exhausted", "耗尽"))}</strong>
        <span>${exhausted}</span>
      </div>
    </div>
  `;
}

function renderRetryQueue(items, filters = {}) {
  const result = filterRetryQueue(items, filters);
  if (!result.totalItems) {
    return {
      html: `<div class="directory-empty">${escapeHTML(t("tasks.retry_queue_empty", "当前没有需要后续重试的队列项。"))}</div>`,
      summaryText: t("tasks.retry_queue_none", "当前没有重试队列项。"),
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
          <div class="muted">${escapeHTML(tf("status.retry_attempt_summary", { attempt: item.attemptCount, limit: item.retryLimit || 0, remaining: item.remainingCount }, `尝试 ${item.attemptCount} / 上限 ${item.retryLimit || 0} / 剩余 ${item.remainingCount}`))}</div>
          ${item.cooldownTier || item.cooldownSeconds ? `<div class="muted">${escapeHTML(tf("status.retry_cooldown_summary", { tier: item.cooldownTier || "custom", seconds: item.cooldownSeconds || 0 }, `冷却：${item.cooldownTier || "custom"} / ${item.cooldownSeconds || 0} 秒`))}</div>` : ""}
          ${item.eligibleAt ? `<div class="muted">${escapeHTML(tf("status.retry_eligible_at", { time: item.eligibleAt }, `可重试时间：${item.eligibleAt}`))}</div>` : ""}
          ${item.rootPath ? `<div class="muted">${escapeHTML(tf("status.retry_root", { value: item.rootPath }, `根目录：${item.rootPath}`))}</div>` : ""}
          ${item.reason ? `<div class="muted">${escapeHTML(tf("status.retry_reason", { value: item.reason }, `原因：${item.reason}`))}</div>` : ""}
          <div class="muted">${escapeHTML(tf("status.retry_next_step", { value: renderBlockedSummary(item.retryAction, item.reason, item.eligibleAt || "") }, `下一步：${renderBlockedSummary(item.retryAction, item.reason, item.eligibleAt || "")}`))}</div>
          ${item.uploadCheckpoint ? `<div class="muted">${escapeHTML(tf("status.retry_checkpoint_summary", { uploadId: stringifyValue(item.uploadCheckpoint.uploadId, "-"), nextPart: stringifyValue(item.uploadCheckpoint.nextPartNumber, "-"), uploaded: stringifyValue(item.uploadCheckpoint.uploadedPartCount, "0") }, `续传断点：上传 ${stringifyValue(item.uploadCheckpoint.uploadId, "-")} / 下一分片 ${stringifyValue(item.uploadCheckpoint.nextPartNumber, "-")} / 已上传 ${stringifyValue(item.uploadCheckpoint.uploadedPartCount, "0")}`))}</div>` : ""}
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-retry-focus-pending="${escapeHTML(item.rootPath || item.path)}"
            >${escapeHTML(t("tasks.focus_pending_tree", "定位待补传树"))}</button>
            <button
              type="button"
              class="ghost"
              data-retry-focus-class="${escapeHTML(item.retryClass)}"
              data-retry-focus-state="${escapeHTML(stateValue)}"
            >${escapeHTML(t("tasks.focus_same_retry_class", "只看同类队列"))}</button>
          </div>
        </div>
      `;
    })
    .join("");
  const summaryText = result.filterActive
    ? tf("tasks.showing_filtered_tree_nodes", { visible: result.visibleItems, total: result.totalItems, label: "重试队列项", suffix: "" }, `当前显示 ${result.visibleItems} / ${result.totalItems} 个重试队列项。`).replace(/\s+\.$/, "。")
    : tf("tasks.showing_all_tree_nodes", { visible: result.totalItems, label: "重试队列项", suffix: "" }, `显示全部 ${result.totalItems} 个重试队列项。`).replace(/\s+\.$/, "。");
  return {
    html: `${renderRetryQueueSummary(result)}${rows || `<div class="directory-empty">${escapeHTML(t("tasks.retry_queue_filtered_empty", "筛选后没有命中的重试队列项。"))}</div>`}`,
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
    refresh_auth_profile: t("tasks.action_refresh_auth_profile", "刷新授权后继续"),
    restore_local_source_file: t("tasks.action_restore_local_source_file", "补回本地文件后继续"),
    manual_intervention_required: t("tasks.action_manual_intervention_required", "修复 provider 会话后继续"),
    wait_for_cooldown: normalizedNextRetryAt ? tf("tasks.action_wait_for_cooldown", { time: normalizedNextRetryAt }, `等待冷却到 ${normalizedNextRetryAt}`) : t("tasks.action_wait_for_cooldown_fallback", "等待冷却结束后继续"),
    wait_for_retry_window: normalizedNextRetryAt ? tf("tasks.action_wait_for_retry_window", { time: normalizedNextRetryAt }, `等待时间窗到 ${normalizedNextRetryAt}`) : t("tasks.action_wait_for_retry_window_fallback", "等待时间窗开放后继续"),
    manual_confirmation_required: t("tasks.action_manual_confirmation_required", "人工确认后继续"),
    review_and_reset_retry_strategy: t("tasks.action_review_and_reset_retry_strategy", "调整重试策略后继续"),
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
        <strong>${escapeHTML(t("tasks.runtime_checkpoint", "运行检查点"))}</strong>
        <span>${escapeHTML(t("tasks.runtime_empty", "暂无运行时信息"))}</span>
      </div>
    `;
  }
  return `
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.runtime_state", "执行状态"))}</strong>
      <span>${stringifyValue(runtime.executionState)}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.pause_request", "暂停请求"))}</strong>
      <span>${runtime.pauseRequested ? "waiting_current_item" : "-"}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.requested_at", "请求时间"))}</strong>
      <span>${stringifyValue(runtime.pauseRequestedAt, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.current_root", "当前根目录"))}</strong>
      <span><code>${escapeHTML(stringifyValue(runtime.currentRoot, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.current_directory", "当前目录"))}</strong>
      <span><code>${escapeHTML(stringifyValue(runtime.currentDirectory, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.last_completed", "上次完成"))}</strong>
      <span><code>${escapeHTML(stringifyValue(runtime.lastCompletedPath, "-"))}</code></span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.progress", "处理进度"))}</strong>
      <span>${stringifyValue(runtime.processedCount, "0")} / next ${stringifyValue(runtime.nextSequence, "1")}</span>
    </div>
    ${metadata && typeof metadata === "object" ? renderRuntimePathChips(t("tasks.selected_roots", "选定根目录"), metadata.selectedRoots || [], scope, "roots") : ""}
    ${metadata && typeof metadata === "object" ? renderRuntimePathChips(t("tasks.scan_trace", "扫描轨迹"), metadata.scanTrace || [], scope, "scan") : ""}
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.result_count", "结果计数"))}</strong>
      <span>done ${stringifyValue(runtime.doneCount, "0")} / skipped ${stringifyValue(runtime.skippedCount, "0")} / failed ${stringifyValue(runtime.failedCount, "0")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.source_deletion_count", "源端删除记录"))}</strong>
      <span>${stringifyValue(runtime.sourceDeletionCount, "0")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.risk_hits", "风控命中"))}</strong>
      <span>${stringifyValue(runtime.riskHitCount, "0")} / last ${stringifyValue(runtime.lastRiskStatus, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.retry_queue_label", "重试队列"))}</strong>
      <span>可重试 ${stringifyValue(runtime.retryableCount, "0")} / 阻塞 ${stringifyValue(runtime.blockedRetryCount, "0")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.blocked_reason", "阻塞原因"))}</strong>
      <span>${stringifyValue(runtime.blockedReason, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.handling_action", "处理动作"))}</strong>
      <span>${stringifyValue(runtime.blockedAction, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.handling_advice", "处理建议"))}</strong>
      <span>${stringifyValue(runtime.blockedAdvice, "-")}</span>
    </div>
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.blocked_summary", "阻塞摘要"))}</strong>
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
      <strong>${escapeHTML(t("tasks.next_auto_recover", "下次自动补传"))}</strong>
      <span>${stringifyValue(runtime.nextRetryAt, "-")}</span>
    </div>
    ${
      metadata?.retrySummary
        ? `
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.auto_recover_candidate", "后台补传候选"))}</strong>
            <span>${escapeHTML(renderAutoRecoverMode(metadata.retrySummary))}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.queue_breakdown", "队列拆分"))}</strong>
            <span>${escapeHTML(renderRetrySummaryBreakdown(metadata.retrySummary))}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.auto_recover_advice", "自动补传提示"))}</strong>
            <span>${escapeHTML(stringifyValue(metadata.retrySummary.autoRecoverAdvice, "-"))}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.wait_auth_refresh", "恢复等待 - Auth 刷新"))}</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingAuthRefreshTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.wait_local_restore", "恢复等待 - 本地恢复"))}</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingLocalRestoreTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.wait_provider_session", "恢复等待 - 网盘会话"))}</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingProviderSessionTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.wait_manual_confirmation", "恢复等待 - 手动确认"))}</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingManualTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.wait_retry_limit", "恢复等待 - 限额超限"))}</strong>
            <span>${stringifyValue(metadata.retrySummary.autoRecoverWaitingRetryLimitTasks, "0")}</span>
          </div>
          <div class="insight-card checkpoint-card">
            <strong>${escapeHTML(t("tasks.wait_retry_window", "恢复等待 - 时间窗"))}</strong>
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
              >${escapeHTML(t("tasks.prefill_from_deletion", "用此删除记录重建向导"))}</button>
            ` : ""}
          </div>
          <span> | ${escapeHTML(t("tasks.reason_label", "原因"))} ${escapeHTML(reason)} | ${escapeHTML(t("tasks.deleted_at_label", "删除时间"))} ${escapeHTML(deletedAt)}</span>
        </div>
      `;
    })
    .join("");
  return `
    <div class="insight-card checkpoint-card">
      <strong>${escapeHTML(t("tasks.deletion_summary", "删除记录摘要"))}</strong>
      <div>${escapeHTML(tf("tasks.deletion_summary_desc", { count: resolvedCount }, `${resolvedCount} 条，默认只记录，不会自动删除目标端真实文件。`))}</div>
      ${canPrefill ? `
        <div class="actions compact">
          <button
            type="button"
            class="ghost"
            data-source-delete-prefill-paths="${escapeHTML(JSON.stringify(paths))}"
            data-source-delete-prefill-scope="${escapeHTML(prefillScope)}"
            data-source-delete-prefill-label="全部删除记录"
          >${escapeHTML(t("tasks.prefill_all_deletions", "按全部删除记录重建向导"))}</button>
        </div>
      ` : ""}
      ${rows || `<div class="muted">${escapeHTML(t("tasks.no_deletion_samples", "暂无可展开样本。"))}</div>`}
      ${items.length > shown.length ? `<div class="muted">${escapeHTML(tf("tasks.more_deletion_samples", { count: items.length - shown.length }, `还有 ${items.length - shown.length} 条未展开。`))}</div>` : ""}
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
    return t("status.matrix_state_pending", "待补齐");
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
    return t("status.matrix_state_ready", "已就绪");
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
    return t("status.matrix_state_partial", "进行中");
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
    const syncLabel = panel === "directory"
      ? t("tasks.sync_to_pending_tree", "待补传树")
      : t("tasks.sync_to_directory_tree", "目录树");

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
                >${escapeHTML(collapsed ? t("tasks.expand_subtree", "展开子树") : t("tasks.collapse_subtree", "收起子树"))}</button>
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
                >${escapeHTML(t("tasks.rebuild_from_current_path", "按当前路径重建向导"))}</button>
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
                >${escapeHTML(t("tasks.retry_current_path", "重试当前路径"))}</button>
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
                >${escapeHTML(t("tasks.auto_recover_current_path", "后台补传当前路径"))}</button>
              `
              : ""
          }
          <button
            type="button"
            class="ghost"
            data-tree-copy-path="${escapeHTML(node.path)}"
            data-tree-copy-scope="${escapeHTML(scope)}"
            data-tree-copy-panel="${escapeHTML(panel)}"
          >${escapeHTML(t("tasks.copy_current_subtree", "复制当前子树"))}</button>
          ${
            parentPath
              ? `
                <button
                  type="button"
                  class="ghost"
                  data-tree-parent-path="${escapeHTML(node.path)}"
                  data-tree-parent-scope="${escapeHTML(scope)}"
                  data-tree-parent-panel="${escapeHTML(panel)}"
                >${escapeHTML(t("tasks.focus_parent", "只看父级"))}</button>
              `
              : ""
          }
          <button
            type="button"
            class="ghost"
            data-tree-focus-path="${escapeHTML(node.path)}"
            data-tree-focus-scope="${escapeHTML(scope)}"
            data-tree-focus-panel="${escapeHTML(panel)}"
          >${escapeHTML(t("tasks.focus_current_path", "只看当前路径"))}</button>
          <button
            type="button"
            class="ghost"
            data-tree-sync-path="${escapeHTML(node.path)}"
            data-tree-sync-scope="${escapeHTML(scope)}"
            data-tree-sync-panel="${escapeHTML(panel)}"
          >${escapeHTML(tf("tasks.sync_to_tree", { label: syncLabel }, `同步到${syncLabel}`))}</button>
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
                      >${escapeHTML(t("tasks.rebuild_from_root", "按当前 root 重建向导"))}</button>
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
                      >${escapeHTML(t("tasks.retry_current_root", "重试当前 root"))}</button>
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
                      >${escapeHTML(t("tasks.auto_recover_current_root", "后台补传当前 root"))}</button>
                    `
                    : ""
                }
                <button
                  type="button"
                  class="ghost"
                  data-tree-focus-path="${escapeHTML(rootPath)}"
                  data-tree-focus-scope="${escapeHTML(scope)}"
                  data-tree-focus-panel="${escapeHTML(panel)}"
                >${escapeHTML(t("tasks.focus_root", "只看 root"))}</button>
                <button
                  type="button"
                  class="ghost"
                  data-tree-sync-path="${escapeHTML(rootPath)}"
                  data-tree-sync-scope="${escapeHTML(scope)}"
                  data-tree-sync-panel="${escapeHTML(panel)}"
                >${escapeHTML(t("tasks.sync_other_tree", "同步另一棵树"))}</button>
                <button
                  type="button"
                  class="ghost tree-group-toggle"
                  data-tree-group-toggle
                  data-tree-group-scope="${escapeHTML(scope)}"
                  data-tree-group-panel="${escapeHTML(panel)}"
                  data-tree-group-path="${escapeHTML(rootPath)}"
                  aria-expanded="${collapsed ? "false" : "true"}"
                >${escapeHTML(collapsed ? t("tasks.expand", "展开") : t("tasks.collapse", "收起"))}</button>
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
    return `<div class="directory-empty">${escapeHTML(t("tasks.no_directory_states", "暂无目录状态。"))}</div>`;
  }
  return renderTreeNodes(states, { mode: "directory", emptyMessage: t("tasks.no_directory_states", "暂无目录状态。") });
}

function renderPendingTree(nodes) {
  return renderTreeNodes(nodes, { mode: "pending", emptyMessage: t("tasks.no_pending_items", "暂无待补传项。") });
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
    emptyMessage: t("tasks.no_directory_states", "暂无目录状态。"),
    normalized: true,
    scope: "task",
    panel: "directory",
  });
  $("#task-pending-tree").innerHTML = renderTreeNodes(pendingResult.nodes, {
    mode: "pending",
    emptyMessage: t("tasks.no_pending_items", "暂无待补传项。"),
    normalized: true,
    scope: "task",
    panel: "pending",
  });
  $("#task-directory-filter-summary").textContent = detail
    ? renderTreeFilterSummary(directoryResult, "目录节点", "directory")
    : t("tasks.waiting_directory_data", "等待任务数据...");
  $("#task-pending-filter-summary").textContent = detail
    ? renderTreeFilterSummary(pendingResult, "待补传节点", "pending")
    : t("tasks.waiting_directory_data", "等待任务数据...");
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
    emptyMessage: t("tasks.no_directory_states", "暂无目录状态。"),
    normalized: true,
    scope: "status",
    panel: "directory",
  });
  $("#status-pending-tree").innerHTML = renderTreeNodes(pendingResult.nodes, {
    mode: "pending",
    emptyMessage: t("tasks.no_pending_items", "暂无待补传项。"),
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
    showFlash(
      tf(
        "tasks.flash_no_visible_tree_paths",
        { label: panel === "pending" ? t("tasks.sync_to_pending_tree", "待补传树") : t("tasks.sync_to_directory_tree", "目录树") },
        `当前${panel === "pending" ? "待补传树" : "目录树"}没有可复制的路径`,
      ),
      true,
    );
    return;
  }
  await copyTextToClipboard(paths.join("\n"));
  showFlash(
    tf(
      "tasks.flash_copied_tree_paths",
      { count: paths.length, label: panel === "pending" ? "待补传" : "目录" },
      `已复制 ${paths.length} 条${panel === "pending" ? "待补传" : "目录"}路径`,
    ),
  );
}

async function copyTreeNodePaths(scope, panel, path) {
  const nodes = currentVisibleTreeNodes(scope, panel);
  const node = findTreeNodeByPath(nodes, path);
  if (!node) {
    showFlash(t("tasks.flash_tree_subtree_not_found", "当前筛选结果里未找到对应子树"), true);
    return;
  }
  const paths = flattenTreeNodePaths(node, panel === "pending" ? "pending" : "directory");
  if (!paths.length) {
    showFlash(t("tasks.flash_tree_subtree_empty", "当前子树没有可复制的路径"), true);
    return;
  }
  await copyTextToClipboard(paths.join("\n"));
  showFlash(
    tf(
      "tasks.flash_copied_tree_subtree_paths",
      { count: paths.length, label: panel === "pending" ? "待补传" : "目录" },
      `已复制 ${paths.length} 条${panel === "pending" ? "待补传" : "目录"}子树路径`,
    ),
  );
}

function focusTreeParentPath(scope, panel, path) {
  const parentPath = parentTreePath(path);
  if (!parentPath) {
    showFlash(t("tasks.flash_tree_already_top", "当前已经是最上层路径"), true);
    return;
  }
  focusTreePanelByPath(scope, panel, parentPath);
  showFlash(
    tf(
      "tasks.flash_tree_focused_by_parent",
      { path: parentPath, label: panel === "pending" ? t("tasks.sync_to_pending_tree", "待补传树") : t("tasks.sync_to_directory_tree", "目录树") },
      `已按父级路径 ${parentPath} 收敛${panel === "pending" ? "待补传树" : "目录树"}`,
    ),
  );
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
  showFlash(
    tf(
      "tasks.flash_tree_focused_by_path",
      { path: normalized, label: panel === "pending" ? t("tasks.sync_to_pending_tree", "待补传树") : t("tasks.sync_to_directory_tree", "目录树") },
      `已按 ${normalized} 收敛${panel === "pending" ? "待补传树" : "目录树"}`,
    ),
  );
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
    showFlash(t("tasks.flash_task_pending_focused_by_retry", "已按当前重试项定位待补传树"));
    return;
  }
  state.treeFilters.statusPending.query = path;
  setFilterControlValue("#status-pending-filter-query", path);
  updateStatusTreePanels(recentRuntimePayload());
  showFlash(t("tasks.flash_status_pending_focused_by_retry", "已按当前重试项定位最近待补传树"));
}

function focusRetryClass(scope, retryClass, retryState) {
  if (scope === "task") {
    state.treeFilters.taskRetry.retryClass = retryClass || "";
    state.treeFilters.taskRetry.retryState = retryState || "";
    setFilterControlValue("#task-retry-filter-class", retryClass || "");
    setFilterControlValue("#task-retry-filter-state", retryState || "");
    updateTaskRetryQueue(currentSelectedTaskDetail());
    showFlash(t("tasks.flash_task_retry_class_focused", "已收敛到当前同类重试队列"));
    return;
  }
  state.treeFilters.statusRetry.retryClass = retryClass || "";
  state.treeFilters.statusRetry.retryState = retryState || "";
  setFilterControlValue("#status-retry-filter-class", retryClass || "");
  setFilterControlValue("#status-retry-filter-state", retryState || "");
  updateStatusRetryQueue(recentRuntimePayload());
  showFlash(t("tasks.flash_status_retry_class_focused", "已收敛到最近同类重试队列"));
}

function focusTaskRetryByBlockedAction(action) {
  const preset = blockedActionFilterPreset(action);
  state.treeFilters.taskRetry.retryClass = preset.retryClass;
  state.treeFilters.taskRetry.retryState = preset.retryState;
  setFilterControlValue("#task-retry-filter-class", preset.retryClass);
  setFilterControlValue("#task-retry-filter-state", preset.retryState);
  updateTaskRetryQueue(currentSelectedTaskDetail());
  showFlash(t("tasks.flash_task_retry_blocked_action_focused", "已按当前 blocked action 收敛任务重试队列"));
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
    showFlash(t("tasks.flash_task_pending_not_found", "当前任务没有可定位的待补传路径"), true);
    return;
  }
  state.treeFilters.taskPending.query = path;
  setFilterControlValue("#task-pending-filter-query", path);
  updateTaskTreePanels(detail || currentSelectedTaskDetail());
  showFlash(t("tasks.flash_task_pending_focused", "已按当前任务定位待补传树"));
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
    showFlash(t("tasks.flash_blocked_task_missing", "当前 blocked 摘要没有可用样本任务"), true);
    return;
  }
  activateTab("tasks");
  if (!state.tasks.some((item) => item.task.id === taskID)) {
    await loadTasks();
  }
  if (!state.tasks.some((item) => item.task.id === taskID)) {
    showFlash(t("tasks.flash_blocked_task_not_found", "未找到对应样本任务，可能已被清理"), true);
    return;
  }
  state.selectedTaskId = taskID;
  renderTasks();
  renderSelectedTask();
  showFlash(t("tasks.flash_blocked_task_opened", "已打开 blocked 摘要对应的样本任务"));
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
  showFlash(t("tasks.flash_status_retry_blocked_action_focused", "已按 blocked action 收敛最近重试队列"));
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
    return "当前未附带真实样本来源说明，将沿用网盘源默认模板。";
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
  $("#profile-submit").textContent = t("providers.submit_create", "创建授权档案");
  syncAuthAssistInputs();
  syncAuthModes();
  syncProfileAuthGuide();
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
  $("#profile-submit").textContent = t("providers.submit_update", "更新授权档案");
  syncAuthAssistInputs();
  syncProfileAuthGuide();
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
    throw new Error(localizeAPIError(payload?.error, response.status));
  }
  return payload.data;
}

function syncSessionState() {
  $("#session-state").textContent = state.authenticated ? t("session_logged_in", "已登录") : t("session_logged_out", "未登录");
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
    targetRoot: detail.plan.metadata?.targetRoot || "/",
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

function parentComparePath(path) {
  const normalized = normalizeComparePath(path) || "/";
  if (normalized === "/") {
    return "/";
  }
  const index = normalized.lastIndexOf("/");
  if (index <= 0) {
    return "/";
  }
  return normalized.slice(0, index);
}

function getDirectoryBrowserConfig(kind) {
  if (kind === "target") {
    return {
      providerSelector: "#plan-target-provider",
      profileSelector: "#plan-target-profile",
      pathSelector: "#plan-target-browser-path",
      selectionSelector: "#plan-target-browser-selection",
      listSelector: "#plan-target-browser-list",
      emptyMessage: t("wizard.target_browser_empty_default", "选择目标网盘源和目标授权档案后可浏览目录。"),
    };
  }
  return {
    providerSelector: "#plan-source-provider",
    profileSelector: "#plan-source-profile",
    pathSelector: "#plan-source-browser-path",
    selectionSelector: "#plan-source-browser-selection",
    listSelector: "#plan-source-browser-list",
    emptyMessage: t("wizard.source_browser_empty_default", "选择源网盘源和源授权档案后可浏览目录。"),
  };
}

function directoryBrowserScope(kind) {
  const config = getDirectoryBrowserConfig(kind);
  return {
    providerKey: $(config.providerSelector)?.value || "",
    profileId: $(config.profileSelector)?.value || "",
  };
}

function directoryBrowserHint(kind, browser, providerKey) {
  const provider = findProviderEntry(providerKey);
  const displayName = provider?.meta?.displayName || providerKey || "当前网盘源";
  const capability = provider?.capability || {};
  if (!providerKey) {
    return t("wizard.browser_choose_provider_first", "请先选择网盘源，再继续浏览目录。");
  }
  if (!capability.supportsList) {
    return kind === "target"
      ? tf("wizard.browser_no_list_target", { provider: displayName }, `${displayName} 当前未声明目录浏览能力，请直接在“目标根目录”输入框手动填写目标路径。`)
      : tf("wizard.browser_no_list_source", { provider: displayName }, `${displayName} 当前未声明目录浏览能力，请直接在“选定根目录(JSON)”输入框手动填写源目录路径。`);
  }
  if (!browser.currentFileId && normalizeComparePath(browser.currentPath) !== "/") {
    return tf("wizard.browser_missing_ids", { provider: displayName }, `${displayName} 当前目录未返回稳定 fileId / parentId，子目录创建或部分跳转可能受限。`);
  }
  if (browser.error) {
    return kind === "target"
      ? tf("wizard.browser_load_fail_target", { provider: displayName }, `${displayName} 目录加载失败时，可先验证授权档案、切回根目录，或直接在“目标根目录”输入框手动填写路径。`)
      : tf("wizard.browser_load_fail_source", { provider: displayName }, `${displayName} 目录加载失败时，可先验证授权档案、切回根目录，或直接在“选定根目录(JSON)”输入框手动填写路径。`);
  }
  return tf("wizard.browser_ready", { provider: displayName }, `${displayName} 已启用目录浏览，可直接点选目录回填到任务向导。`);
}

function syncManualDirectoryInputHints() {
  const selectedRootsHint = $("#plan-selected-roots-hint");
  const targetRootHint = $("#plan-target-root-hint");
  if (!selectedRootsHint || !targetRootHint) {
    return;
  }
  const sourceScope = directoryBrowserScope("source");
  const targetScope = directoryBrowserScope("target");
  const sourceProvider = findProviderEntry(sourceScope.providerKey);
  const targetProvider = findProviderEntry(targetScope.providerKey);
  const sourceSupportsList = Boolean(sourceProvider?.capability?.supportsList);
  const targetSupportsList = Boolean(targetProvider?.capability?.supportsList);
  selectedRootsHint.textContent = sourceSupportsList
    ? t("wizard.selected_roots_manual_supported", '也可手动填写，例如 ["/电影","/相册/旅行"]；只同步整个根目录时填写 ["/"]。')
    : t("wizard.selected_roots_manual_required", '当前源网盘源可能不支持目录浏览，建议直接手动填写 JSON 路径数组，例如 ["/电影","/相册/旅行"]；只同步根目录时填写 ["/"]。');
  targetRootHint.textContent = targetSupportsList
    ? t("wizard.target_root_manual_supported", "也可手动填写目标目录，例如 /归档/2026；如果直接写入目标根目录，可保留 /。")
    : t("wizard.target_root_manual_required", "当前目标网盘源可能不支持目录浏览，建议直接手动填写目标目录路径，例如 /归档/2026；如果直接写入目标根目录，可保留 /。");
}

function spotlightDirectoryBrowserSelection(kind) {
  const listNode = $(kind === "target" ? "#plan-target-browser-list" : "#plan-source-browser-list");
  if (!listNode) {
    return;
  }
  const activeItem = listNode.querySelector(".directory-browser-item.active");
  if (!activeItem) {
    return;
  }
  activeItem.scrollIntoView({ block: "center", behavior: "smooth" });
  activeItem.classList.remove("selection-spotlight");
  void activeItem.offsetWidth;
  activeItem.classList.add("selection-spotlight");
}

function renderDirectoryBrowserLevel(path) {
  const currentPath = normalizeComparePath(path) || "/";
  if (currentPath === "/") {
    return t("wizard.browser_level_root", "当前层级：根目录");
  }
  const segments = currentPath.split("/").filter(Boolean);
  const currentName = segments[segments.length - 1] || "/";
  return tf("wizard.browser_level_named", { level: segments.length, name: currentName }, `当前层级：第 ${segments.length} 层目录（${currentName}）`);
}

function renderDirectoryBrowserEmptyState(kind, browser) {
  if (browser.loading) {
    return t("wizard.browser_loading", "正在加载目录列表...");
  }
  if (browser.error) {
    return kind === "target"
      ? tf("wizard.browser_error_target", { error: browser.error }, `目录加载失败：${browser.error}。建议先刷新目录、切回根目录；如果仍失败，可直接改填“目标根目录”。`)
      : tf("wizard.browser_error_source", { error: browser.error }, `目录加载失败：${browser.error}。建议先刷新目录、切回根目录；如果仍失败，可直接改填“选定根目录(JSON)”。`);
  }
  if (kind === "target") {
    return t("wizard.browser_empty_target", "当前目录下没有可继续浏览的子目录。你可以直接使用当前目录，或在这里新建子目录。");
  }
  return t("wizard.browser_empty_source", "当前目录下没有可继续浏览的子目录。你可以直接使用当前目录作为源目录。");
}

function renderDirectoryBrowserSelectionSummary(kind, selectedPath) {
  const normalized = normalizeComparePath(selectedPath) || "/";
  if (kind === "target") {
    return tf("wizard.browser_target_manual_fill", { path: normalized }, `已回填到目标根目录：${normalized}`);
  }
  return tf("wizard.browser_source_manual_fill", { path: normalized }, `已回填到选定根目录(JSON)：${normalized}`);
}

function resetDirectoryBrowser(kind) {
  state.directoryBrowsers[kind] = {
    currentPath: "/",
    items: [],
    loading: false,
    lastLoadedPath: "",
    selectedPath: "",
    error: "",
    scopeKey: "",
    currentFileId: "",
    currentParentId: "",
  };
}

function readSourceBrowserSelection() {
  try {
    const parsed = parseJSONInput($("#plan-selected-roots").value, []);
    return Array.isArray(parsed) && parsed.length ? normalizeComparePath(parsed[0]) || "/" : "/";
  } catch (error) {
    return "/";
  }
}

function readTargetBrowserSelection() {
  return normalizeComparePath($("#plan-target-root").value) || "/";
}

function directoryBrowserSelection(kind) {
  return kind === "target" ? readTargetBrowserSelection() : readSourceBrowserSelection();
}

function normalizeDirectoryBrowserItems(items, currentPath) {
  if (!Array.isArray(items)) {
    return [];
  }
  return items
    .filter((item) => item && typeof item === "object")
    .map((item) => {
      const name = String(item.name || "").trim();
      const basePath = normalizeComparePath(currentPath) || "/";
      const fallbackPath = name ? (basePath === "/" ? `/${name}` : `${basePath}/${name}`) : basePath;
      const path = normalizeComparePath(item.path || fallbackPath) || "/";
      return {
        name: name || (path === "/" ? "/" : path.split("/").pop() || path),
        path,
        isDir: Boolean(item.isDir),
        fileId: String(item.fileId || ""),
        parentId: String(item.parentId || ""),
      };
    })
    .filter((item) => item.isDir)
    .sort((left, right) => left.path.localeCompare(right.path, "zh-CN"));
}

function renderDirectoryBrowser(kind) {
  const browser = state.directoryBrowsers[kind];
  const config = getDirectoryBrowserConfig(kind);
  const { providerKey, profileId } = directoryBrowserScope(kind);
  const selectedPath = directoryBrowserSelection(kind);
  const pathNode = $(config.pathSelector);
  const breadcrumbsNode = $(kind === "target" ? "#plan-target-browser-breadcrumbs" : "#plan-source-browser-breadcrumbs");
  const levelNode = $(kind === "target" ? "#plan-target-browser-level" : "#plan-source-browser-level");
  const selectionNode = $(config.selectionSelector);
  const hintNode = $(kind === "target" ? "#plan-target-browser-hint" : "#plan-source-browser-hint");
  const listNode = $(config.listSelector);
  if (!pathNode || !breadcrumbsNode || !levelNode || !selectionNode || !hintNode || !listNode) {
    return;
  }

  pathNode.innerHTML = `<code>${escapeHTML(browser.currentPath || "/")}</code>`;
  const currentPath = normalizeComparePath(browser.currentPath) || "/";
  const segments = currentPath === "/" ? [] : currentPath.split("/").filter(Boolean);
  const breadcrumbParts = [
    `
      <button type="button" class="ghost" data-browser-breadcrumb="${escapeHTML(kind)}" data-browser-path="/">${escapeHTML(t("wizard.browser_root", "根目录"))}</button>
    `,
  ];
  let partialPath = "";
  segments.forEach((segment) => {
    partialPath += `/${segment}`;
    breadcrumbParts.push(`<span class="separator">/</span>`);
    breadcrumbParts.push(`
      <button
        type="button"
        class="ghost"
        data-browser-breadcrumb="${escapeHTML(kind)}"
        data-browser-path="${escapeHTML(partialPath)}"
      >${escapeHTML(segment)}</button>
    `);
  });
  breadcrumbsNode.innerHTML = breadcrumbParts.join("");
  levelNode.textContent = renderDirectoryBrowserLevel(currentPath);
  selectionNode.innerHTML = `
    <div class="browser-selection-note">${escapeHTML(renderDirectoryBrowserSelectionSummary(kind, selectedPath || "/"))}</div>
    <code>${escapeHTML(selectedPath || "/")}</code>
  `;
  hintNode.textContent = directoryBrowserHint(kind, browser, providerKey);
  const refreshButton = kind === "target" ? $("#plan-target-browser-refresh") : $("#plan-source-browser-refresh");
  const upButton = kind === "target" ? $("#plan-target-browser-up") : $("#plan-source-browser-up");
  const selectCurrentButton = kind === "target" ? $("#plan-target-browser-select-current") : $("#plan-source-browser-select-current");
  const createButton = kind === "target" ? $("#plan-target-browser-create") : null;
  const createInput = kind === "target" ? $("#plan-target-browser-create-name") : null;
  if (refreshButton) {
    refreshButton.disabled = browser.loading || !providerKey || !profileId;
  }
  if (upButton) {
    upButton.disabled = browser.loading || !providerKey || !profileId || normalizeComparePath(browser.currentPath) === "/";
  }
  if (selectCurrentButton) {
    selectCurrentButton.disabled = browser.loading || !providerKey || !profileId;
  }
  if (createButton) {
    createButton.disabled = browser.loading || !providerKey || !profileId;
  }
  if (createInput) {
    createInput.disabled = browser.loading || !providerKey || !profileId;
  }

  if (!providerKey || !profileId) {
    listNode.innerHTML = `<div class="directory-empty">${escapeHTML(config.emptyMessage)}</div>`;
    return;
  }
  if (browser.loading) {
    listNode.innerHTML = `<div class="directory-empty">${escapeHTML(renderDirectoryBrowserEmptyState(kind, browser))}</div>`;
    return;
  }
  if (browser.error) {
    listNode.innerHTML = `<div class="directory-empty">${escapeHTML(renderDirectoryBrowserEmptyState(kind, browser))}</div>`;
    return;
  }
  if (!Array.isArray(browser.items) || !browser.items.length) {
    listNode.innerHTML = `<div class="directory-empty">${escapeHTML(renderDirectoryBrowserEmptyState(kind, browser))}</div>`;
    return;
  }
  listNode.innerHTML = browser.items
    .map(
      (item) => {
        const isSelected = normalizeComparePath(item.path) === normalizeComparePath(selectedPath);
        const isCurrent = normalizeComparePath(item.path) === normalizeComparePath(browser.currentPath);
        return `
        <article class="directory-browser-item${isSelected ? " active" : ""}" data-directory-path="${escapeHTML(item.path)}">
          <strong>${escapeHTML(item.name)}</strong>
          <div class="muted"><code>${escapeHTML(item.path)}</code></div>
          <div class="meta-row">
            ${isCurrent ? `<span class="pill">${escapeHTML(t("wizard.browser_pill_current", "当前目录"))}</span>` : ""}
            ${isSelected ? `<span class="pill">${escapeHTML(t("wizard.browser_pill_selected", "已选目录"))}</span>` : ""}
          </div>
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-browser-open="${escapeHTML(kind)}"
              data-browser-path="${escapeHTML(item.path)}"
              data-browser-file-id="${escapeHTML(item.fileId)}"
            >${escapeHTML(t("wizard.browser_open", "打开目录"))}</button>
            <button
              type="button"
              class="ghost"
              data-browser-select="${escapeHTML(kind)}"
              data-browser-path="${escapeHTML(item.path)}"
            >${escapeHTML(kind === "target" ? t("wizard.browser_use_this", "使用此目录") : t("wizard.browser_use_this", "使用此目录"))}</button>
          </div>
        </article>
      `;
      },
    )
    .join("");
  spotlightDirectoryBrowserSelection(kind);
  syncManualDirectoryInputHints();
}

function renderAllDirectoryBrowsers() {
  renderDirectoryBrowser("source");
  renderDirectoryBrowser("target");
  syncManualDirectoryInputHints();
}

async function loadDirectoryBrowser(kind, path = "/", options = {}) {
  const { providerKey, profileId } = directoryBrowserScope(kind);
  const config = getDirectoryBrowserConfig(kind);
  if (!providerKey || !profileId) {
    resetDirectoryBrowser(kind);
    renderDirectoryBrowser(kind);
    return;
  }
  const scopeKey = `${providerKey}:${profileId}`;
  if (state.directoryBrowsers[kind].scopeKey !== scopeKey) {
    resetDirectoryBrowser(kind);
    state.directoryBrowsers[kind].scopeKey = scopeKey;
  }
  const browser = state.directoryBrowsers[kind];
  const targetPath = normalizeComparePath(path || browser.currentPath || "/") || "/";
  browser.loading = true;
  browser.error = "";
  renderDirectoryBrowser(kind);
  try {
    const listResult = await api(`/api/providers/${encodeURIComponent(providerKey)}/list`, {
      method: "POST",
      body: {
        profileId,
        path: targetPath,
        parentId: options.parentId || "",
        pageSize: 200,
      },
    });
    let metadataResult = null;
    try {
      metadataResult = await api(`/api/providers/${encodeURIComponent(providerKey)}/metadata`, {
        method: "POST",
        body: {
          profileId,
          path: targetPath,
        },
      });
    } catch (error) {
      metadataResult = null;
    }
    browser.currentPath = targetPath;
    browser.lastLoadedPath = targetPath;
    browser.currentFileId = String(metadataResult?.entry?.fileId || options.fileId || "");
    browser.currentParentId = String(metadataResult?.entry?.parentId || "");
    browser.items = normalizeDirectoryBrowserItems(listResult.items || [], targetPath);
    browser.loading = false;
    browser.error = "";
    renderDirectoryBrowser(kind);
  } catch (error) {
    browser.loading = false;
    browser.error = error.message;
    renderDirectoryBrowser(kind);
  }
}

async function refreshDirectoryBrowsers() {
  await Promise.all([loadDirectoryBrowser("source"), loadDirectoryBrowser("target")]);
}

function applyDirectoryBrowserSelection(kind, path) {
  const normalized = normalizeComparePath(path) || "/";
  if (kind === "target") {
    setInputValueIfPresent("#plan-target-root", normalized);
    state.directoryBrowsers.target.selectedPath = normalized;
    renderDirectoryBrowser("target");
    showFlash(tf("wizard.flash_target_root_selected", { path: normalized }, `已将 ${normalized} 回填到“目标根目录”`));
    return;
  }
  $("#plan-selected-roots").value = JSON.stringify([normalized], null, 2);
  state.directoryBrowsers.source.selectedPath = normalized;
  renderDirectoryBrowser("source");
  showFlash(tf("wizard.flash_source_root_selected", { path: normalized }, `已将 ${normalized} 回填到“选定根目录(JSON)”`));
}

async function createTargetBrowserDirectory() {
  const { providerKey, profileId } = directoryBrowserScope("target");
  const browser = state.directoryBrowsers.target;
  const dirName = $("#plan-target-browser-create-name").value.trim();
  if (!providerKey || !profileId) {
    showFlash(t("wizard.flash_choose_target_profile_first", "请先选择目标网盘源和目标授权档案"), true);
    return;
  }
  if (!dirName) {
    showFlash(t("wizard.flash_enter_new_dir_name", "请先输入新建目录名称"), true);
    return;
  }
  if (browser.currentPath !== "/" && !browser.currentFileId) {
    showFlash(t("wizard.flash_browser_missing_parent_id", "当前目录缺少 parentId / fileId，暂时无法创建子目录"), true);
    return;
  }
  await api(`/api/providers/${encodeURIComponent(providerKey)}/create_dir`, {
    method: "POST",
    body: {
      profileId,
      parentId: browser.currentPath === "/" ? "" : browser.currentFileId,
      dirName,
    },
  });
  const createdPath = normalizeComparePath(`${browser.currentPath === "/" ? "" : browser.currentPath}/${dirName}`) || "/";
  $("#plan-target-browser-create-name").value = "";
  state.directoryBrowsers.target.selectedPath = createdPath;
  showFlash(tf("wizard.flash_target_dir_created", { path: browser.currentPath, name: dirName }, `已在 ${browser.currentPath} 下创建目录 ${dirName}，并自动进入该目录回填到“目标根目录”`));
  await loadDirectoryBrowser("target", createdPath);
  applyDirectoryBrowserSelection("target", createdPath);
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
  setInputValueIfPresent("#plan-target-root", detail.plan.metadata?.targetRoot || "/");
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
      ? () => showFlash(t("wizard.flash_prefill_task_required", "请先选择任务"), true)
      : () => {
          activateTab("wizard");
          prefillWizardFromTask(detail);
          showFlash(t("wizard.flash_prefill_task_loaded", "已按当前任务重建向导参数"));
        };
  }

  if (copyButton) {
    copyButton.disabled = disabled;
    copyButton.onclick = disabled
      ? () => showFlash(t("wizard.flash_copy_task_required", "请先选择任务"), true)
      : async () => {
          try {
            await copyTextToClipboard(formatJSON(payload));
            showFlash(t("wizard.flash_copy_task_payload_done", "任务创建参数已复制到剪贴板"));
          } catch (error) {
            showFlash(t("wizard.flash_copy_failed", "复制失败：{error}").replace("{error}", error.message), true);
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
          <h3>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? t("tasks.auto_recover_takeover_title", "等待后台自动补传接管") : t("tasks.upload_checkpoint_resume_title", "等待上传会话自动续跑"))}</h3>
          <div class="meta-row">
            <span class="pill">${escapeHTML(stringifyValue(retrySummary.autoRecoverMode, "upload_checkpoint_auto_resume"))}</span>
            ${nextRetryAt ? `<span class="pill">${escapeHTML(nextRetryAt)}</span>` : ""}
          </div>
          <ol class="checklist">
            <li>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? t("tasks.auto_recover_takeover_step_1", "当前队列满足后台自动补传条件，系统随后会自动尝试继续执行。") : t("tasks.upload_checkpoint_resume_step_1", "当前失败队列携带可恢复的 upload checkpoint，单机执行进程会优先尝试恢复上传会话。"))}</li>
            <li>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? t("tasks.auto_recover_takeover_step_2", "先到状态矩阵查看后台补传候选池，确认该任务是否已经进入重试队列自动补传通道（retry_queue_auto_retry）。") : t("tasks.upload_checkpoint_resume_step_2", "先到状态矩阵查看后台补传候选池，确认该任务是否已经进入上传断点自动续跑通道（upload checkpoint auto-resume）。"))}</li>
            <li>${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? t("tasks.auto_recover_takeover_step_3", "如果长时间未自动推进，再检查重试摘要（retrySummary）、网盘返回状态和风险时间窗是否把它留在等待态。") : t("tasks.upload_checkpoint_resume_step_3", "如果长时间未自动推进，再检查 providerData / uploadId / nextPartNumber 等恢复线索是否完整。"))}</li>
          </ol>
          <div class="muted">${escapeHTML(retrySummary.autoRecoverAdvice || t("tasks.auto_recover_takeover_default_advice", "当前失败队列都带可恢复的 upload checkpoint，单机执行进程会优先尝试恢复上传会话。"))}</div>
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-task-guide-view="status"
              data-task-guide-intent="focus_status_auto_recover_mode"
              data-task-guide-mode="${escapeHTML(stringifyValue(retrySummary.autoRecoverMode, "upload_checkpoint_auto_resume"))}"
              data-task-guide-provider="${escapeHTML(providerKey)}"
              data-task-guide-profile="${escapeHTML(profileId)}"
            >${escapeHTML(retrySummary.autoRecoverMode === "retry_queue_auto_retry" ? t("tasks.focus_auto_recover_only", "只看自动补传候选") : t("tasks.focus_checkpoint_only", "只看自动续跑候选"))}</button>
            <button
              type="button"
              class="ghost"
              data-task-guide-view="status"
              data-task-guide-intent="focus_status_open"
              data-task-guide-provider="${escapeHTML(providerKey)}"
              data-task-guide-profile="${escapeHTML(profileId)}"
            >${escapeHTML(t("tasks.open_status_matrix", "打开状态矩阵"))}</button>
          </div>
        </div>
      `;
    }
    return `
      <div class="insight-card">
        <strong>${escapeHTML(t("tasks.task_next_steps_title", "下一步处理"))}</strong>
        <span>${escapeHTML(t("tasks.task_next_steps_idle", "当前任务没有需要人工处理的阻塞动作（blocked），可直接继续运行或观察状态矩阵。"))}</span>
      </div>
    `;
  }

  const stepsByAction = {
    refresh_auth_profile: {
      title: t("tasks.guide_refresh_auth_title", "刷新授权档案"),
      steps: [
        t("tasks.guide_refresh_auth_step_1", "切到“网盘源 / 授权”面板，定位当前目标端授权档案。"),
        t("tasks.guide_refresh_auth_step_2", "更新 token / cookie 后先执行授权验证，确认授权恢复正常。"),
        t("tasks.guide_refresh_auth_step_3", "回到任务详情页，再执行继续或重试。"),
      ],
      buttons: [
        { label: t("tasks.guide_open_providers", "打开授权面板"), view: "providers", providerKey, profileId, intent: "focus_profile" },
        { label: t("tasks.guide_focus_auth_queue", "只看授权失效队列"), view: "tasks", intent: "focus_task_retry" },
        { label: t("tasks.guide_open_status_blocked", "按当前阻塞打开状态矩阵"), view: "status", intent: "focus_status_blocked" },
      ],
    },
    restore_local_source_file: {
      title: t("tasks.guide_restore_local_title", "补回本地回退文件"),
      steps: [
        t("tasks.guide_restore_local_step_1", "先补回源文件或校正本地兜底路径，确保 localPath 对应文件真实存在。"),
        t("tasks.guide_restore_local_step_2", "如果路径配置有误，建议回到任务向导核对 entries / selectedRoots。"),
        t("tasks.guide_restore_local_step_3", "补齐后返回任务详情页重新重试。"),
      ],
      buttons: [
        { label: t("tasks.focus_pending_tree", "定位待补传树"), view: "tasks", intent: "focus_task_pending" },
        { label: t("tasks.guide_focus_local_missing", "只看本地缺失队列"), view: "tasks", intent: "focus_task_retry" },
        { label: t("tasks.guide_open_wizard", "打开任务向导"), view: "wizard", providerKey, intent: "prefill_wizard" },
        { label: t("tasks.guide_open_status_blocked", "按当前阻塞打开状态矩阵"), view: "status", intent: "focus_status_blocked" },
      ],
    },
    manual_intervention_required: {
      title: t("tasks.guide_manual_intervention_title", "修复 provider 会话缺口"),
      steps: [
        t("tasks.guide_manual_intervention_step_1", "当前重试分类（retryClass）是 provider_session_missing，说明网盘返回体缺少 uploadid、upload session 这类关键会话字段。"),
        t("tasks.guide_manual_intervention_step_2", "先核对网盘返回体、上传会话构建逻辑和目标端授权档案，确认是否需要重新生成会话或刷新授权。"),
        t("tasks.guide_manual_intervention_step_3", "修复后回到状态矩阵，确认这类阻塞项已经收敛，再执行重试。"),
      ],
      buttons: [
        { label: t("tasks.guide_focus_session_missing", "只看会话缺口队列"), view: "tasks", intent: "focus_task_retry" },
        { label: t("tasks.guide_open_providers", "打开授权面板"), view: "providers", providerKey, profileId, intent: "focus_profile" },
        { label: t("tasks.guide_open_status_blocked", "按当前阻塞打开状态矩阵"), view: "status", intent: "focus_status_blocked" },
      ],
    },
    wait_for_cooldown: {
      title: t("tasks.guide_wait_cooldown_title", "等待冷却窗口结束"),
      steps: [
        nextRetryAt ? tf("tasks.guide_wait_cooldown_step_1", { time: nextRetryAt }, `当前最早自动补传时间是 ${nextRetryAt}。`) : t("tasks.guide_wait_cooldown_step_1_fallback", "当前处于风控冷却窗口。"),
        t("tasks.guide_wait_cooldown_step_2", "冷却期间无需手动重试，系统会在窗口结束后自动尝试补传。"),
        t("tasks.guide_wait_cooldown_step_3", "如果想确认整体阻塞分布，可切到状态矩阵查看阻塞聚合看板。"),
      ],
      buttons: [
        { label: t("tasks.guide_focus_cooldown", "只看冷却队列"), view: "tasks", intent: "focus_task_retry" },
        { label: t("tasks.guide_open_status_blocked", "按当前阻塞打开状态矩阵"), view: "status", intent: "focus_status_blocked" },
      ],
    },
    wait_for_retry_window: {
      title: t("tasks.guide_wait_window_title", "等待自动补传时间窗"),
      steps: [
        nextRetryAt ? tf("tasks.guide_wait_window_step_1", { time: nextRetryAt }, `当前下一次允许自动补传的时间是 ${nextRetryAt}。`) : t("tasks.guide_wait_window_step_1_fallback", "当前不在允许的自动补传时间窗内。"),
        t("tasks.guide_wait_window_step_2", "这类任务仍会留在自动补传候选池里，但在时间窗开始前不会被后台执行进程实际执行。"),
        t("tasks.guide_wait_window_step_3", "如果需要排查影响范围，可切到状态矩阵按阻塞动作（blocked action）或通道（lane）直接聚焦。"),
      ],
      buttons: [
        { label: t("tasks.guide_focus_window", "只看时间窗等待态"), view: "status" },
        { label: t("tasks.guide_focus_current_retry", "只看当前任务重试队列"), view: "tasks", intent: "focus_task_retry" },
      ],
    },
    manual_confirmation_required: {
      title: t("tasks.guide_manual_confirmation_title", "等待人工确认"),
      steps: [
        t("tasks.guide_manual_confirmation_step_1", "当前任务存在 pending_manual 项，说明网盘仍需要人工确认，或要等待后续兜底运行能力补齐。"),
        t("tasks.guide_manual_confirmation_step_2", "先在状态矩阵和待补传树里确认影响范围，再决定是否拆分任务或等待后续链路补齐。"),
        t("tasks.guide_manual_confirmation_step_3", "确认后再回到任务详情执行重试。"),
      ],
      buttons: [
        { label: t("tasks.focus_pending_tree", "定位待补传树"), view: "tasks", intent: "focus_task_pending" },
        { label: t("tasks.guide_focus_manual_pending", "只看待确认队列"), view: "tasks", intent: "focus_task_retry" },
        { label: t("tasks.guide_open_status_blocked", "按当前阻塞打开状态矩阵"), view: "status", intent: "focus_status_blocked" },
        { label: t("tasks.guide_stay_task_detail", "留在任务详情"), view: "tasks" },
      ],
    },
    review_and_reset_retry_strategy: {
      title: t("tasks.guide_review_retry_title", "调整重试策略"),
      steps: [
        t("tasks.guide_review_retry_step_1", "当前任务已经达到重试上限（retryLimit），继续原样重试也不会再推进。"),
        t("tasks.guide_review_retry_step_2", "建议回到任务向导调整 riskOverride / retryLimit / 执行策略，必要时拆成更小任务。"),
        t("tasks.guide_review_retry_step_3", "创建新任务后，用状态矩阵对比新的阻塞分布是否收敛。"),
      ],
      buttons: [
        { label: t("tasks.guide_focus_exhausted", "只看已耗尽队列"), view: "tasks", intent: "focus_task_retry" },
        { label: t("tasks.guide_open_wizard", "打开任务向导"), view: "wizard", providerKey, intent: "prefill_wizard" },
        { label: t("tasks.guide_open_status_blocked", "按当前阻塞打开状态矩阵"), view: "status", intent: "focus_status_blocked" },
      ],
    },
  };

  const config = stepsByAction[action] || {
    title: t("tasks.guide_manual_fallback_title", "人工处理建议"),
    steps: [advice || t("tasks.guide_manual_fallback_step", "请根据阻塞原因检查授权、源文件和网盘返回状态。")],
    buttons: [{ label: t("tasks.open_status_matrix", "打开状态矩阵"), view: "status" }],
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
          showFlash(t("wizard.flash_focus_profile", "已定位到当前授权档案"));
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
          showFlash(t("wizard.flash_focus_auto_recover_mode", "已按 {mode} 收敛后台补传候选").replace("{mode}", button.dataset.taskGuideMode || "-"));
        }
        if (intent === "focus_status_open") {
          activateTab("status");
          showFlash(t("wizard.flash_open_status", "已打开状态矩阵"));
        }
      }
      if (view === "wizard") {
        if (intent === "prefill_wizard") {
          prefillWizardFromTask(detail);
          showFlash(t("wizard.flash_prefill_task_from_detail", "已按当前任务预填任务向导参数"));
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
            <span class="pill">${escapeHTML(renderProviderLifecycleStatusLabel(entry.meta.status))}</span>
          </div>
          <div class="meta-row">
            ${entry.meta.authModes.map((mode) => `<span class="pill">${escapeHTML(renderAuthModeLabel(mode))}</span>`).join("")}
          </div>
          <div class="provider-card-summary">
            <div class="muted">核心能力：${escapeHTML(renderProviderCapabilityCompact(entry.capability))}</div>
            <div class="muted">推荐风控：${escapeHTML(renderRiskModeLabel(defaultRiskTemplate?.recommendedMode))}</div>
            <div class="muted">授权入口：先创建授权档案，再继续目录选择与任务创建。</div>
          </div>
          <details class="provider-card-advanced">
            <summary>展开高级信息</summary>
            <div class="provider-card-advanced-body">
              <div class="muted">兜底策略：${escapeHTML(fallbackModes.join(", ") || "-")}</div>
              <div class="muted">冲突策略：${escapeHTML(conflictPolicies.join(", ") || "-")}</div>
              <div class="muted">风控特征：${escapeHTML(riskTraits.join(", ") || "-")}</div>
              <div class="muted">风控提示：${escapeHTML(riskHints.join(" / ") || "-")}</div>
              <div class="muted">默认风控：${escapeHTML(renderRiskProfileCompact(defaultRiskTemplate?.calibrated))}</div>
              <div class="muted">推荐档位：${escapeHTML(renderRiskModeLabel(defaultRiskTemplate?.recommendedMode))}</div>
              <div class="muted">校准依据：${escapeHTML((defaultRiskTemplate?.calibrationReasons || []).join(" / ") || "-")}</div>
              <div class="muted">校准就绪度：${escapeHTML(stringifyValue(defaultRiskTemplate?.calibrationReadiness, "-"))}</div>
              <div class="muted">优先校准动作：${escapeHTML(stringifyValue(defaultRiskTemplate?.calibrationPriorityAction, "-"))}</div>
              <div class="muted">恢复预算：${escapeHTML(renderRecoverBudgetCompact(defaultRiskTemplate?.recoverBudget))}</div>
              <div class="muted">账号默认来源：${escapeHTML(renderRiskDefaultsSourceBadge(profileSource))}</div>
              <div class="muted">账号默认建议：${escapeHTML(renderProfileRiskDefaultSourceAdvice(profileSource))}</div>
            </div>
          </details>
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
  renderAllDirectoryBrowsers();
}

function prefillWizardFromTaskPath(detail, path, options = {}) {
  const payload = buildCreatePayloadFromTaskPath(detail, path, options);
  if (!payload) {
    showFlash(t("wizard.flash_prefill_task_required", "请先选择任务"), true);
    return;
  }
  prefillWizardFromPayload(payload);
  showFlash(t("wizard.flash_prefill_task_range", "已按 {path} 重建向导范围").replace("{path}", path));
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
  setInputValueIfPresent("#plan-target-root", payload.targetRoot || "/");
  $("#plan-entries").value = JSON.stringify(payload.entries || [], null, 2);
  syncExecutionModeHint();
  loadDirectoryBrowser("source", readSourceBrowserSelection()).catch(() => {});
  loadDirectoryBrowser("target", readTargetBrowserSelection()).catch(() => {});
}

function prefillWizardFromTaskPaths(detail, paths, label = "当前筛选", options = {}) {
  const payload = buildCreatePayloadFromTaskPaths(detail, paths, options);
  if (!payload) {
    showFlash(t("wizard.flash_prefill_task_required", "请先选择任务"), true);
    return;
  }
  prefillWizardFromPayload(payload);
  showFlash(t("wizard.flash_prefill_label_range", "已按{label}重建向导范围").replace("{label}", label));
}

function prefillWizardFromPreviewPaths(paths, label = "全部删除记录") {
  let payload;
  try {
    payload = buildPlanPayload();
  } catch (error) {
    showFlash(
      t("wizard.flash_prefill_payload_parse_error", "当前向导参数无法解析：{error}").replace("{error}", error.message),
      true,
    );
    return;
  }
  prefillWizardFromPayload(buildCreatePayloadFromPayloadPaths(payload, paths, { exactRoots: true }));
  showFlash(t("wizard.flash_prefill_label_range", "已按{label}重建向导范围").replace("{label}", label));
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
        showFlash(t("wizard.flash_prefill_missing_path", "缺少可重建路径"), true);
        return;
      }
      if (focusScope === "preview") {
        prefillWizardFromPreviewPaths([path], "此删除记录");
        return;
      }
      const context = taskContextByScope(focusScope);
      if (!context?.detail) {
        showFlash(
          focusScope === "task"
            ? t("wizard.flash_prefill_task_required", "请先选择任务")
            : t("wizard.flash_prefill_status_task_missing", "当前状态样本没有可用任务"),
          true,
        );
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
      showFlash(t("wizard.flash_prefill_missing_delete_paths", "当前没有可重建的删除记录"), true);
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
    syncProfileAuthGuide();
    return;
  }
  authModeSelect.innerHTML = provider.meta.authModes
    .map((mode) => `<option value="${mode}">${escapeHTML(renderAuthModeLabel(mode))}</option>`)
    .join("");
  syncProfileAuthGuide();
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
          <th>网盘源</th>
          <th>授权方式</th>
          <th>账号默认风控</th>
          <th>状态</th>
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
                <td>${escapeHTML(renderAuthModeLabel(profile.authMode))}</td>
                <td>
                  <div>${escapeHTML(profileRisk)}</div>
                  <div class="muted">账号默认来源: ${escapeHTML(renderRiskDefaultsSourceBadge(profileSource))}</div>
                  <div class="muted">账号默认建议: ${escapeHTML(profileAdvice)}</div>
                </td>
                <td>${escapeHTML(renderProfileStatusLabel(profile.status))}</td>
                <td>
                  <div class="actions compact">
                    <button type="button" class="ghost" data-profile-edit="${profile.id}">编辑</button>
                    <button type="button" class="ghost" data-profile-validate="${profile.id}">验证授权</button>
                    <button type="button" class="ghost" data-profile-delete="${profile.id}">删除</button>
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
        showFlash(t("wizard.flash_profile_missing_edit", "未找到要编辑的授权档案"), true);
        return;
      }
      setProfileFormEditing(profile);
      focusProfile(profile.id);
      showFlash(t("wizard.flash_profile_edit_loaded", "已载入授权档案编辑表单"));
    });
  });

  wrap.querySelectorAll("[data-profile-validate]").forEach((button) => {
    button.addEventListener("click", async () => {
      try {
        const result = await api(`/api/auth/profiles/${button.dataset.profileValidate}/validate`, { method: "POST" });
        showFlash(
          t("wizard.flash_profile_validate_done", "授权校验完成：{status}").replace(
            "{status}",
            renderProfileStatusLabel(result.status),
          ),
        );
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
        showFlash(t("wizard.flash_profile_deleted", "授权档案已删除"));
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
  resetDirectoryBrowser("target");
  syncTargetProviderInsight();
  syncTargetProfileInsight();
  renderDirectoryBrowser("target");
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
  resetDirectoryBrowser("source");
  renderDirectoryBrowser("source");
}

function syncAutoRecoverProviders() {
  const select = $("#auto-recover-provider");
  if (!select) {
    return;
  }
  const current = state.autoRecoverFilters.providerKey || select.value || "";
  const providerKeys = Array.from(new Set((state.providers || []).map((item) => item?.meta?.key).filter(Boolean))).sort();
  select.innerHTML = `<option value="">${escapeHTML(t("status.all_providers", "全部 provider"))}</option>${providerKeys
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
  select.innerHTML = `<option value="">${escapeHTML(t("status.all_protocol_groups", "全部协议族"))}</option>${values
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
  select.innerHTML = `<option value="">${escapeHTML(t("status.all_profiles", "全部授权档案"))}</option>${sorted
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
  select.innerHTML = `<option value="">${escapeHTML(t("status.all_blocked_actions", "全部阻塞动作"))}</option>${values
    .map((value) => `<option value="${value}">${value}</option>`)
    .join("")}`;
  setFilterControlValue("#auto-recover-blocked-action", values.includes(current) ? current : "");
}

function syncExecutionModeHint() {
  const mode = $("#plan-execution-mode").value;
  const hint = $("#plan-execution-hint");
  if (mode === "pre_scan_flat") {
    hint.textContent = t("wizard.execution_mode_prescan", "先完整扫描再执行（pre_scan_flat）适合目录较小、希望先拿到完整扫描结果后再统一执行的场景。");
    return;
  }
  hint.textContent = t("wizard.execution_mode_leaf_first", "按目录逐棵推进（leaf_first_lazy）是默认优先推荐模式，会按顶层目录顺序逐棵子树推进，只扫描当前真正需要传的目录。");
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

  const executionReason = localizeRecommendationReason(metadata.recommendedExecutionModeReason, t("wizard.recommendation_no_execution_reason", "暂无执行模式推荐原因"));
  const riskReason = localizeRecommendationReason(metadata.recommendedRiskModeReason, t("wizard.recommendation_no_risk_reason", "暂无风控推荐原因"));
  const aggressiveWarning = stringifyValue(metadata.aggressiveRiskWarning, "-");
  card.classList.remove("hidden");

  const titleParts = [];
  if (recommendedExecution) {
    titleParts.push(
      recommendedExecution === selectedExecution
        ? tf("wizard.recommendation_execution_applied", { mode: renderExecutionModeLabel(recommendedExecution) }, `执行模式已采用推荐值：${renderExecutionModeLabel(recommendedExecution)}`)
        : tf("wizard.recommendation_execution_suggested", { mode: renderExecutionModeLabel(recommendedExecution) }, `建议执行模式：${renderExecutionModeLabel(recommendedExecution)}`),
    );
  }
  if (recommendedRisk) {
    titleParts.push(
      recommendedRisk === selectedRisk
        ? tf("wizard.recommendation_risk_applied", { mode: recommendedRisk }, `风控档位已采用推荐值：${recommendedRisk}`)
        : tf("wizard.recommendation_risk_suggested", { mode: recommendedRisk }, `建议风控档位：${recommendedRisk}`),
    );
  }
  $("#plan-recommendation-title").textContent = titleParts.join(" / ");

  const reasonParts = [];
  if (recommendedExecution) {
    reasonParts.push(tf("wizard.recommendation_execution_reason", { reason: executionReason }, `执行模式：${executionReason}`));
  }
  if (recommendedRisk) {
    reasonParts.push(tf("wizard.recommendation_risk_reason", { reason: riskReason }, `风控档位：${riskReason}`));
  }
  if (aggressiveWarning && aggressiveWarning !== "-") {
    reasonParts.push(tf("wizard.recommendation_warning", { warning: aggressiveWarning }, `提示：${aggressiveWarning}`));
  }
  $("#plan-recommendation-reason").textContent = reasonParts.join(" | ");

  executionButton.disabled = !recommendedExecution || recommendedExecution === selectedExecution;
  riskButton.disabled = !recommendedRisk || recommendedRisk === selectedRisk;
}

function renderTasks() {
  const wrap = $("#tasks-list");
  if (!state.tasks.length) {
    wrap.innerHTML = `<div class="task-item">${escapeHTML(state.language === "en-US" ? "No tasks yet." : "暂无任务。")}</div>`;
    wireTaskQuickActions(null);
    $("#task-summary").innerHTML = `
      <div class="insight-card">
        <strong>${escapeHTML(t("tasks.execution_mode", "执行模式"))}</strong>
        <span>${escapeHTML(t("tasks.waiting_selected", "选择任务后显示"))}</span>
      </div>
    `;
    $("#task-runtime").innerHTML = `
      <div class="insight-card">
        <strong>${escapeHTML(t("tasks.runtime_checkpoint", "运行检查点"))}</strong>
        <span>${escapeHTML(t("tasks.waiting_selected", "选择任务后显示"))}</span>
      </div>
    `;
    updateTaskRetryQueue(null);
    $("#task-resolution-guide").innerHTML = `<div class="directory-empty">${escapeHTML(t("tasks.waiting_task_resolution", "选择任务后显示处理建议。"))}</div>`;
    updateTaskTreePanels(null);
    $("#task-detail").textContent = t("tasks.detail_waiting", "选择一条任务查看详情...");
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
        <strong>${escapeHTML(t("tasks.execution_mode", "执行模式"))}</strong>
        <span>${escapeHTML(t("tasks.waiting_selected", "选择任务后显示"))}</span>
      </div>
    `;
    $("#task-runtime").innerHTML = `
      <div class="insight-card">
        <strong>${escapeHTML(t("tasks.runtime_checkpoint", "运行检查点"))}</strong>
        <span>${escapeHTML(t("tasks.waiting_selected", "选择任务后显示"))}</span>
      </div>
    `;
    updateTaskRetryQueue(null);
    updateTaskTreePanels(null);
    $("#task-resolution-guide").innerHTML = `<div class="directory-empty">${escapeHTML(t("tasks.waiting_task_resolution", "选择任务后显示处理建议。"))}</div>`;
    $("#task-detail").textContent = t("tasks.detail_waiting", "选择一条任务查看详情...");
    return;
  }
  const metadata = detail.plan?.metadata || {};
  const runtime = detail.runtime || metadata.runtime || {};
  syncTaskActionButtons();
  wireTaskQuickActions(detail);
  $("#task-summary").innerHTML = `
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.execution_mode", "执行模式"))}</strong>
      <span>${stringifyValue(metadata.executionMode)}</span>
    </div>
    ${renderRuntimePathChips(t("tasks.selected_roots", "选定根目录"), metadata.selectedRoots || [], "task", "roots")}
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.target_root", "目标根目录"))}</strong>
      <span><code>${escapeHTML(stringifyValue(metadata.targetRoot, "/"))}</code></span>
    </div>
    ${renderRuntimePathChips(t("tasks.scan_trace", "扫描轨迹"), metadata.scanTrace || [], "task", "scan")}
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.recommended_mode", "推荐模式"))}</strong>
      <span>${stringifyValue(metadata.recommendedExecutionMode)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.recommended_reason", "推荐原因"))}</strong>
      <span>${stringifyValue(metadata.recommendedExecutionModeReason)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.scan_mode", "扫描方式"))}</strong>
      <span>${stringifyValue(metadata.scanMode, "尚未运行或无需扫描")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.risk_throttle", "风险节流"))}</strong>
      <span>${stringifyValue(metadata.riskProfile?.requestIntervalMs, "0")}ms / dir ${stringifyValue(metadata.riskProfile?.directoryIntervalMs, "0")}ms / retry ${stringifyValue(metadata.riskProfile?.retryLimit, "0")} / conc ${stringifyValue(metadata.riskProfile?.maxConcurrent, "0")}</span>
    </div>
    ${renderRiskResolutionMetaCards(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.recommended_risk", "推荐风控"))}</strong>
      <span>${stringifyValue(metadata.recommendedRiskMode, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.recommended_risk_reason", "推荐风控原因"))}</strong>
      <span>${stringifyValue(metadata.recommendedRiskModeReason, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.aggressive_warning", "激进风险提示"))}</strong>
      <span>${stringifyValue(metadata.aggressiveRiskWarning, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.source_delete_policy", "源端删除策略"))}</strong>
      <span>${renderSourceDeletePolicy(metadata.sourceDeletePolicy)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.risk_resolution", "风险模板解释"))}</strong>
      <span>${escapeHTML(renderRiskResolutionSummary(metadata.riskProfileResolution))}</span>
      ${renderRiskResolutionDetail(metadata.riskProfileResolution)}
    </div>
    ${renderRiskResolutionFlow(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_window", "自动补传时间窗"))}</strong>
      <span>${escapeHTML(renderRiskWindow(metadata.riskProfile))}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_scope", "重试范围"))}</strong>
      <span>${metadata.retryPendingOnly ? tf("tasks.pending_only_with_count", { count: Array.isArray(metadata.retryPendingPaths) ? metadata.retryPendingPaths.length : 0 }, `仅待处理项 (${Array.isArray(metadata.retryPendingPaths) ? metadata.retryPendingPaths.length : 0} 项)`) : t("tasks.full_task", "整个任务")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_mode", "重试模式"))}</strong>
      <span>${stringifyValue(metadata.retryMode, "default")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_source", "重试来源"))}</strong>
      <span>${stringifyValue(metadata.retryScope, metadata.retrySelectedPaths ? "selected_subset" : "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_selected_paths", "重试选中路径"))}</strong>
      <span>${Array.isArray(metadata.retrySelectedPaths) && metadata.retrySelectedPaths.length ? summarizePathList(metadata.retrySelectedPaths, 4) : "-"}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_selected_path_count", "重试路径数"))}</strong>
      <span>${stringifyValue(metadata.retrySelectedPathCount, Array.isArray(metadata.retrySelectedPaths) ? metadata.retrySelectedPaths.length : 0)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_checkpoint_count", "重试 checkpoint 数"))}</strong>
      <span>${stringifyValue(metadata.retryUploadCheckpointCount, "0")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.retry_summary", "重试摘要"))}</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedReason || (metadata.retrySummary?.shouldBlock ? "blocked" : "ready"), "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.next_summary", "下一步摘要"))}</strong>
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
      <strong>${escapeHTML(t("tasks.source_deletion_count", "源端删除记录"))}</strong>
      <span>${stringifyValue(runtime.sourceDeletionCount || metadata.deletedEntryCount, "0")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.suggested_action", "建议动作"))}</strong>
      <span>${stringifyValue(metadata.retrySummary?.blockedAction, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.auto_recover_candidate", "后台补传候选"))}</strong>
      <span>${renderAutoRecoverMode(metadata.retrySummary)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("tasks.queue_breakdown", "队列拆分"))}</strong>
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
        <strong>${escapeHTML(t("wizard.preview_meta_current_mode", "当前模式"))}</strong>
        <span>${escapeHTML(t("wizard.preview_waiting", "等待预览"))}</span>
      </div>
    `;
    $("#plan-preview").textContent = t("wizard.preview_waiting_text", "等待预览...");
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
      <strong>${escapeHTML(t("wizard.preview_meta_current_mode", "当前模式"))}</strong>
      <span>${stringifyValue(metadata.executionMode)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_selected_roots", "选定根目录"))}</strong>
      <span><code>${escapeHTML(summarizePathList(metadata.selectedRoots || []))}</code></span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_target_root", "目标根目录"))}</strong>
      <span><code>${escapeHTML(stringifyValue(metadata.targetRoot, "/"))}</code></span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_recommended_mode", "推荐模式"))}</strong>
      <span>${stringifyValue(metadata.recommendedExecutionMode)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_recommended_reason", "推荐原因"))}</strong>
      <span>${stringifyValue(metadata.recommendedExecutionModeReason)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_execution_order", "执行顺序"))}</strong>
      <span>${stringifyValue(metadata.executionOrder)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_risk_mode", "风险档位"))}</strong>
      <span>${stringifyValue(metadata.riskProfile?.mode, "balanced")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_risk_throttle", "风险节流"))}</strong>
      <span>${stringifyValue(metadata.riskProfile?.requestIntervalMs, "0")}ms / dir ${stringifyValue(metadata.riskProfile?.directoryIntervalMs, "0")}ms / retry ${stringifyValue(metadata.riskProfile?.retryLimit, "0")} / conc ${stringifyValue(metadata.riskProfile?.maxConcurrent, "0")}</span>
    </div>
    ${renderRiskResolutionMetaCards(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_recommended_risk", "推荐风控"))}</strong>
      <span>${stringifyValue(metadata.recommendedRiskMode, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_recommended_risk_reason", "推荐风控原因"))}</strong>
      <span>${stringifyValue(metadata.recommendedRiskModeReason, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_aggressive_warning", "激进风险提示"))}</strong>
      <span>${stringifyValue(metadata.aggressiveRiskWarning, "-")}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_risk_resolution", "风险模板解释"))}</strong>
      <span>${escapeHTML(renderRiskResolutionSummary(metadata.riskProfileResolution))}</span>
      ${renderRiskResolutionDetail(metadata.riskProfileResolution)}
    </div>
    ${renderRiskResolutionFlow(metadata.riskProfileResolution)}
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_retry_window", "自动补传时间窗"))}</strong>
      <span>${escapeHTML(renderRiskWindow(metadata.riskProfile))}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_source_delete_policy", "源端删除策略"))}</strong>
      <span>${renderSourceDeletePolicy(metadata.sourceDeletePolicy)}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("wizard.preview_meta_entry_counts", "有效条目 / 删除记录"))}</strong>
      <span>${stringifyValue(metadata.activeEntryCount, "0")} / ${stringifyValue(metadata.deletedEntryCount, "0")}</span>
    </div>
    ${hasDeletedRecords ? `
      <div class="insight-card checkpoint-card">
        <strong>${escapeHTML(t("wizard.preview_meta_delete_only", "删除记录仅用于定位"))}</strong>
        <div>${escapeHTML(deletionOnlyPreview ? t("wizard.preview_meta_delete_only_hint", "当前预览只剩删除记录，没有可执行条目；请先恢复源文件并重新预览。") : t("wizard.preview_meta_delete_mix_hint", "当前预览包含删除记录，它们只会用于定位，不会生成可执行条目。"))}</div>
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
  const smokePriorityActionCounts = evidence.smokeMatrixPriorityActionCounts && typeof evidence.smokeMatrixPriorityActionCounts === "object" ? evidence.smokeMatrixPriorityActionCounts : {};
  const smokePriorityActionSummary = Object.entries(smokePriorityActionCounts)
    .sort((left, right) => Number(right[1] || 0) - Number(left[1] || 0) || String(left[0]).localeCompare(String(right[0])))
    .map(([label, count]) => `${label} x${count}`)
    .join(" / ");
  syncAutoRecoverProtocolGroups();
  syncAutoRecoverProfiles();
  syncAutoRecoverBlockedActions();
  $("#evidence-summary").innerHTML = `
    <div class="metric"><span>${escapeHTML(t("status.total_tasks", "任务总数"))}</span><strong>${evidence.totalTasks}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.completed_tasks", "已完成任务"))}</span><strong>${evidence.completedTasks}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.blocked_tasks", "阻塞任务"))}</span><strong>${evidence.blockedTasks}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.execution_mode", "执行模式"))}</span><strong>${stringifyValue(evidence.executionMode, "-")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.scan_mode", "扫描模式"))}</span><strong>${stringifyValue(evidence.scanMode, "-")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.source_delete_policy", "源端删除策略"))}</span><strong>${renderSourceDeletePolicy(evidence.sourceDeletePolicy)}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.auto_recover_tasks", "自动补传任务"))}</span><strong>${stringifyValue(evidence.autoRecoverTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.runnable_now", "可直接放行"))}</span><strong>${stringifyValue(evidence.autoRecoverRunnableTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_cooldown", "等待冷却"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingCooldownTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_retry_window", "等待时间窗"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingRetryWindowTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_auth_refresh", "等待授权刷新"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingAuthRefreshTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_local_restore", "等待本地恢复"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingLocalRestoreTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_provider_session", "等待会话恢复"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingProviderSessionTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_manual_confirmation", "等待人工确认"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingManualTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_retry_limit", "等待重试上限"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingRetryLimitTasks, "0")}</strong></div>
    <div class="metric"><span>${escapeHTML(t("status.waiting_other", "其它等待"))}</span><strong>${stringifyValue(evidence.autoRecoverWaitingOtherTasks, "0")}</strong></div>
    <div class="metric"><span>成功结果</span><strong>${evidence.doneResultCount}</strong></div>
    <div class="metric"><span>跳过结果</span><strong>${evidence.skippedResultCount}</strong></div>
    <div class="metric"><span>待人工处理</span><strong>${evidence.pendingResultCount}</strong></div>
    <div class="metric"><span>源端删除记录</span><strong>${stringifyValue(evidence.sourceDeletionCount, "0")}</strong></div>
    <div class="metric"><span>失败结果</span><strong>${evidence.failedResultCount}</strong></div>
    <div class="metric"><span>风控命中</span><strong>${evidence.riskHitCount}</strong></div>
    <div class="metric"><span>自动补传周期</span><strong>${escapeHTML(stringifyValue(autoRetryPolicy.tick, "-"))}</strong></div>
    <div class="metric"><span>自动补传批量</span><strong>${stringifyValue(autoRetryPolicy.batchLimit, "-")}</strong></div>
    <div class="metric"><span>单 lane 限额</span><strong>${stringifyValue(autoRetryPolicy.limitPerLane, "-")}</strong></div>
    <div class="metric"><span>协议组数量</span><strong>${protocolCoverage.length}</strong></div>
    <div class="metric"><span>已取样协议组</span><strong>${protocolCoverageWithSamples}</strong></div>
    <div class="metric"><span>已验收协议组</span><strong>${acceptedSmokeGroups}</strong></div>
    <div class="metric"><span>进行中协议组</span><strong>${inProgressSmokeGroups}</strong></div>
    <div class="metric"><span>待验收协议组</span><strong>${pendingSmokeGroups}</strong></div>
    <div class="metric"><span>已有上传成功协议组</span><strong>${stringifyValue(evidence.uploadSuccessGroups, String(uploadSuccessSmokeGroups))}</strong></div>
    <div class="metric"><span>上传成功样本</span><strong>${stringifyValue(evidence.uploadSuccessSamples, "0")}</strong></div>
    <div class="metric"><span>缺上传样本协议组</span><strong>${escapeHTML(summarizePathList(evidence.smokeMatrixMissingUploadGroups, 4))}</strong></div>
    <div class="metric"><span>缺覆盖样本协议组</span><strong>${escapeHTML(summarizePathList(evidence.smokeMatrixMissingCoverageGroups, 4))}</strong></div>
    <div class="metric"><span>缺异常样本协议组</span><strong>${escapeHTML(summarizePathList(evidence.smokeMatrixMissingAnomalyGroups, 4))}</strong></div>
    <div class="metric"><span>缺代表性样本协议组</span><strong>${escapeHTML(summarizePathList(evidence.smokeMatrixMissingRepresentativeGroups, 4))}</strong></div>
    <div class="metric"><span>验收优先动作</span><strong>${escapeHTML(smokePriorityActionSummary || "-")}</strong></div>
    <div class="metric"><span>断点续传就绪度</span><strong>${escapeHTML(renderUploadCheckpointReadiness(evidence))}</strong></div>
    <div class="metric"><span>断点续传优先动作</span><strong>${escapeHTML(renderUploadCheckpointPriorityAction(evidence))}</strong></div>
    <div class="metric"><span>补传优先动作</span><strong>${escapeHTML(renderAutoRecoverPriorityAction(evidence))}</strong></div>
    <div class="metric"><span>补传就绪度</span><strong>${escapeHTML(renderAutoRecoverReadiness(evidence))}</strong></div>
    <div class="metric"><span>公平性就绪度</span><strong>${escapeHTML(renderAutoRecoverFairnessReadiness(evidence))}</strong></div>
    <div class="metric"><span>公平性缺口</span><strong>${escapeHTML(renderAutoRecoverFairnessMissing(evidence))}</strong></div>
    <div class="metric"><span>公平性优先动作</span><strong>${escapeHTML(renderAutoRecoverFairnessPriorityAction(evidence))}</strong></div>
    <div class="metric"><span>验收动作汇总</span><strong>${escapeHTML(acceptanceActionSummary || "-")}</strong></div>
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
  $("#auto-retry-policy-summary").textContent = `${t("status.auto_retry_policy_summary_prefix", "自动补传默认调度")}：${autoRetryPolicySummary}`;
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
          <th>网盘源</th>
          <th>协议组</th>
          <th>授权档案数</th>
          <th>任务数</th>
          <th>已完成</th>
          <th>覆盖情况</th>
          <th>执行模式</th>
          <th>扫描模式</th>
          <th>源端删除策略</th>
          <th>风控档位</th>
          <th>最近 Probe</th>
          <th>最近任务状态</th>
          <th>阻塞数</th>
          <th>自动补传数</th>
          <th>当前主动作</th>
          <th>快照摘要</th>
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
    return `<div class="directory-empty">${escapeHTML(t("status.blocked_empty", "当前没有需要人工处理的 blocked 聚合项。"))}</div>`;
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
            <span class="pill">${escapeHTML(t("status.blocked_metric_tasks", "任务"))} ${stringifyValue(item.taskCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.blocked_metric_providers", "网盘源"))} ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.blocked_metric_next", "下次重试"))} ${stringifyValue(item.nextRetryAt, "-")}</span>
          </div>
          <div class="muted">${escapeHTML(stringifyValue(item.advice, "-"))}</div>
          <div class="muted">next-step: ${escapeHTML(renderBlockedSummary(item.action, item.advice, item.nextRetryAt))}</div>
          <div class="actions compact">
            <button
              type="button"
              class="ghost"
              data-blocked-focus-action="${escapeHTML(stringifyValue(item.action))}"
            >${escapeHTML(t("status.blocked_focus_action", "只看这一类阻塞"))}</button>
            <button
              type="button"
              class="ghost"
              data-blocked-run-action="${escapeHTML(stringifyValue(item.action))}"
            >${escapeHTML(t("status.blocked_run_action", "执行此阻塞动作"))}</button>
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
  return parts.length ? ` / ${tf("status.auto_recover_profile_counts", { value: parts.join(" / ") }, `授权档案 ${parts.join(" / ")}`)}` : "";
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
    return t("status.no_auto_recover_result", "尚未执行后台补传预演或实际放行。");
  }
  const label = result.dryRun
    ? t("status.auto_recover_last_preview", "最近预演")
    : t("status.auto_recover_last_run", "最近执行");
  const recoveredLabel = result.dryRun
    ? t("status.auto_recover_last_recoverable", "可放行")
    : t("status.auto_recover_last_recovered", "已恢复");
  const extraSummary = `${renderAutoRecoverOutcomeCounts(result)}${renderAutoRecoverRetryClassCounts(result)}${renderAutoRecoverRecoverStateCounts(result)}${renderAutoRecoverBlockedActionCounts(result)}${renderAutoRecoverProtocolGroupCounts(result)}${renderAutoRecoverProviderCounts(result)}${renderAutoRecoverStrategyCounts(result)}${renderAutoRecoverProfileCounts(result)}${renderAutoRecoverLaneCounts(result)}${renderAutoRecoverSuggestedBudgets(result)}`;
  const earliestSummary = result.earliestNextRetryAt
    ? tf("status.auto_recover_earliest_retry", { value: result.earliestNextRetryAt }, ` / 最早重试 ${result.earliestNextRetryAt}`)
    : "";
  return tf(
    "status.auto_recover_last_result_summary",
    {
      label,
      matched: stringifyValue(result.matchedCount, "0"),
      recoveredLabel,
      recovered: stringifyValue(result.recoveredCount, "0"),
      limit: stringifyValue(result.skippedByLimit, "0"),
      modeBudget: stringifyValue(result.skippedByModeBudget, "0"),
      laneBudget: stringifyValue(result.skippedByLaneBudget, "0"),
      groupBudget: stringifyValue(result.skippedByProtocolGroupBudget, "0"),
      providerBudget: stringifyValue(result.skippedByProviderBudget, "0"),
      profileBudget: stringifyValue(result.skippedByProfileBudget, "0"),
      cooldownWait: stringifyValue(result.skippedByCooldownWait, "0"),
      retryWindowWait: stringifyValue(result.skippedByRetryWindowWait, "0"),
      blocked: stringifyValue(result.skippedByBlockedReason, "0"),
      extra: extraSummary,
      earliest: earliestSummary,
    },
    `${label}：匹配 ${stringifyValue(result.matchedCount, "0")} / ${recoveredLabel} ${stringifyValue(result.recoveredCount, "0")} / 重试耗尽 ${stringifyValue(result.skippedByLimit, "0")} / 模式预算 ${stringifyValue(result.skippedByModeBudget, "0")} / lane 预算 ${stringifyValue(result.skippedByLaneBudget, "0")} / 协议组预算 ${stringifyValue(result.skippedByProtocolGroupBudget, "0")} / 网盘源预算 ${stringifyValue(result.skippedByProviderBudget, "0")} / 授权档案预算 ${stringifyValue(result.skippedByProfileBudget, "0")} / 冷却等待 ${stringifyValue(result.skippedByCooldownWait, "0")} / 时间窗等待 ${stringifyValue(result.skippedByRetryWindowWait, "0")} / 阻塞 ${stringifyValue(result.skippedByBlockedReason, "0")}${extraSummary}${earliestSummary}`,
  );
}

  // Static test anchors: mode budget current 1 -> suggest 2 / lane budget current 1 -> suggest 2 / group budget current 1 -> suggest 2 / provider budget current 1 -> suggest 2 / profile budget current 1 -> suggest 2
function renderAutoRecoverDecisionBudgetHint(label, currentValue, suggestedValue) {
  const currentNumeric = Number(currentValue || 0);
  const suggestedNumeric = Number(suggestedValue || 0);
  if (!Number.isFinite(currentNumeric) || currentNumeric <= 0 || !Number.isFinite(suggestedNumeric) || suggestedNumeric <= 0) {
    return "";
  }
  return tf("status.auto_recover_budget_hint_item", { label, current: stringifyValue(currentNumeric, "0"), suggested: stringifyValue(suggestedNumeric, "0") }, `${label} 当前 ${stringifyValue(currentNumeric, "0")} -> 建议 ${stringifyValue(suggestedNumeric, "0")}`);
}

function renderAutoRecoverDecisionBudgetHints(item) {
  if (!item || typeof item !== "object") {
    return "";
  }
  const hints = [
    renderAutoRecoverDecisionBudgetHint(t("status.auto_recover_budget_hint_label_mode", "模式预算"), item.currentModeBudget, item.suggestedModeBudget),
    renderAutoRecoverDecisionBudgetHint(t("status.auto_recover_budget_hint_label_lane", "lane 预算"), item.currentLaneBudget, item.suggestedLaneBudget),
    renderAutoRecoverDecisionBudgetHint(t("status.auto_recover_budget_hint_label_group", "协议组预算"), item.currentProtocolGroupBudget, item.suggestedProtocolGroupBudget),
    renderAutoRecoverDecisionBudgetHint(t("status.auto_recover_budget_hint_label_provider", "网盘源预算"), item.currentProviderBudget, item.suggestedProviderBudget),
    renderAutoRecoverDecisionBudgetHint(t("status.auto_recover_budget_hint_label_profile", "授权档案预算"), item.currentProfileBudget, item.suggestedProfileBudget),
  ].filter(Boolean);
  if (!hints.length) {
    return "";
  }
  return `${t("status.auto_recover_budget_hint_prefix", "预算占用")}：${hints.join(" / ")}`;
}

function renderAutoRecoverLastResultDetail() {
  const result = state.autoRecoverLastResult;
  const decisions = Array.isArray(result?.decisions) ? result.decisions : [];
  if (!decisions.length) {
    return `<div class="directory-empty">${escapeHTML(t("status.auto_recover_decision_empty", "最近一次后台补传预演或执行暂无决策明细。"))}</div>`;
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
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_provider", { value: stringifyValue(item.providerKey, "-") }, `网盘源 ${stringifyValue(item.providerKey, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_profile", { value: stringifyValue(item.profileId, "-") }, `授权档案 ${stringifyValue(item.profileId, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_strategy", { value: stringifyValue(item.strategy, "-") }, `策略 ${stringifyValue(item.strategy, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_mode", { value: stringifyValue(item.mode, "-") }, `模式 ${stringifyValue(item.mode, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_state", { value: autoRecoverStateLabel(item.recoverState) }, `状态 ${autoRecoverStateLabel(item.recoverState)}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_mode_budget", { value: stringifyValue(item.suggestedModeBudget, "-") }, `模式预算 ${stringifyValue(item.suggestedModeBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_lane_budget", { value: stringifyValue(item.suggestedLaneBudget, "-") }, `lane 预算 ${stringifyValue(item.suggestedLaneBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_group_budget", { value: stringifyValue(item.suggestedProtocolGroupBudget, "-") }, `协议组预算 ${stringifyValue(item.suggestedProtocolGroupBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_provider_budget", { value: stringifyValue(item.suggestedProviderBudget, "-") }, `网盘源预算 ${stringifyValue(item.suggestedProviderBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_decision_profile_budget", { value: stringifyValue(item.suggestedProfileBudget, "-") }, `授权档案预算 ${stringifyValue(item.suggestedProfileBudget, "-")}`))}</span>
          </div>
          <div class="muted">${escapeHTML(tf("status.auto_recover_decision_path_protocol", { path: stringifyValue(item.path, "-"), protocolGroup: stringifyValue(item.protocolGroup, "-") }, `路径：${stringifyValue(item.path, "-")} / 协议组：${stringifyValue(item.protocolGroup, "-")}`))}</div>
          <div class="muted">${escapeHTML(tf("status.auto_recover_decision_retry_detail", { retryClass: stringifyValue(item.retryClass, "-"), blockedAction: stringifyValue(item.blockedAction, "-"), blockedReason: stringifyValue(item.blockedReason, "-"), nextRetryAt: stringifyValue(item.nextRetryAt, "-") }, `重试类型：${stringifyValue(item.retryClass, "-")} / 阻塞动作：${stringifyValue(item.blockedAction, "-")} / 阻塞原因：${stringifyValue(item.blockedReason, "-")} / 下次重试：${stringifyValue(item.nextRetryAt, "-")}`))}</div>
          <div class="muted">next-step: ${escapeHTML(renderBlockedSummary(item.blockedAction, item.message, item.nextRetryAt, autoRecoverDecisionAdvice(item)))}</div>
          <div class="muted">${escapeHTML(renderAutoRecoverDecisionBudgetHints(item) || t("status.auto_recover_budget_hint_empty", "预算占用：当前决策未返回可复用的预算占用信息。"))}</div>
          <div class="muted">${escapeHTML(t("status.auto_recover_waiting_advice", "等待态说明"))}：${escapeHTML(autoRecoverDecisionAdvice(item))}</div>
          <div class="muted">${escapeHTML(stringifyValue(item.message, "-"))}</div>
          <div class="tree-actions">
            <button type="button" class="link-button" data-auto-recover-decision-focus-state="${escapeHTML(stringifyValue(item.recoverState, ""))}">${escapeHTML(t("status.focus_current_state", "只看该状态"))}</button>
            <button type="button" class="link-button" data-auto-recover-decision-focus-lane-mode="${escapeHTML(stringifyValue(item.mode, ""))}" data-auto-recover-decision-focus-lane-strategy="${escapeHTML(stringifyValue(item.strategy, ""))}" data-auto-recover-decision-focus-lane-retry-class="${escapeHTML(stringifyValue(item.retryClass, ""))}" data-auto-recover-decision-focus-lane-blocked-action="${escapeHTML(stringifyValue(item.blockedAction, ""))}">${escapeHTML(t("status.focus_current_lane", "只看该通道"))}</button>
            <button type="button" class="link-button" data-auto-recover-decision-apply-budgets="1" data-auto-recover-decision-apply-mode-budget="${escapeHTML(stringifyValue(item.suggestedModeBudget, ""))}" data-auto-recover-decision-apply-lane-budget="${escapeHTML(stringifyValue(item.suggestedLaneBudget, ""))}" data-auto-recover-decision-apply-group-budget="${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, ""))}" data-auto-recover-decision-apply-provider-budget="${escapeHTML(stringifyValue(item.suggestedProviderBudget, ""))}" data-auto-recover-decision-apply-profile-budget="${escapeHTML(stringifyValue(item.suggestedProfileBudget, ""))}">${escapeHTML(t("status.apply_suggested_budgets", "采用建议预算"))}</button>
            <button type="button" class="link-button" data-auto-recover-decision-preview="1" data-auto-recover-decision-task-id="${escapeHTML(stringifyValue(item.taskId, ""))}" data-auto-recover-decision-mode="${escapeHTML(stringifyValue(item.mode, ""))}" data-auto-recover-decision-strategy="${escapeHTML(stringifyValue(item.strategy, ""))}" data-auto-recover-decision-protocol-group="${escapeHTML(stringifyValue(item.protocolGroup, ""))}" data-auto-recover-decision-provider="${escapeHTML(stringifyValue(item.providerKey, ""))}" data-auto-recover-decision-profile="${escapeHTML(stringifyValue(item.profileId, ""))}" data-auto-recover-decision-retry-class="${escapeHTML(stringifyValue(item.retryClass, ""))}" data-auto-recover-decision-blocked-action="${escapeHTML(stringifyValue(item.blockedAction, ""))}" data-auto-recover-decision-recover-state="${escapeHTML(stringifyValue(item.recoverState, ""))}" data-auto-recover-decision-mode-budget="${escapeHTML(stringifyValue(item.suggestedModeBudget, ""))}" data-auto-recover-decision-lane-budget="${escapeHTML(stringifyValue(item.suggestedLaneBudget, ""))}" data-auto-recover-decision-group-budget="${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, ""))}" data-auto-recover-decision-provider-budget="${escapeHTML(stringifyValue(item.suggestedProviderBudget, ""))}" data-auto-recover-decision-profile-budget="${escapeHTML(stringifyValue(item.suggestedProfileBudget, ""))}">${escapeHTML(t("status.preview_current_decision", "预演该决策"))}</button>
            <button type="button" class="link-button" data-auto-recover-decision-run="1" data-auto-recover-decision-task-id="${escapeHTML(stringifyValue(item.taskId, ""))}" data-auto-recover-decision-mode="${escapeHTML(stringifyValue(item.mode, ""))}" data-auto-recover-decision-strategy="${escapeHTML(stringifyValue(item.strategy, ""))}" data-auto-recover-decision-protocol-group="${escapeHTML(stringifyValue(item.protocolGroup, ""))}" data-auto-recover-decision-provider="${escapeHTML(stringifyValue(item.providerKey, ""))}" data-auto-recover-decision-profile="${escapeHTML(stringifyValue(item.profileId, ""))}" data-auto-recover-decision-retry-class="${escapeHTML(stringifyValue(item.retryClass, ""))}" data-auto-recover-decision-blocked-action="${escapeHTML(stringifyValue(item.blockedAction, ""))}" data-auto-recover-decision-recover-state="${escapeHTML(stringifyValue(item.recoverState, ""))}" data-auto-recover-decision-mode-budget="${escapeHTML(stringifyValue(item.suggestedModeBudget, ""))}" data-auto-recover-decision-lane-budget="${escapeHTML(stringifyValue(item.suggestedLaneBudget, ""))}" data-auto-recover-decision-group-budget="${escapeHTML(stringifyValue(item.suggestedProtocolGroupBudget, ""))}" data-auto-recover-decision-provider-budget="${escapeHTML(stringifyValue(item.suggestedProviderBudget, ""))}" data-auto-recover-decision-profile-budget="${escapeHTML(stringifyValue(item.suggestedProfileBudget, ""))}">${escapeHTML(t("status.run_current_decision", "执行该决策"))}</button>
            <button type="button" class="link-button" data-auto-recover-decision-open-task="${escapeHTML(stringifyValue(item.taskId, ""))}">${escapeHTML(t("status.open_sample_task", "打开样本任务"))}</button>
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
    showFlash(tf("tasks.flash_auto_recover_decision_previewed", { taskId: stringifyValue(payload.taskId, "-") }, `已按决策预演后台补传：${stringifyValue(payload.taskId, "-")}`));
    return result;
  }
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(tf("tasks.flash_auto_recover_decision_executed", { taskId: stringifyValue(payload.taskId, "-") }, `已按决策执行后台补传：${stringifyValue(payload.taskId, "-")}`));
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
      showFlash(tf("tasks.flash_auto_recover_decision_state_focused", { state: recoverState }, `已按决策状态 ${recoverState} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-decision-focus-lane-mode]").forEach((button) => {
    button.addEventListener("click", () => {
      const mode = button.dataset.autoRecoverDecisionFocusLaneMode || "";
      const strategy = button.dataset.autoRecoverDecisionFocusLaneStrategy || "";
      const retryClass = button.dataset.autoRecoverDecisionFocusLaneRetryClass || "";
      const blockedAction = button.dataset.autoRecoverDecisionFocusLaneBlockedAction || "";
      applyAutoRecoverFilters({ mode, strategy, retryClass, blockedAction });
      showFlash(tf("tasks.flash_auto_recover_decision_lane_focused", { lane: [mode, strategy, retryClass, blockedAction].filter(Boolean).join(" / ") }, `已按决策 lane 收敛后台补传候选：${[mode, strategy, retryClass, blockedAction].filter(Boolean).join(" / ")}`));
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
      showFlash(tf("tasks.flash_auto_recover_decision_budgets_applied", { mode: limitPerMode || "-", lane: limitPerLane || "-", group: limitPerProtocolGroup || "-", provider: limitPerProvider || "-", profile: limitPerProfile || "-" }, `已按决策采用建议预算：mode ${limitPerMode || "-"} / lane ${limitPerLane || "-"} / group ${limitPerProtocolGroup || "-"} / provider ${limitPerProvider || "-"} / profile ${limitPerProfile || "-"}`));
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
      return `<div class="directory-empty">${escapeHTML(t("status.auto_recover_filtered_empty", "当前筛选条件下没有命中的后台补传候选。"))}</div>`;
    }
    return `<div class="directory-empty">${escapeHTML(t("status.auto_recover_pool_empty", "当前没有进入后台补传候选池的任务。"))}</div>`;
  }
  const aggregate = summarizeAutoRecoverVisibleItems(visibleItems);
  return `
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>${escapeHTML(t("status.auto_recover_summary_title", "当前可见候选池摘要"))}</strong>
        <code>${escapeHTML(tf("status.auto_recover_summary_compact", { lanes: String(visibleItems.length), tasks: String(aggregate.tasks) }, `${visibleItems.length} 个 lane / ${aggregate.tasks} 个任务`))}</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_runnable", { value: stringifyValue(aggregate.runnable, "0") }, `可运行 ${stringifyValue(aggregate.runnable, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_cooldown", { value: stringifyValue(aggregate.waitingCooldown, "0") }, `等冷却 ${stringifyValue(aggregate.waitingCooldown, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_window", { value: stringifyValue(aggregate.waitingRetryWindow, "0") }, `等时间窗 ${stringifyValue(aggregate.waitingRetryWindow, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_auth", { value: stringifyValue(aggregate.waitingAuthRefresh, "0") }, `等刷新授权 ${stringifyValue(aggregate.waitingAuthRefresh, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_local", { value: stringifyValue(aggregate.waitingLocalRestore, "0") }, `等本地恢复 ${stringifyValue(aggregate.waitingLocalRestore, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_manual", { value: stringifyValue(aggregate.waitingManual, "0") }, `等人工确认 ${stringifyValue(aggregate.waitingManual, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_limit", { value: stringifyValue(aggregate.waitingRetryLimit, "0") }, `重试耗尽 ${stringifyValue(aggregate.waitingRetryLimit, "0")}`))}</span>
        <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_other", { value: stringifyValue(aggregate.waitingOther, "0") }, `其它等待 ${stringifyValue(aggregate.waitingOther, "0")}`))}</span>
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
            <span class="pill">${escapeHTML(t("status.blocked_metric_tasks", "任务"))} ${stringifyValue(item.taskCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.blocked_metric_providers", "网盘源"))} ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_counts_profiles", { value: stringifyValue(item.profileCount, "0") }, `授权档案 ${stringifyValue(item.profileCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_queue_count", { value: stringifyValue(item.queueItemCount, "0") }, `队列项 ${stringifyValue(item.queueItemCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_group_budget", { value: stringifyValue(item.suggestedProtocolGroupBudget, "-") }, `协议组预算 ${stringifyValue(item.suggestedProtocolGroupBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_provider_budget", { value: stringifyValue(item.suggestedProviderBudget, "-") }, `网盘源预算 ${stringifyValue(item.suggestedProviderBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_profile_budget", { value: stringifyValue(item.suggestedProfileBudget, "-") }, `授权档案预算 ${stringifyValue(item.suggestedProfileBudget, "-")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_ready_count", { value: stringifyValue(item.retryableNowCount, "0") }, `可立即重试 ${stringifyValue(item.retryableNowCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_cooldown_count", { value: stringifyValue(item.cooldownCount, "0") }, `冷却中 ${stringifyValue(item.cooldownCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_runnable_tasks", { value: stringifyValue(item.runnableTaskCount, "0") }, `可运行任务 ${stringifyValue(item.runnableTaskCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_cooldown", { value: stringifyValue(item.waitingCooldownTaskCount, "0") }, `等冷却 ${stringifyValue(item.waitingCooldownTaskCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_window", { value: stringifyValue(item.waitingRetryWindowTaskCount, "0") }, `等时间窗 ${stringifyValue(item.waitingRetryWindowTaskCount, "0")}`))}</span>
            <span class="pill">${escapeHTML(tf("status.auto_recover_candidate_wait_other", { value: stringifyValue(item.waitingOtherTaskCount, "0") }, `其它等待 ${stringifyValue(item.waitingOtherTaskCount, "0")}`))}</span>
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
            >只看该通道</button>`
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
            >预演该通道</button>
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
            >${escapeHTML(t("status.auto_recover_run_wait_auth", "只执行等刷新授权"))}</button>`
                : ""
            }
            ${
              Number(item?.waitingLocalRestoreTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_local_restore"
            >${escapeHTML(t("status.auto_recover_run_wait_local", "只执行等补源文件"))}</button>`
                : ""
            }
            ${
              Number(item?.waitingManualTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_manual_confirmation"
            >${escapeHTML(t("status.auto_recover_run_wait_manual", "只执行等人工确认"))}</button>`
                : ""
            }
            ${
              Number(item?.waitingRetryLimitTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_retry_limit"
            >${escapeHTML(t("status.auto_recover_run_wait_limit", "只执行重试耗尽"))}</button>`
                : ""
            }
            ${
              Number(item?.waitingOtherTaskCount || 0) > 0
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-state="waiting_other"
            >${escapeHTML(t("status.auto_recover_run_wait_other", "只执行其它等待"))}</button>`
                : ""
            }
            ${
              item.primaryRetryClass
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-retry-class="${escapeHTML(stringifyValue(item.primaryRetryClass, ""))}"
            >${escapeHTML(t("status.auto_recover_run_primary_retry", "执行主重试类型"))}</button>`
                : ""
            }
            ${
              item.primaryBlockedAction
                ? `<button
              type="button"
              class="ghost"
              data-auto-recover-run-primary-blocked-action="${escapeHTML(stringifyValue(item.primaryBlockedAction, ""))}"
            >${escapeHTML(t("status.auto_recover_run_primary_blocked", "执行主阻塞动作"))}</button>`
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
            >${escapeHTML(t("status.auto_recover_run_lane", "执行该通道"))}</button>`
                : ""
            }
            <button
              type="button"
              class="ghost"
              data-auto-recover-run-mode="${escapeHTML(stringifyValue(item.mode, ""))}"
            >${escapeHTML(t("status.auto_recover_run_mode", "执行该模式"))}</button>
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
    return `<div class="directory-empty">${escapeHTML(t("status.protocol_coverage_empty", "当前没有协议族覆盖数据。"))}</div>`;
  }
  return items
    .map(
      (item) => `
        <div class="directory-row tree-node">
          <div class="directory-row-header">
            <strong>${escapeHTML(stringifyValue(item.protocolGroup))}</strong>
            <code>${escapeHTML(
              item.hasRealSuccessSample
                ? t("status.protocol_sampled", "已取样")
                : t("status.protocol_pending_sample", "待取样"),
            )}</code>
          </div>
          <div class="directory-metrics">
            <span class="pill">${escapeHTML(t("common.provider", "网盘源"))} ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">${escapeHTML(t("common.tasks", "任务"))} ${stringifyValue(item.taskCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.completed_tasks", "已完成任务"))} ${stringifyValue(item.completedTaskCount, "0")}</span>
            <span class="pill">${escapeHTML(t("common.real_success", "真实成功"))} ${stringifyValue(item.realSuccessTaskCount, "0")}</span>
          </div>
          <div class="muted">${escapeHTML(t("status.protocol_related_providers", "涉及网盘源："))}${escapeHTML((item.providerKeys || []).join(", ") || "-")}</div>
          <div class="muted">${escapeHTML(t("status.protocol_sample_context", "样本上下文："))}${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleTaskId, "-"))}</div>
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
  // Static test anchors:
  // <th>状态</th>
  // <th>模式</th>
  // <th>执行模式</th>
  // <th>重试模式</th>
  // <th>重试范围</th>
  // <th>重试路径数</th>
  // <th>重试路径</th>
  // <th>源端删除策略</th>
  // <th>推荐结果</th>
  // <th>消息</th>
  // <th>风控命中</th>
  // <th>冲突处理</th>
  // <th>创建时间</th>
  if (!items.length) {
    return `<div class="provider-card">${escapeHTML(t("status.recent_results_empty", "暂无结果证据。"))}</div>`;
  }
  return `
    <table>
      <thead>
        <tr>
          <th>${escapeHTML(t("status.recent_results_col_status", "状态"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_mode", "模式"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_execution_mode", "执行模式"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_retry_mode", "重试模式"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_retry_scope", "重试范围"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_retry_path_count", "重试路径数"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_retry_paths", "重试路径"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_source_delete_policy", "源端删除策略"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_recommendation", "推荐结果"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_message", "消息"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_risk_hit", "风控命中"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_conflict", "冲突处理"))}</th>
          <th>${escapeHTML(t("status.recent_results_col_created_at", "创建时间"))}</th>
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
  // Static test anchors:
  // <th>网盘源</th>
  // <th>状态</th>
  // <th>授权档案</th>
  // <th>执行模式</th>
  // <th>扫描模式</th>
  // <th>重试模式</th>
  // <th>重试范围</th>
  // <th>重试路径数</th>
  // <th>重试路径</th>
  // <th>源端删除策略</th>
  // <th>风控命中</th>
  // <th>载荷</th>
  // <th>创建时间</th>
  if (!items.length) {
    return `<div class="provider-card">${escapeHTML(t("status.recent_probes_empty", "暂无探测证据。"))}</div>`;
  }
  return `
    <table>
      <thead>
        <tr>
          <th>${escapeHTML(t("status.recent_probes_col_provider", "网盘源"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_status", "状态"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_profile", "授权档案"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_execution_mode", "执行模式"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_scan_mode", "扫描模式"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_retry_mode", "重试模式"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_retry_scope", "重试范围"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_retry_path_count", "重试路径数"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_retry_paths", "重试路径"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_source_delete_policy", "源端删除策略"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_risk_hit", "风控命中"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_payload", "载荷"))}</th>
          <th>${escapeHTML(t("status.recent_probes_col_created_at", "创建时间"))}</th>
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
    missingBasicProviders: [],
    missingUploadProviders: [],
    missingAnomalyProviders: [],
    missingRepresentativeProviders: [],
  };
  for (const item of Array.isArray(items) ? items : []) {
    counts.total += 1;
    const providerKey = stringifyValue(item?.providerKey, "unknown");
    if (!item?.hasBasicSuccessSample) {
      counts.missingBasic += 1;
      counts.missingBasicProviders.push(providerKey);
    }
    if (!item?.hasUploadSuccessSample) {
      counts.missingUpload += 1;
      counts.missingUploadProviders.push(providerKey);
    }
    if (Array.isArray(item?.anomalyMissing) && item.anomalyMissing.length) {
      counts.missingAnomaly += 1;
      counts.missingAnomalyProviders.push(providerKey);
    }
    if (Array.isArray(item?.representativeMissing) && item.representativeMissing.length) {
      counts.missingRepresentative += 1;
      counts.missingRepresentativeProviders.push(providerKey);
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
  // Static test anchors:
  // ready（基础、上传、异常、代表性样本齐）
  // partial（已有样本，仍缺验收项）
  // pending（待补真实网盘样本）
  const readiness = String(value || "").trim().toLowerCase();
  if (readiness === "ready") {
    return t("status.provider_smoke_readiness_ready", "已就绪（基础、上传、异常、代表样本齐）");
  }
  if (readiness === "partial") {
    return t("status.provider_smoke_readiness_partial", "进行中（已有样本，仍缺验收项）");
  }
  return t("status.provider_smoke_readiness_pending", "待补齐（待补真实网盘样本）");
}

function renderEvidenceProviderSmokeProviders(report) {
  const items = Array.isArray(report?.providerSmokeProviders) ? report.providerSmokeProviders : [];
  if (!items.length) {
    return `
      <div class="insight-card">
        <strong>${escapeHTML(t("status.provider_smoke_acceptance_title", "网盘源级真实样本验收"))}</strong>
        <span>${escapeHTML(t("status.provider_smoke_acceptance_empty", "暂无 providerSmokeProviders 数据，请先刷新或保存新版验收报告。"))}</span>
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
  if (Array.isArray(summary.providerSmokeProviderMissingBasicProviders)) {
    counts.missingBasicProviders = summary.providerSmokeProviderMissingBasicProviders;
  }
  if (Array.isArray(summary.providerSmokeProviderMissingUploadProviders)) {
    counts.missingUploadProviders = summary.providerSmokeProviderMissingUploadProviders;
  }
  if (Array.isArray(summary.providerSmokeProviderMissingAnomalyProviders)) {
    counts.missingAnomalyProviders = summary.providerSmokeProviderMissingAnomalyProviders;
  }
  if (Array.isArray(summary.providerSmokeProviderMissingRepresentativeProviders)) {
    counts.missingRepresentativeProviders = summary.providerSmokeProviderMissingRepresentativeProviders;
  }
  const providerPriorityActionCounts =
    summary.providerSmokeProviderPriorityActionCounts && typeof summary.providerSmokeProviderPriorityActionCounts === "object"
      ? summary.providerSmokeProviderPriorityActionCounts
      : {};
  const providerPriorityActionSummary = Object.entries(providerPriorityActionCounts)
    .sort((left, right) => Number(right[1] || 0) - Number(left[1] || 0) || String(left[0]).localeCompare(String(right[0])))
    .slice(0, 8)
    .map(([label, count]) => `${label} x${count}`)
    .join(" / ");
  const focusItems = items
    .filter((item) => String(item?.readiness || "").toLowerCase() !== "ready")
    .slice(0, 6);
  return `
    <div class="insight-card">
      <strong>${escapeHTML(t("status.provider_smoke_acceptance_title", "网盘源级真实样本验收"))}</strong>
      <span>${escapeHTML(t("status.provider_smoke_ready", "网盘就绪"))} ${counts.ready} / ${counts.total}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>${escapeHTML(t("status.provider_smoke_gap_summary", "网盘源验收缺口速览"))}</strong>
        <code>providerSmokeProviders</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">${escapeHTML(t("status.risk_calibration_ready", "校准就绪").replace("校准", "")) || "就绪"} ${counts.ready}</span>
        <span class="pill">${escapeHTML(t("common.partial_ready", "部分就绪"))} ${counts.partial}</span>
        <span class="pill">${escapeHTML(t("common.pending_completion", "待补齐"))} ${counts.pending}</span>
        <span class="pill">${escapeHTML(t("status.provider_smoke_missing_basic_providers", "缺基础样本网盘源").replace("网盘源", ""))} ${counts.missingBasic}</span>
        <span class="pill">${escapeHTML(t("status.provider_smoke_missing_upload_providers", "缺上传样本网盘源").replace("网盘源", ""))} ${counts.missingUpload}</span>
        <span class="pill">${escapeHTML(t("status.provider_smoke_missing_anomaly_providers", "缺异常样本网盘源").replace("网盘源", ""))} ${counts.missingAnomaly}</span>
        <span class="pill">${escapeHTML(t("status.provider_smoke_missing_representative_providers", "缺代表样本网盘源").replace("网盘源", ""))} ${counts.missingRepresentative}</span>
      </div>
      <div class="muted">${escapeHTML(t("status.provider_smoke_missing_basic_providers", "缺基础样本网盘源:"))} ${escapeHTML(summarizePathList(counts.missingBasicProviders, 8))}</div>
      <div class="muted">${escapeHTML(t("status.provider_smoke_missing_upload_providers", "缺上传样本网盘源:"))} ${escapeHTML(summarizePathList(counts.missingUploadProviders, 8))}</div>
      <div class="muted">${escapeHTML(t("status.provider_smoke_missing_anomaly_providers", "缺异常样本网盘源:"))} ${escapeHTML(summarizePathList(counts.missingAnomalyProviders, 8))}</div>
      <div class="muted">${escapeHTML(t("status.provider_smoke_missing_representative_providers", "缺代表样本网盘源:"))} ${escapeHTML(summarizePathList(counts.missingRepresentativeProviders, 8))}</div>
      <div class="muted">${escapeHTML(t("status.provider_smoke_priority_counts", "网盘源优先动作统计:"))} ${escapeHTML(providerPriorityActionSummary || "-")}</div>
      ${
        focusItems.length
          ? focusItems
              .map(
                (item) => `
                  <div class="muted">
                    ${escapeHTML(stringifyValue(item.providerKey, "-"))}:
                    ${escapeHTML(renderProviderSmokeProviderReadinessLabel(item.readiness))}
                    / ${escapeHTML(t("status.provider_smoke_basic_ready", "基础"))} ${item.hasBasicSuccessSample ? escapeHTML(t("status.matrix_state_ready", "已就绪")) : escapeHTML(t("status.matrix_state_pending", "待补齐"))}
                    / ${escapeHTML(t("status.provider_smoke_upload_ready", "上传"))} ${item.hasUploadSuccessSample ? escapeHTML(t("status.matrix_state_ready", "已就绪")) : escapeHTML(t("status.matrix_state_pending", "待补齐"))}
                    / ${escapeHTML(t("status.provider_smoke_anomaly_ready", "异常"))} ${escapeHTML(stringifyValue(item.anomalyCompletedCount, "0"))}/${escapeHTML(stringifyValue(item.anomalyTargetCount, "0"))}
                    / ${escapeHTML(t("status.provider_smoke_representative_ready", "代表"))} ${escapeHTML(stringifyValue(item.representativeCompletedCount, "0"))}/${escapeHTML(stringifyValue(item.representativeTargetCount, "0"))}
                    / ${escapeHTML(t("status.provider_smoke_default_action", "网盘源优先动作"))} ${escapeHTML(stringifyValue(item.priorityAction, t("status.provider_smoke_default_action_complete", "已补齐")))}
                  </div>
                  <div class="muted">${escapeHTML(t("status.provider_smoke_basic_sample", "优先基础样本:"))} ${escapeHTML(stringifyValue(item.preferredSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredSamplePriority, "-"))}</div>
                  <div class="muted">${escapeHTML(t("status.provider_smoke_upload_sample", "优先上传样本:"))} ${escapeHTML(stringifyValue(item.preferredUploadSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredUploadPriority, "-"))}</div>
                  <div class="muted">${escapeHTML(t("status.provider_smoke_anomaly_sample", "优先异常样本:"))} ${escapeHTML(stringifyValue(item.preferredAnomalySampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredAnomalyPriority, "-"))}</div>
                  <div class="muted">${escapeHTML(t("status.provider_smoke_representative_sample", "优先代表样本:"))} ${escapeHTML(stringifyValue(item.preferredRepresentativeSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredRepresentativePriority, "-"))}</div>
                  ${Array.isArray(item.representativeMissing) && item.representativeMissing.length ? `<div class="muted">${escapeHTML(t("status.provider_smoke_representative_missing", "代表样本缺口:"))} ${escapeHTML(item.representativeMissing.join(", "))}</div>` : ""}
                  ${Array.isArray(item.representativeActions) && item.representativeActions.length ? `<div class="muted">${escapeHTML(t("status.provider_smoke_representative_actions", "代表样本动作建议:"))} ${escapeHTML(item.representativeActions.join("；"))}</div>` : ""}
                  ${item.representativeAdvice ? `<div class="muted">${escapeHTML(t("status.provider_smoke_representative_advice", "代表样本补齐建议:"))} ${escapeHTML(item.representativeAdvice)}</div>` : ""}
                `,
              )
              .join("")
          : `<div class="muted">${escapeHTML(t("status.provider_smoke_default_action", "网盘源优先动作"))}: ${escapeHTML(t("status.provider_smoke_default_action_complete", "已补齐"))}</div>`
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
      <strong>${escapeHTML(t("status.upload_checkpoint_acceptance_title", "上传断点续传默认恢复验收"))}</strong>
      <span>${escapeHTML(tf("status.upload_checkpoint_readiness", { value: readiness }, `断点续传就绪度：${readiness}`))}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>${escapeHTML(t("status.upload_checkpoint_resume_summary", "大文件/长链路恢复摘要"))}</strong>
        <code>uploadCheckpointResume</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">${escapeHTML(t("status.upload_checkpoint_tasks", "断点任务"))} ${checkpointCount}</span>
        <span class="pill">${escapeHTML(t("status.upload_checkpoint_resume_tasks", "自动续传"))} ${resumeCount}</span>
        <span class="pill">${escapeHTML(t("status.upload_checkpoint_readiness_compact", "就绪度"))} ${escapeHTML(readiness)}</span>
      </div>
      <div class="muted">${escapeHTML(tf("status.upload_checkpoint_priority_action", { value: priorityAction }, `优先恢复动作：${priorityAction}`))}</div>
      <div class="muted">${escapeHTML(tf("status.upload_checkpoint_sample_context", { provider: stringifyValue(summary.uploadCheckpointResumeSampleProvider, "-"), protocolGroup: stringifyValue(summary.uploadCheckpointResumeSampleProtocol, "-"), taskId: stringifyValue(summary.uploadCheckpointResumeSampleTaskId, "-"), profileId: stringifyValue(summary.uploadCheckpointResumeSampleProfileId, "-") }, `样本上下文：网盘源 ${stringifyValue(summary.uploadCheckpointResumeSampleProvider, "-")} / 协议组 ${stringifyValue(summary.uploadCheckpointResumeSampleProtocol, "-")} / 任务 ${stringifyValue(summary.uploadCheckpointResumeSampleTaskId, "-")} / 授权档案 ${stringifyValue(summary.uploadCheckpointResumeSampleProfileId, "-")}`))}</div>
      <div class="muted">${escapeHTML(tf("status.upload_checkpoint_resume_detail", { uploadId: stringifyValue(summary.uploadCheckpointResumeSampleUploadId, "-"), nextPart: stringifyValue(summary.uploadCheckpointResumeSampleNextPart, "0"), uploaded: stringifyValue(summary.uploadCheckpointResumeSampleUploaded, "0"), partCount: stringifyValue(summary.uploadCheckpointResumeSamplePartCount, "0") }, `续传详情：上传 ${stringifyValue(summary.uploadCheckpointResumeSampleUploadId, "-")} / 下一分片 ${stringifyValue(summary.uploadCheckpointResumeSampleNextPart, "0")} / 已上传 ${stringifyValue(summary.uploadCheckpointResumeSampleUploaded, "0")}/${stringifyValue(summary.uploadCheckpointResumeSamplePartCount, "0")}`))}</div>
      <div class="muted">${escapeHTML(tf("status.upload_checkpoint_sample_paths", { value: paths.length ? paths.join(" -> ") : "-" }, `样本路径：${paths.length ? paths.join(" -> ") : "-"}`))}</div>
      <div class="muted">${escapeHTML(tf("status.upload_checkpoint_priority_action_label", { value: priorityAction }, `恢复优先动作：${priorityAction}`))}</div>
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
    const readiness = String(item?.meta?.defaultRiskTemplate?.calibrationReadiness || "").trim().toLowerCase();
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
        <strong>${escapeHTML(t("status.risk_calibration_title", "Provider 默认风控校准"))}</strong>
        <span>${escapeHTML(t("status.risk_calibration_empty", "暂无网盘源默认模板校准数据，请先刷新网盘源列表。"))}</span>
      </div>
    `;
  }
  const counts = providerRiskCalibrationCounts(items);
  const summary = report?.summary && typeof report.summary === "object" ? report.summary : {};
  if (Object.prototype.hasOwnProperty.call(summary, "providerRiskCalibrationTotalCount")) {
    counts.total = Number(summary.providerRiskCalibrationTotalCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerRiskCalibrationReadyCount")) {
    counts.ready = Number(summary.providerRiskCalibrationReadyCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerRiskCalibrationPartialCount")) {
    counts.partial = Number(summary.providerRiskCalibrationPartialCount || 0);
  }
  if (Object.prototype.hasOwnProperty.call(summary, "providerRiskCalibrationPendingCount")) {
    counts.pending = Number(summary.providerRiskCalibrationPendingCount || 0);
  }
  const calibrationMissingFieldCounts =
    summary.providerRiskCalibrationMissingFieldCounts && typeof summary.providerRiskCalibrationMissingFieldCounts === "object"
      ? summary.providerRiskCalibrationMissingFieldCounts
      : {};
  const calibrationPriorityActionCounts =
    summary.providerRiskCalibrationPriorityActionCounts && typeof summary.providerRiskCalibrationPriorityActionCounts === "object"
      ? summary.providerRiskCalibrationPriorityActionCounts
      : {};
  const calibrationMissingFieldSummary = Object.entries(calibrationMissingFieldCounts)
    .sort((left, right) => Number(right[1] || 0) - Number(left[1] || 0) || String(left[0]).localeCompare(String(right[0])))
    .slice(0, 8)
    .map(([label, count]) => `${label} x${count}`)
    .join(" / ");
  const calibrationPriorityActionSummary = Object.entries(calibrationPriorityActionCounts)
    .sort((left, right) => Number(right[1] || 0) - Number(left[1] || 0) || String(left[0]).localeCompare(String(right[0])))
    .slice(0, 8)
    .map(([label, count]) => `${label} x${count}`)
    .join(" / ");
  const focusItems = items
    .filter((item) => String(item?.meta?.defaultRiskTemplate?.calibrationReadiness || "").toLowerCase() !== "ready")
    .slice(0, 6);
  return `
    <div class="insight-card">
      <strong>${escapeHTML(t("status.risk_calibration_title", "Provider 默认风控校准"))}</strong>
      <span>${escapeHTML(t("status.risk_calibration_ready", "校准就绪"))} ${counts.ready} / ${counts.total}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>${escapeHTML(t("status.risk_calibration_gap_summary", "默认模板缺口速览"))}</strong>
        <code>defaultRiskTemplate</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">${escapeHTML(t("status.risk_calibration_ready", "校准就绪").replace("校准", "")) || "就绪"} ${counts.ready}</span>
        <span class="pill">${escapeHTML(t("common.partial_ready", "部分就绪"))} ${counts.partial}</span>
        <span class="pill">${escapeHTML(t("common.pending_completion", "待补齐"))} ${counts.pending}</span>
      </div>
      <div class="muted">${escapeHTML(t("status.risk_calibration_missing_counts", "校准缺失字段统计:"))} ${escapeHTML(calibrationMissingFieldSummary || "-")}</div>
      <div class="muted">${escapeHTML(t("status.risk_calibration_priority_counts", "校准优先动作统计:"))} ${escapeHTML(calibrationPriorityActionSummary || "-")}</div>
      ${
        focusItems.length
          ? focusItems
              .map((item) => {
                const template = item?.meta?.defaultRiskTemplate || {};
                return `
                  <div class="muted">
                    ${escapeHTML(stringifyValue(item?.meta?.key, "-"))}:
                    ${escapeHTML(t("status.risk_calibration_readiness", "校准就绪度"))} ${escapeHTML(stringifyValue(template.calibrationReadiness, "pending"))}
                    / ${escapeHTML(t("common.recommended_risk", "推荐风控"))} ${escapeHTML(stringifyValue(template.recommendedMode, "-"))}
                    / ${escapeHTML(t("status.risk_calibration_priority_action", "优先校准"))} ${escapeHTML(stringifyValue(template.calibrationPriorityAction, "-"))}
                  </div>
                  <div class="muted">
                    ${escapeHTML(t("status.risk_calibration_window_source", "时间窗来源"))} ${escapeHTML(renderAutoRetryWindowSource(template.autoRetryWindowSource))}
                    / ${escapeHTML(t("status.risk_calibration_window_advice", "时间窗建议"))} ${escapeHTML(stringifyValue(template.autoRetryWindowAdvice, "-"))}
                  </div>
                  <div class="muted">
                    ${escapeHTML(t("status.risk_calibration_missing", "校准缺失"))} ${escapeHTML((template.calibrationMissing || []).join(", ") || "-")}
                    / ${escapeHTML(t("status.risk_calibration_coverage", "校准覆盖"))} ${escapeHTML(stringifyValue(template.calibrationCoverage, "-"))}
                    / ${escapeHTML(t("status.risk_calibration_covered_fields", "已覆盖字段"))} ${escapeHTML(stringifyValue(template.calibrationCoveredCount, "0"))}/${escapeHTML(stringifyValue(template.calibrationTargetCount, "0"))}
                  </div>
                  <div class="muted">
                    ${escapeHTML(t("status.risk_calibration_covered_fields", "已覆盖字段"))} ${escapeHTML((template.calibrationCoveredFields || []).join(", ") || "-")}
                  </div>
                  <div class="muted">
                    ${escapeHTML(t("status.risk_calibration_sample_advice", "校准样本建议"))} ${escapeHTML(stringifyValue(template.calibrationSampleAdvice, "-"))}
                  </div>
                `;
              })
              .join("")
          : `<div class="muted">${escapeHTML(t("status.risk_calibration_default_action", "优先校准动作:"))} complete</div>`
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
  const autoRecoverPriorityActionCounts =
    summary.autoRecoverPriorityActionCounts && typeof summary.autoRecoverPriorityActionCounts === "object"
      ? summary.autoRecoverPriorityActionCounts
      : {};
  const autoRecoverPriorityActionSummary = Object.entries(autoRecoverPriorityActionCounts)
    .sort((left, right) => Number(right[1] || 0) - Number(left[1] || 0) || String(left[0]).localeCompare(String(right[0])))
    .slice(0, 8)
    .map(([label, count]) => `${label} x${count}`)
    .join(" / ");
  const focusLanes = fairnessPool.slice(0, 4);
  return `
    <div class="insight-card">
      <strong>${escapeHTML(t("status.auto_recover_acceptance", "自动补传验收"))}</strong>
      <span>恢复就绪 ${escapeHTML(recoveryReadiness)} / 公平性就绪 ${escapeHTML(fairnessReadiness)}</span>
    </div>
    <div class="directory-row tree-node">
      <div class="directory-row-header">
        <strong>${escapeHTML(t("status.auto_recover_fairness_summary", "自动补传恢复与公平性摘要"))}</strong>
        <code>autoRecoverPool</code>
      </div>
      <div class="directory-metrics">
        <span class="pill">任务 ${escapeHTML(stringifyValue(summary.autoRecoverTasks, "0"))}</span>
        <span class="pill">可运行 ${escapeHTML(stringifyValue(summary.autoRecoverRunnableTasks, "0"))}</span>
        <span class="pill">恢复 ${escapeHTML(recoveryReadiness)}</span>
        <span class="pill">公平性 ${escapeHTML(fairnessReadiness)}</span>
      </div>
      <div class="muted">${escapeHTML(t("status.recovery_priority_action", "恢复优先动作"))}: ${escapeHTML(recoveryPriorityAction)}</div>
      <div class="muted">${escapeHTML(t("status.recovery_priority_action_counts", "恢复优先动作统计"))}: ${escapeHTML(autoRecoverPriorityActionSummary || "-")}</div>
      <div class="muted">${escapeHTML(t("status.fairness_gap", "公平性缺口"))}: ${escapeHTML(renderAutoRecoverFairnessMissing(summary))}</div>
      <div class="muted">${escapeHTML(t("status.fairness_priority_action", "公平性优先动作"))}: ${escapeHTML(fairnessPriorityAction)}</div>
      <div class="muted">${escapeHTML(t("status.waiting_reason_summary", "等待原因"))}: 冷却 ${escapeHTML(stringifyValue(summary.autoRecoverWaitingCooldownTasks, "0"))} / 时间窗 ${escapeHTML(stringifyValue(summary.autoRecoverWaitingRetryWindowTasks, "0"))} / 授权 ${escapeHTML(stringifyValue(summary.autoRecoverWaitingAuthRefreshTasks, "0"))} / 本地文件 ${escapeHTML(stringifyValue(summary.autoRecoverWaitingLocalRestoreTasks, "0"))} / 人工处理 ${escapeHTML(stringifyValue(summary.autoRecoverWaitingManualTasks, "0"))}</div>
      ${
        focusLanes.length
          ? focusLanes
              .map(
                (item) => `
                  <div class="muted">
                    ${escapeHTML(t("status.lane_summary_prefix", "通道"))} ${escapeHTML(stringifyValue(item.mode, "-"))}:
                    网盘源 ${escapeHTML(stringifyValue(item.sampleProvider, "-"))}
                    / 协议组 ${escapeHTML(stringifyValue(item.sampleProtocolGroup, "-"))}
                    / 授权档案 ${escapeHTML(stringifyValue(item.sampleProfileId, "-"))}
                    / 网盘源数 ${escapeHTML(stringifyValue(item.providerCount, "0"))}
                    / 授权档案数 ${escapeHTML(stringifyValue(item.profileCount, "0"))}
                  </div>
                `,
              )
              .join("")
          : `<div class="muted">${escapeHTML(t("status.no_auto_recover_pool_samples", "当前没有自动补传候选池样本。"))}</div>`
      }
    </div>
  `;
}

function renderEvidenceReport(report) {
  if (!report || typeof report !== "object") {
    return `<div class="directory-empty">${escapeHTML(t("status.acceptance_report_empty", "暂无验收报告，请先刷新或保存一份报告。"))}</div>`;
  }
  return `
    <div class="insight-card">
      <strong>${escapeHTML(t("status.report_title_label", "报告标题"))}</strong>
      <span>${escapeHTML(stringifyValue(report.title, "-"))}</span>
    </div>
    <div class="insight-card">
      <strong>${escapeHTML(t("status.report_generated_at", "生成时间"))}</strong>
      <span>${escapeHTML(stringifyValue(report.generatedAt, "-"))}</span>
    </div>
    ${report.note ? `
      <div class="insight-card">
        <strong>${escapeHTML(t("status.report_note_label", "报告备注"))}</strong>
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
      showFlash(tf("tasks.flash_auto_recover_mode_focused", { mode }, `已按 ${mode} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-protocol-group]").forEach((button) => {
    button.addEventListener("click", () => {
      const protocolGroup = button.dataset.autoRecoverFocusProtocolGroup || "";
      applyAutoRecoverFilters({ protocolGroup });
      showFlash(tf("tasks.flash_auto_recover_protocol_group_focused", { protocolGroup }, `已按协议族 ${protocolGroup} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-state]").forEach((button) => {
    button.addEventListener("click", () => {
      const recoverState = button.dataset.autoRecoverFocusState || "";
      applyAutoRecoverFilters({ recoverState });
      showFlash(tf("tasks.flash_auto_recover_state_focused", { state: recoverState }, `已按执行状态 ${recoverState} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-retry-class]").forEach((button) => {
    button.addEventListener("click", () => {
      const retryClass = button.dataset.autoRecoverFocusRetryClass || "";
      applyAutoRecoverFilters({ retryClass });
      showFlash(tf("tasks.flash_auto_recover_retry_class_focused", { retryClass }, `已按主失败类型 ${retryClass} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-primary-blocked-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const blockedAction = button.dataset.autoRecoverFocusPrimaryBlockedAction || "";
      applyAutoRecoverFilters({ blockedAction });
      showFlash(tf("tasks.flash_auto_recover_primary_blocked_action_focused", { blockedAction }, `已按主阻塞动作 ${blockedAction} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-blocked-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const blockedAction = button.dataset.autoRecoverFocusBlockedAction || "";
      applyAutoRecoverFilters({ blockedAction });
      showFlash(tf("tasks.flash_auto_recover_blocked_action_focused", { blockedAction }, `已按 ${blockedAction} 收敛后台补传候选`));
    });
  });
  wrap.querySelectorAll("[data-auto-recover-focus-lane-mode]").forEach((button) => {
    button.addEventListener("click", () => {
      const mode = button.dataset.autoRecoverFocusLaneMode || "";
      const strategy = button.dataset.autoRecoverFocusLaneStrategy || "";
      const retryClass = button.dataset.autoRecoverFocusLaneRetryClass || "";
      const blockedAction = button.dataset.autoRecoverFocusLaneBlockedAction || "";
      applyAutoRecoverFilters({ mode, strategy, retryClass, blockedAction });
      showFlash(tf("tasks.flash_auto_recover_lane_focused", { lane: [mode, strategy, retryClass, blockedAction].filter(Boolean).join(" / ") }, `已按 lane 收敛后台补传候选：${[mode, strategy, retryClass, blockedAction].filter(Boolean).join(" / ")}`));
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
      showFlash(tf("tasks.flash_auto_recover_budgets_applied", { mode: limitPerMode || "-", lane: limitPerLane || "-", group: limitPerProtocolGroup || "-", provider: limitPerProvider || "-", profile: limitPerProfile || "-" }, `已采用建议预算：mode ${limitPerMode || "-"} / lane ${limitPerLane || "-"} / group ${limitPerProtocolGroup || "-"} / provider ${limitPerProvider || "-"} / profile ${limitPerProfile || "-"}`));
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
      tf("status.auto_recover_preview_completed", { matched: stringifyValue(result.matchedCount, "0"), recovered: stringifyValue(result.recoveredCount, "0"), providerBudget: stringifyValue(result.skippedByProviderBudget, "0"), profileBudget: stringifyValue(result.skippedByProfileBudget, "0") }, `后台补传预演完成：匹配 ${stringifyValue(result.matchedCount, "0")} / 可放行 ${stringifyValue(result.recoveredCount, "0")} / 网盘源预算 ${stringifyValue(result.skippedByProviderBudget, "0")} / 授权档案预算 ${stringifyValue(result.skippedByProfileBudget, "0")}`),
    );
    return result;
  }
  await Promise.all([loadTasks(), loadStatus()]);
  showFlash(
    tf("status.auto_recover_run_completed", { matched: stringifyValue(result.matchedCount, "0"), recovered: stringifyValue(result.recoveredCount, "0"), limit: stringifyValue(result.skippedByLimit, "0"), modeBudget: stringifyValue(result.skippedByModeBudget, "0"), laneBudget: stringifyValue(result.skippedByLaneBudget, "0"), groupBudget: stringifyValue(result.skippedByProtocolGroupBudget, "0"), providerBudget: stringifyValue(result.skippedByProviderBudget, "0"), profileBudget: stringifyValue(result.skippedByProfileBudget, "0"), cooldownWait: stringifyValue(result.skippedByCooldownWait, "0"), retryWindowWait: stringifyValue(result.skippedByRetryWindowWait, "0"), blocked: stringifyValue(result.skippedByBlockedReason, "0") }, `后台补传已执行：匹配 ${stringifyValue(result.matchedCount, "0")} / 已恢复 ${stringifyValue(result.recoveredCount, "0")} / 重试耗尽 ${stringifyValue(result.skippedByLimit, "0")} / 模式预算 ${stringifyValue(result.skippedByModeBudget, "0")} / lane 预算 ${stringifyValue(result.skippedByLaneBudget, "0")} / 协议组预算 ${stringifyValue(result.skippedByProtocolGroupBudget, "0")} / 网盘源预算 ${stringifyValue(result.skippedByProviderBudget, "0")} / 授权档案预算 ${stringifyValue(result.skippedByProfileBudget, "0")} / 冷却等待 ${stringifyValue(result.skippedByCooldownWait, "0")} / 时间窗等待 ${stringifyValue(result.skippedByRetryWindowWait, "0")} / 阻塞 ${stringifyValue(result.skippedByBlockedReason, "0")}`),
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
    throw new Error(scope === "task" ? t("wizard.flash_task_required", "请先选择任务") : t("wizard.flash_prefill_status_task_missing", "当前状态样本没有可用任务"));
  }
  if (!normalizedPath) {
    throw new Error(t("status.blocked_missing_path", "缺少可恢复路径"));
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
    tf("status.auto_recover_subtree_completed", { path: normalizedPath, scope: selectionScopeLabel(autoRecoverScopeFromPanel(panel)), matched: stringifyValue(result.matchedCount, "0"), recovered: stringifyValue(result.recoveredCount, "0"), modeBudget: stringifyValue(result.skippedByModeBudget, "0"), laneBudget: stringifyValue(result.skippedByLaneBudget, "0"), providerBudget: stringifyValue(result.skippedByProviderBudget, "0") }, `后台补传子树已执行：${normalizedPath} / ${selectionScopeLabel(autoRecoverScopeFromPanel(panel))} / 匹配 ${stringifyValue(result.matchedCount, "0")} / 已恢复 ${stringifyValue(result.recoveredCount, "0")} / 模式预算 ${stringifyValue(result.skippedByModeBudget, "0")} / lane 预算 ${stringifyValue(result.skippedByLaneBudget, "0")} / 网盘源预算 ${stringifyValue(result.skippedByProviderBudget, "0")}`),
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
  showFlash(t("status.flash_auto_recover_filters_cleared", "后台补传筛选已清空"));
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
      showFlash(t("tasks.flash_task_tree_focused_by_scan", "已按扫描轨迹定位任务目录树"));
      return;
    }
    state.treeFilters.taskDirectory.query = normalized;
    setFilterControlValue("#task-directory-filter-query", normalized);
    updateTaskTreePanels(currentSelectedTaskDetail());
    showFlash(t("tasks.flash_task_tree_focused_by_root", "已按选定根目录定位任务目录树"));
    return;
  }
  if (kind === "scan") {
    state.treeFilters.statusDirectory.query = normalized;
    setFilterControlValue("#status-directory-filter-query", normalized);
    updateStatusTreePanels(recentRuntimePayload());
    showFlash(t("tasks.flash_status_tree_focused_by_scan", "已按最近扫描轨迹定位状态目录树"));
    return;
  }
  state.treeFilters.statusDirectory.query = normalized;
  setFilterControlValue("#status-directory-filter-query", normalized);
  updateStatusTreePanels(recentRuntimePayload());
  showFlash(t("tasks.flash_status_tree_focused_by_root", "已按选定根目录定位状态目录树"));
}


  // Static test anchors: template: / completeness / reuse: / priority / regression entry: / representative: / auto recover focus: / operations: / createdAt:
function renderProviderSmokeRecords(items) {
  const result = filterProviderSmokeRecords(items, state.providerSmokeRecordFilters);
  if (!result.totalItems) {
    return `<div class="directory-empty">${escapeHTML(t("status.sample_record_empty", "暂无真实网盘样本记录。"))}</div>`;
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
          <div class="muted">${escapeHTML(tf("status.provider_smoke_record_template", { template: stringifyValue(item.templateVersion, "-"), type: stringifyValue(item.sampleType, "-"), completeness: stringifyValue(item.evidenceCompleteness, "-") }, `模板：${stringifyValue(item.templateVersion, "-")} / 类型 ${stringifyValue(item.sampleType, "-")} / 完整度 ${stringifyValue(item.evidenceCompleteness, "-")}`))}</div>
          <div class="muted">${escapeHTML(tf("status.provider_smoke_record_reuse", { advice: stringifyValue(item.reuseAdvice, "-"), priority: stringifyValue(item.reusePriority, "-") }, `复用建议：${stringifyValue(item.reuseAdvice, "-")} / 优先级 ${stringifyValue(item.reusePriority, "-")}`))}</div>
          <div class="muted">${escapeHTML(tf("status.provider_smoke_record_regression", { value: stringifyValue(item.regressionEntry, "-") }, `回归入口：${stringifyValue(item.regressionEntry, "-")}`))}</div>
          <div class="muted">${escapeHTML(tf("status.provider_smoke_record_representative", { labels: (item.representativeLabels || []).join(" / ") || "-", focus: stringifyValue(item.autoRecoverFocus, "-") }, `代表标签：${(item.representativeLabels || []).join(" / ") || "-"} / 自动补传关注点：${stringifyValue(item.autoRecoverFocus, "-")}`))}</div>
          <div class="muted">${escapeHTML(tf("status.provider_smoke_record_operations", { value: (item.operations || []).join(", ") || "-" }, `操作：${(item.operations || []).join(", ") || "-"}`))}</div>
          <div class="muted">${escapeHTML(tf("status.provider_smoke_record_created_at", { value: stringifyValue(item.createdAt, "-") }, `创建时间：${stringifyValue(item.createdAt, "-")}`))}</div>
          <div class="actions compact">
            <button type="button" class="ghost" data-provider-smoke-view="${escapeHTML(item.id || "")}">${escapeHTML(t("status.smoke_view_markdown", "查看样本文档"))}</button>
            <button type="button" class="ghost" data-provider-smoke-download="${escapeHTML(item.id || "")}">${escapeHTML(t("status.smoke_download_markdown", "下载样本文档"))}</button>
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
  const sampleType = String(filters.sampleType || "").trim().toLowerCase();
  const result = String(filters.result || "").trim().toLowerCase();
  const filterActive = Boolean(query || protocolGroup || sampleType || result);
  const visible = records.filter((item) => {
    const matchesQuery = includesFilterText(
      [item.title, item.providerKey, item.note, item.sampleType, item.evidenceCompleteness, item.reuseAdvice, item.reusePriority, item.regressionEntry, Array.isArray(item.representativeLabels) ? item.representativeLabels.join("/") : "", item.autoRecoverFocus, Array.isArray(item.operations) ? item.operations.join(",") : ""],
      query,
    );
    const matchesGroup = includesFilterText([item.protocolGroup], protocolGroup);
    const matchesSampleType = includesFilterText([item.sampleType, item.reusePriority, item.autoRecoverFocus], sampleType);
    const matchesResult = includesFilterText([item.result], result);
    return matchesQuery && matchesGroup && matchesSampleType && matchesResult;
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
    return t("status.smoke_record_empty", "当前没有样本记录。");
  }
  if (!result.filterActive) {
    return tf("status.smoke_record_showing_all", { visible: result.visibleItems }, `显示全部 ${result.visibleItems} 条样本记录。`);
  }
  return tf("status.smoke_record_showing_filtered", { visible: result.visibleItems, total: result.totalItems }, `当前显示 ${result.visibleItems} / ${result.totalItems} 条样本记录。`);
}

function renderProviderSmokeSummary(items) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="directory-empty">${escapeHTML(t("status.sample_matrix_empty", "暂无真实样本矩阵。"))}</div>`;
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
            <span class="pill">${escapeHTML(t("status.smoke_summary_smokes", "样本数"))} ${stringifyValue(item.smokeCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.smoke_summary_success", "成功"))} ${stringifyValue(item.successCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.smoke_summary_upload", "上传成功"))} ${stringifyValue(item.uploadSuccessCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.smoke_summary_failure", "失败"))} ${stringifyValue(item.failureCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.smoke_summary_providers", "网盘源"))} ${stringifyValue(item.providerCount, "0")}</span>
            <span class="pill">${escapeHTML(item.hasRealSuccessSample ? t("status.protocol_sampled", "已取样") : t("status.protocol_pending_sample", "待取样"))}</span>
          </div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_providers_label", "涉及网盘源："))}${escapeHTML((item.providerKeys || []).join(", ") || "-")}</div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_sample_label", "样本："))}${escapeHTML(stringifyValue(item.sampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleCategory, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_preferred_sample", "优先基础样本："))}${escapeHTML(stringifyValue(item.preferredSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredSampleProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredSamplePriority, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_preferred_upload", "优先上传样本："))}${escapeHTML(stringifyValue(item.preferredUploadSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredUploadProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredUploadPriority, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_preferred_anomaly", "优先异常样本："))}${escapeHTML(stringifyValue(item.preferredAnomalySampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredAnomalyProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredAnomalyPriority, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_preferred_representative", "优先代表样本："))}${escapeHTML(stringifyValue(item.preferredRepresentativeSampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.preferredRepresentativeProvider, "-"))} / ${escapeHTML(stringifyValue(item.preferredRepresentativePriority, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.smoke_summary_latest_smoke_at", "最近样本时间："))}<code>${escapeHTML(stringifyValue(item.latestSmokeAt, "-"))}</code></div>
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
      return t("status.matrix_filter_accepted", "已验收");
    case "in_progress":
      return t("status.matrix_filter_in_progress", "进行中");
    case "pending":
      return t("status.matrix_filter_pending", "待补齐");
    default:
      return t("status.matrix_filter_all", "全部");
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

function renderProviderSmokeMatrixState(stateValue) {
  switch (String(stateValue || "").trim().toLowerCase()) {
    case "ready":
      return t("status.matrix_state_ready", "已就绪");
    case "partial":
      return t("status.matrix_state_partial", "进行中");
    case "pending":
      return t("status.matrix_state_pending", "待补齐");
    case "accepted":
      return t("status.matrix_state_accepted", "已验收");
    default:
      return stringifyValue(stateValue, "-");
  }
}

function renderProviderSmokeMatrixControls(items) {
  const counts = providerSmokeMatrixCounts(items);
  const filters = [
    { key: "all", label: `${t("status.matrix_filter_all", "全部")} ${counts.total}` },
    { key: "accepted", label: `${t("status.matrix_filter_accepted", "已验收")} ${counts.accepted}` },
    { key: "in_progress", label: `${t("status.matrix_filter_in_progress", "进行中")} ${counts.inProgress}` },
    { key: "pending", label: `${t("status.matrix_filter_pending", "待补齐")} ${counts.pending}` },
  ];
  return `
    <div class="provider-smoke-matrix-toolbar">
      <div class="directory-row-header">
        <strong>${escapeHTML(t("status.acceptance_matrix_view", "验收矩阵视图"))}</strong>
        <code>${escapeHTML(providerSmokeMatrixFilterLabel(state.providerSmokeMatrixFilter))}</code>
      </div>
      <div class="muted">${escapeHTML(t("status.acceptance_matrix_hint", "可按验收状态快速筛选，也能直接跳到对应样本记录或任务样本，方便继续补齐真实联调样本。"))}</div>
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
    `${t("status.matrix_checklist_upload", "上传")} ${renderProviderSmokeMatrixState(item?.hasUploadSuccessSample ? "ready" : "pending")}`,
    `${t("status.matrix_checklist_coverage", "覆盖")} ${stringifyValue(item?.coverageRealSuccessTaskCount, "0")}/${stringifyValue(item?.coverageTaskCount, "0")}`,
    `${t("status.matrix_checklist_anomaly", "异常")} ${stringifyValue(item?.anomalyCompletedCount, "0")}/${stringifyValue(item?.anomalyTargetCount, "0")}`,
    `${t("status.matrix_checklist_representative", "代表")} ${stringifyValue(item?.representativeCompletedCount, "0")}/${stringifyValue(item?.representativeTargetCount, "0")}`,
  ].join(" / ");
}

function renderProviderSmokeGaps(item) {
  const gaps = [];
  if (!item?.hasUploadSuccessSample) {
    gaps.push(t("status.matrix_gap_upload", "上传"));
  }
  const anomalyMissing = Array.isArray(item?.anomalyMissing) ? item.anomalyMissing.filter(Boolean) : [];
  if (anomalyMissing.length) {
    gaps.push(tf("status.matrix_gap_anomaly", { value: anomalyMissing.join(",") }, `异常(${anomalyMissing.join(",")})`));
  }
  const representativeMissing = Array.isArray(item?.representativeMissing) ? item.representativeMissing.filter(Boolean) : [];
  if (representativeMissing.length) {
    gaps.push(tf("status.matrix_gap_representative", { value: representativeMissing.join(",") }, `代表(${representativeMissing.join(",")})`));
  }
  return gaps.length ? gaps.join(" / ") : t("status.matrix_gap_complete", "已补齐");
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
    actions.push(t("status.smoke_action_fill_upload_success", "补 1 条真实上传成功样本"));
  }
  if (Number(item?.coverageRealSuccessTaskCount || 0) < Number(item?.coverageTaskCount || 0)) {
    actions.push(t("status.smoke_action_fill_task_coverage", "补 1 条真实任务覆盖样本"));
  }
  if (Array.isArray(item?.anomalyActions) && item.anomalyActions.length) {
    actions.push(item.anomalyActions[0]);
  }
  if (Array.isArray(item?.representativeActions) && item.representativeActions.length) {
    actions.push(item.representativeActions[0]);
  }
  if (!actions.length) {
    return t("status.matrix_state_complete", "已完成");
  }
  return Array.from(new Set(actions.filter(Boolean))).join("；") || t("status.matrix_state_complete", "已完成");
}

function renderProviderSmokePriorityAction(item) {
  if (!item?.hasUploadSuccessSample) {
    return t("status.smoke_action_fill_upload_success", "补 1 条真实上传成功样本");
  }
  if (Number(item?.coverageRealSuccessTaskCount || 0) < Number(item?.coverageTaskCount || 0)) {
    return t("status.smoke_action_fill_task_coverage", "补 1 条真实任务覆盖样本");
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
  // Static test anchors:
  // checklist:
  // next action:
  // priority action:
  // anomaly missing:
  // anomaly actions:
  // anomaly advice:
  // representative missing:
  // representative actions:
  // representative advice:
  // accepted / in_progress / pending
  // ready / pending
  // 异常样本：auth / rate / local / manual
  // 代表样本：large / nested / retry
  // 当前筛选 全部 没有真实样本验收矩阵。
  if (!Array.isArray(visibleItems) || !visibleItems.length) {
    return `<div class="directory-empty">${escapeHTML(
      tf(
        "status.matrix_filter_empty",
        { filter: providerSmokeMatrixFilterLabel(state.providerSmokeMatrixFilter) },
        `当前筛选 ${providerSmokeMatrixFilterLabel(state.providerSmokeMatrixFilter)} 没有真实样本验收矩阵。`,
      ),
    )}</div>`;
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
            <span class="pill">${escapeHTML(t("status.matrix_smoke_count", "样本"))} ${stringifyValue(item.smokeCount, "0")}</span>
            <span class="pill">${escapeHTML(t("status.matrix_upload_smoke", "上传样本"))} ${stringifyValue(item.uploadSuccessCount, "0")} / ${renderProviderSmokeMatrixState(item.hasUploadSuccessSample ? "ready" : "pending")}</span>
            <span class="pill">${escapeHTML(t("status.matrix_coverage", "任务覆盖"))} ${stringifyValue(item.coverageRealSuccessTaskCount, "0")}/${stringifyValue(item.coverageTaskCount, "0")}</span>
            <span class="pill">${renderProviderSmokeMatrixState(item.accepted ? "accepted" : item.acceptanceStatus || "pending")}</span>
          </div>
          <div class="muted">${escapeHTML(t("status.matrix_smoke_sample", "样本记录："))}${escapeHTML(stringifyValue(item.sampleTitle, "-"))} / ${escapeHTML(stringifyValue(item.sampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.sampleCategory, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_coverage_sample", "任务样本："))}${escapeHTML(stringifyValue(item.coverageSampleProviderKey, "-"))} / ${escapeHTML(stringifyValue(item.coverageSampleTaskState, "-"))} / ${escapeHTML(stringifyValue(item.coverageSampleCompletionKind, "-"))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_readiness", "就绪度："))}${escapeHTML(renderProviderSmokeReadiness(item))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_checklist", "验收清单："))}${escapeHTML(renderProviderSmokeChecklist(item))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_gaps", "缺口："))}${escapeHTML(renderProviderSmokeGaps(item))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_next_action", "下一步动作："))}${escapeHTML(renderProviderSmokeNextAction(item))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_priority_action", "验收优先动作："))}${escapeHTML(renderProviderSmokePriorityAction(item))}</div>
          <div class="muted">${escapeHTML(t("status.matrix_anomaly_summary", "异常样本："))}auth ${renderProviderSmokeMatrixState(item.hasAuthExpiredSample ? "ready" : "pending")} / rate ${renderProviderSmokeMatrixState(item.hasRateLimitedSample ? "ready" : "pending")} / local ${renderProviderSmokeMatrixState(item.hasLocalFileMissingSample ? "ready" : "pending")} / manual ${renderProviderSmokeMatrixState(item.hasPendingManualSample ? "ready" : "pending")}</div>
          <div class="muted">${escapeHTML(t("status.matrix_representative_summary", "代表样本："))}large ${renderProviderSmokeMatrixState(item.hasLargeFileSample ? "ready" : "pending")} / nested ${renderProviderSmokeMatrixState(item.hasNestedDirectorySample ? "ready" : "pending")} / retry ${renderProviderSmokeMatrixState(item.hasRetryRecoverySample ? "ready" : "pending")}</div>
          ${Array.isArray(item.anomalyMissing) && item.anomalyMissing.length ? `<div class="muted">${escapeHTML(t("status.matrix_anomaly_missing", "异常样本缺口："))}${escapeHTML(item.anomalyMissing.join(", "))}</div>` : ""}
          ${Array.isArray(item.anomalyActions) && item.anomalyActions.length ? `<div class="muted">${escapeHTML(t("status.matrix_anomaly_actions", "异常样本动作："))}${escapeHTML(item.anomalyActions.join("；"))}</div>` : ""}
          ${item.anomalyAdvice ? `<div class="muted">${escapeHTML(t("status.matrix_anomaly_advice", "异常样本建议："))}${escapeHTML(item.anomalyAdvice)}</div>` : ""}
          ${Array.isArray(item.representativeMissing) && item.representativeMissing.length ? `<div class="muted">${escapeHTML(t("status.matrix_representative_missing", "代表样本缺口："))}${escapeHTML(item.representativeMissing.join(", "))}</div>` : ""}
          ${Array.isArray(item.representativeActions) && item.representativeActions.length ? `<div class="muted">${escapeHTML(t("status.matrix_representative_actions", "代表样本动作："))}${escapeHTML(item.representativeActions.join("；"))}</div>` : ""}
          ${item.representativeAdvice ? `<div class="muted">${escapeHTML(t("status.matrix_representative_advice", "代表样本建议："))}${escapeHTML(item.representativeAdvice)}</div>` : ""}
          ${Array.isArray(item.acceptanceMissing) && item.acceptanceMissing.length ? `<div class="muted">${escapeHTML(t("status.matrix_missing", "验收缺口："))}${escapeHTML(item.acceptanceMissing.join(", "))}</div>` : ""}
          ${Array.isArray(item.acceptanceActions) && item.acceptanceActions.length ? `<div class="muted">${escapeHTML(t("status.matrix_actions", "验收动作："))}${escapeHTML(item.acceptanceActions.join("；"))}</div>` : ""}
          ${item.acceptanceAdvice ? `<div class="muted">${escapeHTML(t("status.matrix_advice", "验收建议："))}${escapeHTML(item.acceptanceAdvice)}</div>` : ""}
          <div class="actions compact">
            ${item.sampleRecordId ? `<button type="button" class="ghost" data-provider-smoke-open-record="${escapeHTML(stringifyValue(item.sampleRecordId))}">${escapeHTML(t("status.matrix_open_smoke_record", "打开样本记录"))}</button>` : ""}
            ${item.coverageSampleTaskId ? `<button type="button" class="ghost" data-provider-smoke-open-task="${escapeHTML(stringifyValue(item.coverageSampleTaskId))}">${escapeHTML(t("status.matrix_open_task_sample", "打开任务样本"))}</button>` : ""}
            <button type="button" class="ghost" data-provider-smoke-draft="${escapeHTML(stringifyValue(item.protocolGroup))}">${escapeHTML(t("status.matrix_prefill_smoke_form", "预填样本表单"))}</button>
            <button type="button" class="ghost" data-provider-smoke-draft-action="${escapeHTML(stringifyValue(item.protocolGroup))}">${escapeHTML(providerSmokeDraftActionLabel(item))}</button>
            <button type="button" class="ghost" data-provider-smoke-prefill-profile-risk="${escapeHTML(stringifyValue(item.protocolGroup))}">${escapeHTML(t("status.matrix_prefill_profile_risk", "预填账号默认风控"))}</button>
            <button type="button" class="ghost" data-provider-smoke-focus-group="${escapeHTML(stringifyValue(item.protocolGroup))}">${escapeHTML(t("status.matrix_focus_group_records", "只看该组记录"))}</button>
            <button type="button" class="ghost" data-provider-smoke-filter-status="${escapeHTML(item.accepted ? "accepted" : item.acceptanceStatus || "pending")}">${escapeHTML(t("status.matrix_focus_acceptance_type", "只看此类"))}</button>
          </div>
          <div class="muted">${escapeHTML(t("status.matrix_latest_observed", "最近样本 / 覆盖观察："))}<code>${escapeHTML(stringifyValue(item.latestSmokeAt, "-"))}</code> / <code>${escapeHTML(stringifyValue(item.coverageLastObservedAt, "-"))}</code></div>
        </div>
      `,
    )
    .join("");
}

function renderProviderSmokeMarkdown(markdown) {
  if (!markdown) {
    return `<div class="directory-empty">${escapeHTML(t("status.provider_smoke_markdown_empty", "请选择一条样本记录查看样本文档。"))}</div>`;
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
      label: t("status.smoke_draft_label_auth_expired", "补授权失效样本"),
      category: "failed",
      result: "failure",
      operations: ["ValidateAuth", "Upload", "refresh_auth_profile"],
      focusResult: "failure",
      note: t("status.smoke_draft_note_auth_expired", "目标异常：auth_expired"),
    };
  }
  if (missing.includes("rate_limited_sample_missing")) {
    return {
      label: t("status.smoke_draft_label_rate_limited", "补限流样本"),
      category: "partial_blocked",
      result: "failure",
      operations: ["ValidateAuth", "List", "Metadata", "Upload", "cooldown", "retry_window"],
      focusResult: "failure",
      note: t("status.smoke_draft_note_rate_limited", "目标异常：rate_limited / risk_control"),
    };
  }
  if (missing.includes("local_file_missing_sample_missing")) {
    return {
      label: t("status.smoke_draft_label_local_file_missing", "补本地文件缺失样本"),
      category: "failed",
      result: "failure",
      operations: ["ValidateAuth", "Upload", "restore_local_source_file"],
      focusResult: "failure",
      note: t("status.smoke_draft_note_local_file_missing", "目标异常：local_file_missing"),
    };
  }
  if (missing.includes("pending_manual_sample_missing")) {
    return {
      label: t("status.smoke_draft_label_pending_manual", "补人工确认样本"),
      category: "partial_blocked",
      result: "failure",
      operations: ["ValidateAuth", "List", "Metadata", "Upload", "manual_confirmation", "blocked_recovery"],
      focusResult: "failure",
      note: t("status.smoke_draft_note_pending_manual", "目标异常：pending_manual / overwrite downgrade"),
    };
  }
  return null;
}

function providerSmokeDraftSpecFromRepresentative(item) {
  const missing = Array.isArray(item?.representativeMissing) ? item.representativeMissing : [];
  if (missing.includes("large_file_sample_missing")) {
    return {
      label: t("status.smoke_draft_label_large_file", "补大文件样本"),
      category: "binary_upload_success",
      result: "success",
      operations: ["ValidateAuth", "List", "Metadata", "CreateDir", "Upload", "checkpoint"],
      focusResult: "success",
      note: t("status.smoke_draft_note_large_file", "目标代表样本：large_file / multipart / 大文件上传恢复"),
    };
  }
  if (missing.includes("nested_directory_sample_missing")) {
    return {
      label: t("status.smoke_draft_label_nested_directory", "补多层目录样本"),
      category: "browse_only",
      result: "success",
      operations: ["ValidateAuth", "List", "Metadata", "CreateDir", "leaf_first"],
      focusResult: "success",
      note: t("status.smoke_draft_note_nested_directory", "目标代表样本：nested_directory / 多层目录 / subtree"),
    };
  }
  if (missing.includes("retry_recovery_sample_missing")) {
    return {
      label: t("status.smoke_draft_label_retry_recovery", "补重试恢复样本"),
      category: "binary_upload_success",
      result: "success",
      operations: ["ValidateAuth", "List", "Metadata", "Upload", "checkpoint"],
      focusResult: "success",
      note: t("status.smoke_draft_note_retry_recovery", "目标代表样本：retry_recovery / checkpoint / resume / 续传"),
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
    protocolGroup ? tf("status.smoke_note_protocol_group", { value: protocolGroup }, `协议族：${protocolGroup}`) : "",
    providerKey ? tf("status.smoke_note_provider", { value: providerKey }, `建议 provider：${providerKey}`) : "",
    tf("status.smoke_note_status", { value: status }, `验收状态：${status}`),
    missing.length ? tf("status.smoke_note_missing", { value: missing.join(", ") }, `缺口：${missing.join(", ")}`) : "",
    actions.length ? tf("status.smoke_note_action", { value: actions.join("；") }, `动作：${actions.join("；")}`) : "",
    item?.acceptanceAdvice ? tf("status.smoke_note_advice", { value: item.acceptanceAdvice }, `建议：${item.acceptanceAdvice}`) : "",
  ].filter(Boolean);
  return {
    providerKey,
    protocolGroup,
    authMode: "",
    category,
    result: "success",
    title: protocolGroup ? tf("status.smoke_draft_title_status", { protocolGroup, status }, `${protocolGroup} ${status} smoke`) : t("status.smoke_draft_title_provider", "网盘样本"),
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
    draft.note = [draft.note, tf("status.smoke_note_goal", { value: actions.join("；") }, `本次目标：${actions.join("；")}`)].filter(Boolean).join("；");
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
    return t("status.smoke_action_fill_upload_smoke", "补上传成功样本");
  }
  if (missing.includes("task_coverage_missing")) {
    return t("status.smoke_action_fill_task_sample", "补任务覆盖样本");
  }
  if (missing.includes("real_smoke_success_missing")) {
    return t("status.smoke_action_fill_smoke_success", "补成功样本");
  }
  return t("status.smoke_action_prefill_by_gap", "按缺口预填动作");
}

function focusProviderSmokeRecordsByResult(result) {
  const normalized = String(result || "").trim().toLowerCase();
  state.providerSmokeRecordFilters.result = normalized;
  setFilterControlValue("#provider-smoke-records-filter-result", normalized);
  renderStatus();
  const label = normalized
    ? t(
      normalized === "success" ? "status.result_success" : "status.result_failure",
      normalized === "success" ? t("status.result_success", "成功（success）") : t("status.result_failure", "失败（failure）"),
    )
    : "";
  showFlash(
    normalized
      ? tf("status.flash_smoke_records_filtered", { label }, `已按 ${label} 收敛样本记录`)
      : t("status.flash_smoke_records_result_cleared", "已清空样本记录结果筛选"),
  );
}

function focusProviderSmokeMatrixByStatus(status) {
  setProviderSmokeMatrixFilter(status || "all");
  const raw = status || "all";
  const label = state.language === "en-US" ? providerSmokeMatrixFilterLabel(raw) : raw;
  showFlash(tf("status.flash_smoke_matrix_filtered", { label }, `已按 ${label} 收敛验收矩阵`));
}

function openProviderSmokeRecordInMatrix(id) {
  const record = (state.providerSmokes || []).find((item) => item?.id === id) || null;
  if (record) {
    hydrateProviderSmokeForm(record);
    state.selectedProviderSmokeId = id;
    focusProviderSmokeRecordsByGroup(record.protocolGroup || "");
  }
  return loadProviderSmokeMarkdown(id).then(() => {
    showFlash(t("status.flash_smoke_record_opened", "已打开样本记录并回填表单"));
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
    showFlash(t("status.flash_smoke_gap_prefilled", "已按验收缺口预填样本动作"));
    return;
  }
  showFlash(t("status.flash_smoke_matrix_prefilled", "已按验收矩阵预填样本表单"));
}

function buildProviderSmokeDraftByProtocolGroup(protocolGroup, { fromGap = false } = {}) {
  const row = (state.providerSmokeMatrix || []).find((item) => item.protocolGroup === protocolGroup);
  if (!row) {
    showFlash(t("wizard.flash_smoke_matrix_group_missing", "未找到对应协议组的验收矩阵项"), true);
    return false;
  }
  draftProviderSmokeAndFocus(row, { fromGap });
  return true;
}

function draftProviderSmokeAndOpenTask(taskID) {
  if (!taskID) {
    showFlash(t("wizard.flash_smoke_matrix_task_missing", "当前验收矩阵项没有可打开的任务样本"), true);
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
  const sourceDisplay = draft.protocolGroup ? tf("status.smoke_matrix_source_display", { protocolGroup: draft.protocolGroup, status }, `Smoke Matrix ${draft.protocolGroup} (${status})`) : tf("status.smoke_matrix_source_display_fallback", { status }, `Smoke Matrix (${status})`);
  activateTab("providers");
  setSelectValueIfPresent("#profile-provider", draft.providerKey || "");
  setInputValueIfPresent("#profile-display-name", draft.protocolGroup ? tf("status.smoke_matrix_risk_profile_title", { protocolGroup: draft.protocolGroup }, `${draft.protocolGroup} 风控模板`) : t("status.smoke_matrix_risk_profile_title_fallback", "真实样本风控模板"));
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
  showFlash(t("status.flash_smoke_profile_risk_prefilled", "已按真实样本预填账号默认风控"));
}

function draftProviderSmokeFromAccepted(item) {
  const draft = draftProviderSmokeFromMatrix(item);
  const boundarySmokeTitle = t("status.smoke_title_boundary_sample", "边界场景样本");
  draft.title = draft.protocolGroup ? `${draft.protocolGroup} ${boundarySmokeTitle}` : boundarySmokeTitle;
  draft.note = [draft.note, t("status.smoke_note_boundary_sample", "当前协议组已验收，建议继续补充边界场景样本。")].filter(Boolean).join("；");
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
  showFlash(
    normalized
      ? tf("status.flash_smoke_records_filtered", { label: normalized }, `已按 ${normalized} 收敛样本记录`)
      : t("status.flash_smoke_records_group_cleared", "已清空样本记录协议组筛选"),
  );
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
    throw new Error(tf("status.smoke_markdown_load_failed", { status: response.status }, `加载样本文档失败：${response.status}`));
  }
  state.selectedProviderSmokeId = id;
  state.selectedProviderSmokeMarkdown = await response.text();
  renderStatus();
}

async function bootstrapData() {
  await Promise.all([loadProviders(), loadProfiles(), loadTasks(), loadStatus()]);
  renderAllDirectoryBrowsers();
  await refreshDirectoryBrowsers();
}

function wireLanguage() {
  const select = $("#language-select");
  if (!select) {
    return;
  }
  select.value = state.language || "zh-CN";
  select.addEventListener("change", () => {
    const next = String(select.value || "zh-CN").trim();
    state.language = translations[next] ? next : "zh-CN";
    saveLanguage();
    applyI18n();
    syncSessionState();
    syncAuthAssistInputs();
    syncProfileAuthGuide();
    renderPreview();
    renderTasks();
    renderStatus();
    renderAllDirectoryBrowsers();
  });
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
      showFlash(t("flash_login_success", "登录验证成功"));
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
    showFlash(t("flash_logout_success", "本地控制台会话已清理"));
  });
}

function wireProfiles() {
  $("#profile-provider").addEventListener("change", syncAuthModes);
  $("#profile-auth-mode").addEventListener("change", syncProfileAuthGuide);
  ["#profile-assist-openlist-url", "#profile-assist-openlist-token", "#profile-assist-alist-url", "#profile-assist-alist-token"].forEach((selector) => {
    $(selector).addEventListener("input", () => {
      persistAuthAssistState(collectAuthAssistStateFromForm());
    });
  });
  $("#profile-assist-use-openlist").addEventListener("click", () => {
    switchAuthAssistMode("openlist");
    showFlash(t("providers.flash_use_openlist", "已切换为 OpenList 优先引导"));
  });
  $("#profile-assist-use-alist").addEventListener("click", () => {
    switchAuthAssistMode("alist");
    showFlash(t("providers.flash_use_alist", "已切换为 Alist 兜底引导"));
  });
  $("#profile-assist-use-manual").addEventListener("click", () => {
    switchAuthAssistMode("manual");
    showFlash(t("providers.flash_use_manual", "已切换为手动高级模式"));
  });
  $("#profile-assist-discover-openlist").addEventListener("click", async () => {
    try {
      const result = await discoverAuthAssist("openlist");
      showFlash(t("providers.flash_detect_openlist", "已检测 OpenList，可见存储 {count} 项").replace("{count}", String(Array.isArray(result.storages) ? result.storages.length : 0)));
    } catch (error) {
      syncAuthAssistDiscovery(error.message);
      showFlash(error.message, true);
    }
  });
  $("#profile-assist-discover-alist").addEventListener("click", async () => {
    try {
      const result = await discoverAuthAssist("alist");
      showFlash(t("providers.flash_detect_alist", "已检测 Alist，可见存储 {count} 项").replace("{count}", String(Array.isArray(result.storages) ? result.storages.length : 0)));
    } catch (error) {
      syncAuthAssistDiscovery(error.message);
      showFlash(error.message, true);
    }
  });
  $("#profile-assist-open-openlist").addEventListener("click", () => openAuthAssistURL("openlist"));
  $("#profile-assist-open-alist").addEventListener("click", () => openAuthAssistURL("alist"));
  $("#profile-assist-clear").addEventListener("click", () => {
    persistAuthAssistState(defaultAuthAssistState());
    syncAuthAssistDiscovery("");
    showFlash(t("providers.flash_reset_assist", "授权引导配置已清空，已恢复 OpenList 优先"));
  });
  $("#profile-assist-discovery").addEventListener("click", (event) => {
    const button = event.target.closest("[data-assist-select-index]");
    if (!button) {
      return;
    }
    try {
      applyAuthAssistDiscoverySelection(Number(button.dataset.assistSelectIndex));
    } catch (error) {
      showFlash(
        t("providers.assist_discovery_apply_failed", "回填发现结果失败：{error}").replace("{error}", error.message),
        true,
      );
    }
  });
  $("#plan-source-provider").addEventListener("change", async () => {
    syncSourceProfiles();
    await loadDirectoryBrowser("source");
  });
  $("#plan-source-profile").addEventListener("change", async () => {
    resetDirectoryBrowser("source");
    renderDirectoryBrowser("source");
    await loadDirectoryBrowser("source");
  });
  $("#plan-target-profile").addEventListener("change", async () => {
    syncTargetProfileInsight();
    resetDirectoryBrowser("target");
    renderDirectoryBrowser("target");
    await loadDirectoryBrowser("target");
  });
  $("#plan-target-provider").addEventListener("change", async () => {
    syncTargetProfiles();
    await loadDirectoryBrowser("target");
    const providerKey = $("#plan-target-provider").value;
    if (!providerKey) {
      return;
    }
    try {
      await loadProviderCapabilityDetail(providerKey);
    } catch (error) {
      showFlash(
        t("providers.provider_detail_load_failed", "加载网盘能力详情失败：{error}").replace("{error}", error.message),
        true,
      );
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
      showFlash(
        t("providers.provider_detail_loaded", "已加载 {provider} 网盘能力详情").replace(
          "{provider}",
          button.dataset.providerDetailOpen || "",
        ),
      );
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#refresh-providers").addEventListener("click", async () => {
    try {
      await loadProviders();
      showFlash(t("providers.flash_refresh_catalog", "网盘列表已刷新"));
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#refresh-profiles").addEventListener("click", async () => {
    try {
      await loadProfiles();
      await refreshDirectoryBrowsers();
      showFlash(t("providers.flash_refresh_profiles", "授权档案已刷新"));
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#sync-profile-risk-defaults").addEventListener("click", () => {
    try {
      const extra = parseJSONInput($("#profile-extra").value, {});
      const merged = mergeProfileRiskDefaultsIntoExtra(extra, collectRiskProfileFromForm("profile-risk"));
      $("#profile-extra").value = Object.keys(merged).length ? JSON.stringify(merged, null, 2) : "";
      showFlash(t("providers.flash_sync_risk", "账号默认风控已同步到 Extra JSON"));
    } catch (error) {
      showFlash(
        t("wizard.flash_extra_json_parse_error", "Extra JSON 无法解析：{error}").replace("{error}", error.message),
        true,
      );
    }
  });

  $("#clear-profile-risk-defaults").addEventListener("click", () => {
    try {
      const extra = parseJSONInput($("#profile-extra").value, {});
      const merged = mergeProfileRiskDefaultsIntoExtra(extra, null);
      $("#profile-extra").value = Object.keys(merged).length ? JSON.stringify(merged, null, 2) : "";
      hydrateRiskProfileForm("profile-risk", null);
      showFlash(t("providers.flash_clear_risk", "账号默认风控已清空"));
    } catch (error) {
      showFlash(
        t("wizard.flash_extra_json_parse_error", "Extra JSON 无法解析：{error}").replace("{error}", error.message),
        true,
      );
    }
  });

  $("#profile-cancel-edit").addEventListener("click", () => {
    resetProfileForm();
    focusProfile("");
    showFlash(t("providers.flash_cancel_edit", "已退出授权档案编辑"));
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
      await refreshDirectoryBrowsers();
      showFlash(profileID ? t("providers.flash_profile_updated", "授权档案已更新") : t("providers.flash_profile_created", "授权档案已创建"));
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
    targetRoot: $("#plan-target-root").value.trim() || "/",
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
    showFlash(t("providers.risk_override_synced_note", "风控覆盖已同步到 JSON"));
  });
  $("#clear-risk-override").addEventListener("click", () => {
    hydrateRiskOverrideForm(null);
    $("#plan-risk-override").value = "";
    showFlash(t("providers.risk_override_cleared_note", "风控覆盖已清空，将使用默认档位和网盘源校准"));
  });
  $("#plan-selected-roots").addEventListener("input", () => renderDirectoryBrowser("source"));
  $("#plan-target-root").addEventListener("input", () => renderDirectoryBrowser("target"));
  $("#plan-source-browser-refresh").addEventListener("click", async () => {
    await loadDirectoryBrowser("source");
    showFlash(
      t("providers.browser_source_refreshed", "源目录已刷新：{path}").replace(
        "{path}",
        state.directoryBrowsers.source.currentPath || "/",
      ),
    );
  });
  $("#plan-target-browser-refresh").addEventListener("click", async () => {
    await loadDirectoryBrowser("target");
    showFlash(
      t("providers.browser_target_refreshed", "目标目录已刷新：{path}").replace(
        "{path}",
        state.directoryBrowsers.target.currentPath || "/",
      ),
    );
  });
  $("#plan-source-browser-up").addEventListener("click", async () => {
    await loadDirectoryBrowser("source", parentComparePath(state.directoryBrowsers.source.currentPath));
    showFlash(
      t("providers.browser_source_up", "已返回上级源目录：{path}").replace(
        "{path}",
        state.directoryBrowsers.source.currentPath || "/",
      ),
    );
  });
  $("#plan-target-browser-up").addEventListener("click", async () => {
    await loadDirectoryBrowser("target", parentComparePath(state.directoryBrowsers.target.currentPath));
    showFlash(
      t("providers.browser_target_up", "已返回上级目标目录：{path}").replace(
        "{path}",
        state.directoryBrowsers.target.currentPath || "/",
      ),
    );
  });
  $("#plan-source-browser-select-current").addEventListener("click", () => {
    applyDirectoryBrowserSelection("source", state.directoryBrowsers.source.currentPath);
  });
  $("#plan-target-browser-select-current").addEventListener("click", () => {
    applyDirectoryBrowserSelection("target", state.directoryBrowsers.target.currentPath);
  });
  $("#plan-target-browser-create").addEventListener("click", async () => {
    try {
      await createTargetBrowserDirectory();
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#plan-source-browser-list").addEventListener("click", async (event) => {
    const openButton = event.target.closest("[data-browser-open='source']");
    const selectButton = event.target.closest("[data-browser-select='source']");
    if (openButton) {
      try {
        await loadDirectoryBrowser("source", openButton.dataset.browserPath || "/", { fileId: openButton.dataset.browserFileId || "" });
        showFlash(
          t("providers.browser_source_opened", "已打开源目录：{path}").replace(
            "{path}",
            state.directoryBrowsers.source.currentPath || "/",
          ),
        );
      } catch (error) {
        showFlash(error.message, true);
      }
      return;
    }
    if (selectButton) {
      applyDirectoryBrowserSelection("source", selectButton.dataset.browserPath || "/");
    }
  });
  $("#plan-target-browser-list").addEventListener("click", async (event) => {
    const openButton = event.target.closest("[data-browser-open='target']");
    const selectButton = event.target.closest("[data-browser-select='target']");
    if (openButton) {
      try {
        await loadDirectoryBrowser("target", openButton.dataset.browserPath || "/", { fileId: openButton.dataset.browserFileId || "" });
        showFlash(
          t("providers.browser_target_opened", "已打开目标目录：{path}").replace(
            "{path}",
            state.directoryBrowsers.target.currentPath || "/",
          ),
        );
      } catch (error) {
        showFlash(error.message, true);
      }
      return;
    }
    if (selectButton) {
      applyDirectoryBrowserSelection("target", selectButton.dataset.browserPath || "/");
    }
  });
  $("#plan-source-browser-breadcrumbs").addEventListener("click", async (event) => {
    const button = event.target.closest("[data-browser-breadcrumb='source']");
    if (!button) {
      return;
    }
    try {
      await loadDirectoryBrowser("source", button.dataset.browserPath || "/");
      showFlash(
        t("providers.browser_source_jumped", "已跳转到源目录：{path}").replace(
          "{path}",
          state.directoryBrowsers.source.currentPath || "/",
        ),
      );
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#plan-target-browser-breadcrumbs").addEventListener("click", async (event) => {
    const button = event.target.closest("[data-browser-breadcrumb='target']");
    if (!button) {
      return;
    }
    try {
      await loadDirectoryBrowser("target", button.dataset.browserPath || "/");
      showFlash(
        t("providers.browser_target_jumped", "已跳转到目标目录：{path}").replace(
          "{path}",
          state.directoryBrowsers.target.currentPath || "/",
        ),
      );
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#plan-risk-override").addEventListener("blur", () => {
    try {
      hydrateRiskOverrideForm(parseJSONInput($("#plan-risk-override").value, null));
    } catch (error) {
      showFlash(
        t("wizard.flash_risk_override_parse_error", "风控覆盖 JSON 无法解析：{error}").replace("{error}", error.message),
        true,
      );
    }
  });
  $("#apply-recommended-execution").addEventListener("click", () => {
    const recommended = state.preview?.metadata?.recommendedExecutionMode;
    if (!recommended) {
      showFlash(t("wizard.flash_preview_required_execution", "请先生成计划预览"), true);
      return;
    }
    setSelectValueIfPresent("#plan-execution-mode", recommended);
    showFlash(t("wizard.flash_execution_applied", "已采用推荐执行模式：{mode}").replace("{mode}", recommended));
  });
  $("#apply-recommended-risk").addEventListener("click", () => {
    const recommended = state.preview?.metadata?.recommendedRiskMode;
    if (!recommended) {
      showFlash(t("wizard.flash_preview_required_execution", "请先生成计划预览"), true);
      return;
    }
    setSelectValueIfPresent("#plan-risk-mode", recommended);
    showFlash(t("wizard.flash_risk_applied", "已采用推荐风控档位：{mode}").replace("{mode}", recommended));
  });

  $("#preview-plan").addEventListener("click", async () => {
    try {
      const payload = buildPlanPayload();
      state.preview = await api("/api/plans/preview", {
        method: "POST",
        body: {
          sourceProvider: payload.sourceProvider,
          targetProvider: payload.targetProvider,
          targetRoot: payload.targetRoot,
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
      showFlash(t("wizard.flash_preview_generated", "计划预览已生成"));
    } catch (error) {
      showFlash(error.message, true);
    }
  });

  $("#plan-form").addEventListener("submit", async (event) => {
    event.preventDefault();
    try {
      const payload = buildPlanPayload();
      if (payloadHasOnlyDeletedEntries(payload)) {
        showFlash(t("wizard.flash_delete_only_blocked", "当前只有删除记录，没有可执行条目；请先恢复源文件并重新预览"), true);
        return;
      }
      const detail = await api("/api/tasks", {
        method: "POST",
        body: payload,
      });
      state.selectedTaskId = detail.task.id;
      showFlash(t("wizard.flash_task_created", "任务已创建"));
      await loadTasks();
      document.querySelector('[data-view="tasks"]').click();
    } catch (error) {
      showFlash(error.message, true);
    }
  });
}

async function runTaskAction(action, body = undefined) {
  if (!state.selectedTaskId) {
    showFlash(t("wizard.flash_task_required", "请先选择任务"), true);
    return false;
  }
  return runTaskActionForTask(state.selectedTaskId, action, body);
}

function taskActionFeedbackLabel(action) {
  switch (String(action || "").trim()) {
    case "run":
      return t("tasks.run", "运行");
    case "pause":
      return t("tasks.pause", "暂停");
    case "resume":
      return t("tasks.resume", "继续");
    case "retry":
      return t("tasks.retry", "重试");
    default:
      return String(action || "").trim() || "-";
  }
}

async function runTaskActionForTask(taskId, action, body = undefined) {
  const normalizedTaskId = String(taskId || "").trim();
  if (!normalizedTaskId) {
    showFlash(t("wizard.flash_task_id_missing", "缺少任务标识"), true);
    return false;
  }
  if (state.taskActionPending) {
    showFlash(t("wizard.flash_task_action_pending", "任务操作执行中，请稍候"), true);
    return false;
  }
  state.taskActionPending = true;
  syncTaskActionButtons();
  try {
    await api(`/api/tasks/${normalizedTaskId}/${action}`, { method: "POST", body });
    showFlash(tf("wizard.flash_task_action_executed", { action: taskActionFeedbackLabel(action) }, `任务${taskActionFeedbackLabel(action)}已执行`));
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
    return t("status.selection_scope_directory", "目录子集");
  }
  return source === "pending_tree" || source === "selected_pending_subset" ? "selected_pending_subset" : "selected_retry_subset";
}
function selectionSourceLabel(source) {
  if (source === "directory_tree" || source === "selected_directory_subset") {
    return t("status.selection_source_directory", "目录子集");
  }
  if (source === "pending_tree" || source === "selected_pending_subset") {
    return t("status.selection_source_pending", "待补传子集");
  }
  return t("status.selection_source_retry", "重试队列子集");
}

function selectionScopeLabel(scope) {
  const normalized = String(scope || "").trim();
  if (normalized === "selected_directory_subset") {
    return t("status.selection_scope_directory", "目录子集");
  }
  if (normalized === "selected_pending_subset") {
    return t("status.selection_scope_pending", "待补传子集");
  }
  return t("status.selection_scope_retry", "重试队列子集");
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
    showFlash(scope === "task" ? t("wizard.flash_prefill_task_required", "请先选择任务") : t("wizard.flash_prefill_status_task_missing", "当前状态样本没有可用任务"), true);
    return;
  }
  const paths = visibleSelectionPaths(scope, source);
  if (!paths.length) {
    showFlash(t("wizard.flash_prefill_visible_path_missing", "当前筛选结果里没有可用于重建向导的路径"), true);
    return;
  }
  prefillWizardFromTaskPaths(context.detail, paths, `${selectionSourceLabel(source)} / ${selectionScopeLabel(recoverScopeFromSource(source))}`);
}

async function retryVisibleSelection(scope, source) {
  const context = taskContextByScope(scope);
  if (!context?.taskId) {
    showFlash(scope === "task" ? t("wizard.flash_task_required", "请先选择任务") : t("wizard.flash_prefill_status_task_missing", "当前状态样本没有可用任务"), true);
    return;
  }
  const paths = visibleSelectionPaths(scope, source);
  if (!paths.length) {
    showFlash(t("wizard.flash_retry_visible_path_missing", "当前筛选结果里没有可重试的路径"), true);
    return;
  }
  const ok = await runTaskActionForTask(context.taskId, "retry", { paths, scope: recoverScopeFromSource(source) });
  if (ok) {
    showFlash(tf("tasks.flash_retry_selection_rebuilt", { source: selectionSourceLabel(source), scope: selectionScopeLabel(recoverScopeFromSource(source)), count: paths.length }, `已按${selectionSourceLabel(source)} (${selectionScopeLabel(recoverScopeFromSource(source))}) 重建 ${paths.length} 条路径的 retry 范围`));
  }
}

async function autoRecoverVisibleSelection(scope, source) {
  const context = taskContextByScope(scope);
  if (!context || !context.taskId) {
    showFlash(scope === "task" ? t("wizard.flash_task_required", "请先选择任务") : t("wizard.flash_prefill_status_task_missing", "当前状态样本没有可用任务"), true);
    return;
  }
  const paths = visibleSelectionPaths(scope, source);
  if (!paths.length) {
    showFlash(t("wizard.flash_auto_recover_visible_path_missing", "当前筛选结果里没有可后台补传的路径"), true);
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
    tf("status.auto_recover_selection_completed", { source: selectionSourceLabel(source), scope: selectionScopeLabel(recoverScopeFromSource(source)), paths: String(paths.length), matched: stringifyValue(result.matchedCount, "0"), recovered: stringifyValue(result.recoveredCount, "0"), modeBudget: stringifyValue(result.skippedByModeBudget, "0"), laneBudget: stringifyValue(result.skippedByLaneBudget, "0"), providerBudget: stringifyValue(result.skippedByProviderBudget, "0") }, `后台补传筛选已执行：${selectionSourceLabel(source)} / ${selectionScopeLabel(recoverScopeFromSource(source))} / 路径 ${paths.length} / 匹配 ${stringifyValue(result.matchedCount, "0")} / 已恢复 ${stringifyValue(result.recoveredCount, "0")} / 模式预算 ${stringifyValue(result.skippedByModeBudget, "0")} / lane 预算 ${stringifyValue(result.skippedByLaneBudget, "0")} / 网盘源预算 ${stringifyValue(result.skippedByProviderBudget, "0")}`),
    result.recoveredCount <= 0,
  );
}

async function retryTaskPath(scope, path) {
  const normalizedPath = normalizeComparePath(path);
  const taskId = scope === "task" ? state.selectedTaskId : currentStatusTaskContext()?.taskId;
  if (!taskId) {
    showFlash(scope === "task" ? t("wizard.flash_task_required", "请先选择任务") : t("wizard.flash_prefill_status_task_missing", "当前状态样本没有可用任务"), true);
    return false;
  }
  if (!normalizedPath) {
    showFlash(t("wizard.flash_retry_invalid_path", "当前路径无效，无法重试"), true);
    return false;
  }
  const ok = await runTaskActionForTask(taskId, "retry", {
    paths: [normalizedPath],
    scope: "selected_pending_subset",
  });
  if (ok) {
    showFlash(tf("tasks.flash_retry_single_path_rebuilt", { path: normalizedPath, scope: selectionScopeLabel("selected_pending_subset") }, `已按 ${normalizedPath} (${selectionScopeLabel("selected_pending_subset")}) 重建 retry 范围`));
  }
  return ok;
}

function wireTasks() {
  $("#refresh-tasks").addEventListener("click", async () => {
    try {
      await loadTasks();
      showFlash(t("tasks.flash_tasks_refreshed", "任务列表已刷新"));
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
      showFlash(t("tasks.flash_status_refreshed", "状态矩阵已刷新"));
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
      showFlash(t("tasks.flash_provider_smoke_saved", "网盘样本记录已保存"));
      await loadStatus();
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#refresh-report").addEventListener("click", async () => {
    try {
      await loadStatus();
      showFlash(t("tasks.flash_report_refreshed", "验收报告已刷新"));
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
      showFlash(t("tasks.flash_report_saved", "验收报告已保存"));
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
        showFlash(t("status.acceptance_report_empty", "暂无可下载的验收报告"), true);
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
      showFlash(t("tasks.flash_report_downloaded", "验收报告已下载"));
    } catch (error) {
      showFlash(error.message, true);
    }
  });
  $("#report-history").addEventListener("click", async (event) => {
    const viewButton = event.target.closest("[data-report-view]");
    if (viewButton) {
      state.selectedReportId = viewButton.dataset.reportView || "";
      renderStatus();
      showFlash(t("tasks.flash_report_switched", "已切换验收报告"));
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
        showFlash(t("tasks.flash_smoke_markdown_switched", "已切换样本文档"));
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
          throw new Error(tf("tasks.flash_smoke_markdown_download_failed", { status: response.status }, `下载样本文档失败：${response.status}`));
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
        showFlash(t("tasks.flash_smoke_markdown_downloaded", "样本文档已下载"));
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
        showFlash(t("wizard.flash_smoke_matrix_group_missing", "未找到对应协议组的验收矩阵项"), true);
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
    showFlash(t("tasks.flash_task_directory_filters_cleared", "已清空任务目录树筛选"));
  });
  $("#task-pending-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("taskPending");
    setFilterControlValue("#task-pending-filter-query", "");
    setFilterControlValue("#task-pending-filter-reason", "");
    setFilterControlValue("#task-pending-filter-leaf-only", false);
    rerenderTask();
    showFlash(t("tasks.flash_task_pending_filters_cleared", "已清空任务待补传筛选"));
  });
  $("#status-directory-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("statusDirectory");
    setFilterControlValue("#status-directory-filter-query", "");
    setFilterControlValue("#status-directory-filter-status", "");
    setFilterControlValue("#status-directory-filter-leaf-only", false);
    setFilterControlValue("#status-directory-filter-problem-only", false);
    rerenderStatus();
    showFlash(t("tasks.flash_status_directory_filters_cleared", "已清空状态目录树筛选"));
  });
  $("#status-pending-filter-clear").addEventListener("click", () => {
    resetTreeFilterSection("statusPending");
    setFilterControlValue("#status-pending-filter-query", "");
    setFilterControlValue("#status-pending-filter-reason", "");
    setFilterControlValue("#status-pending-filter-leaf-only", false);
    rerenderStatus();
    showFlash(t("tasks.flash_status_pending_filters_cleared", "已清空状态待补传筛选"));
  });
  $("#provider-smoke-records-filter-query").addEventListener("input", (event) => {
    state.providerSmokeRecordFilters.query = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-group").addEventListener("input", (event) => {
    state.providerSmokeRecordFilters.protocolGroup = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-sample-type").addEventListener("input", (event) => {
    state.providerSmokeRecordFilters.sampleType = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-result").addEventListener("change", (event) => {
    state.providerSmokeRecordFilters.result = event.target.value;
    renderStatus();
  });
  $("#provider-smoke-records-filter-clear").addEventListener("click", () => {
    state.providerSmokeRecordFilters.query = "";
    state.providerSmokeRecordFilters.protocolGroup = "";
    state.providerSmokeRecordFilters.sampleType = "";
    state.providerSmokeRecordFilters.result = "";
    setFilterControlValue("#provider-smoke-records-filter-query", "");
    setFilterControlValue("#provider-smoke-records-filter-group", "");
    setFilterControlValue("#provider-smoke-records-filter-sample-type", "");
    setFilterControlValue("#provider-smoke-records-filter-result", "");
    renderStatus();
    showFlash(t("status.flash_smoke_records_cleared", "已清空样本记录筛选"));
  });
  $("#task-directory-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("task", "directory");
    } catch (error) {
      showFlash(t("wizard.flash_copy_directory_path_failed", "复制目录路径失败：{error}").replace("{error}", error.message), true);
    }
  });
  $("#task-pending-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("task", "pending");
    } catch (error) {
      showFlash(t("wizard.flash_copy_pending_path_failed", "复制待补传路径失败：{error}").replace("{error}", error.message), true);
    }
  });
  $("#status-directory-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("status", "directory");
    } catch (error) {
      showFlash(t("wizard.flash_copy_directory_path_failed", "复制目录路径失败：{error}").replace("{error}", error.message), true);
    }
  });
  $("#status-pending-copy-visible").addEventListener("click", async () => {
    try {
      await copyVisibleTreePaths("status", "pending");
    } catch (error) {
      showFlash(t("wizard.flash_copy_pending_path_failed", "复制待补传路径失败：{error}").replace("{error}", error.message), true);
    }
  });
}

async function init() {
  applyI18n();
  setupTabs();
  wireLanguage();
  wireLogin();
  wireProfiles();
  wirePlanner();
  wireTasks();
  wireStatus();
  wireTreeFilters();
  syncSessionState();
  syncAuthAssistInputs();
  syncExecutionModeHint();
  renderPreview();
  renderStatus();
  renderAllDirectoryBrowsers();
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


