# tabcli

Chrome拡張、Native Messaging Host、MCP、CLIを組み合わせ、現在開いているGoogle Chromeタブを参照・比較・整理するツールです。Windows 11 x64とChrome Stableを先行対象とし、現在ユーザーのHKCUとLOCALAPPDATAだけへインストールします。

## Documents

- [Windowsインストール・利用ガイド](docs/getting-started-windows.md)
- [要求仕様書](docs/requirements.md)
- [開発者ガイド](docs/development.md)
- [既存macOSガイド（後続再検証対象）](docs/getting-started.md)
- [AIエージェント向けSkill](skills/tabcli/SKILL.md)

## Windows build and release

Go 1.25とNode.js 24を用意し、cleanなリポジトリrootで実行します。

```powershell
.\scripts\publish-windows-release.ps1 -Version 0.3.0
```

`tabcli-VERSION-windows-amd64.zip`には`tabcli.exe`、extension ZIP、`install.ps1`、`INSTALL.txt`、version metadataが含まれます。Release assetとして`install-with-gh.ps1`と`SHA256SUMS`も生成します。バイナリは未署名のためSmartScreenが警告する場合があり、更新時はChromeを完全に終了する必要があります。

成果物検証後にGitHub Releaseへ公開する場合は、cleanなHEADで`-Publish`を明示します。

```powershell
.\scripts\publish-windows-release.ps1 -Version 0.3.0 -Publish
```

インストールとunpacked extensionの読み込みは[Windowsガイド](docs/getting-started-windows.md)を参照してください。インストーラーはChromeを強制終了せず、PATHも変更しません。

## CLI

人間向け表示が既定です。機械処理ではトップレベルの`--json`を指定すると、成功結果または構造化エラーだけをstdoutへ返します。診断やuntrusted page contentの注意はstderrへ分離し、Token、Cookie、private headerは出力しません。

```text
tabcli.exe list
tabcli.exe content TAB_ID
tabcli.exe compare TAB_ID_A TAB_ID_B
tabcli.exe diff TAB_ID_A TAB_ID_B
tabcli.exe close --confirm TAB_ID...
tabcli.exe group list
tabcli.exe group preview --plan FILE
tabcli.exe group apply --preview-id ID
tabcli.exe group undo
```

管理コマンドの`install`、`uninstall`、`status`、`doctor`、`version`と、stdio MCP proxyの`mcp serve`も利用できます。旧`tabs list`と`groups list`に互換aliasはなく、未知のコマンドとして拒否します。

```powershell
$Tabcli = "$env:LOCALAPPDATA\Programs\tabcli\tabcli.exe"
& $Tabcli --json version
& $Tabcli --json doctor
& $Tabcli --json list --inactive-for 7d --include-activity
& $Tabcli --json content 123 --max-chars 10000
& $Tabcli --json compare 123 456
& $Tabcli --json diff 123 456 --max-chars 50000 --max-diff-chars 20000
& $Tabcli --json close --confirm 123 456
& $Tabcli --json group preview --plan .\plan.json
& $Tabcli --json group apply --preview-id PREVIEW_ID
& $Tabcli mcp serve
```

`compare`は可視テキストを拡張内でSHA-256化して一致を判定し、本文を返しません。`diff`は上限付きの変更行だけを返します。どちらもタブを自動的に閉じません。

グループ変更は必ず`group preview`の差分を明示承認した後、その`previewId`で`group apply`します。クローズは一覧で正確なIDを確認し、承認した同じIDだけを`close --confirm`へ渡します。クローズはUndoできません。

## Fixed identities

- Go module: `github.com/masahide/tabcli`
- Native Messaging Host: `io.github.masahide.tabcli`
- Chrome extension ID: `ddgfmgclndpdobieomcjaklboinbaoel`
- Windows product data: `%LOCALAPPDATA%\tabcli`

extension IDを維持するため、[extension/manifest.json](extension/manifest.json)の公開鍵は変更しません。MCP tool名とschemaは既存の`chrome_*`契約を維持します。
