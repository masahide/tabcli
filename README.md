# tabcli

Chrome拡張、Native Messaging Host、MCP、CLIを組み合わせ、現在開いているGoogle Chromeタブを参照・比較・整理するツールです。Windows 11 x64とChrome Stableを先行対象とし、現在ユーザーのHKCUとLOCALAPPDATAだけへインストールします。

## できること

- 開いているタブとタブグループの一覧を取得する
- タブの可視テキストを取得し、2つのタブの一致確認や差分比較を行う
- 変更内容をpreviewしてからタブをグループへ整理し、直前の整理をUndoする
- CLI、MCP、[AIエージェント向けSkill](skills/tabcli/SKILL.md)から同じChrome状態を扱う

## Windowsへインストール

Windows PowerShellから最新のstable Releaseをワンライナーでインストールできます。

```powershell
irm https://raw.githubusercontent.com/masahide/tabcli/main/install.ps1 | iex
```

管理者権限とGitHub CLIは不要です。インストーラーは検証済みのReleaseを現在ユーザー領域へ配置し、`tabcli`をユーザーPATHへ追加します。

インストール後は次の操作を行います。

1. Chromeで`chrome://extensions`を開き、デベロッパーモードを有効にする。
2. 「パッケージ化されていない拡張機能を読み込む」から、インストーラーが表示したextension directoryを選ぶ。
3. 拡張機能を再読み込みするかChromeを再起動する。
4. 新しいPowerShellで接続を確認する。

```powershell
tabcli --json doctor
tabcli --json list
```

Chrome拡張の読み込みや接続エラーへの対処は[Windowsインストール・利用ガイド](docs/getting-started-windows.md)を参照してください。

## 基本操作

```powershell
tabcli list
tabcli group list
tabcli --json content 123 --max-chars 10000
tabcli --json compare 123 456
tabcli --json diff 123 456 --max-chars 50000 --max-diff-chars 20000
tabcli --json close --confirm 123 456
tabcli --json group preview --plan .\plan.json
tabcli --json group apply --preview-id PREVIEW_ID
tabcli --json group undo
```

人間向け表示が既定です。スクリプトやAIエージェントから利用する場合はトップレベルの`--json`を指定します。

グループ変更は`group preview`の内容を確認してから、その`previewId`で`group apply`します。タブを閉じる操作はUndoできないため、一覧でIDを確認してから`close --confirm`を実行してください。

## ドキュメント

- [Windowsインストール・利用ガイド](docs/getting-started-windows.md)
- [AIエージェント向けSkill](skills/tabcli/SKILL.md)
- [開発者ガイド](docs/development.md)
- [要求仕様書](docs/requirements.md)
- [macOSガイド（再検証予定）](docs/getting-started.md)
