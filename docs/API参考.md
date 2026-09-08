# Bee Go SDK API 参考

## 调用方式

每个消息或事件回调都应使用本次回调的机器人上下文创建 SDK：

```go
bee, err := NewBeeAPI(robotJSON)
if err != nil {
    return MessageContinue
}

_, _ = bee.Friend(friendID).SendText("你好哦")
_, _ = bee.Group(groupID).SendText("大家好")
_, _ = bee.Channel(channelID).SendImage("图片", imageURL)
_, _ = bee.ChannelDM(guildID).SendText("频道私信")
```

`BeeAPI` 包含当前回调的 `plugin_id`、`msg_id` 和 `event_id`，禁止全局缓存后跨消息使用。

## 初始化协议

Go SDK 提供与易语言 SDK 初始化段对齐的类型和函数：

```go
info := PluginInfo{
    Name:        "测试插件",
    Author:      "周星星",
    Version:     "1.0",
    Description: "这是一个测试插件\n欢迎使用",
}

metadataJSON, err := InitializeBeePlugin(robotJSON, info)
```

`PluginInfo` 序列化为 Bee 需要的 `name`、`author`、`ver`、`text` JSON 字段。`InitializeBeePlugin` 会解析初始化机器人 JSON 中的 `api` 和 `plugin_id`，并且只在第一次调用时返回元数据 JSON；后续调用返回空字符串。当前模板的 `Bee_初始化` 返回值仍由构建工具根据 `plugin_main.go` 常量生成，普通插件业务通常只需要修改这些常量。

## 插件应用数据目录

使用 SDK 获取当前插件专属的数据目录：

```go
dataDir, err := bee.GetAppDataDir()
if err != nil {
    return
}

configFile := filepath.Join(dataDir, "config.json")
```

生产环境返回：

```text
Bee框架根目录\plugin_data\插件名称
```

该方法根据 Worker 可执行文件所在目录计算，不依赖可能发生变化的当前工作目录。插件配置、数据库、缓存和文件日志都应写入此目录，不要写到 `plugin`、`temp_plugin` 或 DLL 所在目录。

## 协议规则

- 分隔符：`%@#bee#@%`
- 操作码 31、45、48 不携带 `plugin_id`；其余操作码自动携带。
- 机器人 JSON 中的 `api` 字段兼容字符串和数字两种形态。
- 主动消息清空 `msg_id` 和 `event_id`。
- 参数中不能包含协议分隔符。
- C 壳在 Bee 进程内调用 `robot.api`；Go worker 不直接使用函数地址。
- Bee 入站 GBK 参数由 C 壳 Base64 封装，worker 解码为 UTF-8。
- Go 出站 UTF-8 命令由 C 壳转换为 GBK；GBK 不可表示字符由 SDK编码为 UTF-16 `\uXXXX`。

## Markdown 辅助函数

```go
at, _ := bee.At("user-openid")
everyone, _ := bee.AtEveryone()
input, _ := bee.InlineCommandInputText("帮助", "/help")
send, _ := bee.InlineCommand("发送", "/send")
mentionedID, _ := bee.MentionedUserID(message)
```

`BeeAPI` 和 `RobotContext` 的同名方法分别调用 98、99、100、101、102、103、104 号 Bee API。`InlineCommandInputText` 仅用于 Markdown 中嵌入聊天框输入指令，用户点击后会把指令填入输入框但不会自动发送；`InlineCommand` 会在点击后直接发送文字。`MentionedUserID` 通过 102 号 API 从消息中获取被艾特人 ID。

## 1～104 操作码

| 操作码 | Go 常量 | 功能 |
|---:|---|---|
| 1 | `OpLog` | 输出日志 |
| 2 | `OpSendChannelMessage` | 发送频道消息 |
| 3 | `OpSendChannelDM` | 发送频道私信 |
| 4 | `OpListGuilds` | 取频道列表 |
| 5 | `OpGetGuild` | 取频道详细信息 |
| 6 | `OpListChannels` | 取子频道列表 |
| 7 | `OpGetChannel` | 取子频道详细信息 |
| 8 | `OpCreateChannel` | 创建子频道 |
| 9 | `OpGetChannelOnlineCount` | 取子频道在线人数 |
| 10 | `OpGetGuildMember` | 取频道成员详细 |
| 11 | `OpUpdateChannel` | 修改子频道 |
| 12 | `OpDeleteChannel` | 删除子频道 |
| 13 | `OpDeleteGuildMember` | 删除频道成员 |
| 14 | `OpListGuildRoles` | 取频道身份组列表 |
| 15 | `OpCreateGuildRole` | 创建频道身份组 |
| 16 | `OpUpdateGuildRole` | 修改频道身份组 |
| 17 | `OpIsGuildOwner` | 取是否频道主 |
| 18 | `OpIsGuildAdmin` | 取是否频道管理员 |
| 19 | `OpIsChannelAdmin` | 取是否子频道管理员 |
| 20 | `OpHasGuildRole` | 取是否指定频道身份组 |
| 21 | `OpDeleteGuildRole` | 删除频道身份组 |
| 22 | `OpAddGuildMemberRole` | 添加频道成员到身份组 |
| 23 | `OpRemoveGuildMemberRole` | 从身份组删除频道成员 |
| 24 | `OpRecallChannelMessage` | 撤回频道消息 |
| 25 | `OpSendChannelReply` | 发送频道引用消息 |
| 26 | `OpSendChannelTextCard` | 发送频道文字卡片 |
| 27 | `OpSendChannelCustom` | 发送频道自定义消息 |
| 28 | `OpSendChannelLargeCard` | 发送频道大图卡片 |
| 29 | `OpMuteGuildMember` | 频道成员禁言 |
| 30 | `OpMuteGuild` | 频道全员禁言 |
| 31 | `OpGetRobotID` | 取机器人 ID |
| 32 | `OpGetRobotInfo` | 取机器人信息 |
| 33 | `OpSendAdaptiveMessage` | 发送自适应消息 |
| 34 | `OpSendGroupMessage` | 发送群消息 |
| 35 | `OpSendGroupVideo` | 发送群视频 |
| 36 | `OpSendGroupAudio` | 发送群语音 |
| 37 | `OpGetFrameworkInfo` | 取框架信息 |
| 38 | `OpSendGroupMarkdown` | 发送群 Markdown |
| 39 | `OpSendGroupTextCard` | 发送群文字卡片 |
| 40 | `OpGetQQNickname` | 取某人昵称 |
| 41 | `OpSendGroupLargeCard` | 发送群大图卡片 |
| 42 | `OpSendAdaptiveLargeCard` | 发送自适应大图卡片 |
| 43 | `OpSendGroupThumbnailCard` | 发送群缩略图卡片 |
| 44 | `OpSendChannelThumbnailCard` | 发送频道缩略图卡片 |
| 45 | `OpUploadImage` | 上传图片到图床 |
| 46 | `OpRespondButton` | 响应按钮事件 |
| 47 | `OpSendChannelMarkdown` | 发送频道 Markdown |
| 48 | `OpGetRobotAppID` | 取机器人 AppID |
| 49 | `OpGetAvatar` | 取用户头像 |
| 50 | `OpGetQQAvatar` | 取 QQ 头像 |
| 51 | `OpRecallGroupMessage` | 撤回群消息 |
| 52 | `OpSendFriendMessage` | 发送好友消息 |
| 53 | `OpSendFriendVideo` | 发送好友视频 |
| 54 | `OpSendFriendAudio` | 发送好友语音 |
| 55 | `OpSendFriendMarkdown` | 发送好友 Markdown |
| 56 | `OpSendFriendTextCard` | 发送好友文字卡片 |
| 57 | `OpSendFriendLargeCard` | 发送好友大图卡片 |
| 58 | `OpSendFriendThumbnailCard` | 发送好友缩略图卡片 |
| 59 | `OpRecallFriendMessage` | 撤回好友消息 |
| 60 | `OpSendAdaptivePrivateMessage` | 发送自适应私信消息 |
| 61 | `OpAddChannelReaction` | 添加频道表情表态 |
| 62 | `OpDeleteChannelReaction` | 删除频道表情表态 |
| 63 | `OpListChannelReactionUsers` | 取表情表态用户列表 |
| 64 | `OpGetRobotStats` | 取机器人统计信息 |
| 65 | `OpSendGroupButton` | 发送群按钮消息 |
| 66 | `OpSendFriendButton` | 发送好友按钮消息 |
| 67 | `OpGetRobotToken` | 取机器人 Token |
| 68 | `OpGetRobotSecret` | 取机器人密钥 |
| 69 | `OpSendGroupFile` | 发送群文件 |
| 70 | `OpSendFriendFile` | 发送好友文件 |
| 71 | `OpSendGroupReply` | 发送群引用消息 |
| 72 | `OpSendFriendReply` | 发送好友引用消息 |
| 73 | `OpGetRobotShareLink` | 取机器人分享链接 |
| 74 | `OpGetGroupBasicInfo` | 取群基本信息 |
| 75 | `OpGetRobotGroupStatus` | 取机器人群内状态 |
| 76 | `OpListGroupJoinRequests` | 取入群申请列表 |
| 77 | `OpHandleGroupJoinRequest` | 处理入群请求 |
| 78 | `OpGetGroupMuteInfo` | 取群内禁言信息 |
| 79 | `OpIsGroupMemberMuted` | 取群内某人是否被禁言 |
| 80 | `OpIsGroupMuted` | 取群内是否全员禁言中 |
| 81 | `OpMuteGroupMembers` | 设置群成员禁言 |
| 82 | `OpIsGroupManagement` | 取是否为群管理高层 |
| 83 | `OpListGroupAutoApprovalStrategies` | 查询入群自动审批策略列表 |
| 84 | `OpCreateGroupAutoApprovalStrategy` | 创建入群自动审批策略 |
| 85 | `OpUpdateGroupAutoApprovalStrategy` | 修改入群自动审批策略 |
| 86 | `OpDeleteGroupAutoApprovalStrategy` | 删除入群自动审批策略 |
| 87 | `OpExecuteGroupAutoApprovalStrategy` | 执行入群自动审批策略 |
| 88 | `OpEditGroupAutoApprovalStrategyWhitelist` | 编辑入群自动审批策略白名单 |
| 89 | `OpGetGlobalCustomMenu` | 查询全局自定义菜单 |
| 90 | `OpEditGlobalCustomMenu` | 编辑全局自定义菜单 |
| 91 | `OpListCommandPanels` | 查询指令面板列表 |
| 92 | `OpGetCommandPanel` | 查询指令面板详细 |
| 93 | `OpCreateCommandPanel` | 创建指令面板 |
| 94 | `OpUpdateCommandPanel` | 修改指令面板 |
| 95 | `OpDeleteCommandPanel` | 删除指令面板 |
| 96 | `OpEditCommandPanelTargets` | 编辑指令面板关联对象 |
| 97 | `OpGetGroupMuteInfoEx` | 取群内禁言信息 Ex |
| 98 | `OpAt` | 艾特指定用户 |
| 99 | `OpAtEveryone` | 艾特全体成员 |
| 100 | `OpInlineCommandInput` | 生成点击后填入聊天框的 Markdown 指令 |
| 101 | `OpInlineCommandSend` | 生成点击后直接发送的 Markdown 指令 |
| 102 | `OpMentionedUserID` | 获取消息中的被艾特人 ID |
| 103 | `OpIsQuotedMessage` | 判断当前消息是否为引用回复 |
| 104 | `OpQuotedMessageContent` | 获取被引用消息的原内容 |

## 引用消息 API

引用消息接口需要使用当前回调的上下文：

```go
isQuoted, _ := bee.IsQuotedMessage(robotID)
if isQuoted {
    original, _ := bee.QuotedMessageContent(robotID)
    _ = original
}
```

`robotID` 是机器人 ID。非引用消息调用 `QuotedMessageContent` 时，框架返回空字符串。

## 新增群管理 API

`BeeAPI` 和 `RobotContext` 均提供以下方法：

| 方法 | 返回 | 说明 |
|---|---|---|
| `GetRobotShareLink()` | `string` | 返回机器人分享链接，用于邀请用户添加机器人为好友 |
| `GetGroupBasicInfo(groupID string)` | `GroupBasicInfo` | 返回群 ID、名称、简介、分类、标签和成员数 |
| `GetRobotGroupStatus(groupID string)` | `RobotGroupStatus` | 返回机器人在群内的身份、入群时间和消息接收状态 |
| `ListGroupJoinRequests(groupID string)` | `[]GroupJoinRequest` | 返回入群申请列表，需管理员身份调用 |
| `HandleGroupJoinRequest(groupID, userID, requestID string, action int, rejectReason string, blockUser bool)` | `error` | 处理入群请求，`action` 为 0 同意、1 拒绝 |
| `GetGroupMuteInfo(groupID string)` | `GroupMuteInfo` | 返回全员禁言状态和被禁言成员列表，需管理员身份调用 |
| `GetGroupMuteInfoEx(groupID string)` | `[]GroupMuteMemberInfoEx` | 返回被禁言成员 ID、昵称、统一 ID 和禁言结束时间，需管理员身份调用 |
| `IsGroupMemberMuted(groupID, userID string)` | `bool` | 查询某人是否被禁言，需管理员身份调用 |
| `IsGroupMuted(groupID string)` | `bool` | 查询全员禁言是否开启，需管理员身份调用 |
| `MuteGroupMembers(groupID, userIDs string, seconds int)` | `error` | 设置成员禁言，`seconds` 为 0 表示解除禁言；多个用户 ID 用换行分隔 |
| `IsGroupManagement(groupID string)` | `bool` | 查询机器人是否为管理员或群主 |

相关结构体：

- `GroupBasicInfo`
- `RobotGroupStatus`
- `GroupJoinRequest`
- `GroupMuteInfo`
- `GroupMuteMemberInfoEx`

## 昵称 API

`RobotContext` 提供 `GetQQNickname(qqOrUserID string)` 和 `GetUserNickname(userIDOrQQ string)`；`BeeAPI` 提供 `GetUserNickname(userIDOrQQ string)`。二者均调用 40 号操作码，可传 QQ 或用户 ID，失败时框架返回空字符串。

## 入群自动审批策略 API

`BeeAPI` 和 `RobotContext` 均提供以下方法：

| 方法 | 返回 | 说明 |
|---|---|---|
| `ListGroupAutoApprovalStrategies()` | `[]GroupAutoApprovalStrategy` | 查询当前机器人的策略列表，按创建时间倒序 |
| `CreateGroupAutoApprovalStrategy(groupOpenIDs, groupIDs string, enabled bool, expireAt, remark string)` | `string` | 创建策略，成功返回策略 ID；群 OpenID 和群号二选一，多个值用换行分隔 |
| `UpdateGroupAutoApprovalStrategy(strategyID string, editType int, groupOpenIDs, groupIDs string, enabled bool, expireAt, remark string)` | `string` | 修改策略，成功返回过期时间；`editType` 为 0 增加、1 删除 |
| `DeleteGroupAutoApprovalStrategy(strategyID string)` | `error` | 删除策略 |
| `ExecuteGroupAutoApprovalStrategy(strategyID string)` | `error` | 对策略关联的全部群执行自动审批，需机器人是管理员 |
| `EditGroupAutoApprovalStrategyWhitelist(strategyID string, editType int, qq string)` | `string` | 编辑白名单，成功返回更新时间；`editType` 为 0 增加、1 删除，多个 QQ 用换行分隔 |

相关结构体：

- `GroupAutoApprovalStrategy`

## 自定义菜单和指令面板 API

`BeeAPI` 和 `RobotContext` 均提供以下方法：

| 方法 | 返回 | 说明 |
|---|---|---|
| `GetGlobalCustomMenu()` | `string` | 查询全局自定义菜单配置，返回 JSON 字符串 |
| `EditGlobalCustomMenu(menuJSON string)` | `error` | 编辑全局自定义菜单；`menuJSON` 为空表示删除菜单 |
| `ListCommandPanels(scene int)` | `string` | 根据场景查询指令面板列表，返回 JSON 字符串；`scene` 为 0 QQ好友、1 群聊、2 文字频道、3 频道私信 |
| `GetCommandPanel(panelID string)` | `string` | 查询指令面板详细，返回 JSON 字符串 |
| `CreateCommandPanel(scene, targetType int, panelConfigJSON, userIDs, groupOpenIDs string)` | `string` | 创建指令面板，成功返回面板 ID；`targetType` 为 0 所有人、1 指定范围 |
| `UpdateCommandPanel(panelID, panelConfigJSON string)` | `string` | 修改指定指令面板配置，成功返回版本号 |
| `DeleteCommandPanel(panelID string)` | `error` | 删除指定指令面板 |
| `EditCommandPanelTargets(panelID string, editType int, userIDs, groupOpenIDs string)` | `error` | 编辑指令面板关联用户或群；`editType` 为 0 增加、1 删除 |

各方法完整参数签名和类型定义直接查看根目录 `bee_sdk.go`。该文件是 SDK 的单一实现源，不拆分为大量小文件。
