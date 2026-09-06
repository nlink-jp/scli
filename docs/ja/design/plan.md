# scli — 開発計画

## フェーズ概要

| フェーズ | タイトル | 目標 |
|---------|---------|------|
| 0 | スキャフォールディング | ビルド可能なスケルトン、ツールチェーン、Gitフック |
| 1 | 認証・設定 | OAuthログイン、トークン保管、ワークスペース管理 |
| 2 | チャンネル基本機能 | チャンネル一覧・読み取り・投稿 |
| 3 | DM | ダイレクトメッセージの送受信 |
| 4 | 未読・ユーザー | 未読サマリー、ユーザー一覧 |
| 5 | 拡張機能 | 検索、スレッド返信、ファイル添付 |
| 6 | 仕上げ・リリース | クロスコンパイル、セキュリティスキャン、ドキュメント、v1.0.0 |

---

## フェーズ 0 — スキャフォールディング

**目標**: ツールチェーンを整備し、コンパイル・テスト可能なスケルトンを作る。

### タスク

- [ ] `go mod init github.com/<org>/scli`
- [ ] ディレクトリ構成（`cmd/`, `internal/`, `docs/`, `scripts/hooks/`）
- [ ] `Makefile`（ターゲット: `build`, `build-all`, `test`, `lint`, `check`, `setup`, `clean`）
- [ ] クロスコンパイルターゲット: `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64` (darwin は arm64 のみ)
- [ ] `cobra` ルートコマンド（`cmd/root.go`）グローバルフラグ: `--workspace`, `--json`, `--no-color`
- [ ] Gitフック（`scripts/hooks/pre-commit`, `scripts/hooks/pre-push`）で `make check` を実行
- [ ] `make setup` でフックを自動インストール
- [ ] `golangci-lint` 設定（`.golangci.yml`）
- [ ] `CHANGELOG.md` 初期エントリ

### 完了基準

クリーンクローン後に `make check`（lint + test + build-all）がパスすること。

---

## フェーズ 1 — 認証・設定

**目標**: Slackへの認証とワークスペース切り替えができる。

### タスク

- [ ] `internal/keychain` — OSキーチェーン抽象化（`zalando/go-keyring`）
  - インターフェース: `Store{Get, Set, Delete}`
  - モックストアを使ったユニットテスト
- [ ] `internal/config` — 設定ファイル・環境変数リーダー
  - トークン解決チェーン: 環境変数 → `.env` → `config.json` → キーチェーン
  - `~/.config/scli/config.json` の読み書き
  - ユニットテスト
- [ ] `internal/auth` — OAuth 2.0 PKCEフロー
  - コールバック受信用ローカルHTTPサーバー（`localhost:7777`）
  - ブラウザ起動（クロスプラットフォーム対応）
  - トークン交換と保管
  - 統合テスト（Slack OAuthエンドポイントをモック）
- [ ] `cmd/auth.go` — `scli auth login / logout / list`
- [ ] `cmd/workspace.go` — `scli workspace list / use`
- [ ] 実装中に発見した変更を `docs/setup.md` に反映

### 完了基準

`scli auth login`, `scli auth logout`, `scli workspace list`, `scli workspace use` が
実際の Slack ワークスペースに対して動作すること。

---

## フェーズ 2 — チャンネル基本機能

**目標**: チャンネル一覧・メッセージ読み取り・投稿ができる。

### タスク

- [ ] `internal/output` — フォーマッタ
  - ANSIカラー整形テキストレンダラー
  - JSONレンダラー
  - TTY自動検出（ターミナルでない場合はカラーを無効化）
  - ユニットテスト
- [ ] `internal/slack` — Slack APIクライアント（初期実装）
  - 認証ヘッダーを注入するHTTPクライアントラッパー
  - `conversations.list` — チャンネル一覧
  - `conversations.history` — メッセージ履歴
  - `chat.postMessage` — メッセージ投稿
  - チャンネル/ユーザー名解決（`#名前` → ID、`@名前` → ID）
  - モックHTTPレスポンスを使ったユニットテスト
- [ ] `cmd/channel.go` — `scli channel list / read`
- [ ] `cmd/post.go`（またはサブコマンド） — `scli post <channel> <message>`

### 完了基準

チャンネル一覧表示、カラー付きでN件のメッセージ読み取り、メッセージ投稿ができること。
全コマンドで `--json` フラグが有効なJSONを出力すること。

---

## フェーズ 3 — DM

**目標**: ダイレクトメッセージの送受信ができる。

### タスク

- [ ] `internal/slack` 追加実装
  - `conversations.open` — DMチャンネルの開設/検索
  - DM向けにフィルタした `conversations.list`
  - 名前によるユーザー検索（`users.list`）
- [ ] `cmd/dm.go` — `scli dm list / read / send`
- [ ] `cmd/user.go` — `scli user list`

### 完了基準

DM会話一覧の表示、メッセージ読み取り、`@名前` またはユーザーIDへのDM送信ができること。

---

## フェーズ 4 — 未読・ユーザー

**目標**: 確認が必要な内容をすばやく把握できる。

### タスク

- [ ] `internal/slack` 追加実装
  - `users.conversations` — ユーザーが参加しているチャンネル取得
  - `conversations.info` の `unread_count` フィールドで未読数を取得
- [ ] `cmd/unread.go` — `scli unread`
  - チャンネル名と未読件数を未読数の降順で表示
  - 未読ゼロのチャンネルはスキップ

### 完了基準

`scli unread` が未読のあるチャンネル/DMのサマリーを表示すること。

---

## フェーズ 5 — 拡張機能

**目標**: スレッド返信、ファイル添付、検索に対応する。

### タスク

- [ ] スレッドサポート
  - `scli post` と `scli dm send` に `--thread <timestamp>` フラグを追加
  - `scli channel read` に `--thread <timestamp>` フラグを追加してスレッドを表示
  - `conversations.replies` APIの呼び出し
- [ ] ファイル添付
  - `scli post` に `--file <path>` フラグを追加
  - `files.getUploadURLExternal` + `files.completeUploadExternal`（v2アップロードAPI）
- [ ] 検索
  - `cmd/search.go` — `scli search <query> [--in <channel>] [--limit N]`
  - `search.messages` APIの呼び出し

### 完了基準

スレッド返信がエンドツーエンドで動作すること。ファイルがアップロードされSlackに表示されること。
検索結果がチャンネルコンテキスト付きで返されること。

---

## フェーズ 6 — 仕上げ・リリース

**目標**: v1.0.0 としてリリース可能な状態にする。

### タスク

- [ ] `govulncheck ./...` でクリーン
- [ ] `golangci-lint` の警告をすべて解消
- [ ] 全5ターゲットでのクロスコンパイルを検証
- [ ] `docs/dependencies.md` の完成
- [ ] `docs/setup.md` と `docs/ja/setup.md` の最終レビュー
- [ ] 実装との乖離がないか `docs/design/overview.md` を更新
- [ ] `CHANGELOG.md` に v1.0.0 エントリを追加
- [ ] Gitタグ `v1.0.0` を打つ

### 完了基準

`make check` がパスすること。全プラットフォームのバイナリがクリーンにビルドできること。
ドキュメントが実装と同期していること。

---

## 依存ライブラリ計画

| ライブラリ | 用途 | 導入フェーズ |
|-----------|------|------------|
| `github.com/spf13/cobra` | CLIフレームワーク | 0 |
| `github.com/zalando/go-keyring` | OSキーチェーン抽象化 | 1 |
| `github.com/fatih/color` | ANSIカラー出力 | 2 |
| `github.com/joho/godotenv` | `.env`ファイル読み込み | 1 |

すべての依存ライブラリは使用前に `docs/dependencies.md` へ記載すること。

---

*原文（英語）: `docs/design/plan.md`*
