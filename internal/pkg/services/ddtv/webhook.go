package ddtv

import (
	"time"
)

type WebHook struct {
	ID       string    `json:"id,omitempty"`
	Type     int       `json:"type,omitempty"`
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
	BroadcastType      string             `json:"broadcast_type,omitempty"`
	NeedP2P            string             `json:"need_p2p,omitempty"`
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
