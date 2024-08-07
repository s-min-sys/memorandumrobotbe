package server

import "time"

//
//
//

type Memo struct {
	ID       string        `json:"id" yaml:"id"`
	Name     string        `json:"name" yaml:"name"`
	Info     string        `json:"info,omitempty" yaml:"info,omitempty"`
	Span     time.Duration `json:"span" yaml:"span"`
	Disabled bool          `json:"disabled,omitempty" yaml:"disabled,omitempty"`
}

type MemoTouchInfo struct {
	At   time.Time `json:"at" yaml:"at"`
	Info string    `json:"info,omitempty"`
}

type MemoRecord struct {
	ID                   string         `json:"id" yaml:"id"`
	LastTouchAt          time.Time      `json:"last_touch_at,omitempty" yaml:"last_touch_at,omitempty"`
	LastSuccessTouchInfo *MemoTouchInfo `json:"last_success_touch_info,omitempty" yaml:"last_success_touch_info,omitempty"`
	LastFailTouchInfo    *MemoTouchInfo `json:"last_fail_touch_info,omitempty" yaml:"last_fail_touch_info,omitempty"`
}

//
//
//

type AddRequest struct {
	Name            string `json:"name"`
	Info            string `json:"info"`
	InternalSeconds int    `json:"internal_seconds"`
}

func (req AddRequest) Valid() bool {
	return req.Name != "" && req.InternalSeconds > 1
}

type DelRequest struct {
	ID string `json:"id"`
}

func (req DelRequest) Valid() bool {
	return req.ID != ""
}

type TouchRequest struct {
	ID       string `json:"id"`
	FailFlag bool   `json:"fail_flag,omitempty"`
	Info     string `json:"info,omitempty"`
}

func (req TouchRequest) Valid() bool {
	return req.ID != ""
}

type MemoAllItem struct {
	ID       string `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name"`
	Info     string `json:"info,omitempty" yaml:"info,omitempty"`
	Span     string `json:"span" yaml:"span"`
	Disabled bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`

	LastTouchAt          time.Time      `json:"last_touch_at,omitempty" yaml:"last_touch_at,omitempty"`
	LastSuccessTouchInfo *MemoTouchInfo `json:"last_success_touch_info,omitempty" yaml:"last_success_touch_info,omitempty"`
	LastFailTouchInfo    *MemoTouchInfo `json:"last_fail_touch_info,omitempty" yaml:"last_fail_touch_info,omitempty"`

	UntilExpired string `json:"until_expired,omitempty" yaml:"until_expired,omitempty"`
	Expired      bool   `json:"expired,omitempty" yaml:"expired,omitempty"`
}
