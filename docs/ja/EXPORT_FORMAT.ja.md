# エクスポートログのデータ形式

Slack チャンネルのメッセージエクスポートで共有される JSON スキーマ。
この形式は **scat**、**stail**、**scli** で使用される。

## トップレベル構造

| フィールド | 型 | 説明 |
|--------------------|--------|----------------------------------------------|
| `export_timestamp` | string | エクスポートを実行した時刻の RFC 3339 タイムスタンプ |
| `channel_name`     | string | `#` プレフィックス付きのチャンネル名（例: `#general`） |
| `messages`         | array  | [Message](#message-オブジェクト) オブジェクトの配列 |

## Message オブジェクト

| フィールド | 型 | 必須 | 説明 |
|------------------------|---------|----------|--------------------------------------------------|
| `user_id`              | string  | はい     | Slack の User ID または Bot ID（[動作に関する注記](#動作に関する注記)を参照） |
| `user_name`            | string  | いいえ   | 表示名（空のときは省略）                         |
| `post_type`            | string  | はい     | `"user"` または `"bot"`                          |
| `timestamp`            | string  | はい     | RFC 3339 UTC（例: `2025-03-01T10:00:00Z`）      |
| `timestamp_unix`       | string  | はい     | Slack の生の ts（例: `1740823200.000000`）       |
| `text`                 | string  | はい     | メッセージ本文のテキスト                         |
| `files`                | array   | はい     | [File](#file-オブジェクト) オブジェクトの配列（無いときは空の `[]`） |
| `attachments`          | array   | いいえ   | [Attachment](#attachment-オブジェクト) オブジェクトの配列（無いときは省略） |
| `blocks`               | array   | いいえ   | Block Kit の生 JSON 配列（無いときは省略）       |
| `thread_timestamp_unix`| string  | いいえ   | スレッド親の Slack ts（スレッド外のメッセージでは省略） |
| `is_reply`             | boolean | はい     | メッセージがスレッド返信なら `true`              |

## File オブジェクト

| フィールド | 型 | 必須 | 説明 |
|--------------|--------|----------|--------------------------------------------------|
| `id`         | string | はい     | Slack のファイル ID                              |
| `name`       | string | はい     | 元のファイル名                                   |
| `mimetype`   | string | はい     | MIME タイプ（例: `image/png`、`application/pdf`）|
| `local_path` | string | いいえ   | ダウンロードしたファイルの絶対パス（`--save-dir` / `--output-files` を使った場合のみ） |

ダウンロードが失敗した場合 `local_path` は出ません。理由は stderr に `warn:`
行として出力され、エクスポート自体は続行します。理由のひとつは知っておく価値が
あります: Slack のダウンロード URL はリダイレクトを返し、トークンは
**ダウンロードを開始したドメインの内側にのみ**再送されます。そのドメインの外へ
向かうリダイレクトにはトークンを渡さないため、ファイルはダウンロードされず、
警告がそのホスト名を挙げます——そのホストが返すサインインページを、ファイル名で
保存してしまう代わりに。

## Attachment オブジェクト

レガシーなリッチ添付（URL の展開、ボットカードなど）。

| フィールド | 型 | 必須 | 説明 |
|--------------|--------|----------|--------------------------------------|
| `fallback`   | string | いいえ   | プレーンテキストの要約               |
| `color`      | string | いいえ   | サイドバーの色の 16 進数（例: `#ff0000`） |
| `pretext`    | string | いいえ   | 添付の上に表示されるテキスト         |
| `title`      | string | いいえ   | 添付のタイトル                       |
| `title_link` | string | いいえ   | タイトルからリンクされる URL         |
| `text`       | string | いいえ   | 本文のテキスト                       |
| `fields`     | array  | いいえ   | [Attachment Field](#attachment-field-オブジェクト) オブジェクトの配列 |
| `footer`     | string | いいえ   | フッターのテキスト                   |
| `image_url`  | string | いいえ   | フルサイズ画像の URL                 |

全フィールドが `omitempty` を使っている — 存在しないフィールドは JSON 出力から省略される。

## Attachment Field オブジェクト

| フィールド | 型 | 説明 |
|---------|---------|--------------------------------------------|
| `title` | string  | フィールドのラベル                         |
| `value` | string  | フィールドの値                             |
| `short` | boolean | フィールドが横並び表示に十分短ければ `true` |

## Blocks

Block Kit のペイロードは生の JSON 配列として保存され、完全な忠実性が保たれる。
変換もフラット化も行わない。

```json
"blocks": [
  {"type": "section", "text": {"type": "mrkdwn", "text": "Hello *world*"}}
]
```

プレーンテキストが必要な利用側は、blocks をパースするか `text` フィールドにフォールバックすること。

## 動作に関する注記

### ボットメッセージの `user_id`

Slack API が `user` フィールドが空のメッセージを返した場合（incoming webhook や
一部のボット連携でよくある）、`user_id` は `bot_id` の値にフォールバックする。

### `thread_timestamp_unix`

Slack API の `thread_ts` フィールドから直接設定される:
- **スレッド親**: `thread_ts == ts` → フィールドは存在し、`is_reply` は `false`
- **スレッド返信**: `thread_ts != ts` → フィールドは存在し、`is_reply` は `true`
- **スレッド外のメッセージ**: `thread_ts` が空 → フィールドは省略される

### ファイルダウンロードのエラー

ファイルのダウンロードが失敗した場合（HTTP エラー、レート制限の枯渇）、ツールは
stderr に警告を出力してエクスポートを続行する。ファイルのメタデータ（`id`、
`name`、`mimetype`）は保持され、`local_path` は空のままになる。

### スレッド展開

ツールごとの動作:
- **scli**: 親ごとに `conversations.replies` を取得してスレッドを展開する。
- **stail**: スレッドを展開せず、`conversations.history` のページのみをエクスポートする。

## 例

```json
{
  "export_timestamp": "2025-08-15T11:03:53Z",
  "channel_name": "#alerts",
  "messages": [
    {
      "user_id": "U12345ABC",
      "user_name": "Alice",
      "post_type": "user",
      "timestamp": "2025-08-14T10:00:00Z",
      "timestamp_unix": "1755168000.000000",
      "text": "Check this out",
      "files": [
        {
          "id": "F001",
          "name": "report.pdf",
          "mimetype": "application/pdf"
        }
      ],
      "is_reply": false
    },
    {
      "user_id": "B012345DEF",
      "user_name": "monitoring-bot",
      "post_type": "bot",
      "timestamp": "2025-08-14T10:05:00Z",
      "timestamp_unix": "1755168300.000000",
      "text": "",
      "files": [],
      "attachments": [
        {
          "fallback": "Server CPU > 90%",
          "color": "#ff0000",
          "title": "CPU Alert",
          "title_link": "https://grafana.example.com/d/cpu",
          "text": "Production server cpu-1 is at 95%",
          "fields": [
            {"title": "Host", "value": "cpu-1", "short": true},
            {"title": "Severity", "value": "Critical", "short": true}
          ],
          "footer": "Grafana"
        }
      ],
      "blocks": [
        {
          "type": "section",
          "text": {"type": "mrkdwn", "text": "*CPU Alert*\nProduction server cpu-1 is at 95%"}
        }
      ],
      "is_reply": false
    },
    {
      "user_id": "U67890GHI",
      "user_name": "Bob",
      "post_type": "user",
      "timestamp": "2025-08-14T10:10:00Z",
      "timestamp_unix": "1755168600.000000",
      "text": "Looking into it",
      "files": [],
      "thread_timestamp_unix": "1755168300.000000",
      "is_reply": true
    }
  ]
}
```

## 互換性

新しいフィールド（`attachments`、`blocks`）は `omitempty` を使っており、メッセージが
それらを含む場合にのみ存在する。前方互換性のため、利用側は未知のフィールドを無視すること。

---

*原文（英語）: `docs/en/EXPORT_FORMAT.md`*
