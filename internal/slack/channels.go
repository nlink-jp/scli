package slack

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"
)

const channelCacheTTL = time.Hour

// ListChannels returns all channels the authenticated user is a member of.
// Results are cached on disk for channelCacheTTL to avoid repeated paginated fetches.
func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	if c.cacheDir != "" {
		if channels, ok := loadCache[[]Channel](filepath.Join(c.cacheDir, "channels.json"), channelCacheTTL); ok {
			return channels, nil
		}
	}

	var all []Channel
	cursor := ""

	for {
		params := url.Values{
			"types":            {"public_channel,private_channel"},
			"exclude_archived": {"true"},
			"limit":            {"200"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Channels []struct {
				ID         string `json:"id"`
				Name       string `json:"name"`
				IsMember   bool   `json:"is_member"`
				IsArchived bool   `json:"is_archived"`
				Purpose    struct {
					Value string `json:"value"`
				} `json:"purpose"`
				NumMembers  int `json:"num_members"`
				UnreadCount int `json:"unread_count"`
			} `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "conversations.list", params, &resp); err != nil {
			return nil, fmt.Errorf("conversations.list: %w", err)
		}

		for _, ch := range resp.Channels {
			all = append(all, Channel{
				ID:          ch.ID,
				Name:        ch.Name,
				Purpose:     ch.Purpose.Value,
				IsMember:    ch.IsMember,
				UnreadCount: ch.UnreadCount,
			})
		}

		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	if c.cacheDir != "" {
		saveCache(filepath.Join(c.cacheDir, "channels.json"), all)
	}
	return all, nil
}

// ListJoinedChannels returns only the channels the authenticated user is a member of.
// It calls users.conversations instead of conversations.list, which avoids fetching
// the full workspace channel list and is therefore more efficient.
// Results are cached on disk for channelCacheTTL.
func (c *Client) ListJoinedChannels(ctx context.Context) ([]Channel, error) {
	cacheFile := filepath.Join(c.cacheDir, "joined_channels.json")
	if c.cacheDir != "" {
		if channels, ok := loadCache[[]Channel](cacheFile, channelCacheTTL); ok {
			return channels, nil
		}
	}

	var all []Channel
	cursor := ""

	for {
		params := url.Values{
			"types":            {"public_channel,private_channel"},
			"exclude_archived": {"true"},
			"limit":            {"200"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Channels []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Purpose struct {
					Value string `json:"value"`
				} `json:"purpose"`
				UnreadCount int `json:"unread_count"`
			} `json:"channels"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "users.conversations", params, &resp); err != nil {
			return nil, fmt.Errorf("users.conversations: %w", err)
		}

		for _, ch := range resp.Channels {
			all = append(all, Channel{
				ID:          ch.ID,
				Name:        ch.Name,
				Purpose:     ch.Purpose.Value,
				IsMember:    true,
				UnreadCount: ch.UnreadCount,
			})
		}

		if resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	if c.cacheDir != "" {
		saveCache(cacheFile, all)
	}
	return all, nil
}

// GetChannelHistory returns up to limit messages from the channel identified
// by channelID, in chronological order (oldest first).
// If oldest is non-empty, only messages after that Slack timestamp are returned.
func (c *Client) GetChannelHistory(ctx context.Context, channelID string, limit int, oldest string) ([]Message, error) {
	params := url.Values{
		"channel": {channelID},
		"limit":   {fmt.Sprintf("%d", limit)},
	}
	if oldest != "" {
		params.Set("oldest", oldest)
		params.Set("inclusive", "false")
	}

	var resp struct {
		Messages []struct {
			TS         string `json:"ts"`
			User       string `json:"user"`
			BotID      string `json:"bot_id"`
			Username   string `json:"username"` // bot display name
			Text       string `json:"text"`
			ThreadTS   string `json:"thread_ts"`
			ReplyCount int    `json:"reply_count"`
			Files      []struct {
				ID                 string `json:"id"`
				Name               string `json:"name"`
				Mimetype           string `json:"mimetype"`
				URLPrivateDownload string `json:"url_private_download"`
			} `json:"files"`
		} `json:"messages"`
		HasMore bool `json:"has_more"`
	}

	if err := c.get(ctx, "conversations.history", params, &resp); err != nil {
		return nil, fmt.Errorf("conversations.history: %w", err)
	}

	// Slack returns newest-first; reverse to chronological order.
	msgs := make([]Message, len(resp.Messages))
	for i, m := range resp.Messages {
		files := make([]File, len(m.Files))
		for j, f := range m.Files {
			files[j] = File{
				ID:       f.ID,
				Name:     f.Name,
				MIMEType: f.Mimetype,
				URL:      f.URLPrivateDownload,
			}
		}
		msgs[len(resp.Messages)-1-i] = Message{
			Timestamp:   m.TS,
			UserID:      m.User,
			BotUsername: m.Username,
			Text:        m.Text,
			ThreadTS:    m.ThreadTS,
			ReplyCount:  m.ReplyCount,
			Files:       files,
		}
	}

	return msgs, nil
}

// ChannelInfo holds per-channel metadata returned by conversations.info.
type ChannelInfo struct {
	LastRead    string
	UnreadCount int
}

// GetChannelDetail returns detailed metadata for a channel, including topic,
// member count, creator, and creation time.
func (c *Client) GetChannelDetail(ctx context.Context, channelID string) (ChannelDetail, error) {
	params := url.Values{"channel": {channelID}}

	var resp struct {
		Channel struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			IsArchived bool   `json:"is_archived"`
			IsGeneral  bool   `json:"is_general"`
			IsPrivate  bool   `json:"is_private"`
			Created    int64  `json:"created"`
			Creator    string `json:"creator"`
			NumMembers int    `json:"num_members"`
			Topic      struct {
				Value string `json:"value"`
			} `json:"topic"`
			Purpose struct {
				Value string `json:"value"`
			} `json:"purpose"`
		} `json:"channel"`
	}

	if err := c.get(ctx, "conversations.info", params, &resp); err != nil {
		return ChannelDetail{}, fmt.Errorf("conversations.info: %w", err)
	}

	ch := resp.Channel
	return ChannelDetail{
		Channel: Channel{
			ID:      ch.ID,
			Name:    ch.Name,
			Purpose: ch.Purpose.Value,
		},
		Topic:      ch.Topic.Value,
		NumMembers: ch.NumMembers,
		Creator:    ch.Creator,
		Created:    ch.Created,
		IsArchived: ch.IsArchived,
		IsGeneral:  ch.IsGeneral,
		IsPrivate:  ch.IsPrivate,
	}, nil
}

// GetChannelInfo returns metadata for the given channel, including last-read
// timestamp and unread message count.
func (c *Client) GetChannelInfo(ctx context.Context, channelID string) (ChannelInfo, error) {
	params := url.Values{"channel": {channelID}}

	var resp struct {
		Channel struct {
			LastRead    string `json:"last_read"`
			UnreadCount int    `json:"unread_count"`
		} `json:"channel"`
	}

	if err := c.get(ctx, "conversations.info", params, &resp); err != nil {
		return ChannelInfo{}, fmt.Errorf("conversations.info: %w", err)
	}
	return ChannelInfo{
		LastRead:    resp.Channel.LastRead,
		UnreadCount: resp.Channel.UnreadCount,
	}, nil
}

// GetChannelLastRead returns the last-read timestamp for the given channel.
// Returns an empty string if the information is unavailable.
func (c *Client) GetChannelLastRead(ctx context.Context, channelID string) (string, error) {
	info, err := c.GetChannelInfo(ctx, channelID)
	if err != nil {
		return "", err
	}
	return info.LastRead, nil
}

// PostMessage sends a text message to the given channel.
// Returns the message timestamp on success.
func (c *Client) PostMessage(ctx context.Context, channelID, text string) (string, error) {
	params := url.Values{
		"channel": {channelID},
		"text":    {text},
	}

	var resp struct {
		TS string `json:"ts"`
	}

	if err := c.post(ctx, "chat.postMessage", params, &resp); err != nil {
		return "", fmt.Errorf("chat.postMessage: %w", err)
	}
	return resp.TS, nil
}

// PostThreadReply sends a message as a reply to a thread.
func (c *Client) PostThreadReply(ctx context.Context, channelID, threadTS, text string) (string, error) {
	params := url.Values{
		"channel":   {channelID},
		"text":      {text},
		"thread_ts": {threadTS},
	}

	var resp struct {
		TS string `json:"ts"`
	}

	if err := c.post(ctx, "chat.postMessage", params, &resp); err != nil {
		return "", fmt.Errorf("chat.postMessage (thread): %w", err)
	}
	return resp.TS, nil
}

// PostMessageWithBlocks sends a message with Block Kit blocks to the given channel.
// text is used as the notification fallback; blocksJSON is a JSON-encoded array of block objects.
func (c *Client) PostMessageWithBlocks(ctx context.Context, channelID, text, blocksJSON string) (string, error) {
	params := url.Values{
		"channel": {channelID},
		"text":    {text},
		"blocks":  {blocksJSON},
	}

	var resp struct {
		TS string `json:"ts"`
	}

	if err := c.post(ctx, "chat.postMessage", params, &resp); err != nil {
		return "", fmt.Errorf("chat.postMessage (blocks): %w", err)
	}
	return resp.TS, nil
}

// PostThreadReplyWithBlocks sends a Block Kit message as a reply to a thread.
func (c *Client) PostThreadReplyWithBlocks(ctx context.Context, channelID, threadTS, text, blocksJSON string) (string, error) {
	params := url.Values{
		"channel":   {channelID},
		"text":      {text},
		"thread_ts": {threadTS},
		"blocks":    {blocksJSON},
	}

	var resp struct {
		TS string `json:"ts"`
	}

	if err := c.post(ctx, "chat.postMessage", params, &resp); err != nil {
		return "", fmt.Errorf("chat.postMessage (blocks thread): %w", err)
	}
	return resp.TS, nil
}
