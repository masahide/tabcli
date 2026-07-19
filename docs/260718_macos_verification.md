# macOS MVP検証記録（2026-07-18）

## 検証環境

- macOS / Apple M2 Pro / `darwin/arm64`
- ローカルChrome Stable: 150.0.7871.125
- 実ブラウザ自動結合テスト: Chrome for Testing Stable 151.0.7922.34
- 拡張機能ID: `ddgfmgclndpdobieomcjaklboinbaoel`

## 実Chrome結合テスト

`CHROME_REAL_INTEGRATION=1 go test -tags=integration ./integration -run TestRealChromeNativeHTTPCLIAndStdioMCP`を実行し、次の経路を一つのテストで確認した。

- unpacked MV3拡張機能のservice worker起動
- Native Messaging Host起動とloopback discovery
- 直接HTTP MCP呼び出し
- CLI呼び出し
- stdio MCP proxy呼び出し

結果はPASS。要求適合後の最終成果物では、service workerの準備完了からNative Messaging・discoveryの利用可能化まで202.950 ms、stdio MCP初期化は26.809 msだった。目標の1秒・500 msを満たした。

## ベンチマーク

`go test -run '^$' -bench 'Benchmark(TabsList500|Apply200Tabs|ProxyInitialization)$' -benchmem ./internal/app ./internal/plan ./internal/tools`を実行した。

| 対象 | 実測 | 目標 | 判定 |
| --- | ---: | ---: | --- |
| 500タブ一覧 | 118,495 ns/op（約0.118 ms） | 1秒以内 | PASS |
| 200タブapply | 147,239 ns/op（約0.147 ms） | 5秒以内 | PASS |
| proxy初期化 | 1,487,306 ns/op（約1.487 ms） | 500 ms以内 | PASS |

全項目が性能目標を十分に満たしたため、実測に基づく追加最適化は不要と判断した。

## 0.2.1 Native Host署名・更新手順の再検証

### 障害証拠と判断範囲

- 2026-07-18 16:23:02に、インストール済み`tabcli`の最終パスへ`cp`で直接上書きした作業記録がある。
- その5秒後の16:23:07と、16:23:31、16:23:43に、UUID `C07C1CDB-85F7-3862-CE76-98645DD0F634`の`tabcli`が`SIGKILL (Code Signature Invalid)`、`Taskgated Invalid Signature`で終了したmacOS診断レポートが3件ある。
- 16:24:09に一時ファイルから`mv`する原子的置換へ変更し、それ以後に`tabcli`のコード署名クラッシュレポートは発生していない。
- 障害以前に動作していたバイナリの現物・checksum・署名検証ログは残っていない。そのバイナリがアドホック署名付きだったかは不明であり、推測を監査根拠には使用しない。

以上から、今回の直接原因は使用中の最終パスへの直接上書きと判断し、配布品質の追加対策として全macOS成果物への明示的なアドホック署名も実施した。

### 対策後のrelease検証

- release entrypointはarm64・amd64の両方へidentifier `io.github.masahide.tabcli.tabcli`のアドホック署名を付け、直後に`codesign --verify --strict --verbose=2`を実行する。
- 両成果物で`Signature=adhoc`、`TeamIdentifier=not set`を確認した。Developer ID署名・notarizationは使用していない。
- version `0.2.1`、commit `validation`の同一入力でreleaseを2回生成し、`SHA256SUMS`が完全一致した。各release内の全checksumと、CPU別ZIPから展開した`tabcli`の厳格署名検証も成功した。
- Go全package、Vitest 53件、TypeScript typecheck、extension clean buildが成功した。

### 隔離Chrome結合試験

Chrome Stable 150.0.7871.129を、一時プロファイル・一時HOME・診断専用Native Host名で起動し、0.2.1成果物だけを使用した。

- extension 0.2.1とhost 0.2.1のprotocol 2 handshake: PASS
- `tabcli --json doctor`: executable、native_manifest、discovery、chrome、mcpの全checkがtrue
- `tabcli --json tabs list`: PASS
- `tabcli --json groups list`: PASS
- 存在しないtab ID `2147483647`への確認済みclose: `TAB_NOT_FOUND`となり、タブ変更なし

自動化した隔離Chromeの終了ではNative Hostが強制終了されstale discoveryが残ったが、`doctor`はPID不在として正しく拒否した。通常Chromeプロファイルと実タブはこの試験で操作していない。
