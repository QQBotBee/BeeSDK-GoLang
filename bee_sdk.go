package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// ==================== opcodes.go ====================
const (
	OpLog                        = 1
	OpSendChannelMessage         = 2
	OpSendChannelDM              = 3
	OpListGuilds                 = 4
	OpGetGuild                   = 5
	OpListChannels               = 6
	OpGetChannel                 = 7
	OpCreateChannel              = 8
	OpGetChannelOnlineCount      = 9
	OpGetGuildMember             = 10
	OpUpdateChannel              = 11
	OpDeleteChannel              = 12
	OpDeleteGuildMember          = 13
	OpListGuildRoles             = 14
	OpCreateGuildRole            = 15
	OpUpdateGuildRole            = 16
	OpIsGuildOwner               = 17
	OpIsGuildAdmin               = 18
	OpIsChannelAdmin             = 19
	OpHasGuildRole               = 20
	OpDeleteGuildRole            = 21
	OpAddGuildMemberRole         = 22
	OpRemoveGuildMemberRole      = 23
	OpRecallChannelMessage       = 24
	OpSendChannelReply           = 25
	OpSendChannelTextCard        = 26
	OpSendChannelCustom          = 27
	OpSendChannelLargeCard       = 28
	OpMuteGuildMember            = 29
	OpMuteGuild                  = 30
	OpGetRobotID                 = 31
	OpGetRobotInfo               = 32
	OpSendAdaptiveMessage        = 33
	OpSendGroupMessage           = 34
	OpSendGroupVideo             = 35
	OpSendGroupAudio             = 36
	OpGetFrameworkInfo           = 37
	OpSendGroupMarkdown          = 38
	OpSendGroupTextCard          = 39
	OpGetQQNickname              = 40
	OpSendGroupLargeCard         = 41
	OpSendAdaptiveLargeCard      = 42
	OpSendGroupThumbnailCard     = 43
	OpSendChannelThumbnailCard   = 44
	OpUploadImage                = 45
	OpRespondButton              = 46
	OpSendChannelMarkdown        = 47
	OpGetRobotAppID              = 48
	OpGetAvatar                  = 49
	OpGetQQAvatar                = 50
	OpRecallGroupMessage         = 51
	OpSendFriendMessage          = 52
	OpSendFriendVideo            = 53
	OpSendFriendAudio            = 54
	OpSendFriendMarkdown         = 55
	OpSendFriendTextCard         = 56
	OpSendFriendLargeCard        = 57
	OpSendFriendThumbnailCard    = 58
	OpRecallFriendMessage        = 59
	OpSendAdaptivePrivateMessage = 60
	OpAddChannelReaction         = 61
	OpDeleteChannelReaction      = 62
	OpListChannelReactionUsers   = 63
	OpGetRobotStats              = 64
	OpSendGroupButton            = 65
	OpSendFriendButton           = 66
	OpGetRobotToken              = 67
	OpGetRobotSecret             = 68
	OpSendGroupFile              = 69
	OpSendFriendFile             = 70
	OpSendGroupReply             = 71
	OpSendFriendReply            = 72
	OpGetRobotShareLink          = 73
	OpGetGroupBasicInfo          = 74
	OpGetRobotGroupStatus        = 75
	OpListGroupJoinRequests      = 76
	OpHandleGroupJoinRequest     = 77
	OpGetGroupMuteInfo           = 78
	OpIsGroupMemberMuted         = 79
	OpIsGroupMuted               = 80
	OpMuteGroupMembers           = 81
	OpIsGroupManagement          = 82
)

// ==================== types.go ====================
// EventType 表示 Bee 事件类型字符串，并允许未来新增事件。
type EventType string

const (
	// EventGuildCreate 表示机器人加入新频道。
	EventGuildCreate EventType = "GUILD_CREATE"
	// EventGuildUpdate 表示频道资料发生变更。
	EventGuildUpdate EventType = "GUILD_UPDATE"
	// EventGuildDelete 表示机器人被移除或频道解散，机器人退出该频道。
	EventGuildDelete EventType = "GUILD_DELETE"
	// EventChannelCreate 表示子频道被创建。
	EventChannelCreate EventType = "CHANNEL_CREATE"
	// EventChannelUpdate 表示子频道被更新，例如修改名称或新置顶精华。
	EventChannelUpdate EventType = "CHANNEL_UPDATE"
	// EventChannelDelete 表示子频道被删除。
	EventChannelDelete EventType = "CHANNEL_DELETE"
	// EventGuildMemberAdd 表示有人加入频道。
	EventGuildMemberAdd EventType = "GUILD_MEMBER_ADD"
	// EventGuildMemberUpdate 表示频道成员资料发生变更。
	EventGuildMemberUpdate EventType = "GUILD_MEMBER_UPDATE"
	// EventGuildMemberRemove 表示某人被踢出或移除频道。
	EventGuildMemberRemove EventType = "GUILD_MEMBER_REMOVE"
	// EventMessageDelete 表示频道中有人撤回消息。
	EventMessageDelete EventType = "MESSAGE_DELETE"
	// EventMessageReactionAdd 表示有人为频道消息添加表情表态。
	EventMessageReactionAdd EventType = "MESSAGE_REACTION_ADD"
	// EventMessageReactionRemove 表示有人为频道消息删除表情表态。
	EventMessageReactionRemove EventType = "MESSAGE_REACTION_REMOVE"
	// EventForumThreadCreate 表示有人发布主题。
	EventForumThreadCreate EventType = "FORUM_THREAD_CREATE"
	// EventForumThreadUpdate 表示有人更新或修改主题。
	EventForumThreadUpdate EventType = "FORUM_THREAD_UPDATE"
	// EventForumThreadDelete 表示有人删除主题。
	EventForumThreadDelete EventType = "FORUM_THREAD_DELETE"
	// EventForumPostCreate 表示有人发布帖子。
	EventForumPostCreate EventType = "FORUM_POST_CREATE"
	// EventForumReplyDelete 表示有人删除评论。
	EventForumReplyDelete EventType = "FORUM_REPLY_DELETE"
	// EventForumPostDelete 表示有人删除帖子。
	EventForumPostDelete EventType = "FORUM_POST_DELETE"
	// EventForumReplyCreate 表示有人回复帖子或主题。
	EventForumReplyCreate EventType = "FORUM_REPLY_CREATE"
	// EventAudioLiveEnter 表示有人进入音视直播子频道。
	EventAudioLiveEnter EventType = "AUDIO_OR_LIVE_CHANNEL_MEMBER_ENTER"
	// EventAudioLiveExit 表示有人离开音视直播子频道。
	EventAudioLiveExit EventType = "AUDIO_OR_LIVE_CHANNEL_MEMBER_EXIT"

	// EventGroupAddRobot 表示机器人被添加到群聊。
	EventGroupAddRobot EventType = "GROUP_ADD_ROBOT"
	// EventGroupDelRobot 表示机器人被移出群聊。
	EventGroupDelRobot EventType = "GROUP_DEL_ROBOT"
	// EventGroupMessageReject 表示群管理员关闭通知，拒绝机器人主动消息。
	EventGroupMessageReject EventType = "GROUP_MSG_REJECT"
	// EventGroupMessageReceive 表示群管理员开启通知，接受机器人主动消息。
	EventGroupMessageReceive EventType = "GROUP_MSG_RECEIVE"
	// EventFriendAdd 表示用户添加机器人为好友。
	EventFriendAdd EventType = "FRIEND_ADD"
	// EventFriendDelete 表示用户删除机器人好友。
	EventFriendDelete EventType = "FRIEND_DEL"
	// EventC2CMessageReject 表示好友关闭通知，拒绝机器人主动私聊消息。
	EventC2CMessageReject EventType = "C2C_MSG_REJECT"
	// EventC2CMessageReceive 表示好友开启通知，接受机器人主动私聊消息。
	EventC2CMessageReceive EventType = "C2C_MSG_RECEIVE"
	// EventGroupMemberAdd 表示有人加入群聊。
	EventGroupMemberAdd EventType = "GROUP_MEMBER_ADD"
	// EventGroupMemberRemove 表示某人被踢出或移除群聊。
	EventGroupMemberRemove EventType = "GROUP_MEMBER_REMOVE"
	// EventGroupJoinRequest 表示有人申请加入群聊。
	EventGroupJoinRequest EventType = "GROUP_JOIN_REQUEST"

	// EventInteractionCreate 表示有人触发按钮回调，频道、群聊和私聊场景共用此事件值。
	EventInteractionCreate EventType = "INTERACTION_CREATE"
)

// GuildInfo 表示频道详细信息。
type GuildInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	IconURL     string `json:"icon"`
	OwnerID     string `json:"owner_id"`
	JoinedAt    string `json:"joined_at"`
	MemberCount int    `json:"member_count"`
	MaxMembers  int    `json:"max_members"`
	Description string `json:"description"`
}

// ChannelInfo 表示子频道详细信息。
type ChannelInfo struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            int    `json:"type"`
	SubType         int    `json:"sub_type"`
	ParentID        string `json:"parent_id"`
	OwnerID         string `json:"owner_id"`
	PrivateType     int    `json:"private_type"`
	SpeakPermission int    `json:"speak_permission"`
	ApplicationID   int    `json:"application_id"`
}

// GuildMemberInfo 表示频道成员详细信息。
type GuildMemberInfo struct {
	UserName  string
	NickName  string
	AvatarURL string
	JoinedAt  string
	Roles     []string
}

// UnmarshalJSON 解析框架返回的频道成员 JSON。
func (v *GuildMemberInfo) UnmarshalJSON(data []byte) error {
	var raw struct {
		User struct {
			Name   string `json:"name"`
			Avatar string `json:"avatar"`
		} `json:"user"`
		Nick     string   `json:"nick"`
		JoinedAt string   `json:"joined_at"`
		Roles    []string `json:"roles"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v.UserName, v.NickName, v.AvatarURL = raw.User.Name, raw.Nick, raw.User.Avatar
	v.JoinedAt, v.Roles = raw.JoinedAt, raw.Roles
	return nil
}

// FrameworkInfo 表示 Bee 框架信息。
type FrameworkInfo struct {
	Name        string `json:"name"`
	Version     string `json:"ver"`
	Sent        string `json:"send"`
	Received    string `json:"receive"`
	AccountType string `json:"type"`
	Uptime      string `json:"time"`
}

// RobotInfo 表示机器人账号信息。
type RobotInfo struct {
	ID        string `json:"id"`
	Name      string `json:"username"`
	AvatarURL string `json:"avatar"`
}

// GroupBasicInfo 表示群聊基本信息。
type GroupBasicInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"desc"`
	Type        string `json:"type"`
	Tags        string `json:"tag"`
	MemberCount int    `json:"num"`
}

// RobotGroupStatus 表示机器人在指定群内的状态。
type RobotGroupStatus struct {
	RobotID     string `json:"id"`
	JoinedAt    string `json:"date"`
	ReceivePush bool   `json:"send"`
	MessageType int    `json:"type"`
	Power       int    `json:"power"`
}

// GroupJoinRequest 表示一条入群申请。
type GroupJoinRequest struct {
	RequestID string `json:"joinId"`
	RiskTips  string `json:"tips"`
	OpenID    string `json:"openId"`
	UserID    string `json:"userId"`
	UserNick  string `json:"userNick"`
	Date      string `json:"date"`
	Type      int    `json:"type"`
	InvitedBy string `json:"invitedBy"`
	IsBot     bool   `json:"bot"`
	Content   string `json:"content"`
}

// GroupMuteInfo 表示群禁言状态。
type GroupMuteInfo struct {
	AllMuted     bool   `json:"all"`
	MutedMembers string `json:"list"`
}

// MarkdownParam 表示 Markdown 模板的一组键和值。
type MarkdownParam struct{ Key, Value string }

// MarkdownMessage 表示原生或模板 Markdown 消息参数。
type MarkdownMessage struct {
	Native        string
	TemplateIndex int
	TemplateID    string
	Params        []MarkdownParam
	KeyboardJSON  string
	KeyboardID    string
}

// Button 表示 Markdown 自定义键盘中的一个按钮。
type Button struct {
	Label                string
	VisitedLabel         string
	Style                int
	Type                 int
	Permission           int
	UserIDs              []string
	RoleIDs              []string
	Data                 string
	Reply                bool
	Enter                bool
	Anchor               int
	ClickLimit           int
	AtBotShowChannelList bool
	UnsupportTips        string
}

// ==================== api.go ====================
func decodeCall[T any](ctx *RobotContext, op int, args ...string) (T, error) {
	var result T
	out, err := ctx.Call(op, args...)
	if err != nil {
		return result, err
	}
	if out == "" {
		return result, errors.New("框架返回空数据")
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return result, err
	}
	return result, nil
}

// Log 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) Log(content string) error { _, err := ctx.Call(OpLog, content); return err }

// ListGuilds 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) ListGuilds() (string, error) { return ctx.Call(OpListGuilds) }

// GetGuild 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetGuild(id string) (GuildInfo, error) {
	return decodeCall[GuildInfo](ctx, OpGetGuild, id)
}

// ListChannels 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) ListChannels(guildID string) (string, error) {
	return ctx.Call(OpListChannels, guildID)
}

// GetChannel 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetChannel(id string) (ChannelInfo, error) {
	return decodeCall[ChannelInfo](ctx, OpGetChannel, id)
}

// CreateChannel 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) CreateChannel(guildID, name string, channelType, subType, privateType, speakPermission, applicationID int) (bool, error) {
	return ctx.CallBool(OpCreateChannel, guildID, name, intText(channelType), intText(subType), intText(privateType), intText(speakPermission), intText(applicationID))
}

// GetChannelOnlineCount 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetChannelOnlineCount(id string) (string, error) {
	return ctx.Call(OpGetChannelOnlineCount, id)
}

// GetGuildMember 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetGuildMember(guildID, userID string) (GuildMemberInfo, error) {
	return decodeCall[GuildMemberInfo](ctx, OpGetGuildMember, guildID, userID)
}

// UpdateChannel 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) UpdateChannel(channelID, name string, privateType, speakPermission int) (bool, error) {
	return ctx.CallBool(OpUpdateChannel, channelID, name, intText(privateType), intText(speakPermission))
}

// DeleteChannel 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) DeleteChannel(id string) (bool, error) {
	return ctx.CallBool(OpDeleteChannel, id)
}

// DeleteGuildMember 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) DeleteGuildMember(guildID, userID string, recallDays int, blacklist bool) (bool, error) {
	// 修复原 SDK：显式传 false 不应被改成 true。
	return ctx.CallBool(OpDeleteGuildMember, guildID, userID, intText(recallDays), boolText(blacklist))
}

// ListGuildRoles 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) ListGuildRoles(guildID string) (string, error) {
	return ctx.Call(OpListGuildRoles, guildID)
}

// CreateGuildRole 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) CreateGuildRole(guildID, name string, color int, hoist bool) (string, error) {
	return ctx.Call(OpCreateGuildRole, guildID, name, intText(color), boolText(hoist))
}

// UpdateGuildRole 修改频道身份组。name 为空、color 为 -1 时表示对应项不修改；hoist 为 nil 时不修改展示状态。
func (ctx *RobotContext) UpdateGuildRole(guildID, roleID, name string, color int, hoist *bool) error {
	// 修复原 SDK：名称、颜色、展示状态必须属于同一个协议字符串。
	hoistText := ""
	if hoist != nil {
		hoistText = boolText(*hoist)
	}
	_, err := ctx.Call(OpUpdateGuildRole, guildID, roleID, name, intText(color), hoistText)
	return err
}

// IsGuildOwner 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) IsGuildOwner(guildID, userID string) (bool, error) {
	return ctx.CallBool(OpIsGuildOwner, guildID, userID)
}

// IsGuildAdmin 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) IsGuildAdmin(guildID, userID string) (bool, error) {
	return ctx.CallBool(OpIsGuildAdmin, guildID, userID)
}

// IsChannelAdmin 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) IsChannelAdmin(guildID, userID string) (bool, error) {
	return ctx.CallBool(OpIsChannelAdmin, guildID, userID)
}

// HasGuildRole 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) HasGuildRole(guildID, userID, roleID string) (bool, error) {
	return ctx.CallBool(OpHasGuildRole, guildID, userID, roleID)
}

// DeleteGuildRole 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) DeleteGuildRole(guildID, roleID string) error {
	_, err := ctx.Call(OpDeleteGuildRole, guildID, roleID)
	return err
}

// AddGuildMemberRole 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) AddGuildMemberRole(guildID, userID, roleID string) error {
	_, err := ctx.Call(OpAddGuildMemberRole, guildID, userID, roleID)
	return err
}

// RemoveGuildMemberRole 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) RemoveGuildMemberRole(guildID, userID, roleID string) error {
	_, err := ctx.Call(OpRemoveGuildMemberRole, guildID, userID, roleID)
	return err
}

// RecallChannelMessage 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) RecallChannelMessage(channelID, messageID string, hideTip bool) error {
	_, err := ctx.Call(OpRecallChannelMessage, channelID, messageID, boolText(hideTip))
	return err
}

// MuteGuildMember 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) MuteGuildMember(guildID, userID string, seconds int) error {
	_, err := ctx.Call(OpMuteGuildMember, guildID, userID, intText(seconds))
	return err
}

// MuteGuild 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) MuteGuild(guildID string, seconds int) error {
	_, err := ctx.Call(OpMuteGuild, guildID, intText(seconds))
	return err
}

// GetRobotID 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetRobotID() (string, error) { return ctx.Call(OpGetRobotID) }

// GetRobotInfo 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetRobotInfo() (RobotInfo, error) {
	return decodeCall[RobotInfo](ctx, OpGetRobotInfo)
}

// GetFrameworkInfo 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetFrameworkInfo() (FrameworkInfo, error) {
	return decodeCall[FrameworkInfo](ctx, OpGetFrameworkInfo)
}

// GetQQNickname 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetQQNickname(qq string) (string, error) {
	return ctx.Call(OpGetQQNickname, qq)
}

// UploadImage 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) UploadImage(pathOrURL string) (string, error) {
	return ctx.Call(OpUploadImage, pathOrURL)
}

// RespondButton 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) RespondButton(eventID string, responseType int) (string, error) {
	return ctx.Call(OpRespondButton, eventID, intText(responseType))
}

// GetRobotAppID 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetRobotAppID() (string, error) { return ctx.Call(OpGetRobotAppID) }

// GetAvatar 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetAvatar(userID string, size int) (string, error) {
	return ctx.Call(OpGetAvatar, userID, intText(size))
}

// GetQQAvatar 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetQQAvatar(qq string, size int) (string, error) {
	return ctx.Call(OpGetQQAvatar, qq, intText(size))
}

// RecallGroupMessage 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) RecallGroupMessage(groupID, messageID string) (string, error) {
	return ctx.Call(OpRecallGroupMessage, groupID, messageID)
}

// RecallFriendMessage 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) RecallFriendMessage(friendID, messageID string) (string, error) {
	return ctx.Call(OpRecallFriendMessage, friendID, messageID)
}

// AddChannelReaction 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) AddChannelReaction(channelID, messageID string, reactionType, reactionID int) (string, error) {
	return ctx.Call(OpAddChannelReaction, channelID, messageID, intText(reactionType), intText(reactionID))
}

// DeleteChannelReaction 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) DeleteChannelReaction(channelID, messageID string, reactionType, reactionID int) (string, error) {
	return ctx.Call(OpDeleteChannelReaction, channelID, messageID, intText(reactionType), intText(reactionID))
}

// ListChannelReactionUsers 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) ListChannelReactionUsers(channelID, messageID string, reactionType, reactionID int) (string, error) {
	return ctx.Call(OpListChannelReactionUsers, channelID, messageID, intText(reactionType), intText(reactionID))
}

// GetRobotStats 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetRobotStats(limit *int) (string, error) {
	value := ""
	if limit != nil {
		value = strconv.Itoa(*limit)
	}
	return ctx.Call(OpGetRobotStats, value)
}

// GetRobotToken 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetRobotToken() (string, error) { return ctx.Call(OpGetRobotToken) }

// GetRobotSecret 封装对应的 Bee 框架 API；参数和返回值与方法签名一致。
func (ctx *RobotContext) GetRobotSecret() (string, error) { return ctx.Call(OpGetRobotSecret) }

// GetRobotShareLink 返回机器人分享链接，用于邀请用户添加机器人为好友。
func (ctx *RobotContext) GetRobotShareLink() (string, error) {
	return ctx.Call(OpGetRobotShareLink)
}

// GetGroupBasicInfo 返回群聊基本信息。
func (ctx *RobotContext) GetGroupBasicInfo(groupID string) (GroupBasicInfo, error) {
	return decodeCall[GroupBasicInfo](ctx, OpGetGroupBasicInfo, groupID)
}

// GetRobotGroupStatus 返回机器人在指定群内的状态。
func (ctx *RobotContext) GetRobotGroupStatus(groupID string) (RobotGroupStatus, error) {
	return decodeCall[RobotGroupStatus](ctx, OpGetRobotGroupStatus, groupID)
}

// ListGroupJoinRequests 返回指定群的入群申请列表；需管理员身份调用。
func (ctx *RobotContext) ListGroupJoinRequests(groupID string) ([]GroupJoinRequest, error) {
	return decodeCall[[]GroupJoinRequest](ctx, OpListGroupJoinRequests, groupID)
}

// HandleGroupJoinRequest 处理入群请求；action 为 0 表示同意，1 表示拒绝。
func (ctx *RobotContext) HandleGroupJoinRequest(groupID, userID, requestID string, action int, rejectReason string, blockUser bool) error {
	_, err := ctx.Call(OpHandleGroupJoinRequest, groupID, userID, requestID, intText(action), rejectReason, boolText(blockUser))
	return err
}

// GetGroupMuteInfo 返回指定群的禁言状态；需管理员身份调用。
func (ctx *RobotContext) GetGroupMuteInfo(groupID string) (GroupMuteInfo, error) {
	return decodeCall[GroupMuteInfo](ctx, OpGetGroupMuteInfo, groupID)
}

// IsGroupMemberMuted 查询指定群成员是否被禁言；需管理员身份调用。
func (ctx *RobotContext) IsGroupMemberMuted(groupID, userID string) (bool, error) {
	return ctx.CallBool(OpIsGroupMemberMuted, groupID, userID)
}

// IsGroupMuted 查询指定群是否开启全员禁言；需管理员身份调用。
func (ctx *RobotContext) IsGroupMuted(groupID string) (bool, error) {
	return ctx.CallBool(OpIsGroupMuted, groupID)
}

// MuteGroupMembers 设置群成员禁言秒数，传 0 表示解除禁言；userIDs 支持用换行分隔多个用户。
func (ctx *RobotContext) MuteGroupMembers(groupID, userIDs string, seconds int) error {
	_, err := ctx.Call(OpMuteGroupMembers, groupID, userIDs, intText(seconds))
	return err
}

// IsGroupManagement 查询机器人是否为指定群的管理员或群主。
func (ctx *RobotContext) IsGroupManagement(groupID string) (bool, error) {
	return ctx.CallBool(OpIsGroupManagement, groupID)
}

// ==================== messages.go ====================
func sendMessage(ctx *RobotContext, op int, target, content, media string, deleteMedia, active bool, recallInteraction *bool) (string, error) {
	messageID, eventID := activeIDs(ctx, active)
	args := []string{target, content, media, boolText(deleteMedia), messageID, eventID}
	if recallInteraction != nil {
		args = append(args, boolText(*recallInteraction))
	}
	return ctx.Call(op, args...)
}

func sendMedia(ctx *RobotContext, op int, target, file string, deleteFile, active bool, recallInteraction *bool) (string, error) {
	messageID, eventID := activeIDs(ctx, active)
	args := []string{target, file, boolText(deleteFile), messageID, eventID}
	if recallInteraction != nil {
		args = append(args, boolText(*recallInteraction))
	}
	return ctx.Call(op, args...)
}

func sendReply(ctx *RobotContext, op int, target, quotedMessageID, content, image string, deleteImage, active bool, recallInteraction *bool) (string, error) {
	messageID, eventID := activeIDs(ctx, active)
	args := []string{target, quotedMessageID, content, image, boolText(deleteImage), messageID, eventID}
	if recallInteraction != nil {
		args = append(args, boolText(*recallInteraction))
	}
	return ctx.Call(op, args...)
}

func sendCard(ctx *RobotContext, op int, target string, fields []string, active bool, recallInteraction *bool) (string, error) {
	messageID, eventID := activeIDs(ctx, active)
	args := append([]string{target}, fields...)
	args = append(args, messageID, eventID)
	if recallInteraction != nil {
		args = append(args, boolText(*recallInteraction))
	}
	return ctx.Call(op, args...)
}

func markdownArgs(ctx *RobotContext, target string, message MarkdownMessage, active bool, recallInteraction *bool) []string {
	args := []string{target, message.Native, intText(message.TemplateIndex), message.TemplateID}
	for i := 0; i < 10; i++ {
		if i < len(message.Params) {
			args = append(args, message.Params[i].Key, message.Params[i].Value)
		} else {
			args = append(args, "", "")
		}
	}
	messageID, eventID := activeIDs(ctx, active)
	args = append(args, messageID, message.KeyboardJSON, eventID, message.KeyboardID)
	if recallInteraction != nil {
		args = append(args, boolText(*recallInteraction))
	}
	return args
}

// SendChannelMessage 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelMessage(channelID, content, image string, deleteImage, active bool) (string, error) {
	return sendMessage(ctx, OpSendChannelMessage, channelID, content, image, deleteImage, active, nil)
}

// SendChannelDM 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelDM(guildID, content, image string, deleteImage, active bool) (string, error) {
	return sendMessage(ctx, OpSendChannelDM, guildID, content, image, deleteImage, active, nil)
}

// SendAdaptiveMessage 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendAdaptiveMessage(sourceID, content, image string, deleteImage, active bool) (string, error) {
	return sendMessage(ctx, OpSendAdaptiveMessage, sourceID, content, image, deleteImage, active, nil)
}

// SendGroupMessage 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupMessage(groupID, content, image string, deleteImage, active bool) (string, error) {
	return sendMessage(ctx, OpSendGroupMessage, groupID, content, image, deleteImage, active, nil)
}

// SendFriendMessage 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendMessage(friendID, content, image string, deleteImage, active, recallInteraction bool) (string, error) {
	return sendMessage(ctx, OpSendFriendMessage, friendID, content, image, deleteImage, active, &recallInteraction)
}

// SendAdaptivePrivateMessage 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendAdaptivePrivateMessage(sourceID, content, image string, deleteImage, active bool) (string, error) {
	return sendMessage(ctx, OpSendAdaptivePrivateMessage, sourceID, content, image, deleteImage, active, nil)
}

// SendGroupVideo 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupVideo(groupID, file string, deleteFile, active bool) (string, error) {
	return sendMedia(ctx, OpSendGroupVideo, groupID, file, deleteFile, active, nil)
}

// SendGroupAudio 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupAudio(groupID, file string, deleteFile, active bool) (string, error) {
	return sendMedia(ctx, OpSendGroupAudio, groupID, file, deleteFile, active, nil)
}

// SendFriendVideo 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendVideo(friendID, file string, deleteFile, active, recallInteraction bool) (string, error) {
	return sendMedia(ctx, OpSendFriendVideo, friendID, file, deleteFile, active, &recallInteraction)
}

// SendFriendAudio 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendAudio(friendID, file string, deleteFile, active, recallInteraction bool) (string, error) {
	return sendMedia(ctx, OpSendFriendAudio, friendID, file, deleteFile, active, &recallInteraction)
}

// SendGroupFile 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupFile(groupID, file string, deleteFile, active bool) (string, error) {
	return sendMedia(ctx, OpSendGroupFile, groupID, file, deleteFile, active, nil)
}

// SendFriendFile 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendFile(friendID, file string, deleteFile, active, recallInteraction bool) (string, error) {
	return sendMedia(ctx, OpSendFriendFile, friendID, file, deleteFile, active, &recallInteraction)
}

// SendChannelReply 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelReply(channelID, quotedMessageID, content, image string, active bool) (string, error) {
	messageID, eventID := activeIDs(ctx, active)
	return ctx.Call(OpSendChannelReply, channelID, quotedMessageID, content, image, messageID, eventID)
}

// SendGroupReply 发送群引用消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupReply(groupID, quotedMessageID, content, image string, deleteImage, active bool) (string, error) {
	return sendReply(ctx, OpSendGroupReply, groupID, quotedMessageID, content, image, deleteImage, active, nil)
}

// SendFriendReply 发送好友引用消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendReply(friendID, quotedMessageID, content, image string, deleteImage, active, recallInteraction bool) (string, error) {
	return sendReply(ctx, OpSendFriendReply, friendID, quotedMessageID, content, image, deleteImage, active, &recallInteraction)
}

// SendChannelCustom 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelCustom(channelID, rawJSON string, active bool) (string, error) {
	messageID, eventID := activeIDs(ctx, active)
	return ctx.Call(OpSendChannelCustom, channelID, rawJSON, messageID, eventID)
}

// SendChannelTextCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelTextCard(channelID, title, preview, content, imageURL string, active bool) (string, error) {
	return sendCard(ctx, OpSendChannelTextCard, channelID, []string{title, preview, content, imageURL}, active, nil)
}

// SendGroupTextCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupTextCard(groupID, preview, content string, active bool) (string, error) {
	return sendCard(ctx, OpSendGroupTextCard, groupID, []string{preview, content}, active, nil)
}

// SendFriendTextCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendTextCard(friendID, preview, content string, active, recallInteraction bool) (string, error) {
	return sendCard(ctx, OpSendFriendTextCard, friendID, []string{preview, content}, active, &recallInteraction)
}

// SendChannelLargeCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelLargeCard(channelID, title, subtitle, preview, imageURL, jumpURL string, active bool) (string, error) {
	return sendCard(ctx, OpSendChannelLargeCard, channelID, []string{title, subtitle, preview, imageURL, jumpURL}, active, nil)
}

// SendGroupLargeCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupLargeCard(groupID, title, subtitle, preview, imageURL, jumpURL string, active bool) (string, error) {
	return sendCard(ctx, OpSendGroupLargeCard, groupID, []string{title, subtitle, preview, imageURL, jumpURL}, active, nil)
}

// SendAdaptiveLargeCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendAdaptiveLargeCard(sourceID, title, subtitle, preview, imageURL, jumpURL string, active bool) (string, error) {
	return sendCard(ctx, OpSendAdaptiveLargeCard, sourceID, []string{title, subtitle, preview, imageURL, jumpURL}, active, nil)
}

// SendFriendLargeCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendLargeCard(friendID, title, subtitle, preview, imageURL, jumpURL string, active, recallInteraction bool) (string, error) {
	return sendCard(ctx, OpSendFriendLargeCard, friendID, []string{title, subtitle, preview, imageURL, jumpURL}, active, &recallInteraction)
}

// SendGroupThumbnailCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupThumbnailCard(groupID, title, subtitle, preview, imageURL, jumpURL string, active bool) (string, error) {
	return sendCard(ctx, OpSendGroupThumbnailCard, groupID, []string{title, subtitle, preview, imageURL, jumpURL}, active, nil)
}

// SendChannelThumbnailCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelThumbnailCard(channelID, title, subtitle, preview, imageURL, jumpURL string, active bool) (string, error) {
	return sendCard(ctx, OpSendChannelThumbnailCard, channelID, []string{title, subtitle, preview, imageURL, jumpURL}, active, nil)
}

// SendFriendThumbnailCard 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendThumbnailCard(friendID, title, subtitle, preview, imageURL, jumpURL string, active, recallInteraction bool) (string, error) {
	return sendCard(ctx, OpSendFriendThumbnailCard, friendID, []string{title, subtitle, preview, imageURL, jumpURL}, active, &recallInteraction)
}

// SendGroupMarkdown 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupMarkdown(groupID string, message MarkdownMessage, active bool) (string, error) {
	return ctx.Call(OpSendGroupMarkdown, markdownArgs(ctx, groupID, message, active, nil)...)
}

// SendChannelMarkdown 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendChannelMarkdown(channelID string, message MarkdownMessage, active bool) (string, error) {
	return ctx.Call(OpSendChannelMarkdown, markdownArgs(ctx, channelID, message, active, nil)...)
}

// SendFriendMarkdown 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendMarkdown(friendID string, message MarkdownMessage, active, recallInteraction bool) (string, error) {
	return ctx.Call(OpSendFriendMarkdown, markdownArgs(ctx, friendID, message, active, &recallInteraction)...)
}

// SendGroupButton 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendGroupButton(groupID, keyboardID string, active bool) (string, error) {
	return sendCard(ctx, OpSendGroupButton, groupID, []string{keyboardID}, active, nil)
}

// SendFriendButton 发送对应类型的 Bee 消息；active 为 true 时使用主动消息模式。
func (ctx *RobotContext) SendFriendButton(friendID, keyboardID string, active, recallInteraction bool) (string, error) {
	return sendCard(ctx, OpSendFriendButton, friendID, []string{keyboardID}, active, &recallInteraction)
}

// ==================== helpers.go ====================
// At 生成艾特指定用户的文本代码。
func At(userID string) string { return "<@!" + userID + ">" }

// AtEveryone 返回艾特全体成员的文本代码。
func AtEveryone() string { return "@everyone" }

// MentionedUserID 从 <@!用户ID> 中提取用户 ID。
func MentionedUserID(text string) string {
	start := strings.Index(text, "<@!")
	if start < 0 {
		return ""
	}
	rest := text[start+3:]
	end := strings.IndexByte(rest, '>')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// ImageDownloadURL 从 Bee 图片消息代码中提取下载地址。
func ImageDownloadURL(message string) string {
	start := strings.Index(message, ",url=")
	if start < 0 {
		return ""
	}
	rest := message[start+5:]
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// InlineCommand 生成可嵌入 Markdown 的 QQ 指令链接。
func InlineCommand(label, command string, send bool) string {
	enter := "false"
	if send {
		enter = "true"
	}
	return fmt.Sprintf("[%s](mqqapi://aio/inlinecmd?command=%s&reply=false&enter=%s)", label, url.QueryEscape(command), enter)
}

// ResolveRedirect 请求网址并返回重定向后的地址。
func ResolveRedirect(rawURL string) (string, error) {
	client := &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(rawURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	location := resp.Header.Get("Location")
	if location == "" {
		return rawURL, nil
	}
	base, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	target, err := base.Parse(location)
	if err != nil {
		return "", err
	}
	return target.String(), nil
}

type keyboard struct {
	Content keyboardContent `json:"content"`
}
type keyboardContent struct {
	BotAppID string        `json:"bot_appid"`
	Rows     []keyboardRow `json:"rows"`
}
type keyboardRow struct {
	Buttons []keyboardButton `json:"buttons"`
}
type keyboardButton struct {
	ID         string         `json:"id"`
	RenderData keyboardRender `json:"render_data"`
	Action     keyboardAction `json:"action"`
}
type keyboardRender struct {
	Label        string `json:"label"`
	VisitedLabel string `json:"visited_label"`
	Style        int    `json:"style"`
}
type keyboardAction struct {
	Type                 int                `json:"type"`
	Permission           keyboardPermission `json:"permission"`
	Data                 string             `json:"data"`
	Reply                bool               `json:"reply,omitempty"`
	Enter                bool               `json:"enter,omitempty"`
	Anchor               int                `json:"anchor,omitempty"`
	ClickLimit           int                `json:"click_limit,omitempty"`
	AtBotShowChannelList bool               `json:"at_bot_show_channel_list,omitempty"`
	UnsupportTips        string             `json:"unsupport_tips,omitempty"`
}
type keyboardPermission struct {
	Type           int      `json:"type"`
	SpecifyUserIDs []string `json:"specify_user_ids,omitempty"`
	SpecifyRoleIDs []string `json:"specify_role_ids,omitempty"`
}

// BuildKeyboard 使用传入的机器人 AppID，避免原 SDK 将 AppID 固定为 102069021。
func BuildKeyboard(botAppID string, rows [][]Button) (string, error) {
	if botAppID == "" {
		return "", errors.New("botAppID 不能为空")
	}
	if len(rows) > 5 {
		return "", errors.New("按钮最多 5 行")
	}
	result := keyboard{Content: keyboardContent{BotAppID: botAppID}}
	for rowIndex, row := range rows {
		if len(row) > 10 {
			return "", fmt.Errorf("第 %d 行按钮超过 10 个", rowIndex+1)
		}
		outRow := keyboardRow{}
		for columnIndex, button := range row {
			label := button.Label
			if label == "" {
				label = "按钮"
			}
			visited := button.VisitedLabel
			if visited == "" {
				visited = "按钮"
			}
			data := button.Data
			if data == "" {
				data = "你好"
			}
			outRow.Buttons = append(outRow.Buttons, keyboardButton{
				ID:         fmt.Sprintf("%d_%d", rowIndex, columnIndex),
				RenderData: keyboardRender{Label: label, VisitedLabel: visited, Style: button.Style},
				Action: keyboardAction{
					Type:                 button.Type,
					Permission:           keyboardPermission{Type: button.Permission, SpecifyUserIDs: button.UserIDs, SpecifyRoleIDs: button.RoleIDs},
					Data:                 data,
					Reply:                button.Reply,
					Enter:                button.Enter,
					Anchor:               button.Anchor,
					ClickLimit:           button.ClickLimit,
					AtBotShowChannelList: button.AtBotShowChannelList,
					UnsupportTips:        button.UnsupportTips,
				},
			})
		}
		result.Content.Rows = append(result.Content.Rows, outRow)
	}
	data, err := json.Marshal(result)
	return string(data), err
}

// ==================== sdk.go ====================
// BeeSeparator 是 Bee 框架命令协议使用的字段分隔符。
const BeeSeparator = "%@#bee#@%"

const (
	// MessageIgnore 与 MessageContinue 都是 0，是原 Bee SDK 中的语义别名。
	MessageIgnore = 0
	// MessageContinue 表示消息处理完成后继续投递给后续插件。
	MessageContinue = 0
	// MessageIntercept 表示拦截消息，不再投递给后续插件。
	MessageIntercept = 1
)

// RobotContext 保存框架随事件传入的机器人上下文。
type RobotContext struct {
	APIText   string          `json:"api"`
	transport APITransport    `json:"-"`
	Message   string          `json:"msg"`
	MessageID string          `json:"msg_id"`
	ChannelID string          `json:"channel_id"`
	GuildID   string          `json:"guild_id"`
	FromID    string          `json:"form_id"`
	RobotID   string          `json:"robot_id"`
	PluginID  string          `json:"plugin_id"`
	EventID   string          `json:"event_id"`
	Raw       json.RawMessage `json:"raw"`
}

// BeeAPI 是面向插件开发者的简化 SDK 入口。
// 它保存了创建时的回调上下文快照。依赖 msg_id、event_id 等当前上下文的操作不能跨回调复用；
// 输出日志、取框架信息等不依赖消息或事件上下文的框架级操作可以复用有效对象。
type BeeAPI struct {
	ctx *RobotContext
}

// MessageTarget 是已经绑定消息目标的快捷发送器。
type MessageTarget struct {
	ctx      *RobotContext
	targetID string
	kind     int
}

const (
	targetFriend = iota
	targetGroup
	targetChannel
	targetChannelDM
)

// NewBeeAPI 根据当前 Bee 回调传入的最新机器人 JSON 创建简化 SDK 入口。
// 消息回复、撤回、按钮响应等上下文相关操作应在每次回调中重新创建，
// 确保 msg_id、event_id 等数据属于当前回调。
func NewBeeAPI(robotJSON string) (*BeeAPI, error) {
	ctx, err := ParseRobotContext(robotJSON)
	if err != nil {
		return nil, err
	}
	return &BeeAPI{ctx: ctx}, nil
}

// Log 向 Bee 框架输出日志。
func (api *BeeAPI) Log(content string) error {
	return api.ctx.Log(content)
}

// GetAppDataDir 返回当前插件的应用数据目录。
// 生产环境中 Worker 固定释放并运行于“Bee框架根目录\plugin_data\插件名称\”，
// 因此使用 Worker 可执行文件所在目录，避免受到进程当前工作目录变化的影响。
func (api *BeeAPI) GetAppDataDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("获取插件应用数据目录: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return "", fmt.Errorf("规范化插件应用数据目录: %w", err)
	}
	return filepath.Clean(filepath.Dir(executable)), nil
}

// EventID 返回当前回调的事件 ID，按钮响应等上下文相关 API 会使用。
func (api *BeeAPI) EventID() string {
	if api == nil || api.ctx == nil {
		return ""
	}
	return api.ctx.EventID
}

// RespondButton 响应当前按钮事件；responseType 的具体值按 Bee 按钮响应协议填写。
func (api *BeeAPI) RespondButton(responseType int) (string, error) {
	if api == nil || api.ctx == nil || api.ctx.EventID == "" {
		return "", errors.New("当前回调缺少 event_id，无法响应按钮事件")
	}
	return api.ctx.RespondButton(api.ctx.EventID, responseType)
}

// Friend 绑定好友 ID，之后可直接调用 SendText、SendImage 等方法。
func (api *BeeAPI) Friend(friendID string) *MessageTarget {
	return &MessageTarget{ctx: api.ctx, targetID: friendID, kind: targetFriend}
}

// Group 绑定群 ID，之后可直接调用 SendText、SendImage 等方法。
func (api *BeeAPI) Group(groupID string) *MessageTarget {
	return &MessageTarget{ctx: api.ctx, targetID: groupID, kind: targetGroup}
}

// Channel 绑定子频道 ID，之后可直接调用 SendText、SendImage 等方法。
func (api *BeeAPI) Channel(channelID string) *MessageTarget {
	return &MessageTarget{ctx: api.ctx, targetID: channelID, kind: targetChannel}
}

// ChannelDM 绑定频道 ID，用于发送频道私信。
func (api *BeeAPI) ChannelDM(guildID string) *MessageTarget {
	return &MessageTarget{ctx: api.ctx, targetID: guildID, kind: targetChannelDM}
}

// IsGuildOwner 判断指定用户是否为频道主。
func (api *BeeAPI) IsGuildOwner(guildID, userID string) (bool, error) {
	return api.ctx.IsGuildOwner(guildID, userID)
}

// MuteMember 设置频道成员禁言秒数，传 0 表示解除禁言。
func (api *BeeAPI) MuteMember(guildID, userID string, seconds int) error {
	return api.ctx.MuteGuildMember(guildID, userID, seconds)
}

// MuteAll 设置频道全员禁言秒数，传 0 表示关闭全员禁言。
func (api *BeeAPI) MuteAll(guildID string, seconds int) error {
	return api.ctx.MuteGuild(guildID, seconds)
}

// GetRobotShareLink 返回机器人分享链接，用于邀请用户添加机器人为好友。
func (api *BeeAPI) GetRobotShareLink() (string, error) {
	return api.ctx.GetRobotShareLink()
}

// GetGroupBasicInfo 返回群聊基本信息。
func (api *BeeAPI) GetGroupBasicInfo(groupID string) (GroupBasicInfo, error) {
	return api.ctx.GetGroupBasicInfo(groupID)
}

// GetRobotGroupStatus 返回机器人在指定群内的状态。
func (api *BeeAPI) GetRobotGroupStatus(groupID string) (RobotGroupStatus, error) {
	return api.ctx.GetRobotGroupStatus(groupID)
}

// ListGroupJoinRequests 返回指定群的入群申请列表；需管理员身份调用。
func (api *BeeAPI) ListGroupJoinRequests(groupID string) ([]GroupJoinRequest, error) {
	return api.ctx.ListGroupJoinRequests(groupID)
}

// HandleGroupJoinRequest 处理入群请求；action 为 0 表示同意，1 表示拒绝。
func (api *BeeAPI) HandleGroupJoinRequest(groupID, userID, requestID string, action int, rejectReason string, blockUser bool) error {
	return api.ctx.HandleGroupJoinRequest(groupID, userID, requestID, action, rejectReason, blockUser)
}

// GetGroupMuteInfo 返回指定群的禁言状态；需管理员身份调用。
func (api *BeeAPI) GetGroupMuteInfo(groupID string) (GroupMuteInfo, error) {
	return api.ctx.GetGroupMuteInfo(groupID)
}

// IsGroupMemberMuted 查询指定群成员是否被禁言；需管理员身份调用。
func (api *BeeAPI) IsGroupMemberMuted(groupID, userID string) (bool, error) {
	return api.ctx.IsGroupMemberMuted(groupID, userID)
}

// IsGroupMuted 查询指定群是否开启全员禁言；需管理员身份调用。
func (api *BeeAPI) IsGroupMuted(groupID string) (bool, error) {
	return api.ctx.IsGroupMuted(groupID)
}

// MuteGroupMembers 设置群成员禁言秒数，传 0 表示解除禁言；userIDs 支持用换行分隔多个用户。
func (api *BeeAPI) MuteGroupMembers(groupID, userIDs string, seconds int) error {
	return api.ctx.MuteGroupMembers(groupID, userIDs, seconds)
}

// IsGroupManagement 查询机器人是否为指定群的管理员或群主。
func (api *BeeAPI) IsGroupManagement(groupID string) (bool, error) {
	return api.ctx.IsGroupManagement(groupID)
}

// ParseMention 判断消息是否艾特当前机器人，并返回移除艾特代码后的消息内容。
func (api *BeeAPI) ParseMention(content string) (bool, string, error) {
	robotID, err := api.ctx.GetRobotID()
	if err != nil {
		return false, content, err
	}
	mention := At(robotID)
	if !strings.Contains(content, mention) {
		return false, strings.TrimSpace(content), nil
	}
	return true, strings.TrimSpace(strings.ReplaceAll(content, mention, "")), nil
}

// SendText 发送纯文本消息，默认使用当前消息进行被动回复。
func (target *MessageTarget) SendText(content string) (string, error) {
	switch target.kind {
	case targetFriend:
		return target.ctx.SendFriendMessage(target.targetID, content, "", false, false, false)
	case targetGroup:
		return target.ctx.SendGroupMessage(target.targetID, content, "", false, false)
	case targetChannelDM:
		return target.ctx.SendChannelDM(target.targetID, content, "", false, false)
	default:
		return target.ctx.SendChannelMessage(target.targetID, content, "", false, false)
	}
}

// SendImage 发送图片消息，可同时携带文字。
func (target *MessageTarget) SendImage(content, image string) (string, error) {
	switch target.kind {
	case targetFriend:
		return target.ctx.SendFriendMessage(target.targetID, content, image, false, false, false)
	case targetGroup:
		return target.ctx.SendGroupMessage(target.targetID, content, image, false, false)
	case targetChannelDM:
		return target.ctx.SendChannelDM(target.targetID, content, image, false, false)
	default:
		return target.ctx.SendChannelMessage(target.targetID, content, image, false, false)
	}
}

// SendActiveText 主动发送纯文本消息；主动消息受平台频率和次数限制。
func (target *MessageTarget) SendActiveText(content string) (string, error) {
	switch target.kind {
	case targetFriend:
		return target.ctx.SendFriendMessage(target.targetID, content, "", false, true, false)
	case targetGroup:
		return target.ctx.SendGroupMessage(target.targetID, content, "", false, true)
	case targetChannelDM:
		return target.ctx.SendChannelDM(target.targetID, content, "", false, true)
	default:
		return target.ctx.SendChannelMessage(target.targetID, content, "", false, true)
	}
}

// SendTextCard 发送频道文字卡片消息；仅用于 Channel 目标。
func (target *MessageTarget) SendTextCard(title, preview, content, imageURL string) (string, error) {
	return target.ctx.SendChannelTextCard(target.targetID, title, preview, content, imageURL, false)
}

// SendCustom 发送频道自定义 JSON 消息；仅用于 Channel 目标。
func (target *MessageTarget) SendCustom(rawJSON string) (string, error) {
	return target.ctx.SendChannelCustom(target.targetID, rawJSON, false)
}

// SendLargeCard 发送频道大图卡片消息；仅用于 Channel 目标。
func (target *MessageTarget) SendLargeCard(title, subtitle, preview, imageURL, jumpURL string) (string, error) {
	return target.ctx.SendChannelLargeCard(target.targetID, title, subtitle, preview, imageURL, jumpURL, false)
}

// Reply 发送引用消息，按当前目标自动选择频道、群或好友引用接口。
func (target *MessageTarget) Reply(messageID, content, image string) (string, error) {
	switch target.kind {
	case targetFriend:
		return target.ctx.SendFriendReply(target.targetID, messageID, content, image, false, false, false)
	case targetGroup:
		return target.ctx.SendGroupReply(target.targetID, messageID, content, image, false, false)
	default:
		return target.ctx.SendChannelReply(target.targetID, messageID, content, image, false)
	}
}

// ParseRobotContext 从 JSON 解析机器人上下文和框架 API 地址。
func ParseRobotContext(text string) (*RobotContext, error) {
	return ParseRobotContextWithTransport(text, currentAPITransport())
}

func ParseRobotContextWithTransport(text string, transport APITransport) (*RobotContext, error) {
	var ctx RobotContext
	if err := json.Unmarshal([]byte(text), &ctx); err != nil {
		return nil, fmt.Errorf("解析机器人上下文: %w", err)
	}
	if transport == nil {
		return nil, errors.New("Bee API IPC transport 未连接")
	}
	ctx.transport = transport
	return &ctx, nil
}

// MustRobotContext 解析机器人上下文，失败时触发 panic。
func MustRobotContext(text string) *RobotContext {
	ctx, err := ParseRobotContext(text)
	if err != nil {
		panic(err)
	}
	return ctx
}

func (ctx *RobotContext) command(op int, args ...string) string {
	fields := make([]string, 0, len(args)+2)
	fields = append(fields, strconv.Itoa(op))
	// 操作码 31、45、48 按原 SDK 不带 plugin_id。
	if op != OpGetRobotID && op != OpUploadImage && op != OpGetRobotAppID {
		fields = append(fields, ctx.PluginID)
	}
	fields = append(fields, args...)
	return strings.Join(fields, BeeSeparator)
}

func encodeFrameworkText(value string) string {
	var out strings.Builder
	for _, r := range value {
		text := string(r)
		encoded, _, err := transform.String(simplifiedchinese.GBK.NewEncoder(), text)
		if err == nil {
			decoded, _, decodeErr := transform.String(simplifiedchinese.GBK.NewDecoder(), encoded)
			if decodeErr == nil && decoded == text {
				out.WriteString(text)
				continue
			}
		}
		for _, unit := range utf16.Encode([]rune{r}) {
			fmt.Fprintf(&out, `\u%04X`, unit)
		}
	}
	return out.String()
}

// Call 按 Bee 协议拼接命令，调用机器人上下文中的 32 位 stdcall 框架函数。
// 所有传给框架的内容会先把 GBK 无法表示的 Unicode 字符编码为 UTF-16 \uXXXX；
// 例如“哈哈😄”会转换为“哈哈\uD83D\uDE04”，再将整条命令转为 GBK。
// 返回值按 GBK解码为 UTF-8；返回内存归框架所有。
func (ctx *RobotContext) Call(op int, args ...string) (string, error) {
	if ctx == nil || ctx.transport == nil {
		return "", errors.New("无有效 Bee API IPC transport")
	}
	command := encodeFrameworkText(ctx.command(op, args...))
	result, err := ctx.transport.Call([]byte(command))
	return string(result), err
}

// CallBool 调用框架 API，并将返回值“1”或“真”转换为 true。
func (ctx *RobotContext) CallBool(op int, args ...string) (bool, error) {
	out, err := ctx.Call(op, args...)
	return out == "1" || out == "真", err
}

func boolText(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
func intText(v int) string { return strconv.Itoa(v) }
func activeIDs(ctx *RobotContext, active bool) (string, string) {
	if active {
		return "", ctx.EventID
	}
	return ctx.MessageID, ctx.EventID
}

var OpcodeNames = map[int]string{
	1: "输出日志", 2: "发送频道消息", 3: "发送频道私信", 4: "取频道列表",
	5: "取频道详细信息", 6: "取子频道列表", 7: "取子频道详细信息", 8: "创建子频道",
	9: "取子频道在线人数", 10: "取频道成员详细", 11: "修改子频道", 12: "删除子频道",
	13: "删除频道成员", 14: "取频道身份组列表", 15: "创建频道身份组", 16: "修改频道身份组",
	17: "取是否频道主", 18: "取是否频道管理员", 19: "取是否子频道管理员", 20: "取是否指定频道身份组",
	21: "删除频道身份组", 22: "添加频道成员到身份组", 23: "删除频道成员从身份组", 24: "撤回频道消息",
	25: "发送频道引用消息", 26: "发送频道文字卡片消息", 27: "发送频道自定义消息", 28: "发送频道大图卡片消息",
	29: "置频道禁言指定成员", 30: "置频道全员禁言", 31: "取机器人ID", 32: "取机器人信息",
	33: "发送自适应消息", 34: "发送群消息", 35: "发送群视频消息", 36: "发送群语音消息",
	37: "取框架信息", 38: "发送群Markdown消息", 39: "发送群文字卡片消息", 40: "取某人昵称",
	41: "发送群大图卡片消息", 42: "发送自适应大图卡片消息", 43: "发送群缩略图卡片消息", 44: "发送频道缩略图卡片消息",
	45: "上传图片到图床", 46: "响应按钮事件", 47: "发送频道Markdown消息", 48: "取机器人AppID",
	49: "取某人头像", 50: "取某人头像_QQ", 51: "撤回群消息", 52: "发送好友消息",
	53: "发送好友视频消息", 54: "发送好友语音消息", 55: "发送好友Markdown消息", 56: "发送好友文字卡片消息",
	57: "发送好友大图卡片消息", 58: "发送好友缩略图卡片消息", 59: "撤回好友消息", 60: "发送自适应私信消息",
	61: "发表频道表情表态", 62: "删除频道表情表态", 63: "取频道表情表态用户列表", 64: "取机器人统计信息",
	65: "发送群按钮消息", 66: "发送好友按钮消息", 67: "取机器人Token", 68: "取机器人密钥",
	69: "发送群文件", 70: "发送好友文件", 71: "发送群引用消息", 72: "发送好友引用消息",
	73: "取机器人分享链接", 74: "取群基本信息", 75: "取机器人群内状态", 76: "取入群申请列表",
	77: "处理入群请求", 78: "取群内禁言信息", 79: "取群内某人是否被禁言", 80: "取群内是否全员禁言中",
	81: "设置群成员禁言", 82: "取是否为群管理高层",
}

// ==================== IPC transport ====================
// IPCMessage 定义 Go 工作进程与 Bee C 壳之间传输的 IPC 消息格式。
type IPCMessage struct {
	Type       string   `json:"type"`
	ID         string   `json:"id,omitempty"`
	Event      string   `json:"event,omitempty"`
	ArgsB64    []string `json:"args_b64,omitempty"`
	CommandB64 string   `json:"command_b64,omitempty"`
	ValueB64   string   `json:"value_b64,omitempty"`
	Result     int      `json:"result,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type APITransport interface {
	Call(command []byte) ([]byte, error)
}

type messageWriter func(IPCMessage) error
type messageReader func() (IPCMessage, error)

type IPCClient struct {
	mu     sync.Mutex
	write  messageWriter
	read   messageReader
	nextID uint64
}

var activeTransport APITransport

func setCurrentAPITransport(transport APITransport) { activeTransport = transport }
func currentAPITransport() APITransport             { return activeTransport }

func newIPCClientForTest(write messageWriter, read messageReader) *IPCClient {
	return &IPCClient{write: write, read: read}
}

func (client *IPCClient) Call(command []byte) ([]byte, error) {
	if client == nil || client.write == nil || client.read == nil {
		return nil, errors.New("api_call IPC 未连接")
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	id := fmt.Sprintf("api-%d", atomic.AddUint64(&client.nextID, 1))
	request := IPCMessage{Type: "api_call", ID: id, CommandB64: base64.StdEncoding.EncodeToString(command)}
	if err := client.write(request); err != nil {
		return nil, err
	}
	response, err := client.read()
	if err != nil {
		return nil, err
	}
	if response.Type != "api_result" || response.ID != id {
		return nil, fmt.Errorf("api_result 不匹配: type=%q id=%q, want id=%q", response.Type, response.ID, id)
	}
	if response.Error != "" {
		return nil, errors.New(response.Error)
	}
	value, err := base64.StdEncoding.DecodeString(response.ValueB64)
	if err != nil {
		return nil, fmt.Errorf("解码 api_result: %w", err)
	}
	return value, nil
}
