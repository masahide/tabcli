# tabcli開発者ガイド

Windows 11 x64を先行開発・配布対象とする。Go 1.25、Node.js 24、npm、PowerShell 5.1以降、Google Chrome Stable 121以降を用意する。

## 固定IDと互換性

実装と配布物では次のIDと配置を維持する。

- Go module: `github.com/masahide/tabcli`
- Native Messaging Host: `io.github.masahide.tabcli`
- Chrome extension ID: `ddgfmgclndpdobieomcjaklboinbaoel`
- Windows product data: `%LOCALAPPDATA%\tabcli`
- Windows executable: `%LOCALAPPDATA%\Programs\tabcli\tabcli.exe`

extension IDを維持するため、[extension/manifest.json](../extension/manifest.json)の公開鍵は変更しない。MCP tool名とschemaは既存APIとして`chrome_*`契約を維持する。

CLIは`list`、`content`、`compare`、`diff`、`close`、`group`をトップレベルの利用者向けコマンドとする。旧`tabs list`と`groups list`に互換aliasはなく、未知のコマンドとして拒否する。管理コマンドは`install`、`uninstall`、`status`、`doctor`、`version`、stdio MCP proxyは`mcp serve`で提供する。

機械処理ではトップレベルの`--json`を指定する。成功結果または構造化エラーだけをstdoutへ返し、診断とuntrusted page contentの注意はstderrへ分離する。Token、Cookie、private headerは出力しない。

## 検証

```powershell
go test ./...
Set-Location extension
npm ci
npm test
npm run typecheck
npm run build
Set-Location ..
```

Windows用バイナリだけを作る場合:

```powershell
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o .\dist\tabcli.exe .\cmd\tabcli
```

## Release

cleanなHEADとextensionのversionを指定versionへ揃え、次を実行する。

```powershell
.\scripts\publish-windows-release.ps1 -Version 0.3.0
```

`dist`には`tabcli.exe`、`tabcli-extension.zip`、`install.ps1`、`install-with-gh.ps1`、`version.json`、`INSTALL.txt`、`SHA256SUMS`、`tabcli-VERSION-windows-amd64.zip`が生成される。Release entrypointはGo/TypeScriptテスト、extension ID検証、Windows amd64クロスビルド、再現可能ZIP生成、秘密情報検査を行う。

通常実行はビルドと成果物検証だけを行い、GitHubへ変更を加えない。検証後、同じcleanなHEADから`-Publish`を付けて実行すると、`vVERSION` annotated tagを作成して`origin`へpushし、`gh release create`でWindows成果物を公開する。

```powershell
.\scripts\publish-windows-release.ps1 -Version 0.3.0 -Publish
```

公開には`gh auth status`が成功する認証が必要となる。すでに同じHEADを指すtagが存在する場合は再利用するため、tag push後にRelease作成だけが失敗した場合も再実行できる。別commitを指す同名tagや既存Releaseがある場合は安全のため失敗する。

Windows bundleには`tabcli.exe`、extension ZIP、PowerShell installer、`INSTALL.txt`、version metadataを含める。Release assetとして`install-with-gh.ps1`と`SHA256SUMS`も生成する。バイナリはAuthenticode未署名のためSmartScreenが警告する場合がある。

インストーラーはChromeや`tabcli.exe`を強制終了しない。更新時にバイナリを置換できない場合は、Chromeを完全に終了して再実行する。インストール成功時は実行ファイルの配置先を現在ユーザーのPATHへ重複なく追加する。

## Windows実機統合

配布物を作成してChromeを終了した状態からインストールし、unpacked extensionを読み込む。自動統合テストは明示的に有効化する。

```powershell
$env:CHROME_REAL_INTEGRATION = "1"
$env:TABCLI_INTEGRATION_BINARY = (Resolve-Path .\dist\tabcli.exe)
$env:TABCLI_INTEGRATION_EXTENSION = (Resolve-Path .\dist\tabcli-extension-unpacked)
go test -tags integration .\integration -v
```

テストはHKCUのcanonical Native Messaging登録、discovery生成、HTTP MCP、`tabcli --json list`、stdio MCPを確認する。終了時はテスト用profileと製品が管理する登録だけを削除する。

macOSの既存コードとunit testは維持するが、codesign、notarization、実機統合、Release再検証はWindows MVP後の後続作業とする。
