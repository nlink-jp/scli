# scli — セットアップガイド

## 前提条件

- Go 1.26 以降
- GNU Make
- アプリをインストールできる権限を持つ Slack ワークスペース

---

## ステップ 1: Slack アプリの作成

### 1-1. Slack API ポータルを開く

<https://api.slack.com/apps> にアクセスし、**Create New App** をクリックします。
**From scratch** を選択し、以下を入力します：

- **App Name**: `scli`（任意の名前でもOK）
- **Pick a workspace**: 使用するワークスペースを選択

### 1-2. OAuthスコープの設定

左サイドバーから **OAuth & Permissions** を開きます。

**User Token Scopes**（Bot Token Scopes ではない）までスクロールし、以下を追加します：

| スコープ | 用途 |
|---------|------|
| `channels:read` | パブリックチャンネル一覧 |
| `groups:read` | プライベートチャンネル一覧 |
| `im:read` | DM一覧 |
| `im:write` | DMチャンネルを開く |
| `mpim:read` | グループDM一覧 |
| `channels:history` | パブリックチャンネルのメッセージ読み取り |
| `groups:history` | プライベートチャンネルのメッセージ読み取り |
| `im:history` | DMメッセージ読み取り |
| `mpim:history` | グループDMメッセージ読み取り |
| `chat:write` | メッセージ投稿 |
| `files:write` | ファイルアップロード |
| `files:read` | 添付ファイルのダウンロード（`channel export --save-dir`） |
| `search:read` | メッセージ検索 |
| `users:read` | ユーザー名・プロフィール解決 |

### 1-3. リダイレクトURLの設定

同じ **OAuth & Permissions** ページの **Redirect URLs** に以下を追加します：

```
https://localhost:7777/callback
```

これは `scli auth login` が OAuth フロー中に待ち受けるローカルHTTPSのURLです。
`scli` は起動時に自己署名証明書を動的生成します。ブラウザに**証明書の警告**が表示されますが、
それを承認することでログインが完了します。

> **補足**: Slack はすべてのリダイレクトURIにHTTPSを要求します。`scli` は短命の自己署名証明書を使った
> ローカルHTTPSサーバーを起動することでこれに対応しています。`mkcert` などの外部ツールは不要です。

### 1-4. クレデンシャルの確認

**Basic Information** を開き、以下をメモしておきます：

- **Client ID**
- **Client Secret**

ステップ 3 で使用します。

---

## ステップ 2: scli のビルド

```bash
# リポジトリをクローン
git clone https://github.com/<your-org>/scli.git
cd scli

# Git フックのインストールとツールチェーンの確認
make setup

# 現在のプラットフォーム向けにビルド
make build

# （任意）全プラットフォーム向けにクロスコンパイル
make build-all
```

バイナリは `./dist/` に生成されます。

---

## ステップ 3: scli の設定

### オプション A-1 — インタラクティブログイン（推奨）

```bash
export SLACK_CLIENT_ID=<your-client-id>
export SLACK_CLIENT_SECRET=<your-client-secret>
scli auth login --workspace myteam
```

ブラウザが開き、Slack の OAuth フローが始まります。`scli` が自己署名証明書を使ったローカルHTTPSサーバー（ポート7777）を起動し、コールバックを自動受信します。

**ブラウザに証明書の警告が表示された場合：**
- Chrome/Edge: **詳細設定** → **localhost にアクセスする（安全でない）** をクリック
- Firefox: **詳細情報** → **危険性を承知で続行する** をクリック
- Safari: **Webサイトを表示** をクリック

成功すると、トークンが OS キーチェーンに安全に保存されます。

### オプション A-2 — 手動ログイン（ヘッドレス環境・ブラウザフローが使えない場合）

```bash
export SLACK_CLIENT_ID=<your-client-id>
export SLACK_CLIENT_SECRET=<your-client-secret>
scli auth login --workspace myteam --manual
```

`scli` が認証URLを表示します。ブラウザで開いてアプリを承認すると、`https://localhost:7777/callback?code=xxxxx` にリダイレクトされます（ページはエラーになりますが問題ありません）。
アドレスバーのURL全体をコピーしてターミナルのプロンプトに貼り付けてください。

### オプション B — 環境変数

```bash
export SLACK_TOKEN=xoxp-...            # デフォルトワークスペース
export SLACK_TOKEN_MYTEAM=xoxp-...     # 名前付きワークスペース
```

### オプション C — 設定ファイル

`~/.config/scli/config.json` を編集します：

```json
{
  "default_workspace": "myteam",
  "workspaces": {
    "myteam": {
      "token": "xoxp-...",
      "team_id": "T012AB3C4",
      "user_id": "U012AB3C4"
    }
  }
}
```

> **セキュリティ注記**: オプション A（OSキーチェーン）を強く推奨します。
> 設定ファイルや環境変数はトークンを平文で保存するため、
> キーチェーンが利用できない環境（CIパイプライン、コンテナ等）でのみ使用してください。

---

## ステップ 4: 動作確認

```bash
scli workspace list
scli channel list
```

---

## トラブルシューティング

| 症状 | 原因 | 対処 |
|------|------|------|
| `invalid_auth` | トークンが期限切れまたは失効 | `scli auth login` を再実行 |
| `missing_scope` | Slack アプリにスコープが未追加 | API ポータルでスコープを追加してアプリを再インストール |
| ブラウザが開かない | ヘッドレス環境 | 標準出力に表示されるURLを手動でコピーしてブラウザで開く |
| キーチェーンの確認が繰り返し表示される | OSキーチェーンがロックされている | キーチェーンをアンロック（macOS: キーチェーンアクセスアプリ） |
| リネーム後に `no token found for workspace` | config.json を直接編集してワークスペース名を変更した | `scli auth login --workspace <新名>` で再認証、または `scli workspace rename` で安全にリネーム |

---

*原文（英語）: `docs/setup.md`*
