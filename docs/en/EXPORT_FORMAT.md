# Export Log Data Format

Shared JSON schema for Slack channel message exports.
This format is used by **scat**, **stail**, and **scli**.

## Top-level structure

| Field              | Type   | Description                                  |
|--------------------|--------|----------------------------------------------|
| `export_timestamp` | string | RFC 3339 timestamp of when the export ran    |
| `channel_name`     | string | Channel name with `#` prefix (e.g. `#general`) |
| `messages`         | array  | Array of [Message](#message-object) objects   |

## Message object

| Field                  | Type    | Required | Description                                      |
|------------------------|---------|----------|--------------------------------------------------|
| `user_id`              | string  | yes      | Slack User ID or Bot ID (see [Behavioral notes](#behavioral-notes)) |
| `user_name`            | string  | no       | Display name (omitted when empty)                |
| `post_type`            | string  | yes      | `"user"` or `"bot"`                              |
| `timestamp`            | string  | yes      | RFC 3339 UTC (e.g. `2025-03-01T10:00:00Z`)      |
| `timestamp_unix`       | string  | yes      | Raw Slack ts (e.g. `1740823200.000000`)          |
| `text`                 | string  | yes      | Message body text                                |
| `files`                | array   | yes      | Array of [File](#file-object) objects (empty `[]` when none) |
| `attachments`          | array   | no       | Array of [Attachment](#attachment-object) objects (omitted when none) |
| `blocks`               | array   | no       | Raw Block Kit JSON array (omitted when none)     |
| `thread_timestamp_unix`| string  | no       | Thread parent Slack ts (omitted for non-thread messages) |
| `is_reply`             | boolean | yes      | `true` if the message is a thread reply          |

## File object

| Field        | Type   | Required | Description                                      |
|--------------|--------|----------|--------------------------------------------------|
| `id`         | string | yes      | Slack file ID                                    |
| `name`       | string | yes      | Original filename                                |
| `mimetype`   | string | yes      | MIME type (e.g. `image/png`, `application/pdf`)  |
| `local_path` | string | no       | Absolute path to downloaded file (only when `--save-dir` / `--output-files` is used) |

## Attachment object

Legacy rich attachments (URL unfurls, bot cards, etc.).

| Field        | Type   | Required | Description                          |
|--------------|--------|----------|--------------------------------------|
| `fallback`   | string | no       | Plain-text summary                   |
| `color`      | string | no       | Sidebar color hex (e.g. `#ff0000`)   |
| `pretext`    | string | no       | Text shown above the attachment      |
| `title`      | string | no       | Attachment title                     |
| `title_link` | string | no       | URL linked from the title            |
| `text`       | string | no       | Main body text                       |
| `fields`     | array  | no       | Array of [Attachment Field](#attachment-field-object) objects |
| `footer`     | string | no       | Footer text                          |
| `image_url`  | string | no       | Full-size image URL                  |

All fields use `omitempty` — absent fields are omitted from the JSON output.

## Attachment Field object

| Field   | Type    | Description                                |
|---------|---------|--------------------------------------------|
| `title` | string  | Field label                                |
| `value` | string  | Field value                                |
| `short` | boolean | `true` if the field is short enough for side-by-side display |

## Blocks

Block Kit payloads are stored as a raw JSON array, preserving full fidelity.
No transformation or flattening is applied.

```json
"blocks": [
  {"type": "section", "text": {"type": "mrkdwn", "text": "Hello *world*"}}
]
```

Consumers that need plain text should parse the blocks or fall back to the `text` field.

## Behavioral notes

### `user_id` for bot messages

When the Slack API returns a message with an empty `user` field (common for
incoming webhooks and some bot integrations), `user_id` falls back to the
`bot_id` value.

### `thread_timestamp_unix`

Set directly from the Slack API's `thread_ts` field:
- **Thread parent**: `thread_ts == ts` → field is present, `is_reply` is `false`
- **Thread reply**: `thread_ts != ts` → field is present, `is_reply` is `true`
- **Non-thread message**: `thread_ts` is empty → field is omitted

### File download errors

When a file download fails (HTTP error, rate limit exhaustion), the tool logs
a warning to stderr and continues the export. The file's metadata (`id`,
`name`, `mimetype`) is preserved; `local_path` remains empty.

### Thread expansion

Tool-specific behavior:
- **scli**: Expands threads by fetching `conversations.replies` for each parent.
- **stail**: Does not expand threads; exports only `conversations.history` pages.

## Example

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

## Compatibility

New fields (`attachments`, `blocks`) use `omitempty` and are only present when
the message contains them. Consumers should ignore unknown fields for forward
compatibility.
