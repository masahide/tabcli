# tabcli開発者ガイド

Windows 11 x64を先行開発・配布対象とする。Go 1.25、Node.js 24、npm、PowerShell 5.1以降、Google Chrome Stable 121以降を用意する。

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
