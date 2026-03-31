// Package output provides formatted output rendering for scli.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/nlink-jp/scli/internal/slack"
)

// Printer renders scli output to an io.Writer with optional color and JSON support.
type Printer struct {
	w       io.Writer
	jsonOut bool
	// Color functions — replaced with no-op functions when color is disabled.
	channelName func(a ...interface{}) string
	userName    func(a ...interface{}) string
	timestamp   func(a ...interface{}) string
	marker      func(a ...interface{}) string
	dimText     func(a ...interface{}) string
	msgHeader   func(a ...interface{}) string
}

// New returns a Printer that writes to w.
// jsonOut: output raw JSON instead of human-readable text.
// noColor: disable ANSI color codes (auto-disabled when w is not a TTY).
func New(w io.Writer, jsonOut, noColor bool) *Printer {
	p := &Printer{w: w, jsonOut: jsonOut}

	if noColor || !isTTY(w) {
		plain := func(a ...interface{}) string { return fmt.Sprint(a...) }
		p.channelName = plain
		p.userName = plain
		p.timestamp = plain
		p.marker = plain
		p.dimText = plain
		p.msgHeader = plain
	} else {
		plain := func(a ...interface{}) string { return fmt.Sprint(a...) }
		p.channelName = color.New(color.FgCyan, color.Bold).SprintFunc()
		p.userName = color.New(color.FgCyan).SprintFunc()
		p.timestamp = color.New(color.FgCyan).SprintFunc()
		p.marker = color.New(color.FgYellow, color.Bold).SprintFunc()
		p.dimText = plain
		p.msgHeader = color.New(color.Bold).SprintFunc()
	}

	return p
}

// Channels renders a list of channels.
func (p *Printer) Channels(channels []slack.Channel, defaultMarker string) error {
	if p.jsonOut {
		return p.writeJSON(channels)
	}
	for _, ch := range channels {
		prefix := "  "
		if ch.ID == defaultMarker {
			prefix = p.marker("* ")
		}
		name := p.channelName("#" + ch.Name)
		purpose := ""
		if ch.Purpose != "" {
			purpose = p.dimText("  — " + ch.Purpose)
		}
		fmt.Fprintf(p.w, "%s%s%s\n", prefix, name, purpose)
	}
	return nil
}

// Messages renders a list of messages, including any thread replies nested
// under their parent.
func (p *Printer) Messages(messages []slack.Message) error {
	if p.jsonOut {
		return p.writeJSON(messages)
	}
	for _, m := range messages {
		p.renderMessage(m, "")
		for _, r := range m.Replies {
			fmt.Fprintln(p.w)
			p.renderMessage(r, "  ")
		}
		fmt.Fprintln(p.w)
	}
	return nil
}

// renderMessage prints a single message with the given line prefix (used for
// indenting thread replies).
func (p *Printer) renderMessage(m slack.Message, prefix string) {
	ts := formatTimestamp(m.Timestamp)
	name := m.UserName
	if name == "" {
		name = m.BotUsername
	}
	if name == "" {
		name = m.UserID
	}
	fmt.Fprintf(p.w, "%s%s %s\n",
		prefix,
		p.msgHeader(fmt.Sprintf("[%s] (%s)", ts, m.Timestamp)),
		p.userName(name),
	)
	for _, line := range strings.Split(expandMentions(m.Text), "\n") {
		fmt.Fprintf(p.w, "%s%s\n", prefix, line)
	}
	for _, f := range m.Files {
		fmt.Fprintf(p.w, "%s📎 %s\n", prefix, f.Name)
	}
}

// DMs renders a list of DM conversations.
// Each channel's Name should already be resolved to the other user's display name.
func (p *Printer) DMs(dms []slack.Channel) error {
	if p.jsonOut {
		return p.writeJSON(dms)
	}
	for _, ch := range dms {
		fmt.Fprintf(p.w, "  %s\n", p.userName("@"+ch.Name))
	}
	return nil
}

// SearchResults renders a list of search results grouped by channel.
func (p *Printer) SearchResults(results []slack.SearchResult) error {
	if p.jsonOut {
		return p.writeJSON(results)
	}
	for _, r := range results {
		ts := formatTimestamp(r.Message.Timestamp)
		name := r.Message.UserName
		if name == "" {
			name = r.Message.BotUsername
		}
		if name == "" {
			name = r.Message.UserID
		}
		channel := p.channelName("#" + r.ChannelName)
		fmt.Fprintf(p.w, "%s %s\n",
			p.msgHeader(fmt.Sprintf("[%s] (%s)", ts, r.Message.Timestamp)),
			p.userName(name)+" "+channel,
		)
		for _, line := range strings.Split(expandMentions(r.Message.Text), "\n") {
			fmt.Fprintf(p.w, "%s\n", line)
		}
		for _, f := range r.Message.Files {
			fmt.Fprintf(p.w, "📎 %s\n", f.Name)
		}
		fmt.Fprintln(p.w)
	}
	return nil
}

// Unread renders a summary of channels and DMs that have unread messages.
// Channel names and DM names should already be resolved before calling.
func (p *Printer) Unread(channels, dms []slack.Channel) error {
	if p.jsonOut {
		return p.writeJSON(map[string]interface{}{"channels": channels, "dms": dms})
	}
	if len(channels) == 0 && len(dms) == 0 {
		fmt.Fprintln(p.w, "No unread messages.")
		return nil
	}
	if len(channels) > 0 {
		fmt.Fprintln(p.w, p.marker("Channels:"))
		for _, ch := range channels {
			fmt.Fprintf(p.w, "  %s  %s\n",
				p.channelName("#"+ch.Name),
				p.dimText(fmt.Sprintf("(%d unread)", ch.UnreadCount)),
			)
		}
	}
	if len(dms) > 0 {
		if len(channels) > 0 {
			fmt.Fprintln(p.w)
		}
		fmt.Fprintln(p.w, p.marker("Direct messages:"))
		for _, ch := range dms {
			fmt.Fprintf(p.w, "  %s  %s\n",
				p.userName("@"+ch.Name),
				p.dimText(fmt.Sprintf("(%d unread)", ch.UnreadCount)),
			)
		}
	}
	return nil
}

// Users renders a list of users.
func (p *Printer) Users(users []slack.User) error {
	if p.jsonOut {
		return p.writeJSON(users)
	}
	for _, u := range users {
		name := p.channelName("@" + u.Name)
		realName := ""
		if u.RealName != "" && u.RealName != u.Name {
			realName = p.dimText("  (" + u.RealName + ")")
		}
		fmt.Fprintf(p.w, "  %s%s\n", name, realName)
	}
	return nil
}

// ChannelDetail renders detailed information for a single channel.
func (p *Printer) ChannelDetail(detail slack.ChannelDetail, creatorName string) error {
	if p.jsonOut {
		return p.writeJSON(detail)
	}

	ch := detail.Channel
	fmt.Fprintln(p.w, p.channelName("#"+ch.Name))

	printField := func(label, value string) {
		if value != "" {
			fmt.Fprintf(p.w, "  %-12s %s\n", label+":", value)
		}
	}

	printField("ID", ch.ID)

	kind := "public"
	if detail.IsPrivate {
		kind = "private"
	}
	if detail.IsGeneral {
		kind += ", general"
	}
	if detail.IsArchived {
		kind += ", archived"
	}
	printField("Type", kind)

	if detail.NumMembers > 0 {
		printField("Members", fmt.Sprintf("%d", detail.NumMembers))
	}
	if detail.Created > 0 {
		printField("Created", time.Unix(detail.Created, 0).Local().Format("2006-01-02"))
	}
	printField("Creator", creatorName)
	printField("Topic", detail.Topic)
	printField("Purpose", ch.Purpose)

	return nil
}

// UserProfile renders detailed profile information for a single user.
func (p *Printer) UserProfile(profile slack.UserProfile) error {
	if p.jsonOut {
		return p.writeJSON(profile)
	}

	u := profile.User
	name := p.userName("@" + u.Name)
	if u.IsBot {
		name += p.dimText(" [bot]")
	}
	fmt.Fprintln(p.w, name)

	printField := func(label, value string) {
		if value != "" {
			fmt.Fprintf(p.w, "  %-12s %s\n", label+":", value)
		}
	}

	printField("ID", u.ID)
	printField("Real name", u.RealName)
	printField("Display name", u.DisplayName)
	printField("Title", profile.Title)
	printField("Email", profile.Email)
	printField("Phone", profile.Phone)
	printField("Timezone", profile.Timezone)

	status := profile.Status
	if profile.Emoji != "" {
		status = profile.Emoji + " " + status
	}
	printField("Status", status)

	return nil
}

// Success prints a success message. Suppressed in JSON mode.
func (p *Printer) Success(msg string) {
	if !p.jsonOut {
		fmt.Fprintln(p.w, msg)
	}
}

// PostResult outputs the result of a post/dm send operation.
// In JSON mode, outputs {"ts":"...", "channel":"..."} to stdout.
// In text mode, prints a human-readable success message.
func (p *Printer) PostResult(ts, channel string) error {
	if p.jsonOut {
		return p.writeJSON(map[string]string{"ts": ts, "channel": channel})
	}
	fmt.Fprintf(p.w, "Message posted (ts: %s)\n", ts)
	return nil
}

// writeJSON encodes v as indented JSON to p.w.
func (p *Printer) writeJSON(v interface{}) error {
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}

// formatTimestamp converts a Slack timestamp (Unix seconds) to a human-readable string.
func formatTimestamp(ts string) string {
	// Slack timestamps look like "1234567890.123456"
	dot := strings.Index(ts, ".")
	if dot == -1 {
		return ts
	}
	sec := int64(0)
	for _, c := range ts[:dot] {
		if c < '0' || c > '9' {
			return ts
		}
		sec = sec*10 + int64(c-'0')
	}
	t := time.Unix(sec, 0).Local()
	return t.Format("2006-01-02 15:04")
}

// expandMentions replaces <@UXXXXX> with @UXXXXX for readability.
// Full name resolution happens at the slack layer.
func expandMentions(text string) string {
	// Replace <@UXXXXXX> with @UXXXXXX (user ID inline mention)
	result := strings.ReplaceAll(text, "<@", "@")
	result = strings.ReplaceAll(result, ">", "")
	return result
}

// isTTY reports whether w is connected to a terminal.
func isTTY(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) != 0
	}
	return false
}
