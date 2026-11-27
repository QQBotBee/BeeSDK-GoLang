# Bee Go SDK：AI 开发指南

## 架构边界

```text
Bee.exe
  └─ 纯 C BeeGoPlugin.dll
       ├─ 11 个 stdcall 中文导出
       ├─ worker 生命周期
       ├─ GBK/UTF-8 边界
       └─ robot.api 进程内调用

bee_go_worker.exe
  ├─ plugin_main.go 业务回调
  ├─ bee_sdk.go 完整 1～111 API
  └─ JSON Lines IPC
```

禁止把 Go runtime、cgo 或 `buildmode=c-archive` 放回 Bee DLL。

## 业务代码位置

主要修改 `plugin_main.go` 中的回调：

- `onChannelPrivate`
- `onChannelMessage`
- `onChannelEvent`
- `onPrivateMessage`
- `onGroupMessage`
- `onCommonEvent`

每次回调都必须使用当前机器人 JSON 创建新的 `BeeAPI`。不要缓存上一条消息的上下文，因为 `msg_id`、`event_id`、`plugin_id` 属于单次回调。

## 常用 API

```go
bee, err := beeFromArgs(args)
if err != nil {
    return MessageContinue
}

_, _ = bee.Friend(friendID).SendText("你好哦")
_, _ = bee.Group(groupID).SendImage("说明", imageURL)
_, _ = bee.Channel(channelID).Reply(messageID, "收到", "")
_, _ = bee.ChannelDM(guildID).SendText("私信回复")
```

复杂参数使用 `bee.ctx` 上的完整方法；完整操作码见 `API参考.md`。

群消息发送者与机器人为好友时，在本次回调中用 `bee.Friend(userID).SendText(...)` 被动私聊回复。 [群消息私聊回复](API参考.md#群消息私聊回复)

## Markdown 发送与回执

发送好友和群 Markdown 时，分别优先用 `SendFriendMarkdownEx` 和 `SendGroupMarkdownEx`。有无按钮都优先用 Ex，无按钮时 `keyboardJSON` 传 `""`。频道 Markdown 按场景选择 `SendChannelMarkdown` 或 `SendChannelMarkdownEx`。

`Send*MarkdownEx` 的 `err` 表示 IPC 错误，`result` 是消息 ID 或失败原因。成功回执可形如 `ROBOT1.0_…!!`，完整 ID 原样保留，避免用固定格式校验。IPC 出错或回执无法识别时按发送状态未确认处理，记录原始结果，避免直接重发。只有框架明确返回权限错误时才提示检查权限。 [API 参考](API参考.md#发送-markdown-ex)

## Markdown 图片显示尺寸

为 Bee 生成原生 Markdown 正文时，用 `![#宽px #高px](图片地址)` 指定图片的显示宽高。宽在前、高在后，中间用一个空格分隔，单位为 `px`。例如 `![#320px #180px](https://example.com/image.png)` 指定图片显示为 320 × 180 像素。

```go
content := "![#320px #180px](" + imageURL + ")"
result, err := bee.SendGroupMarkdownEx(groupID, content, "", false)
// 按上面的回执规则处理 result 和 err。
```

调用 `Send*MarkdownEx` 时把正文传给 `content`。使用 `RobotContext` 的 `Send*Markdown` 方法时，把正文填入 `MarkdownMessage.Native`。 [API 参考](API参考.md#发送-markdown-ex)

## Markdown 按钮

用 `BuildKeyboard("", rows)` 生成 `keyboardJSON`。回调按钮用 `Type: 1`，触发 `INTERACTION_CREATE`。指令按钮用 `Type: 2`。回调分组用 `Button.GroupID`，二次确认用 `Button.Modal`。

群和好友在 `onCommonEvent` 收回调，频道在 `onChannelEvent`。校验 `button_data`、会话和点击者，用本次点击的 `event_id` 调用 `bee.RespondButton(0)`，再执行业务。群事件可能由 Bee 用文本包装，点击事件 ID 在 JSON 外时用 `bee.ctx.RespondButton(eventID, 0)`。有副作用的操作要防重复执行。 [按钮发送与回调示例](API参考.md#按钮发送与回调示例)

### 按钮点击后的被动回复

用户发来消息 A，插件回复带按钮的消息 B。用户点击 B 上的按钮后，如果要基于原消息 A 发送聊天回复，被动回复的 `msg_id` 必须使用 A 的 ID。

1. 在处理消息 A 时，从消息回调的 `messageID` 参数取得原消息 ID，把它和动作标识一起写入 `Button.Data`，例如 `"demo:refresh:" + messageID`。`BuildKeyboard` 会把该值写进 `action.data`。
2. 在按钮回调中读取 `button_data`。校验动作、会话和点击者后，取出完整的原消息 ID。缺少原消息 ID 时停止基于 A 的回复。发送 B 的回执和点击载荷里的卡片消息 ID 都属于 B，仅用于对 B 进行引用、撤回等操作。
3. 用本次点击的 `event_id` 应答按钮。发送聊天回复时，复制本次点击创建的 `bee.ctx`，将副本的 `MessageID` 设为 A 的 ID，清空副本的 `EventID`，再以 `active=false` 发送。`Reply` 的消息 ID 参数用于指定被引用的消息，被动发送使用的是上下文的 `MessageID`。 [按钮发送与回调示例](API参考.md#按钮发送与回调示例)

### 连续按钮回调

用户发「我的庄园」（A），插件用 A 的回调上下文发送首张卡片 B。B 上的「查看田地」按钮在 `Button.Data` 中携带动作和 A 的 `msg_id`。

- 点击 B 时，先用本次点击的 `event_id` 应答，再用同一个 `event_id` 被动发送带「返回庄园」按钮的卡片 C。发送前复制当前 `bee.ctx`，明确设置 `MessageID = ""` 和 `EventID = eventID`，调用 `SendGroupMarkdownEx` 时传 `active=false`。
- 点击 C 时，使用这次点击创建的新 `BeeAPI`。用 C 上这次点击的新 `event_id` 应答并发下一张卡片，发送时仍将 `msg_id` 置空。每一轮都只用本轮点击的事件 ID 作为被动发送凭据。

Bee 文本包装里的点击 ID 也适用。校验并取出外层的 `eventID` 后，将同一个 ID 用于应答和发卡。缺少本次点击 ID 时停止这轮发卡。 [连续按钮回调示例](API参考.md#连续按钮回调示例)

## 转发框架文本

Bee 回调或 API 返回的文本可能同时包含普通中文和字面的 `\uXXXX`，例如 `\uD83D\uDE04你好`。再次发送时，直接将收到的字符串传给 SDK，保留其中的反斜杠和原有字符。Bee 会在处理消息时解析这些序列。手动反转义（如调用 `strconv.Unquote`）、替换转义序列或额外添加反斜杠，都可能导致框架转换失败并出现乱码。

```go
// message 是 Bee 回调收到的消息正文。
_, _ = bee.Group(groupID).SendText(message)
```

同样的规则适用于引用回复、Markdown 和卡片文案。需要在 Go 源码里构造示例时，可以用原始字符串字面量 `` `\uD83D\uDE04你好` ``，使字符串值保留反斜杠。

如果回调载荷是 JSON，按 JSON 结构读取字段后，保留字段值中仍然存在的 `\uXXXX`。构造按钮 JSON 时使用 `BuildKeyboard`，构造自定义消息 JSON 时可使用 `json.Marshal`。把原始字段值直接交给序列化函数，由它处理 JSON 语法所需的转义。

## IPC 纪律

1. worker stdout 只能输出 JSON Lines IPC，日志写 stderr 或插件数据目录。
2. worker 发出 `api_call` 后同步等待匹配 ID 的 `api_result`。
3. C 壳等待 `event_result` 时必须处理中途出现的 `api_call`，否则同步 SDK 调用会死锁。
4. Go 进程不得直接调用机器人 JSON 中的 `api` 地址；函数地址只在 Bee 进程有效。
5. 参数不能包含 `%@#bee#@%`，原 Bee 协议没有转义机制。
6. 机器人 JSON 中的 `api` 字段可能是数字或字符串，Go SDK 和 C 壳都必须兼容。

## 生命周期

- `onInitialize` 是 Go 业务初始化入口，由 C 壳在 `Bee_初始化` 返回插件信息之前调用一次；适合读取配置和准备基础资源，不做耗时业务或长期任务。
- `PluginInfo` 和 `InitializeBeePlugin` 与易语言 SDK 的初始化协议对齐，负责 `name`、`author`、`ver`、`text` 元数据和一次性初始化语义。
- `pluginMetadata` 只描述插件名称、作者、版本和说明；当前模板由构建工具读取这些信息生成 `Bee_初始化` 返回值，不是运行期初始化入口。
- 启用时启动任务。
- 禁用时关闭任务和设置窗口。
- 卸载时完成最终清理。
- worker 还会在宿主 PID 消失或 stdin EOF 时退出。

## 验证边界

WSL 可完成 Go 测试、Windows/386 交叉编译和 PE 静态检查；真实 API 调用、重复加载卸载和消息收发必须在 Windows Bee 中验收。
