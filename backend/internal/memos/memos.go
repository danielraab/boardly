// Package memos maps the notes of an app token's note scope to the Memos v1
// API that the MoeMemos Android app uses: a memo is a note, its content is
// the note as Markdown with the title as a leading level-1 heading, its
// pinned flag is the note's, and its state ARCHIVED is the note's archived
// flag. Memos are always private, and attachments aren't supported yet. The
// shapes follow MoeMemos' MemosV1Api.kt (mudkipme/MoeMemosAndroid, October
// 2026), which talks to Memos 0.27 to 0.31.
package memos

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"boardly/internal/notes"
	"boardly/internal/users"
)

// Version is the Memos version Boardly reports: the newest one MoeMemos
// supports, so it connects without an "unsupported version" warning.
const Version = "0.31.0"

// gRPC status codes of Memos errors.
const (
	CodeInvalidArgument = 3
	CodeNotFound        = 5
	CodeUnauthenticated = 16
)

// Error is a Memos error: {"code", "message", "details"} with an HTTP status.
type Error struct {
	Status  int
	Code    int
	Message string
}

func (e *Error) Error() string { return e.Message }

func invalid(message string) *Error {
	return &Error{http.StatusBadRequest, CodeInvalidArgument, message}
}

// Errors of the service.
var (
	ErrNotFound        = &Error{http.StatusNotFound, CodeNotFound, "Not found."}
	ErrUnauthenticated = &Error{http.StatusUnauthorized, CodeUnauthenticated, "Unauthenticated."}
	ErrAttachments     = invalid("Boardly doesn't support attachments yet.")
	errPageToken       = invalid("Invalid page token.")
	errState           = invalid("The state must be NORMAL or ARCHIVED.")
	errContent         = invalid("A memo needs a content.")
	errTooLarge        = invalid("The memo is too large.")
)

// Time is a time as Memos writes it: RFC 3339 in UTC with milliseconds, so
// the same time is always the same string.
type Time time.Time

// MarshalJSON writes the time as a JSON string.
func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z") + `"`), nil
}

// UnmarshalJSON reads an RFC 3339 time.
func (t *Time) UnmarshalJSON(data []byte) error {
	var v time.Time
	if err := v.UnmarshalJSON(data); err != nil {
		return err
	}
	*t = Time(v)
	return nil
}

// UserName is the Memos name of a user, "users/<id>".
func UserName(id string) string { return "users/" + id }

// User is a Memos user.
type User struct {
	Name        string `json:"name"`
	Role        string `json:"role"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreateTime  Time   `json:"createTime"`
	UpdateTime  Time   `json:"updateTime"`
}

// UserOf is the Memos user of a Boardly user, with the email as username
// and as display name when there is no name.
func UserOf(u users.User) User {
	display := u.Email
	if u.Name != nil && strings.TrimSpace(*u.Name) != "" {
		display = *u.Name
	}
	return User{
		Name: UserName(u.ID), Role: "USER", Username: u.Email, Email: u.Email, DisplayName: display,
		State: "NORMAL", CreateTime: Time(u.CreatedAt), UpdateTime: Time(u.UpdatedAt),
	}
}

// Profile is the public instance profile MoeMemos reads to pick its API.
type Profile struct {
	Version     string `json:"version"`
	Mode        string `json:"mode"`
	InstanceURL string `json:"instanceUrl"`
}

// GeneralSetting is the general user setting; memos are always private.
type GeneralSetting struct {
	Name           string `json:"name"`
	GeneralSetting struct {
		Locale         string `json:"locale"`
		MemoVisibility string `json:"memoVisibility"`
		Theme          string `json:"theme"`
	} `json:"generalSetting"`
}

// SettingOf is the general setting of the user with the given ID.
func SettingOf(userID string) GeneralSetting {
	s := GeneralSetting{Name: UserName(userID) + "/settings/GENERAL"}
	s.GeneralSetting.MemoVisibility = "PRIVATE"
	return s
}

// Stats is a user's statistics; Boardly counts no tags.
type Stats struct {
	Name     string         `json:"name"`
	TagCount map[string]int `json:"tagCount"`
}

// StatsOf is the statistics of the user with the given ID.
func StatsOf(userID string) Stats {
	return Stats{Name: UserName(userID) + "/stats", TagCount: map[string]int{}}
}

// Memo is a note as a Memos memo.
type Memo struct {
	Name        string     `json:"name"`
	State       string     `json:"state"`
	Creator     string     `json:"creator"`
	CreateTime  Time       `json:"createTime"`
	UpdateTime  Time       `json:"updateTime"`
	DisplayTime Time       `json:"displayTime"`
	Content     string     `json:"content"`
	Visibility  string     `json:"visibility"`
	Pinned      bool       `json:"pinned"`
	Attachments []struct{} `json:"attachments"`
	Tags        []string   `json:"tags"`
	Relations   []struct{} `json:"relations"`
}

// Memo states.
const (
	StateNormal   = "NORMAL"
	StateArchived = "ARCHIVED"
)

// memoName is the Memos name of a note, "memos/<id>".
func memoName(id string) string { return "memos/" + id }

// Content is a note's memo content: its content as Markdown, preceded by
// its title as a level-1 heading.
func Content(n notes.Note) string {
	return notes.ToMarkdown(notes.WithTitleHeading(n.Title, n.Content))
}

// MemoOf is the memo of a note of the user with the given ID.
func MemoOf(userID string, n notes.Note) Memo {
	state := StateNormal
	if n.Archived {
		state = StateArchived
	}
	return Memo{
		Name: memoName(n.ID), State: state, Creator: UserName(userID),
		CreateTime: Time(n.CreatedAt), UpdateTime: Time(n.UpdatedAt), DisplayTime: Time(n.CreatedAt),
		Content: Content(n), Visibility: "PRIVATE", Pinned: n.Pinned,
		Attachments: []struct{}{}, Tags: []string{}, Relations: []struct{}{},
	}
}

// pageToken and pageOffset encode the offset of a page as an opaque token.
func pageToken(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func pageOffset(token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, errPageToken
	}
	n, err := strconv.Atoi(string(raw))
	if err != nil || n < 0 {
		return 0, errPageToken
	}
	return n, nil
}

// storeError maps the errors of the note store to Memos errors; other
// errors pass through.
func storeError(err error) error {
	switch {
	case errors.Is(err, notes.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, notes.ErrInvalidDocument):
		return errTooLarge
	case errors.Is(err, notes.ErrInvalidTitle):
		return invalid("The title is too long.")
	}
	return err
}
