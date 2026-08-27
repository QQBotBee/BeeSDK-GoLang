package main

import (
	"errors"
)

const (
	PluginName        = "GO测试插件"
	PluginAuthor      = "周星星"
	PluginVersion     = "1.0"
	PluginDescription = "Bee C shell + Go worker template"
)

// pluginMetadata 只返回插件名称、作者、版本和说明，不是运行时初始化入口。
// 构建工具会读取上方常量生成 Bee_初始化 的返回信息。
func pluginMetadata() PluginInfo {
	return PluginInfo{Name: PluginName, Author: PluginAuthor, Version: PluginVersion, Description: PluginDescription}
}

// 以下是插件业务回调模板。
// 依赖当前消息或事件上下文的 API（例如回复、撤回、按钮响应）必须使用本次回调新建的 BeeAPI，
// 避免复用已经失效的 msg_id、event_id 等数据。
// 不依赖消息或事件上下文的框架级 API（例如输出日志、取框架信息）可以复用有效的 API 对象。

// beeFromArgs 从回调参数的第一项创建 BeeAPI；消息和事件相关操作应使用本次回调创建的对象。
func beeFromArgs(args [][]byte) (*BeeAPI, error) {
	if len(args) == 0 {
		return nil, errors.New("callback missing robot context")
	}
	return NewBeeAPI(string(args[0]))
}

// onInitialize 是插件真正的初始化入口，在 Bee_初始化返回插件信息之前调用一次。
// 适合创建数据目录、读取配置和准备基础资源；不要在这里启动仅启用期间运行的长期任务。
func onInitialize(args [][]byte) {
	bee, err := beeFromArgs(args)
	if err != nil {
		return
	}
	_ = bee.Log("插件初始化完成")
	// dataDir, err := bee.GetAppDataDir()
	// 可在 dataDir 中创建配置、数据库、缓存和日志文件。
}

// onEnable 在插件被启用时调用，可在这里启动插件任务或初始化运行状态。
func onEnable(args [][]byte) {
	bee, err := beeFromArgs(args)
	if err == nil {
		_ = bee.Log("插件被启用")
		// dataDir, err := bee.GetAppDataDir()
		// 应用配置、数据库、缓存和日志统一写入 dataDir。
	}
}

// onDisable 在插件被禁用时调用，负责停止任务并关闭设置窗口等运行资源。
func onDisable(args [][]byte) {
	bee, err := beeFromArgs(args)
	if err == nil {
		_ = bee.Log("插件被禁用")
	}
	closeSettingsWindow()
}

// onUnload 在插件被卸载时调用，负责最终清理插件资源。
// robotJSON 是卸载回调传入的机器人上下文，可用于输出日志等框架级操作。
// Bee 只允许禁用后卸载，设置窗口已在 onDisable 中关闭，这里不重复处理窗口。
func onUnload(args [][]byte) {
	bee, err := beeFromArgs(args)
	if err == nil {
		_ = bee.Log("插件被卸载")
	}
}

// onSettings 在用户点击“设置”时调用，用于打开或聚焦插件设置窗口。
func onSettings(args [][]byte) {
	bee, err := beeFromArgs(args)
	if err == nil {
		_ = bee.Log("插件设置被打开")
	}
	showSettingsWindow()
}

// onChannelPrivate 处理机器人收到的频道私信消息。
// robotJSON 是当前机器人上下文；messageID 用于撤回、引用等上下文相关 API。
// 返回 MessageContinue 表示继续后续处理；返回 MessageIntercept 表示拦截该消息。
func onChannelPrivate(
	robotJSON string, // 当前机器人上下文 JSON，用于创建本次消息的 BeeAPI
	channelID string, // 来源频道 ID
	subChannelID string, // 来源子频道 ID
	userID string, // 触发人 ID，即频道私信发送人
	message string, // 收到的频道私信内容
	messageID string, // 消息 ID，用于撤回、引用等上下文相关 API
) int {
	bee, err := NewBeeAPI(robotJSON)
	if err != nil {
		return MessageContinue
	}

	_ = bee.Log("收到频道私信消息")

	// 示例：向当前频道私信会话发送消息。
	// _, _ = bee.ChannelDM(channelID).SendText("你好哦")
	//
	// 这些具名参数可直接用于业务判断、回复、引用或撤回操作。
	_, _, _, _, _ = channelID, subChannelID, userID, message, messageID

	return MessageContinue
}

// onChannelMessage 处理机器人收到的频道消息。
// robotJSON 是当前机器人上下文；messageID 用于撤回、引用等上下文相关 API。
// 返回 MessageContinue 表示继续后续处理；返回 MessageIntercept 表示拦截该消息。
func onChannelMessage(
	robotJSON string, // 当前机器人上下文 JSON，用于创建本次消息的 BeeAPI
	channelID string, // 来源频道 ID
	subChannelID string, // 来源具体子频道 ID
	userID string, // 触发人 ID，即频道消息发送人
	message string, // 收到的频道消息内容
	messageID string, // 消息 ID，用于撤回、引用等上下文相关 API
) int {
	bee, err := NewBeeAPI(robotJSON)
	if err != nil {
		return MessageContinue
	}

	_ = bee.Log("收到频道消息")

	// 示例：向消息来源子频道发送消息。
	// _, _ = bee.Channel(subChannelID).SendText("你好哦")
	//
	// 这些具名参数可直接用于业务判断、回复、引用或撤回操作。
	_, _, _, _, _ = channelID, subChannelID, userID, message, messageID

	return MessageContinue
}

// onChannelEvent 处理机器人收到的频道事件。
// operatorID 是执行操作的用户，例如“触发人被操作人禁言”；rawMessage 是事件原始内容。
// 返回 MessageContinue 表示继续后续处理；返回 MessageIntercept 表示拦截该事件。
func onChannelEvent(
	robotJSON string, // 当前机器人上下文 JSON，用于创建本次事件的 BeeAPI
	channelID string, // 事件所属频道 ID
	subChannelID string, // 事件所属子频道 ID
	userID string, // 触发人 ID，即这次事件作用或关联的用户
	operatorID string, // 操作人 ID，例如执行禁言、移除等操作的用户
	eventType string, // 事件类型，对应 bee_sdk.go 中的频道 EventType 常量
	rawMessage string, // 事件原始内容，保留 Bee 框架传入的完整事件数据
) int {
	bee, err := NewBeeAPI(robotJSON)
	if err != nil {
		return MessageContinue
	}

	_ = bee.Log("收到频道事件")

	switch EventType(eventType) {
	case EventGuildMemberRemove:
		// 当成员被移除时

	case EventMessageDelete:
		// 有人撤回消息时

	case EventChannelCreate:
		// 当子频道被创建时

	case EventChannelUpdate:
		// 当子频道被更新时

	case EventChannelDelete:
		// 当子频道被删除时

	case EventGuildMemberAdd:
		// 当有人加入频道时

	case EventGuildMemberUpdate:
		// 当成员资料变更时

	case EventForumThreadCreate:
		// 当有人创建主题时

	case EventForumThreadUpdate:
		// 当有人更新主题时

	case EventForumThreadDelete:
		// 当有人删除主题时

	case EventGuildDelete:
		// 当机器人退出频道时

	case EventMessageReactionAdd:
		// 为消息添加表情表态

	case EventMessageReactionRemove:
		// 为消息删除表情表态

	case EventGuildCreate:
		// 当机器人加入新频道时

	case EventGuildUpdate:
		// 当频道资料发生变更时

	case EventAudioLiveEnter:
		// 当有人进入音视直播子频道时

	case EventAudioLiveExit:
		// 当有人离开音视直播子频道时

	case EventForumPostCreate:
		// 当有人创建帖子时

	case EventForumReplyDelete:
		// 当有人删除评论时

	case EventForumPostDelete:
		// 当有人删除帖子时

	case EventForumReplyCreate:
		// 当有人回复帖子时

	case EventInteractionCreate:
		// 按钮事件
		// _, _ = bee.RespondButton(0)

	default:
		// 其他或后续新增的频道事件，可在这里统一处理。
	}

	// 这些具名参数可直接用于判断事件来源、触发人、操作人和原始内容。
	_, _, _, _, _ = channelID, subChannelID, userID, operatorID, rawMessage

	return MessageContinue
}

// onPrivateMessage 处理机器人收到的好友私聊消息。
// robotJSON 是当前机器人上下文；messageID 用于撤回、引用等上下文相关 API。
// 返回 MessageContinue 表示继续后续处理；返回 MessageIntercept 表示拦截该消息。
func onPrivateMessage(
	robotJSON string, // 当前机器人上下文 JSON，用于创建本次消息的 BeeAPI
	friendID string, // 好友 ID，即私聊消息发送人
	message string, // 收到的私聊消息内容
	messageID string, // 消息 ID，用于撤回、引用等上下文相关 API
) int {
	bee, err := NewBeeAPI(robotJSON)
	if err != nil {
		return MessageContinue
	}

	_ = bee.Log("收到好友私聊消息")

	// 示例：向当前好友发送消息。
	// _, _ = bee.Friend(friendID).SendText("你好哦")
	//
	// 这些具名参数可直接用于业务判断、回复、引用或撤回操作。
	_, _, _ = friendID, message, messageID

	return MessageContinue
}

// onGroupMessage 处理机器人收到的群聊消息。
// robotJSON 是当前机器人上下文；messageID 用于撤回、引用等上下文相关 API。
// 返回 MessageContinue 表示继续后续处理；返回 MessageIntercept 表示拦截该消息。
func onGroupMessage(
	robotJSON string, // 当前机器人上下文 JSON，用于创建本次消息的 BeeAPI
	groupID string, // 来源群 ID
	userID string, // 触发人 ID，即群消息发送人
	message string, // 收到的群聊消息内容
	messageID string, // 消息 ID，用于撤回、引用等上下文相关 API
) int {
	bee, err := NewBeeAPI(robotJSON)
	if err != nil {
		return MessageContinue
	}

	_ = bee.Log("收到群消息")

	// 示例：向消息来源群聊发送消息。
	// _, _ = bee.Group(groupID).SendText("大家好")
	//
	// 这些具名参数可直接用于业务判断、回复、引用或撤回操作。
	_, _, _, _ = groupID, userID, message, messageID

	return MessageContinue
}

// onCommonEvent 处理机器人收到的好友、私聊和群聊通常事件。
// sourceID 是群聊 ID 或私聊 ID；operatorID 是执行操作的用户；rawMessage 是事件原始内容。
// 返回 MessageContinue 表示继续后续处理；返回 MessageIntercept 表示拦截该事件。
func onCommonEvent(
	robotJSON string, // 当前机器人上下文 JSON，用于创建本次事件的 BeeAPI
	sourceID string, // 消息来源 ID：群聊事件为群 ID，私聊事件为好友 ID
	userID string, // 触发人 ID，即引发这次事件的用户
	operatorID string, // 操作人 ID，例如执行禁言、移除等操作的用户
	eventType string, // 事件类型，对应 bee_sdk.go 中的 EventType 常量
	rawMessage string, // 事件原始内容，保留 Bee 框架传入的完整事件数据
) int {
	bee, err := NewBeeAPI(robotJSON)
	if err != nil {
		return MessageContinue
	}

	_ = bee.Log("收到通用事件")

	switch EventType(eventType) {
	case EventInteractionCreate:
		// 按钮事件
		// _, _ = bee.RespondButton(0)

	case EventFriendAdd:
		// 用户添加机器人为好友

	case EventFriendDelete:
		// 用户删除机器人好友

	case EventGroupAddRobot:
		// 机器人被添加到群聊

	case EventGroupDelRobot:
		// 机器人被移除群聊

	case EventGroupMessageReceive:
		// 群管理员开启通知，接受机器人主动消息

	case EventGroupMessageReject:
		// 群管理员关闭通知，拒绝机器人主动消息

	case EventC2CMessageReceive:
		// 好友开启通知，接受机器人主动私聊消息

	case EventC2CMessageReject:
		// 好友关闭通知，拒绝机器人主动私聊消息

	case EventGroupMemberAdd:
		// 有人加入群聊

	case EventGroupMemberRemove:
		// 群聊成员被踢出或移除

	case EventGroupJoinRequest:
		// 有人申请加入群聊

	default:
		// 其他或后续新增的通常事件，可在这里统一处理。
	}

	// 这些具名参数可直接用于判断事件来源、触发人、操作人和原始内容。
	_, _, _, _, _ = sourceID, userID, operatorID, eventType, rawMessage

	return MessageContinue
}
