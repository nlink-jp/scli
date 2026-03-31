package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	postFile      string
	postThread    string
	postBlocks    string
	postBlockFile string
)

var postCmd = &cobra.Command{
	Use:   "post <channel> [message]",
	Short: "Post a message to a channel",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPost,
}

func init() {
	postCmd.Flags().StringVar(&postFile, "file", "", "Attach a file to the message")
	postCmd.Flags().StringVar(&postThread, "thread", "", "Reply in a thread (message timestamp)")
	postCmd.Flags().StringVar(&postBlocks, "blocks", "", "Block Kit JSON to post (JSON array string)")
	postCmd.Flags().StringVar(&postBlockFile, "blocks-file", "", "Path to a file containing Block Kit JSON (\"-\" reads from stdin)")
	rootCmd.AddCommand(postCmd)
}

func runPost(cmd *cobra.Command, args []string) error {
	nameOrID := args[0]

	// Resolve blocks JSON from flag or file.
	blocksJSON, err := loadBlocksJSON(postBlocks, postBlockFile)
	if err != nil {
		return err
	}

	// <message> is required when not posting blocks; optional with blocks.
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

	channelID, err := client.ResolveChannelID(cmd.Context(), nameOrID)
	if err != nil {
		return err
	}

	if postFile != "" {
		// File upload: message becomes the initial comment.
		if err := client.UploadFile(cmd.Context(), channelID, postFile, text); err != nil {
			return fmt.Errorf("upload file: %w", err)
		}
		p := newPrinter(cmd)
		p.Success("File uploaded.")
		return nil
	}

	var ts string
	if blocksJSON != "" {
		if postThread != "" {
			ts, err = client.PostThreadReplyWithBlocks(cmd.Context(), channelID, postThread, text, blocksJSON)
		} else {
			ts, err = client.PostMessageWithBlocks(cmd.Context(), channelID, text, blocksJSON)
		}
	} else {
		if postThread != "" {
			ts, err = client.PostThreadReply(cmd.Context(), channelID, postThread, text)
		} else {
			ts, err = client.PostMessage(cmd.Context(), channelID, text)
		}
	}
	if err != nil {
		return fmt.Errorf("post message: %w", err)
	}

	p := newPrinter(cmd)
	return p.PostResult(ts, channelID)
}

