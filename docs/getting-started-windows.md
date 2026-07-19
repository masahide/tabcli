# tabcliをWindowsで使う

Windows 11 x64、Google Chrome Stable 121以降、PowerShell 5.1以降を対象にする。インストールは現在ユーザーの`HKCU`と`LOCALAPPDATA`だけを使い、管理者権限は不要である。実行ファイルの配置先を現在ユーザーのPATHへ追加する。

## インストール

Windows PowerShellで、最新のstable Releaseをインストールする。

```powershell
irm https://raw.githubusercontent.com/masahide/tabcli/main/install.ps1 | iex
```

このbootstrapはGitHub CLIを必要とせず、GitHub APIからWindows amd64 bundleと`SHA256SUMS`を取得してSHA-256を検証する。管理者権限は不要である。

更新時に`tabcli.exe`が使用中なら、インストーラーはプロセスを強制終了せず失敗する。Google Chromeを完全に終了して再実行する。バイナリはAuthenticode未署名のため、Windows SmartScreenが警告する場合がある。

配置先は次のとおり。

- 実行ファイル: `%LOCALAPPDATA%\Programs\tabcli\tabcli.exe`
- 製品データ: `%LOCALAPPDATA%\tabcli`
- Native Messaging manifest: `%LOCALAPPDATA%\tabcli\native-messaging\io.github.masahide.tabcli.json`
- Chrome拡張: `%LOCALAPPDATA%\tabcli\releases\VERSION\tabcli-extension-unpacked`

インストール済みのPowerShellとは別のプロセスで`tabcli`を使う場合は、新しいターミナルを開いてPATHを反映する。再インストールしても同じPATH要素は重複して追加されない。

## インストール直後にすること

Native Hostの登録だけではChromeとの接続はまだ開始されない。次の順序でunpacked拡張をChromeへ読み込み、接続を確認する。

### 1. Chrome拡張を読み込む

1. `chrome://extensions`を開く。
2. デベロッパーモードを有効にする。
3. 「パッケージ化されていない拡張機能を読み込む」を選ぶ。
4. インストーラーが表示した次のextension directoryを選ぶ。

   ```text
   %LOCALAPPDATA%\tabcli\releases\VERSION\tabcli-extension-unpacked
   ```

5. extension IDが`ddgfmgclndpdobieomcjaklboinbaoel`であることを確認する。
6. 拡張機能の再読み込みボタンを押すか、Google Chromeを再起動する。

### 2. 接続を確認する

インストーラーを実行したPowerShellでは、そのまま`tabcli`を実行できる。別のPowerShellでは新しいターミナルを開いてから実行する。

```powershell
tabcli --json doctor
tabcli --json status
tabcli --json list
tabcli --json group list
```

`doctor`の`executable`、`native_manifest`、`registry`、`discovery`、`chrome`、`mcp`がすべて`ok: true`なら利用準備が完了している。

`BROWSER_DISCONNECTED`または`discovery`、`chrome`、`mcp`の失敗がある場合:

1. Google Chromeが起動していることを確認する。
2. `chrome://extensions`でtabcliが有効になっていることを確認する。
3. tabcliの再読み込みボタンを押す。
4. 改善しなければGoogle Chromeを完全に終了して再起動する。
5. 再度`tabcli --json doctor`を実行する。

### 3. タブ一覧を試す

```powershell
tabcli --json list
tabcli --json group list
```

空の配列はエラーではなく、対象となる通常ウィンドウのタブまたはグループがない状態を表す。

## 主な操作

```powershell
tabcli --json content TAB_ID --max-chars 10000
tabcli --json compare TAB_ID_A TAB_ID_B
tabcli --json diff TAB_ID_A TAB_ID_B
tabcli --json close --confirm TAB_ID
tabcli --json group preview --plan .\plan.json
tabcli --json group apply --preview-id PREVIEW_ID
tabcli --json group undo
```

`close`はUndoできない。`group apply`は承認済みpreview IDを必須とし、失敗時に自動再試行しない。

## アンインストール

```powershell
tabcli uninstall
```

製品が管理するHKCU登録、manifest、runtime、settingsだけを削除する。Chrome拡張と実行ファイルは必要に応じて手動で削除する。
