# tabcliをWindowsで使う

Windows 11 x64、Google Chrome Stable 121以降、PowerShell 5.1以降を対象にする。インストールは現在ユーザーの`HKCU`と`LOCALAPPDATA`だけを使い、管理者権限やPATH変更は行わない。

## インストール

Windows PowerShellで、最新のstable Releaseをインストールする。

```powershell
irm https://raw.githubusercontent.com/masahide/tabcli/main/install.ps1 | iex
```

このbootstrapはGitHub CLIを必要とせず、GitHub APIからWindows amd64 bundleと`SHA256SUMS`を取得してSHA-256を検証する。管理者権限は不要である。remote scriptを事前確認してから実行する場合:

```powershell
irm https://raw.githubusercontent.com/masahide/tabcli/main/install.ps1 -OutFile install-tabcli.ps1
Get-Content .\install-tabcli.ps1
.\install-tabcli.ps1
```

ローカルの配布ZIPを展開済みの場合は、展開先で次を実行する。

```powershell
.\install.ps1
```

更新時に`tabcli.exe`が使用中なら、インストーラーはプロセスを強制終了せず失敗する。Google Chromeを完全に終了して再実行する。バイナリはAuthenticode未署名のため、Windows SmartScreenが警告する場合がある。

配置先は次のとおり。

- 実行ファイル: `%LOCALAPPDATA%\Programs\tabcli\tabcli.exe`
- 製品データ: `%LOCALAPPDATA%\tabcli`
- Native Messaging manifest: `%LOCALAPPDATA%\tabcli\native-messaging\io.github.masahide.tabcli.json`
- Chrome拡張: `%LOCALAPPDATA%\tabcli\releases\VERSION\tabcli-extension-unpacked`

## Chrome拡張を読み込む

1. `chrome://extensions`を開く。
2. デベロッパーモードを有効にする。
3. 「パッケージ化されていない拡張機能を読み込む」を選ぶ。
4. インストーラーが表示したextension directoryを選ぶ。
5. extension IDが`ddgfmgclndpdobieomcjaklboinbaoel`であることを確認する。

## 接続確認

```powershell
$Tabcli = "$env:LOCALAPPDATA\Programs\tabcli\tabcli.exe"
& $Tabcli --json doctor
& $Tabcli --json status
& $Tabcli --json list
& $Tabcli --json group list
```

`BROWSER_DISCONNECTED`の場合はChromeを自動起動せず、拡張機能が有効か、`doctor`の`native_manifest`、`registry`、`discovery`を確認する。

## 主な操作

```powershell
& $Tabcli --json content TAB_ID --max-chars 10000
& $Tabcli --json compare TAB_ID_A TAB_ID_B
& $Tabcli --json diff TAB_ID_A TAB_ID_B
& $Tabcli --json close --confirm TAB_ID
& $Tabcli --json group preview --plan .\plan.json
& $Tabcli --json group apply --preview-id PREVIEW_ID
& $Tabcli --json group undo
```

`close`はUndoできない。`group apply`は承認済みpreview IDを必須とし、失敗時に自動再試行しない。

## アンインストール

```powershell
& $Tabcli uninstall
```

製品が管理するHKCU登録、manifest、runtime、settingsだけを削除する。Chrome拡張と実行ファイルは必要に応じて手動で削除する。
