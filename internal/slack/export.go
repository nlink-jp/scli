package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// rawMsg is the internal representation of a Slack message as returned by
// conversations.history and conversations.replies. It captures fields that are
// not present in the public Message type (BotID, raw file download URLs).
type rawMsg struct {
	TS         string `json:"ts"`
	User       string `json:"user"`
	BotID      string `json:"bot_id"`
	Username   string `json:"username"` // bot display name from API
	Text       string `json:"text"`
	ThreadTS   string `json:"thread_ts"`
	ReplyCount int    `json:"reply_count"`
	Files      []struct {
		ID                 string `json:"id"`
		Name               string `json:"name"`
		Mimetype           string `json:"mimetype"`
		URLPrivateDownload string `json:"url_private_download"`
	} `json:"files"`
	Attachments []struct {
		Fallback  string `json:"fallback"`
		Color     string `json:"color"`
		Pretext   string `json:"pretext"`
		Title     string `json:"title"`
		TitleLink string `json:"title_link"`
		Text      string `json:"text"`
		Fields    []struct {
			Title string `json:"title"`
			Value string `json:"value"`
			Short bool   `json:"short"`
		} `json:"fields"`
		Footer   string `json:"footer"`
		ImageURL string `json:"image_url"`
	} `json:"attachments"`
	Blocks json.RawMessage `json:"blocks"`
}

// ExportChannel fetches the full message history of channelID (including all
// thread replies) and returns a ChannelExport ready for JSON serialisation.
//
// oldest and latest are Slack timestamps (e.g. "1740823200.000000"); either
// may be empty to indicate no lower/upper bound.
//
// When saveDir is non-empty the directory is created and every attached file
// is downloaded into it as "<FileID>_<FileName>"; ExportFile.LocalPath is
// then set to the absolute path of the downloaded file.
func (c *Client) ExportChannel(ctx context.Context, channelID, channelName, oldest, latest, saveDir string) (ChannelExport, error) {
	if saveDir != "" {
		if err := os.MkdirAll(saveDir, 0o700); err != nil {
			return ChannelExport{}, fmt.Errorf("create save dir: %w", err)
		}
	}

	raw, err := c.fetchAllHistory(ctx, channelID, oldest, latest)
	if err != nil {
		return ChannelExport{}, err
	}

	messages := make([]ExportMessage, 0, len(raw))

	for _, m := range raw {
		em, err := c.rawToExportMessage(ctx, m, saveDir)
		if err != nil {
			return ChannelExport{}, err
		}
		messages = append(messages, em)

		if m.ReplyCount > 0 {
			replies, err := c.fetchAllReplies(ctx, channelID, m.TS)
			if err != nil {
				return ChannelExport{}, err
			}
			for _, r := range replies {
				rem, err := c.rawToExportMessage(ctx, r, saveDir)
				if err != nil {
					return ChannelExport{}, err
				}
				messages = append(messages, rem)
			}
		}
	}

	return ChannelExport{
		ExportTimestamp: time.Now().UTC().Format(time.RFC3339),
		ChannelName:     "#" + channelName,
		Messages:        messages,
	}, nil
}

// fetchAllHistory retrieves every message in a channel via cursor-based
// pagination (200 messages per page). Messages are returned in chronological
// order (oldest first).
func (c *Client) fetchAllHistory(ctx context.Context, channelID, oldest, latest string) ([]rawMsg, error) {
	var all []rawMsg
	cursor := ""

	for {
		params := url.Values{
			"channel": {channelID},
			"limit":   {"200"},
		}
		if oldest != "" {
			params.Set("oldest", oldest)
		}
		if latest != "" {
			params.Set("latest", latest)
			params.Set("inclusive", "false")
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Messages         []rawMsg `json:"messages"`
			HasMore          bool     `json:"has_more"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "conversations.history", params, &resp); err != nil {
			return nil, fmt.Errorf("conversations.history: %w", err)
		}

		all = append(all, resp.Messages...)

		if !resp.HasMore || resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	// conversations.history returns newest-first; reverse to chronological.
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}

	return all, nil
}

// fetchAllReplies retrieves every reply in a thread via cursor-based pagination.
// The first message (the parent, which is duplicated by the API) is excluded
// from the returned slice. Messages are in chronological order.
func (c *Client) fetchAllReplies(ctx context.Context, channelID, threadTS string) ([]rawMsg, error) {
	var all []rawMsg
	cursor := ""

	for {
		params := url.Values{
			"channel": {channelID},
			"ts":      {threadTS},
			"limit":   {"200"},
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		var resp struct {
			Messages         []rawMsg `json:"messages"`
			HasMore          bool     `json:"has_more"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := c.get(ctx, "conversations.replies", params, &resp); err != nil {
			return nil, fmt.Errorf("conversations.replies: %w", err)
		}

		all = append(all, resp.Messages...)

		if !resp.HasMore || resp.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = resp.ResponseMetadata.NextCursor
	}

	// conversations.replies includes the parent as the first item; skip it.
	if len(all) > 1 {
		return all[1:], nil
	}
	return nil, nil
}

// rawToExportMessage converts a rawMsg to an ExportMessage, resolving user
// names and optionally downloading attached files.
// thread_timestamp_unix and is_reply are derived from the raw message's
// thread_ts field, matching stail's behaviour.
func (c *Client) rawToExportMessage(ctx context.Context, m rawMsg, saveDir string) (ExportMessage, error) {
	postType := "user"
	if m.BotID != "" {
		postType = "bot"
	}

	userID := m.User
	if userID == "" {
		userID = m.BotID
	}

	var userName string
	switch {
	case m.BotID != "":
		userName = m.Username
	case m.User != "":
		userName = c.ResolveUserName(ctx, m.User)
	}

	files := c.buildExportFiles(ctx, m, saveDir)

	attachments := buildExportAttachments(m)

	return ExportMessage{
		UserID:              userID,
		UserName:            userName,
		PostType:            postType,
		Timestamp:           slackTSToRFC3339(m.TS),
		TimestampUnix:       m.TS,
		Text:                m.Text,
		Files:               files,
		Attachments:         attachments,
		Blocks:              m.Blocks,
		ThreadTimestampUnix: m.ThreadTS,
		IsReply:             m.ThreadTS != "" && m.ThreadTS != m.TS,
	}, nil
}

// buildExportAttachments converts the attachment list from a rawMsg into
// []ExportAttachment.
func buildExportAttachments(m rawMsg) []ExportAttachment {
	if len(m.Attachments) == 0 {
		return nil
	}
	attachments := make([]ExportAttachment, 0, len(m.Attachments))
	for _, a := range m.Attachments {
		fields := make([]ExportAttachmentField, 0, len(a.Fields))
		for _, f := range a.Fields {
			fields = append(fields, ExportAttachmentField{Title: f.Title, Value: f.Value, Short: f.Short})
		}
		attachments = append(attachments, ExportAttachment{
			Fallback: a.Fallback, Color: a.Color, Pretext: a.Pretext,
			Title: a.Title, TitleLink: a.TitleLink, Text: a.Text,
			Fields: fields, Footer: a.Footer, ImageURL: a.ImageURL,
		})
	}
	return attachments
}

// buildExportFiles converts the file list from a rawMsg into []ExportFile,
// downloading each file to saveDir when saveDir is non-empty.
// Download errors are logged to stderr and the export continues (matching
// stail's behaviour).
func (c *Client) buildExportFiles(ctx context.Context, m rawMsg, saveDir string) []ExportFile {
	files := make([]ExportFile, len(m.Files))
	for i, f := range m.Files {
		ef := ExportFile{
			ID:       f.ID,
			Name:     f.Name,
			Mimetype: f.Mimetype,
		}
		if saveDir != "" && f.URLPrivateDownload != "" {
			destName := f.ID + "_" + f.Name
			destPath := filepath.Join(saveDir, destName)
			if err := c.downloadFileTo(ctx, f.URLPrivateDownload, destPath); err != nil {
				fmt.Fprintf(os.Stderr, "warn: download %s: %v\n", f.Name, err)
			} else {
				abs, err := filepath.Abs(destPath)
				if err != nil {
					abs = destPath
				}
				ef.LocalPath = abs
			}
		}
		files[i] = ef
	}
	return files
}

// downloadFileTo fetches a Slack private file URL using Bearer authentication
// and writes the contents to destPath. It retries on HTTP 429 responses,
// honouring the Retry-After header.
func (c *Client) downloadFileTo(ctx context.Context, fileURL, destPath string) error {
	const maxRetries = 3

	for attempt := range maxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			wait := retryAfterDuration(resp.Header.Get("Retry-After"))
			resp.Body.Close() //nolint:errcheck
			if attempt == maxRetries-1 {
				break
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close() //nolint:errcheck
			return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
		}

		f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec
		if err != nil {
			resp.Body.Close() //nolint:errcheck
			return fmt.Errorf("create file: %w", err)
		}
		_, copyErr := io.Copy(f, resp.Body)
		resp.Body.Close() //nolint:errcheck
		f.Close()         //nolint:errcheck
		if copyErr != nil {
			return fmt.Errorf("write file: %w", copyErr)
		}
		return nil
	}

	return fmt.Errorf("rate limited downloading file: still throttled after %d attempts", maxRetries)
}

// slackTSToRFC3339 converts a Slack timestamp string ("1234567890.123456") to
// an RFC3339 UTC string. Returns the original string on parse failure.
func slackTSToRFC3339(ts string) string {
	f, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return ts
	}
	sec := int64(f)
	nsec := int64((f - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC().Format(time.RFC3339)
}
