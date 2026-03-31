# scli

ユーザーとして操作するターミナルベースの Slack クライアント（ボットではなく人間として認証）。
チャンネルの閲覧、メッセージ送信、検索、DM の管理をターミナルから実行できる。

[English README is here](README.md)

## 特徴

- チャンネル・DM メッセージの閲覧（スレッド展開対応）
- メッセージ投稿・スレッド返信（`\n` で改行対応）
- [Block Kit](https://api.slack.com/block-kit) メッセージの投稿（ファイル、stdin、インライン JSON）
- ファイルアップロード
- 未読チャンネル・DM の一覧表示
- ワークスペース横断のメッセージ検索
- 複数ワークスペース対応
- カラー出力（`--json` / `--no-color` フラグ対応）
- OS キーチェーン、環境変数、`.env` ファイルによるトークン管理

## インストール

### ソースからビルド

```sh
git clone https://github.com/nlink-jp/scli.git
cd scli
make build          # 現在のプラットフォーム向け → dist/scli
make build-all      # 全プラットフォーム向けクロスコンパイル
```

必要条件: Go 1.26+, `make`

### 初期設定

Slack App の作成と認証の手順は [docs/setup.md](docs/setup.md) を参照。

```sh
scli auth login
```

## コマンド

| コマンド | 説明 |
|---------|------|
| `scli auth login` | Slack で認証（OAuth 2.0 PKCE） |
| `scli auth logout` | 保存された認証情報を削除 |
| `scli auth list` | 認証済みワークスペースの一覧 |
| `scli channel list` | 参加中のチャンネル一覧 |
| `scli channel read <channel>` | チャンネルのメッセージを読む |
| `scli dm list` | DM 会話の一覧 |
| `scli dm read <user>` | DM メッセージを読む |
| `scli dm send <user> [message]` | DM を送信（Block Kit 対応） |
| `scli post <channel> [message]` | チャンネルにメッセージを投稿 |
| `scli search <query>` | ワークスペース内のメッセージを検索 |
| `scli unread` | 未読のチャンネル・DM を表示 |
| `scli user list` | ワークスペースメンバーの一覧 |
| `scli workspace list` | 設定済みワークスペースの一覧 |
| `scli workspace use <name>` | デフォルトワークスペースを切り替え |
| `scli workspace rename <old> <new>` | ワークスペースをリネーム（config + keychain + cache を一括更新） |

### 共通フラグ

```
--workspace, -w   ワークスペース名（デフォルト: "default"）
--json            JSON で出力
--no-color        カラー出力を無効化
```

### channel read / dm read オプション

```
-n, --limit N     取得メッセージ数（デフォルト: 20）
--unread          最終既読以降のメッセージのみ表示
--thread <ts>     指定タイムスタンプのスレッドを表示
```

### post オプション

```
--file <path>         ファイルを添付
--thread <ts>         スレッドに返信
--blocks <json>       Block Kit JSON 配列（インライン文字列）
--blocks-file <path>  Block Kit JSON をファイルから読み込み（"-" で stdin）
```

`--blocks` / `--blocks-file` 使用時、`[message]` は通知フォールバックテキストになり省略可能。
両フラグは排他的。

### dm send オプション

```
--blocks <json>       Block Kit JSON 配列（インライン文字列）
--blocks-file <path>  Block Kit JSON をファイルから読み込み（"-" で stdin）
```

`post` と同じ動作 — blocks 指定時はメッセージ省略可能。

#### Block Kit の例

```sh
# インライン JSON
scli post '#general' 'Hello' --blocks '[{"type":"section","text":{"type":"mrkdwn","text":"*Hello*"}}]'

# ファイルから
scli post '#general' 'Hello' --blocks-file blocks.json

# stdin から（md-to-slack からパイプ）
md-to-slack input.md | scli post '#general' 'Hello' --blocks-file -
```

#### メッセージ ts の取得

`post` と `dm send` は `--json` 指定で `{"ts":"...","channel":"..."}` を出力:

```sh
# ts を取得してスレッド返信やファイル添付に使用
TS=$(scli post '#ops' 'alert' --json | jq -r .ts)
scli post '#ops' 'details' --thread "$TS"
```

### search オプション

```
-n, --limit N     最大結果数（デフォルト: 20）
--asc             古い順に表示（デフォルト: 新しい順）
```

## 複数ワークスペース

```sh
scli auth login --workspace personal
scli auth login --workspace work
scli workspace use work
scli channel read #general --workspace personal
```

### ワークスペースのリネーム

```sh
scli workspace rename personal my-personal
```

> **警告:** `config.json` を直接編集してワークスペース名を変更しないでください。
> 認証トークンは OS キーチェーンにワークスペース名をキーとして保存されています。
> 設定ファイル上の名前だけ変更すると、キーチェーンのトークンが孤立して認証が
> できなくなります。ワークスペース名の変更には必ず `scli workspace rename` を
> 使用してください。config、keychain、cache が一括で更新されます。

## ライセンス

MIT — [LICENSE](LICENSE) を参照
