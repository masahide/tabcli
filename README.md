# tabcli

Chrome拡張機能、Native Messaging Host、MCP、CLIを組み合わせ、現在開いているChromeタブをAIエージェントから参照・分類・整理するためのプロジェクトです。

プロダクト名、リポジトリ名、CLI実行ファイル名はすべて`tabcli`に統一しています。CLIのサブコマンド構造は互換性を保つため現行のネスト型を維持しており、フラット化は別の実装変更として扱います。

## Documents

- [要求仕様書](docs/requirements.md)
- [要求適合監査](docs/260718_requirements_audit.md)
- [macOS検証記録](docs/260718_macos_verification.md)
- [macOSインストール・利用ガイド](docs/getting-started.md)
- [開発者ガイド](docs/development.md)

## Build and release

Go 1.25とNode.js 24を用意し、リポジトリrootで次を実行します。

```bash
go run ./cmd/release --out dist --version 0.3.0
```

この単一entrypointがtest、拡張機能build、`darwin/arm64`・`darwin/amd64`の`CGO_ENABLED=0` build、identity不要のアドホック署名と厳格検証、再現可能ZIP、SHA-256 checksum、version metadata、成果物の秘密情報検査を実行します。ユーザー判断によりDeveloper ID署名・notarizationは行いません。

Releaseは開発者が単一entrypointをローカル実行して検証済み成果物を生成し、対応commitへ`vVERSION`タグを付けて`gh release create`で公開します。

最新Releaseに添付されたインストーラーを`gh`で取得し、そのまま実行するワンライナーです。インストーラーがCPU選択、配布ZIPのchecksum・アドホック署名・version整合の検証、CLIとunpacked extensionの配置、Native Messaging登録を行います。Chromeへのunpacked extensionの読み込みだけは、最後に表示されるdirectoryを使って手動で行います。

```bash
set -o pipefail; gh release download -R masahide/tabcli -p install-with-gh.sh -O - | /bin/bash
```

この形式では、GitHub Releaseから取得した`install-with-gh.sh`自体を事前検証せず実行する。スクリプトは配布ZIPを実行する前に`SHA256SUMS`を検証し、ZIP内インストーラーが署名とversionを再検証する。事前に内容を確認する場合は、パイプせず`--output install-with-gh.sh`で保存してから開く。

Releaseからのインストールは[macOSインストール・利用ガイド](docs/getting-started.md)、ソースビルドとRelease作成は[開発者ガイド](docs/development.md)を参照してください。

## CLI

CLIは人間向け表示を既定とし、機械処理ではトップレベルの`--json`を指定します。成功時は結果オブジェクト、失敗時は`{"error":{"code":"...","message":"...","retryable":false}}`を標準出力へ返し、終了コードを非0にします。

`tabcli --help`は利用可能な名詞・動詞を一覧し、`tabcli tabs list --help`のように各コマンドのliteralなフラグを確認できます。JSONモードではJSONだけをstdoutへ出し、診断と本文取扱い通知はstderrへ分離します。Token、Cookie、private headerは出力しません。`doctor --json`はChrome未接続や設定不備もクラッシュせずchecksとして返します。

```bash
tabcli --json version
tabcli --json doctor
tabcli --json tabs list
tabcli --json tabs content 123 --max-chars 10000
tabcli --json tabs compare 123 456
tabcli --json tabs diff 123 456 --max-chars 50000 --max-diff-chars 20000
tabcli --json tabs close --confirm 123 456
tabcli --json groups preview --plan plan.json
tabcli --json groups apply --preview-id PREVIEW_ID
tabcli mcp serve
```

`tabs compare`は2タブの可視テキスト全体を拡張機能内でSHA-256化し、本文を返さず一致だけを判定します。`tabs diff`は変更行だけを返し、未変更行と比較元全文はNative Messaging以降へ返しません。通常出力も変更行だけで、上限・切り詰め情報が必要な場合は`--json`を使います。どちらもタブを自動的に閉じません。

グループ変更は必ず`groups preview`で差分を確認し、その結果の`previewId`を指定して`groups apply`します。タブのクローズは一覧で正確なtab IDを確認し、対象を明示承認した場合だけ`tabs close --confirm`で実行します。クローズはUndoできず、重複検出や対象の自動選択は行いません。コマンド、引数、出力契約は同梱の[AIエージェント向けSkill](skills/tabcli/SKILL.md)にも記載しています。

終了コードは成功が`0`、usageが`2`で、接続失敗、入力不正、stale plan、本文権限、apply/rollback、Undo、protocol不整合をそれぞれ別コードで返します。空の一覧は正常終了です。

## Install the unpacked extension

Releaseからのインストール、Chromeへの読み込み、動作確認、アンインストールは[macOSインストール・利用ガイド](docs/getting-started.md)にまとめています。

manifestの公開鍵から得られる固定extension IDは`ddgfmgclndpdobieomcjaklboinbaoel`です。

拡張機能は通常のHTTP/HTTPSページに対するhost permissionを持ちますが、常駐content scriptは注入しません。ページ本文の取得・SHA-256比較・変更行差分は、ユーザーが明示した1タブまたは2タブに対して該当コマンドを実行したときだけ、同梱の固定関数を`chrome.scripting.executeScript()`で動的に実行します。
