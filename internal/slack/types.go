// Package slack provides a Slack Web API client for scli.
package slack

import "encoding/json"

// Channel represents a Slack channel (public, private, or DM).
type Channel struct {
	ID          string
	Name        string
	Purpose     string
	IsIM        bool // true for direct messages
	IsMPIM      bool // true for group direct messages
	IsMember    bool
	UnreadCount int
	UserID      string // for DMs: the other user's Slack ID
}

// Message represents a single Slack message.
type Message struct {
	Timestamp   string // Slack timestamp (e.g. "1234567890.123456")
	UserID      string
	UserName    string // resolved display name (may be empty if unresolved)
	BotUsername string // set for bot messages that have no user ID
	Text        string
	ThreadTS    string // non-empty if this is a thread reply or thread parent
	ReplyCount  int    // number of replies (>0 means this is a thread parent)
	Files       []File
	Replies     []Message // populated when thread replies are fetched
}

// File represents a file attached to a message.
type File struct {
	ID       string
	Name     string
	MIMEType string
	URL      string
}

// SearchResult represents a single search match returned by search.messages.
type SearchResult struct {
	ChannelID   string
	ChannelName string
	Message     Message
}

// User represents a Slack workspace member.
type User struct {
	ID          string
	Name        string // username (handle)
	DisplayName string // profile display name
	RealName    string
	IsBot       bool
	IsDeleted   bool
}

// ChannelDetail extends Channel with detailed metadata returned by conversations.info.
type ChannelDetail struct {
	Channel
	Topic      string
	NumMembers int
	Creator    string // creator user ID
	Created    int64  // Unix timestamp
	IsArchived bool
	IsGeneral  bool
	IsPrivate  bool
}

// UserProfile extends User with detailed profile fields returned by users.info.
type UserProfile struct {
	User
	Title    string
	Email    string
	Phone    string
	Status   string // status text
	Emoji    string // status emoji
	Timezone string // tz label, e.g. "Asia/Tokyo"
}

// ExportFile holds metadata for a file attached to an exported message.
// LocalPath is non-empty only when the file was downloaded via --save-dir.
type ExportFile struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Mimetype  string `json:"mimetype"`
	LocalPath string `json:"local_path"`
}

// ExportAttachment holds a legacy rich attachment from a Slack message.
type ExportAttachment struct {
	Fallback  string                  `json:"fallback,omitempty"`
	Color     string                  `json:"color,omitempty"`
	Pretext   string                  `json:"pretext,omitempty"`
	Title     string                  `json:"title,omitempty"`
	TitleLink string                  `json:"title_link,omitempty"`
	Text      string                  `json:"text,omitempty"`
	Fields    []ExportAttachmentField `json:"fields,omitempty"`
	Footer    string                  `json:"footer,omitempty"`
	ImageURL  string                  `json:"image_url,omitempty"`
}

// ExportAttachmentField is a key-value pair inside a legacy attachment.
type ExportAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// ExportMessage is the per-message record written by channel export.
// The schema is compatible with scat and stail exports.
type ExportMessage struct {
	UserID              string             `json:"user_id"`
	UserName            string             `json:"user_name,omitempty"`
	PostType            string             `json:"post_type"` // "user" or "bot"
	Timestamp           string             `json:"timestamp"` // RFC3339
	TimestampUnix       string             `json:"timestamp_unix"`
	Text                string             `json:"text"`
	Files               []ExportFile       `json:"files"`
	Attachments         []ExportAttachment `json:"attachments,omitempty"`
	Blocks              json.RawMessage    `json:"blocks,omitempty"`
	ThreadTimestampUnix string             `json:"thread_timestamp_unix,omitempty"`
	IsReply             bool               `json:"is_reply"`
}

// ChannelExport is the top-level structure of an exported channel.
type ChannelExport struct {
	ExportTimestamp string          `json:"export_timestamp"`
	ChannelName     string          `json:"channel_name"`
	Messages        []ExportMessage `json:"messages"`
}
