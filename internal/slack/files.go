package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// UploadFile uploads a local file and shares it to channelID with an optional
// message as the initial comment.
// Uses the current Slack file upload API (files.getUploadURLExternal +
// files.completeUploadExternal).
func (c *Client) UploadFile(ctx context.Context, channelID, filePath, message, threadTS string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only file; close error is not actionable

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	filename := filepath.Base(filePath)

	// Step 1: get upload URL.
	uploadURL, fileID, err := c.getUploadURL(ctx, filename, len(data))
	if err != nil {
		return err
	}

	// Step 2: upload file content to the provided URL.
	if err := c.uploadToURL(ctx, uploadURL, data); err != nil {
		return err
	}

	// Step 3: complete upload and share to channel.
	return c.completeUpload(ctx, channelID, fileID, filename, message, threadTS)
}

func (c *Client) getUploadURL(ctx context.Context, filename string, length int) (uploadURL, fileID string, err error) {
	params := url.Values{
		"filename": {filename},
		"length":   {fmt.Sprintf("%d", length)},
	}

	var resp struct {
		UploadURL string `json:"upload_url"`
		FileID    string `json:"file_id"`
	}

	if err := c.get(ctx, "files.getUploadURLExternal", params, &resp); err != nil {
		return "", "", fmt.Errorf("files.getUploadURLExternal: %w", err)
	}
	return resp.UploadURL, resp.FileID, nil
}

func (c *Client) uploadToURL(ctx context.Context, uploadURL string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload file: unexpected status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) completeUpload(ctx context.Context, channelID, fileID, filename, message, threadTS string) error {
	filesJSON, err := json.Marshal([]map[string]string{{"id": fileID, "title": filename}})
	if err != nil {
		return fmt.Errorf("marshal files: %w", err)
	}

	params := url.Values{
		"files":      {string(filesJSON)},
		"channel_id": {channelID},
	}
	if message != "" {
		params.Set("initial_comment", message)
	}
	if threadTS != "" {
		params.Set("thread_ts", threadTS)
	}

	var resp struct{}
	if err := c.post(ctx, "files.completeUploadExternal", params, &resp); err != nil {
		return fmt.Errorf("files.completeUploadExternal: %w", err)
	}
	return nil
}
