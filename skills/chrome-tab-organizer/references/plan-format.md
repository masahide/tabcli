# Classification plan形式

`tabctl groups preview --plan FILE`へ渡すJSONは次の形にする。

```json
{
  "policy": "existing_groups_only",
  "assignments": [
    {
      "tabId": 123,
      "destination": { "groupId": 45 },
      "index": 0
    }
  ]
}
```

## policy

- `existing_groups_only`: 既存group IDへの移動だけ。既存グループへの整理依頼で使う。
- `ungrouped_only`: 現在ungroupedのタブだけを対象にする。
- `preserve_existing`: planにない既存所属を維持する。
- `rebuild_selected`: 選択したタブの所属を再構成する。

## assignment

- `tabId`: 一覧で直前に取得した正のtab ID。重複させない。
- `destination`: 次のいずれか一つだけを指定する。
  - 既存group: `{ "groupId": 45 }`
  - ungroup: `{ "ungroup": true }`
  - 新規group: `{ "title": "調査", "color": "blue" }`
- `pinned`: optional boolean。
- `index`: optional non-negative integer。グループ内順序を指定するときに使う。

新規groupのcolorは`grey`、`blue`、`red`、`yellow`、`green`、`pink`、`purple`、`cyan`、`orange`から選ぶ。`existing_groups_only`では新規groupを指定しない。

直前の`tabs content`結果に依存するplanでは、その`contentRevision`を`contentRevisions`の`revision`へ含める。compareとdiffはrevisionを返さないため指定しない。通常のタイトル・URL分類でも省略する。
