# 风控模板清单（Risk Template Catalog）

## 说明

- 本文用于把当前仓库里已经落地的默认风控模板整理成一份可核对清单。
- 口径以 `internal/planner/service.go` 当前实现为准。
- 这份清单描述的是“代码默认值”，不是“所有真实联调样本都已补齐后的最终经验上限”。
- 如果后续真实样本继续校准了默认模板，应同时更新代码、测试和本文。

## 基线档位

| 风控档位 | request interval | page size | directory interval | cooldown | retry limit | max concurrent | auto retry window |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `safe` | 1500ms | 100 | 2500ms | 30s | 2 | 1 | 默认空 |
| `balanced` | 800ms | 300 | 1000ms | 15s | 3 | 2 | 默认空 |
| `fast` | 250ms | 1000 | 300ms | 5s | 5 | 4 | 默认空 |
| `custom` | 0 | 0 | 0 | 0 | 0 | 0 | 默认空 |

## 协议族校准

### `quark_uc`

- 适用 provider：`quark`、`uc`
- 默认校准：
  - `requestIntervalMs >= 1400`
  - `pageSize <= 120`
  - `directoryIntervalMs >= 2200`
  - `cooldownSeconds >= 40`
  - `maxConcurrent <= 1`
- 默认意图：
  - 列表和目录推进更保守
  - 后台补传和恢复预算更偏单账号单轮稳态

### `xunlei_pikpak`

- 适用 provider：`xunlei`、`pikpak`
- 默认校准：
  - `requestIntervalMs >= 700`
  - `pageSize <= 250`
  - `directoryIntervalMs >= 1000`
  - `maxConcurrent <= 2`
- 默认意图：
  - 保留较快节奏
  - 同时压住分页和目录切换频率

### `aliyun_123_open`

- 适用 provider：`aliyundrive_open`、`123_open`
- 默认校准：
  - `pageSize <= 500`
  - `maxConcurrent <= 3`
- 默认意图：
  - 开放接口链路允许更大的分页预算
  - 保留中等并发能力

## Provider 默认模板

| Provider | Protocol Group | 默认档位 | 推荐档位 | request interval | page size | directory interval | cooldown | retry limit | max concurrent | auto retry window | risk keywords |
| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `aliyundrive_open` | `aliyun_123_open` | `balanced` | `balanced` | 800ms | 300 | 1000ms | 15s | 3 | 2 | 默认空 | `429`, `too_many_requests`, `flow_limit` |
| `123_open` | `aliyun_123_open` | `balanced` | `balanced` | 800ms | 300 | 1000ms | 15s | 3 | 2 | 默认空 | `429`, `too_many_requests`, `flow_limit` |
| `quark` | `quark_uc` | `balanced` | `safe` | 1400ms | 120 | 2200ms | 40s | 3 | 1 | 默认空 | `risk_control`, `captcha`, `forbidden` |
| `uc` | `quark_uc` | `balanced` | `safe` | 1400ms | 120 | 2200ms | 40s | 3 | 1 | 默认空 | `risk_control`, `captcha`, `forbidden` |
| `xunlei` | `xunlei_pikpak` | `balanced` | `balanced` | 800ms | 250 | 1000ms | 15s | 3 | 2 | 默认空 | `frequency_limit`, `risk_detected`, `forbidden` |
| `pikpak` | `xunlei_pikpak` | `balanced` | `balanced` | 800ms | 250 | 1000ms | 15s | 3 | 2 | 默认空 | `frequency_limit`, `risk_detected`, `forbidden` |
| `baidu_netdisk` | `baidu_netdisk` | `balanced` | `safe` | 1800ms | 100 | 3000ms | 45s | 2 | 1 | 默认空 | `hit_risk_control`, `captcha`, `too_many_requests` |
| `189cloud` | `189cloud` | `balanced` | `safe` | 1200ms | 150 | 2000ms | 35s | 3 | 1 | 默认空 | `rate_limit`, `token_expired`, `too_many_requests` |
| `115_open` | `115_open` | `balanced` | `safe` | 1000ms | 200 | 1800ms | 30s | 3 | 1 | 默认空 | `rate_limit`, `too_many_requests` |
| `guangya` | `guangya` | `balanced` | `safe` | 900ms | 180 | 1600ms | 25s | 3 | 1 | 默认空 | `rate_limit`, `too_many_requests` |

## 当前边界

- 当前默认模板已经覆盖：
  - `requestIntervalMs`
  - `pageSize`
  - `directoryIntervalMs`
  - `cooldownSeconds`
  - `retryLimit`
  - `maxConcurrent`
  - `riskKeywords`
- 当前默认模板还没有做协议族级内建默认值的字段：
  - `autoRetryStartHour`
  - `autoRetryEndHour`
- 这两个字段目前仍主要通过：
  - auth profile 默认覆盖
  - 任务级 `riskOverride`
  - 后续真实联调样本校准

## 对二期主线三的意义

- 这份清单把“代码里已经有的默认模板”从隐式实现变成了显式资产。
- 后续继续补真实样本时，可以直接对照本文检查：
  - 哪些 provider 已经形成了稳定默认节流建议
  - 哪些字段仍然没有被真实样本充分校准
  - 哪些模板变化需要同步更新控制台推荐语义与默认预算
