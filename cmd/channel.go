package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nlink-jp/scli/internal/slack"
	"github.com/spf13/cobra"
)

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Manage and read Slack channels",
}

var (
	channelReadLimit  int
	channelReadUnread bool
	channelReadThread string
)

var channelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all visible channels (public and private)",
	RunE:  runChannelList,
}

var channelReadCmd = &cobra.Command{
	Use:   "read <channel>",
	Short: "Read messages from a channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelRead,
}

var channelInfoCmd = &cobra.Command{
	Use:   "info <channel>",
	Short: "Show detailed information about a channel",
	Args:  cobra.ExactArgs(1),
	RunE:  runChannelInfo,
}

var channelSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search channels by name or purpose",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runChannelSearch,
}

var channelJoinedCmd = &cobra.Command{
	Use:   "joined",
	Short: "List channels you are a member of",
	RunE:  runChannelJoined,
}

func init() {
	channelReadCmd.Flags().IntVarP(&channelReadLimit, "limit", "n", 20, "Number of messages to fetch")
	channelReadCmd.Flags().BoolVar(&channelReadUnread, "unread", false, "Show only unread messages")
	channelReadCmd.Flags().StringVar(&channelReadThread, "thread", "", "Show a specific thread (message timestamp)")

	channelCmd.AddCommand(channelListCmd, channelReadCmd, channelInfoCmd, channelSearchCmd, channelJoinedCmd)
	rootCmd.AddCommand(channelCmd)
}

func runChannelSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(strings.Join(args, " "))

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channels, err := client.ListChannels(cmd.Context())
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	var matches []slack.Channel
	for _, ch := range channels {
		if strings.Contains(strings.ToLower(ch.Name), query) ||
			strings.Contains(strings.ToLower(ch.Purpose), query) {
			matches = append(matches, ch)
		}
	}

	if len(matches) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No channels found matching %q.\n", query)
		return nil
	}

	p := newPrinter(cmd)
	return p.Channels(matches, "")
}

func runChannelInfo(cmd *cobra.Command, args []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channelID, err := client.ResolveChannelID(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	detail, err := client.GetChannelDetail(cmd.Context(), channelID)
	if err != nil {
		return fmt.Errorf("get channel info: %w", err)
	}

	creatorName := client.ResolveUserName(cmd.Context(), detail.Creator)

	p := newPrinter(cmd)
	return p.ChannelDetail(detail, creatorName)
}

const threadFetchConcurrency = 10

// fetchThreadReplies populates Replies on each message that is a thread parent,
// using up to threadFetchConcurrency parallel requests.
func fetchThreadReplies(ctx context.Context, client *slack.Client, msgs []slack.Message, channelID string) {
	sem := make(chan struct{}, threadFetchConcurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, m := range msgs {
		if m.ReplyCount == 0 {
			continue
		}
		wg.Add(1)
		go func(i int, m slack.Message) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Fetch up to ReplyCount+1 messages (includes parent as first item).
			limit := m.ReplyCount + 1
			replies, err := client.GetThreadReplies(ctx, channelID, m.Timestamp, limit)
			if err != nil || len(replies) <= 1 {
				return
			}
			mu.Lock()
			msgs[i].Replies = replies[1:] // skip the parent message
			mu.Unlock()
		}(i, m)
	}
	wg.Wait()
}

// resolveUserNames resolves UserID to UserName for all messages and their replies.
func resolveUserNames(ctx context.Context, client *slack.Client, msgs []slack.Message) {
	seen := map[string]string{}
	resolve := func(userID string) string {
		if userID == "" {
			return ""
		}
		if name, ok := seen[userID]; ok {
			return name
		}
		name := client.ResolveUserName(ctx, userID)
		seen[userID] = name
		return name
	}
	for i, m := range msgs {
		if m.UserID != "" {
			msgs[i].UserName = resolve(m.UserID)
		}
		for j, r := range m.Replies {
			if r.UserID != "" {
				msgs[i].Replies[j].UserName = resolve(r.UserID)
			}
		}
	}
}

func runChannelList(cmd *cobra.Command, _ []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channels, err := client.ListChannels(cmd.Context())
	if err != nil {
		return fmt.Errorf("list channels: %w", err)
	}

	if len(channels) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No channels found. Try joining some channels in Slack.")
		return nil
	}

	p := newPrinter(cmd)
	return p.Channels(channels, "")
}

func runChannelJoined(cmd *cobra.Command, _ []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channels, err := client.ListJoinedChannels(cmd.Context())
	if err != nil {
		return fmt.Errorf("list joined channels: %w", err)
	}

	if len(channels) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "You are not a member of any channels.")
		return nil
	}

	p := newPrinter(cmd)
	return p.Channels(channels, "")
}

func runChannelRead(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channelID, err := client.ResolveChannelID(cmd.Context(), nameOrID)
	if err != nil {
		return err
	}

	var msgs []slack.Message
	if channelReadThread != "" {
		msgs, err = client.GetThreadReplies(cmd.Context(), channelID, channelReadThread, channelReadLimit)
		if err != nil {
			return fmt.Errorf("read thread: %w", err)
		}
	} else {
		oldest := ""
		if channelReadUnread {
			oldest, err = client.GetChannelLastRead(cmd.Context(), channelID)
			if err != nil {
				return fmt.Errorf("get last read: %w", err)
			}
		}
		msgs, err = client.GetChannelHistory(cmd.Context(), channelID, channelReadLimit, oldest)
		if err != nil {
			return fmt.Errorf("read channel: %w", err)
		}
	}

	if len(msgs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No messages.")
		return nil
	}

	// When reading the channel (not a specific thread), fetch replies for
	// thread parent messages in parallel and attach them.
	if channelReadThread == "" {
		fetchThreadReplies(cmd.Context(), client, msgs, channelID)
	}

	// Resolve usernames for all messages (parents and replies).
	resolveUserNames(cmd.Context(), client, msgs)

	p := newPrinter(cmd)
	return p.Messages(msgs)
}
