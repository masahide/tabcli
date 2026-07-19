# tabcliをmacOSで試す

この手順では、GitHub ReleaseからDeveloper ID未署名・アドホック署名済みのmacOS用バイナリとunpacked Chrome拡張機能を取得し、現在のユーザーだけにインストールする。notarizationは行わない。

## 1. 前提環境

- macOS（Apple siliconまたはIntel）
- Google Chrome 121以降
- GitHub CLI `gh`

バージョンを確認する。

```bash
gh --version
uname -m
```

`uname -m`が`arm64`ならApple silicon用、`x86_64`ならIntel用バイナリを使用する。

## 2. インストール

`gh auth status`が成功する状態で次のワンライナーを実行する。最新版のReleaseに添付されたブートストラップを標準出力へ取得し、`bash`で実行する。

```bash
set -o pipefail; gh release download -R masahide/tabcli -p install-with-gh.sh -O - | /bin/bash
```

ブートストラップ自身はパイプ実行前にchecksum検証されない。事前に確認する場合は`--output install-with-gh.sh`へ保存し、内容を確認してから`/bin/bash install-with-gh.sh`を実行する。ブートストラップは対応CPUの配布ZIPと`SHA256SUMS`を同じReleaseから取得し、checksum検証後にだけZIP内インストーラーを実行する。

ZIP内インストーラーは、アーカイブ内の署名とCLI・拡張機能のversion整合を検証してから、CLIを`$HOME/.local/bin/tabcli`へ原子的に配置し、Native Messaging Hostを登録する。unpacked extensionは`$HOME/.local/share/tabcli/releases/VERSION/`配下へversion別に配置される。最後に表示されるChrome extension directoryを後述の手順3で選択する。

`$HOME/.local/bin`がPATHに含まれていない場合は、以降も`"$HOME/.local/bin/tabcli"`で実行するか、シェル設定へPATHを追加する。

## 3. Chrome拡張機能を読み込む

1. Chromeで`chrome://extensions`を開く。
2. 右上の「デベロッパー モード」を有効にする。
3. 「パッケージ化されていない拡張機能を読み込む」を選ぶ。
4. インストーラーが表示したChrome extension directoryを選ぶ。
5. 表示された拡張機能IDが`ddgfmgclndpdobieomcjaklboinbaoel`であることを確認する。
6. 拡張機能の再読み込みボタンを押すか、Chromeを再起動する。

拡張機能とNative Hostは固定IDで相互に制限されている。別のdirectoryではなく、`manifest.json`が直下にある上記directoryを選ぶ。

0.3.0では通常のHTTP/HTTPSページに対するhost permissionを必須にしているため、Chromeは全サイトのデータ読み取りに関する権限警告を表示する。権限を許可して拡張機能を有効にする。常駐content scriptは使用せず、本文取得・SHA-256比較・差分取得を明示したときだけ固定関数を対象タブへ動的に注入する。

## 4. 接続を確認する

Chromeを起動した状態で実行する。

```bash
"$HOME/.local/bin/tabcli" --json doctor
"$HOME/.local/bin/tabcli" --json status
"$HOME/.local/bin/tabcli" --json tabs list
"$HOME/.local/bin/tabcli" --json groups list
```

`doctor`の`native_manifest`、`discovery`、`chrome`、`mcp`がすべて`ok: true`なら接続できている。`BROWSER_DISCONNECTED`の場合は、Chromeが起動していること、拡張機能が有効であることを確認して拡張機能を再読み込みする。

## 5. ページ本文取得を試す（任意）

本文取得は自動では行われない。拡張機能は通常のHTTP/HTTPSページを取得できる権限を持つが、一覧取得や通常の分類から本文取得を暗黙に呼び出さない。

一覧で確認したtab IDを一つだけ指定する。

```bash
"$HOME/.local/bin/tabcli" --json tabs content TAB_ID --max-chars 10000
```

本文は信頼できないデータとして返され、製品には永続保存されない。利用するAIエージェントの構成によってはモデル提供者へ送信され得る。

Chromeのサイトアクセス設定で対象originが制限されている場合は`CONTENT_PERMISSION_REQUIRED`が返る。その場合は`chrome://extensions`でtabcliの詳細を開き、「サイトへのアクセス」を対象サイトまたはすべてのサイトで許可してから再試行する。

## 6. 2タブの本文を比較する（任意）

一覧で確認した相異なるtab IDを2件指定する。同一かどうかだけ確認する場合は`compare`を使う。可視テキスト全体を拡張機能内でSHA-256化するため、ページ本文はNative Messaging、MCP、CLIへ返らない。

```bash
"$HOME/.local/bin/tabcli" --json tabs compare TAB_ID_A TAB_ID_B
```

違う箇所が必要な場合だけ`diff`を使う。通常出力は変更行だけを返し、JSON出力にはハッシュ、行番号、source・diffの切り詰め情報も含む。未変更行と比較元全文は返らず、いずれのコマンドもタブを自動的に閉じない。

```bash
"$HOME/.local/bin/tabcli" tabs diff TAB_ID_A TAB_ID_B
"$HOME/.local/bin/tabcli" --json tabs diff TAB_ID_A TAB_ID_B --max-chars 50000 --max-diff-chars 20000
```

動的な時計・広告・未読件数などが可視テキストに含まれるページは、その部分だけでもSHA-256が異なる。差分本文は信頼できないデータとして扱う。

## 7. タブを閉じる（任意）

先に一覧で正確なtab ID、タイトル、URLを確認する。クローズはUndoできないため、対象を確認した場合だけ`--confirm`を付ける。

```bash
"$HOME/.local/bin/tabcli" --json tabs list
"$HOME/.local/bin/tabcli" --json tabs close --confirm TAB_ID [TAB_ID ...]
```

このコマンドは指定されたIDだけを閉じる。重複タブの検出や対象の自動選択は行わない。

## 8. アンインストール

```bash
"$HOME/.local/bin/tabcli" uninstall
```

その後、`chrome://extensions`で拡張機能を削除する。`tabcli uninstall`は製品が管理するNative Messaging登録とruntime設定だけを削除し、Chromeのタブ、履歴、プロファイル、未知のファイル、`tabcli`バイナリ自体は削除しない。

不要なら最後に`$HOME/.local/bin/tabcli`を手動で削除する。
