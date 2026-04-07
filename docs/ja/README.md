# scli

ターミナルで動く Slack クライアントです。ボットではなくユーザーとして動作します。
チャンネルの読み書き、メッセージ送信、検索、DM 管理をターミナルから行えます。

## 機能

- チャンネル・DM メッセージの読み取り（スレッド展開対応）
- メッセージ投稿・スレッド返信（`\n` で改行可能）
- [Block Kit](https://api.slack.com/block-kit) JSON によるリッチメッセージ投稿（ファイル・stdin・インライン対応）
- チャンネルへのファイルアップロード
- 未読チャンネル・DM の一覧表示
- ワークスペース全体のメッセージ検索
- 複数ワークスペース対応
- カラー出力 / `--json` / `--no-color` フラグ
- トークンは OS キーチェーン・環境変数・`.env` ファイルで管理

## インストール

### ソースからビルド

```sh
git clone https://github.com/nlink-jp/scli.git
cd scli
make build          # 現在のプラットフォーム向けにビルド → dist/scli
make build-all      # 全対象プラットフォームにクロスコンパイル
```

必要環境: Go 1.26 以上、`make`

### 初回セットアップ

Slack アプリの作成と認証の手順は [docs/ja/setup.md](setup.md) を参照してください。

```sh
scli auth login
```

## コマンド一覧

| コマンド | 説明 |
|---------|------|
| `scli auth login` | Slack 認証（OAuth 2.0 PKCE） |
| `scli auth logout` | 認証情報の削除 |
| `scli auth list` | 認証済みワークスペース一覧 |
| `scli channel list` | 表示可能なチャンネルの一覧（参加・未参加問わず） |
| `scli channel joined` | 参加中のチャンネルの一覧 |
| `scli channel read <channel>` | チャンネルのメッセージを読む |
| `scli dm list` | オープン中 DM 一覧 |
| `scli dm read <user>` | DM を読む |
| `scli dm send <user> <message>` | DM を送る |
| `scli post <channel> [message]` | チャンネルにメッセージを投稿 |
| `scli search <query>` | メッセージを検索 |
| `scli unread` | 未読チャンネル・DM を表示 |
| `scli user list` | ワークスペースメンバー一覧 |
| `scli workspace list` | 設定済みワークスペース一覧 |
| `scli workspace use <name>` | デフォルトワークスペースを切り替え |

### 共通フラグ

```
--workspace, -w   ワークスペース名（デフォルト: "default"）
--json            JSON 形式で出力
--no-color        ANSI カラーコードを無効化
```

### channel read / dm read オプション

```
-n, --limit N     取得するメッセージ数（デフォルト: 20）
--unread          最終既読以降のメッセージのみ表示
--thread <ts>     指定したタイムスタンプのスレッドを表示
```

### post オプション

```
--file <path>         ファイルを添付
--thread <ts>         スレッドに返信
--blocks <json>       Block Kit JSON 配列（インライン文字列）
--blocks-file <path>  Block Kit JSON をファイルから読み込む（"-" は stdin）
```

`--blocks` または `--blocks-file` を使う場合、`[message]` は通知フォールバックテキストとなり省略可能です。
両フラグは同時に使用できません。

#### Block Kit の使用例

```sh
# インライン JSON
scli post '#general' 'Hello' --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*Hello*"}}]'

# ファイルから読み込む
scli post '#general' 'Hello' --blocks-file blocks.json

# stdin から読み込む（md-to-slack などとパイプ接続）
md-to-slack input.md | scli post '#general' 'Hello' --blocks-file -

# フォールバックテキストなし（blocks のみ）
md-to-slack input.md | scli post '#general' --blocks-file -
```

### search オプション

```
-n, --limit N     最大件数（デフォルト: 20）
--asc             古い順に並べる（デフォルト: 新しい順）
```

## 複数ワークスペース

```sh
scli auth login --workspace personal
scli auth login --workspace work
scli workspace use work
scli channel read #general --workspace personal
```

## ライセンス

MIT — [LICENSE](../../LICENSE) を参照
