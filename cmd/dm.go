package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/nlink-jp/scli/internal/slack"
	"github.com/spf13/cobra"
)

var dmCmd = &cobra.Command{
	Use:   "dm",
	Short: "Send and read direct messages",
}

var (
	dmReadLimit   int
	dmReadUnread  bool
	dmReadThread  string
	dmSendBlocks  string
	dmSendBlockFl string
)

var dmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open DM conversations",
	RunE:  runDMList,
}

var dmReadCmd = &cobra.Command{
	Use:   "read <user>",
	Short: "Read messages from a DM conversation",
	Args:  cobra.ExactArgs(1),
	RunE:  runDMRead,
}

var dmSendCmd = &cobra.Command{
	Use:   "send <user> [message]",
	Short: "Send a direct message to a user",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runDMSend,
}

func init() {
	dmReadCmd.Flags().IntVarP(&dmReadLimit, "limit", "n", 20, "Number of messages to fetch")
	dmReadCmd.Flags().BoolVar(&dmReadUnread, "unread", false, "Show only unread messages")
	dmReadCmd.Flags().StringVar(&dmReadThread, "thread", "", "Show a specific thread (message timestamp)")

	dmSendCmd.Flags().StringVar(&dmSendBlocks, "blocks", "", "Block Kit JSON to post (JSON array string)")
	dmSendCmd.Flags().StringVar(&dmSendBlockFl, "blocks-file", "", `Path to a file containing Block Kit JSON ("-" reads from stdin)`)

	dmCmd.AddCommand(dmListCmd, dmReadCmd, dmSendCmd)
	rootCmd.AddCommand(dmCmd)
}

func runDMList(cmd *cobra.Command, _ []string) error {
	client, err := newSlackClient()
	if err != nil {
		return err
	}

	dms, err := client.ListDMs(cmd.Context())
	if err != nil {
		return fmt.Errorf("list DMs: %w", err)
	}

	if len(dms) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No open DM conversations.")
		return nil
	}

	// Resolve each user's display name.
	seen := map[string]string{} // userID -> name
	for i, ch := range dms {
		if ch.UserID == "" {
			continue
		}
		if name, ok := seen[ch.UserID]; ok {
			dms[i].Name = name
			continue
		}
		name := client.ResolveUserName(cmd.Context(), ch.UserID)
		seen[ch.UserID] = name
		dms[i].Name = name
	}

	p := newPrinter(cmd)
	return p.DMs(dms)
}

func runDMRead(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channelID, err := resolveDMChannelID(cmd.Context(), client, nameOrID)
	if err != nil {
		return err
	}

	var msgs []slack.Message
	if dmReadThread != "" {
		msgs, err = client.GetThreadReplies(cmd.Context(), channelID, dmReadThread, dmReadLimit)
		if err != nil {
			return fmt.Errorf("read thread: %w", err)
		}
	} else {
		oldest := ""
		if dmReadUnread {
			oldest, err = client.GetChannelLastRead(cmd.Context(), channelID)
			if err != nil {
				return fmt.Errorf("get last read: %w", err)
			}
		}
		msgs, err = client.GetChannelHistory(cmd.Context(), channelID, dmReadLimit, oldest)
		if err != nil {
			return fmt.Errorf("read DM: %w", err)
		}
	}

	if len(msgs) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No messages.")
		return nil
	}

	// Resolve usernames.
	seen := map[string]string{}
	for i, m := range msgs {
		if m.UserID == "" {
			continue
		}
		if name, ok := seen[m.UserID]; ok {
			msgs[i].UserName = name
			continue
		}
		name := client.ResolveUserName(cmd.Context(), m.UserID)
		seen[m.UserID] = name
		msgs[i].UserName = name
	}

	p := newPrinter(cmd)
	return p.Messages(msgs)
}

func runDMSend(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	// Resolve blocks JSON.
	blocksJSON, err := loadBlocksJSON(dmSendBlocks, dmSendBlockFl)
	if err != nil {
		return err
	}

	text := ""
	if len(args) >= 2 {
		text = unescapeText(args[1])
	} else if blocksJSON == "" {
		return fmt.Errorf("message argument is required when --blocks / --blocks-file is not provided")
	}

	client, err := newSlackClient()
	if err != nil {
		return err
	}

	channelID, err := resolveDMChannelID(cmd.Context(), client, nameOrID)
	if err != nil {
		return err
	}

	var ts string
	if blocksJSON != "" {
		ts, err = client.PostMessageWithBlocks(cmd.Context(), channelID, text, blocksJSON)
	} else {
		ts, err = client.PostMessage(cmd.Context(), channelID, text)
	}
	if err != nil {
		return fmt.Errorf("send DM: %w", err)
	}

	p := newPrinter(cmd)
	p.Success(fmt.Sprintf("Message sent (ts: %s)", ts))
	return nil
}


// resolveDMChannelID resolves a user name, user ID, or DM channel ID to a
// DM channel ID (D-prefixed).
//
// Resolution order:
//  1. D-prefixed string → used as-is
//  2. ResolveUserID (handles U/W IDs and human usernames) → OpenDM
//  3. Fallback: search open DMs by resolved display name (covers bots/apps)
func resolveDMChannelID(ctx context.Context, client *slack.Client, nameOrID string) (string, error) {
	// Direct DM channel ID.
	if len(nameOrID) > 1 && nameOrID[0] == 'D' {
		return nameOrID, nil
	}

	// Try as a regular user first.
	userID, err := client.ResolveUserID(ctx, nameOrID)
	if err == nil {
		return client.OpenDM(ctx, userID)
	}

	// Fallback: search open DMs by display name (bots/apps not in users.list).
	name := strings.ToLower(strings.TrimPrefix(nameOrID, "@"))
	dms, listErr := client.ListDMs(ctx)
	if listErr != nil {
		return "", err // return original ResolveUserID error
	}
	for _, dm := range dms {
		resolved := strings.ToLower(client.ResolveUserName(ctx, dm.UserID))
		if resolved == name {
			return dm.ID, nil
		}
	}

	return "", err // return original ResolveUserID error
}
