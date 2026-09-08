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
  ├─ bee_sdk.go 完整 1～104 中已定义 API
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
