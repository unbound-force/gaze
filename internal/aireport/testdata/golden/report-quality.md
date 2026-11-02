# 🔍 Gaze Quality Report

Project: github.com/example/project · Branch: main
Gaze Version: v2.4.0 · Date: 2026-11-01

---

## 🧪 Quality Summary

| Metric | Value |
|--------|------:|
| Tests analyzed | 8 |
| Average contract coverage | 43.1% |
| Over-specifications | 1 |
| Assertion detection confidence | 92% |
| Skipped tests | 2 |

### Worst Tests by Contract Coverage

| Test | Target | Contract Coverage | Gaps |
|------|--------|------------------:|-----:|
| `TestProcessMessage` | `(*队列处理器).处理消息` | 0.0% | 12 |
| `TestLoadConfig` | `loadConfig` | 0.0% | 2 |
| `TestValidateInput` | `validateInput` | 0.0% | 3 |
| `TestParser_parse\|Pipe` | `(*Parser).parse\|Pipe` | 40.0% | 3 |
| `TestServe` | `(*Handler).ServeHTTP` | 50.0% | 1 |

### Coverage Gaps in Worst Tests

| Target | Effect | Description | Location |
|--------|--------|-------------|----------|
| `(*队列处理器).处理消息` | ReturnValue | returns int at position 0 | internal/队列/处理器.go:50:2 |
| `(*队列处理器).处理消息` | ErrorReturn | returns error at position 1 | internal/队列/处理器.go:55:2 |
| `(*队列处理器).处理消息` | ReceiverMutation | mutates receiver field 已处理 | internal/队列/处理器.go:60:2 |
| `(*队列处理器).处理消息` | ReceiverMutation | mutates receiver field 失败 | internal/队列/处理器.go:65:2 |
| `(*队列处理器).处理消息` | ReceiverMutation | mutates receiver field 重试 | internal/队列/处理器.go:70:2 |
| `(*队列处理器).处理消息` | PointerArgMutation | mutates pointer argument 消息 | internal/队列/处理器.go:75:2 |
| `(*队列处理器).处理消息` | SliceMutation | appends to slice 队列 | internal/队列/处理器.go:80:2 |
| `(*队列处理器).处理消息` | MapMutation | writes map 索引 | internal/队列/处理器.go:85:2 |
| `(*队列处理器).处理消息` | ChannelSend | sends on channel 完成 | internal/队列/处理器.go:90:2 |
| `(*队列处理器).处理消息` | ChannelClose | closes channel 停止 | internal/队列/处理器.go:95:2 |

> ⚠️ SSA analysis degraded for 1 package(s): github.com/example/project/internal/legacy
