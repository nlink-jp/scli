package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nlink-jp/scli/internal/config"
	"github.com/nlink-jp/scli/internal/keychain"
)

func TestWorkspaceRename(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(dir + "/config.json")
	cfg := &config.Config{
		DefaultWorkspace: "old-ws",
		Workspaces: map[string]config.WorkspaceConfig{
			"old-ws": {TeamID: "T001", UserID: "U001"},
		},
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}

	mock := keychain.NewMockStore()
	_ = mock.Set("old-ws", "xoxp-secret")
	SetServicesForTest(mock, mgr)
	defer ResetServices()

	root := rootCmd
	root.SetArgs([]string{"workspace", "rename", "old-ws", "new-ws"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(out.String(), "old-ws") || !strings.Contains(out.String(), "new-ws") {
		t.Errorf("output = %q, expected rename confirmation", out.String())
	}

	// Verify config updated
	newCfg, err := mgr.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := newCfg.Workspaces["old-ws"]; ok {
		t.Error("old workspace still exists in config")
	}
	if ws, ok := newCfg.Workspaces["new-ws"]; !ok {
		t.Error("new workspace not found in config")
	} else if ws.TeamID != "T001" {
		t.Errorf("TeamID = %q, want T001", ws.TeamID)
	}
	if newCfg.DefaultWorkspace != "new-ws" {
		t.Errorf("DefaultWorkspace = %q, want new-ws", newCfg.DefaultWorkspace)
	}

	// Verify keychain updated
	if _, err := mock.Get("old-ws"); err == nil {
		t.Error("old keychain entry still exists")
	}
	token, err := mock.Get("new-ws")
	if err != nil {
		t.Fatalf("keychain get new-ws: %v", err)
	}
	if token != "xoxp-secret" {
		t.Errorf("token = %q, want xoxp-secret", token)
	}
}

func TestWorkspaceRename_NotFound(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(dir + "/config.json")
	cfg := &config.Config{Workspaces: map[string]config.WorkspaceConfig{}}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}

	mock := keychain.NewMockStore()
	SetServicesForTest(mock, mgr)
	defer ResetServices()

	root := rootCmd
	root.SetArgs([]string{"workspace", "rename", "nonexistent", "new-name"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error for nonexistent workspace")
	}
}

func TestWorkspaceRename_TargetExists(t *testing.T) {
	dir := t.TempDir()
	mgr := config.NewManager(dir + "/config.json")
	cfg := &config.Config{
		Workspaces: map[string]config.WorkspaceConfig{
			"ws-a": {TeamID: "T001"},
			"ws-b": {TeamID: "T002"},
		},
	}
	if err := mgr.Save(cfg); err != nil {
		t.Fatal(err)
	}

	mock := keychain.NewMockStore()
	SetServicesForTest(mock, mgr)
	defer ResetServices()

	root := rootCmd
	root.SetArgs([]string{"workspace", "rename", "ws-a", "ws-b"})
	root.SetErr(new(bytes.Buffer))

	if err := root.Execute(); err == nil {
		t.Error("expected error when target already exists")
	}
}
