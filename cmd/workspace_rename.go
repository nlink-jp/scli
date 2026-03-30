package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var workspaceRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a workspace (updates config, keychain, and cache)",
	Long: `Rename a workspace, keeping its token and configuration intact.

This command atomically updates:
  - The workspace entry in config.json
  - The token in the OS keychain
  - The disk cache directory
  - The default workspace pointer (if it was the renamed workspace)

Without this command, manually editing the workspace name in config.json
would orphan the keychain entry and break token resolution.`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkspaceRename,
}

func init() {
	workspaceCmd.AddCommand(workspaceRenameCmd)
}

func runWorkspaceRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}

	mgr, err := newConfigManager()
	if err != nil {
		return err
	}
	cfg, err := mgr.Load()
	if err != nil {
		return err
	}

	// Verify old workspace exists
	oldWS, ok := cfg.Workspaces[oldName]
	if !ok {
		return fmt.Errorf("workspace %q not found", oldName)
	}

	// Verify new name is not taken
	if _, exists := cfg.Workspaces[newName]; exists {
		return fmt.Errorf("workspace %q already exists", newName)
	}

	// 1. Move keychain token (old → new)
	ks := newKeychainStore()
	token, keychainErr := ks.Get(oldName)
	if keychainErr == nil && token != "" {
		if err := ks.Set(newName, token); err != nil {
			return fmt.Errorf("save token under new name: %w", err)
		}
		// Delete old entry only after new one is confirmed saved
		_ = ks.Delete(oldName) // best-effort; don't fail if old entry is already gone
	}

	// 2. Update config
	cfg.Workspaces[newName] = oldWS
	delete(cfg.Workspaces, oldName)

	// 3. Update default workspace if it was the renamed one
	if cfg.DefaultWorkspace == oldName {
		cfg.DefaultWorkspace = newName
	}

	if err := mgr.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// 4. Rename cache directory (best-effort)
	oldCacheDir := filepath.Join(mgr.ConfigDir(), "cache", oldName)
	newCacheDir := filepath.Join(mgr.ConfigDir(), "cache", newName)
	if _, statErr := os.Stat(oldCacheDir); statErr == nil {
		if renameErr := os.Rename(oldCacheDir, newCacheDir); renameErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not rename cache directory: %v\n", renameErr)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Workspace renamed: %q → %q\n", oldName, newName)
	return nil
}
