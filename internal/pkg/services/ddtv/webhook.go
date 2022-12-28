package ddtv

import (
	"dalian-bot/internal/pkg/services/discord"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"reflect"
	"time"
)

func (wh WebHook) DigestEmbed() *discordgo.MessageEmbed {
	switch wh.Type {
	case HookSpaceIsInsufficientWarn:
		return &discordgo.MessageEmbed{
			Title:       "DDTV Insufficient Disk Storage WARNING",
			Description: HookSpaceIsInsufficientWarn.MessagePrompt("", 0),
			Timestamp:   time.Now().Format(time.RFC3339),
			Color:       discord.EmbedColorDanger,
		}
	}
	embed := &discordgo.MessageEmbed{
		Title:       "DDTV Webhook Update",
		Description: wh.Type.MessagePrompt(wh.RoomInfo.Uname, wh.RoomInfo.RoomID),
		Author: &discordgo.MessageEmbedAuthor{
			URL:     fmt.Sprintf("https://space.bilibili.com/%d", wh.UserInfo.UID),
			Name:    fmt.Sprintf("%s [%d]", wh.UserInfo.Name, wh.UserInfo.UID),
			IconURL: wh.RoomInfo.Face,
		},
		Provider: &discordgo.MessageEmbedProvider{
			URL:  "https://ddtv.pro",
			Name: "DDTV",
		},
		Image: &discordgo.MessageEmbedImage{
			URL: wh.RoomInfo.CoverFromUser,
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Color:     discord.EmbedColorNormal,
		URL:       fmt.Sprintf("https://live.bilibili.com/%d", wh.RoomInfo.RoomID),
		Fields: []*discordgo.MessageEmbedField{{
			Name:   wh.RoomInfo.Title,
			Value:  fmt.Sprintf("Code:%d", wh.Type),
			Inline: false,
		}},
	}
	return embed
}

type WebHook struct {
	ID       string    `json:"id,omitempty"`
	Type     HookType  `json:"type,omitempty"`
	Uid      int64     `json:"uid,omitempty"`
	HookTime time.Time `json:"hook_time,omitempty"`
	UserInfo UserInfo  `json:"user_info,omitempty"`
	RoomInfo RoomInfo  `json:"room_info,omitempty"`
}

type UserInfo struct {
	Name      string `json:"name,omitempty"`
	Face      string `json:"face,omitempty"`
	UID       int64  `json:"uid,omitempty"`
	Sign      string `json:"sign,omitempty"`
	Attention int    `json:"attention,omitempty"`
}
type RoomInfo struct {
	Title              string             `json:"title,omitempty"`
	Description        string             `json:"description,omitempty"`
	Attention          int                `json:"attention,omitempty"`
	RoomID             int                `json:"room_id,omitempty"`
	UID                int64              `json:"uid,omitempty"`
	Online             int                `json:"online,omitempty"`
	LiveTime           int64              `json:"live_time,omitempty"`
	LiveStatus         int                `json:"live_status,omitempty"`
	ShortID            int                `json:"short_id,omitempty"`
	Area               int                `json:"area,omitempty"`
	AreaName           string             `json:"area_name,omitempty"`
	AreaV2ID           int                `json:"area_v2_id"`
	AreaV2Name         string             `json:"area_v2_name,omitempty"`
	AreaV2ParentName   string             `json:"area_v2_parent_name,omitempty"`
	AreaV2ParentID     int                `json:"area_v2_parent_id,omitempty"`
	Uname              string             `json:"uname,omitempty"`
	Face               string             `json:"face,omitempty"`
	TagName            string             `json:"tag_name,omitempty"`
	Tags               string             `json:"tags,omitempty"`
	CoverFromUser      string             `json:"cover_from_user,omitempty"`
	Keyframe           string             `json:"keyframe,omitempty"`
	LockTill           string             `json:"lock_till,omitempty"`
	HiddenTill         string             `json:"hidden_till,omitempty"`
	BroadcastType      int                `json:"broadcast_type,omitempty"`
	NeedP2P            int                `json:"need_p2p,omitempty"`
	IsHidden           bool               `json:"is_hidden,omitempty"`
	IsLocked           bool               `json:"is_locked,omitempty"`
	IsPortrait         bool               `json:"is_portrait,omitempty"`
	Encrypted          bool               `json:"encrypted,omitempty"`
	PwdVerified        bool               `json:"pwd_verified,omitempty"`
	IsSP               int                `json:"is_sp,omitempty"`
	SpecialType        int                `json:"special_type,omitempty"`
	RoomStatus         int                `json:"roomStatus,omitempty"`
	RoundStatus        int                `json:"roundStatus,omitempty"`
	URL                string             `json:"url,omitempty"`
	IsAutoRec          bool               `json:"IsAutoRec,omitempty"`
	IsRemind           bool               `json:"IsRemind,omitempty"`
	IsRecDanmu         bool               `json:"IsRecDanmu,omitempty"`
	Level              int                `json:"level,omitempty"`
	Sex                string             `json:"sex,omitempty"`
	Sign               string             `json:"sign,omitempty"`
	DownloadedFileInfo DownloadedFileInfo `json:"DownloadedFileInfo,omitempty"`
	Shell              string             `json:"Shell,omitempty"`
}

type DownloadedFileInfo struct {
	FlvFile   string `json:"FlvFile,omitempty"`
	Mp4File   string `json:"Mp4File,omitempty"`
	DanMuFile string `json:"DanMuFile,omitempty"`
	SCFile    string `json:"SCFile,omitempty"`
	GuardFile string `json:"GuardFile,omitempty"`
	GiftFile  string `json:"GiftFile,omitempty"`
}

type HookType int

func (h HookType) Value() int {
	return int(h)
}

func (h HookType) MessagePrompt(username string, uid int) string {
	switch h {
	case HookStartLive:
		return fmt.Sprintf("Live channel %s[%d] is online.", username, uid)
	case HookStopLive:
		return fmt.Sprintf("Live channel %s[%d] is offline.", username, uid)
	case HookStartRec:
		return fmt.Sprintf("DDTV starts recording live channel %s[%d].", username, uid)
	case HookRecComplete:
		return fmt.Sprintf("DDTV completes recording live channel %s[%d].", username, uid)
	case HookCancelRec:
		return fmt.Sprintf("DDTV cancels recording live channel %s[%d].", username, uid)
	case HookTranscodingComlete:
		return fmt.Sprintf("DDTV completes transcoding video for live channel %s[%d].", username, uid)
	case HookSaveDanmuComplete:
		return fmt.Sprintf("DDTV completes saving danmu for live channel %s[%d].", username, uid)
	case HookSaveSCComplete:
		return fmt.Sprintf("DDTV completes saving superchat for live channel %s[%d].", username, uid)
	case HookSaveGiftComplete:
		return fmt.Sprintf("DDTV completes saving gift info for live channel %s[%d].", username, uid)
	case HookSaveGuardComplete:
		return fmt.Sprintf("DDTV completes saving guard info for live channel %s[%d].", username, uid)
	case HookRunShellComplete:
		return fmt.Sprintf("DDTV completes a shell task for live channel %s[%d].", username, uid)
	case HookDownloadEndMissionSuccess:
		return fmt.Sprintf("DDTV completes a download task for live channel %s[%d].", username, uid)
	case HookSpaceIsInsufficientWarn:
		return "DDTV detects a low disk storage!!"
	default:
		return fmt.Sprintf("Unknown Hook Type: %s", reflect.TypeOf(h))
	}
}

const (
	HookStartLive HookType = iota
	HookStopLive
	HookStartRec
	HookRecComplete
	HookCancelRec
	HookTranscodingComlete
	HookSaveDanmuComplete
	HookSaveSCComplete
	HookSaveGiftComplete
	HookSaveGuardComplete
	HookRunShellComplete
	HookDownloadEndMissionSuccess
	HookSpaceIsInsufficientWarn
)
