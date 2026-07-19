# requirements.md適合監査（2026-07-18）

## 結論

`docs/requirements.md`のmacOS MVP要求を実装、テスト、実Chrome結合試験、release成果物へ照合した。下記の明示除外を除き、監査で見つかった契約差は修正済みである。

## 明示除外

- Windows固有実装・成果物・検証: SYS-021、NFR-008、NFR-012、DIST-001、DIST-005、DIST-009、AC-10のWindows部分
- macOS Developer ID署名・notarization: DIST-008（ユーザー指示）。identity不要のアドホック署名は除外せず、DIST-013として実施する。

## 監査で修正した差

| 対象 | 修正内容 |
| --- | --- |
| 確定済み識別子 | Go module、Native Messaging Host名、`ChromeTabOrganizer`製品・runtime directoryを実装計画と統一 |
| `chrome_tabs_list` | `ungrouped`、`sortBy`、`sortOrder`、position・last accessed・inactive duration・created at sortを要求どおり実装 |
| `chrome_tab_content_get` | `maxChars`、`originalCharCount`、`returnedCharCount`へ公開JSON契約を統一 |
| CLI | `--window`、`--ungrouped`、`--sort`、`--max-chars`、`--preview-id`を要求どおり実装し、versionへGo・OS・CPUを追加 |
| エラー | 要求されたエラーコードを定義し、MCPとCLIで`code`、`message`、`retryable`、任意detailsを返すよう統一 |
| doctor | manifestの存在だけでなく、host名、実行ファイル絶対path、type、allowed originの不一致を検出 |
| lifecycle | handshakeでextension version、profile ID、protocol互換範囲を交換・検証 |
| Undo | 一括適用と無関係なタブ・グループをUndo snapshotへ含めないよう対象を限定 |
| privacy / Skill | 初回本文取得前のモデル提供者送信可能性、事前承認、履歴の不完全性、7日基準の説明を明記 |
| 全サイト権限 | 全HTTP/HTTPS match patternをrequired `host_permissions`へ移し、静的content scriptを使わず、明示された1タブまたは2タブへの要求時だけ固定関数を動的注入する仕様・実装へ更新 |
| 本文比較・差分 | 2タブの可視テキスト全体を拡張機能内でSHA-256化して本文なしで比較するtoolと、変更行だけを返す上限付きdiff toolを追加 |
| host pattern・エラー | port付きlocalhostを有効なport非依存match patternへ正規化し、比較前に消失したtabをprotocolエラーではなく`TAB_NOT_FOUND`へ変換 |
| タブクローズ | 正確なtab ID、明示確認、全対象の事前検証、破壊的annotationを持つCLI/MCP primitiveを追加 |
| macOSバイナリ更新 | arm64・amd64成果物へ再現可能なアドホック署名と厳格検証を追加し、使用中の最終パスを直接上書きしない原子的更新手順へ変更 |
| GitHub Release配布 | 単一release entrypointのローカル実行でversion、checksum、署名、アーカイブ内容をgateし、対応commitのtagへ成果物を添付する構成へ変更 |
| `gh`インストール | Release添付ブートストラップを短いパイプ形式で起動し、CPU別ZIPのchecksum検証後に同梱インストーラーでCLIを原子的に更新し、拡張機能をversion別配置する手順を追加。ブートストラップ自体の信頼境界と事前確認手順も明記 |

## 要求群別の根拠

| 要求群 | 主な実装・検証根拠 |
| --- | --- |
| SYS-001〜027 | MV3 TypeScript拡張、単一Goバイナリ、Native Messaging framing、loopback Streamable HTTP、stdio proxy、origin・protocol・profile handshake tests |
| FR-001〜012 | tab/group snapshot、incognito・window・group・ungrouped filter、operable、inactive duration、全sort、明示確認済みIDだけのclose tests |
| FR-051〜064 | top-level Chrome event listeners、`storage.local` activity、`storage.session` browser session、reconciliation・tab ID reuse tests |
| FR-071〜092 | main-frame固定抽出、SHA-256既知値・本文非返却、変更行だけのdiff、各上限、permission/access/stale判定、非永続化・untrusted tests |
| FR-101〜110 | 4 policy、schema、cross-window、pinned維持、preview無変更、revision・5分expiry tests |
| FR-201〜207 | preview必須、対象限定Undo snapshot、rollback、partial、直前1件Undo、group属性復元 tests |
| MCP-001〜021 | 公式Go SDK typed schema、9 tool catalog、annotations、instructions、構造化retryable error、HTTP/stdio contract tests |
| SKILL-001〜018 | CLI-firstの`skills/tabcli/SKILL.md`、plan形式reference、workflow/safety fixture tests |
| CLI-001〜020 | MCP client-only CLI、共通catalog help、JSON golden、固有exit codes、literal flag mapping、doctor tests |
| SEC-001〜027 | loopback/Host/Origin/Bearer middleware、0600/0700、atomic rename、owner/symlink、tabignore、trusted storage、hash/diff境界、artifact/privacy tests |
| NFR-001〜014 | 実Chrome結合計測、500/200 tab・proxy benchmarks、blocking I/O lifecycle、versioned protocols、reproducible release tests |
| DIST-001〜015 | macOS arm64/amd64 static binaries、アドホック署名検証、原子的更新、単一binary、current-user manifest、fixed extension ID、checksum/version metadata、ローカルGitHub Release手順、検証付き`gh` install、safe uninstall、compatibility guidance |
| AC-01〜13 | Go・Vitest・integration・stress・benchmark・release entrypointとローカルRelease手順でmacOS適用分を確認 |

性能と実Chromeの実測値は`docs/260718_macos_verification.md`に記録した。

## OpenAI agent CLI patterns適合

参照: [OpenAI Codex CLI Patterns](https://github.com/openai/skills/blob/main/skills/.curated/cli-creator/references/agent-cli-patterns.md)

| パターン | 対応 |
| --- | --- |
| composable primitives | `tabs list/content/compare/diff/close`、`groups list/preview/apply/undo`、`doctor`へ分割 |
| product nouns then verbs | `tabs list`、`groups preview`などで統一 |
| help is interface | 共通tool catalog metadataからトップレベルに完全なusageを生成し、各commandの`--help`を成功終了 |
| human text + stable JSON | 人間向け既定表示と全command共通のトップレベル`--json`を提供 |
| stdout/stderr separation | JSONとMCP messageはstdoutだけ、診断・privacy noticeはstderrだけへ出力 |
| documented success/error | READMEに成功object、`error.code/message/retryable`、終了コードを記載 |
| secret redaction | Bearer Token、Cookie、private header、本文を引数・ログ・成果物へ出力しない検査を実施 |
| doctor without connection | Chrome未接続でも`doctor --json`がchecksを返し、クラッシュしない |
| safe writes | group変更はpreview/applyを分離し、不可逆なcloseは正確なIDと`--confirm`、MCPの`confirmed`を必須化 |
| companion skill | READMEより小さいSkillでdoctor開始、JSON利用、読み取り、変更承認境界を案内 |

pagination、download/upload、raw API escape hatchは、外部のページングAPIやファイル転送を提供しない本CLIには該当しない。
