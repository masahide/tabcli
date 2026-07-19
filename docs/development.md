# tabcli開発者ガイド

この資料では、macOSでソースからビルドし、開発用成果物を手動インストールする手順を説明する。一般利用者は[macOSインストール・利用ガイド](getting-started.md)を参照する。

## 前提環境

- macOS（Apple siliconまたはIntel）
- Go 1.25以降
- Node.js 24とnpm
- Google Chrome 121以降

```bash
go version
node --version
npm --version
uname -m
```

## ソースからビルド

リポジトリrootで実行する。

```bash
go run ./cmd/release --out dist --version 0.3.0
```

このコマンドはGo・TypeScriptのテストとtypecheckも実行し、次の成果物を`dist/`へ生成する。

- `tabcli-darwin-arm64`
- `tabcli-darwin-amd64`
- `tabcli-extension-unpacked/`
- `tabcli-extension.zip`
- CPU別配布ZIP
- `install-with-gh.sh`
- `SHA256SUMS`
- `version.json`
- `INSTALL.txt`

checksumを確認する。

```bash
cd dist
shasum -a 256 -c SHA256SUMS
cd ..
```

両CPU向けバイナリはrelease entrypoint内でidentity不要のアドホック署名を付け、`codesign --verify --strict`まで実行する。Developer ID署名とnotarizationは行わない。

## ソースビルドしたtabcliをインストール

Apple siliconの場合:

```bash
mkdir -p "$HOME/.local/bin"
install -m 755 dist/tabcli-darwin-arm64 "$HOME/.local/bin/tabcli.new"
codesign --verify --strict --verbose=2 "$HOME/.local/bin/tabcli.new"
mv -f "$HOME/.local/bin/tabcli.new" "$HOME/.local/bin/tabcli"
"$HOME/.local/bin/tabcli" install
```

Intel Macの場合は`tabcli-darwin-arm64`を`tabcli-darwin-amd64`へ置き換える。

最終パスの実行ファイルを`cp`やリダイレクトで直接上書きしない。ChromeがNative Hostとして実行中のinodeを書き換えるとmacOSがコード署名不整合としてプロセスを終了する可能性があるため、一時ファイルを検証してから同じdirectory内で`mv`する。

`tabcli install`は現在ユーザーのChrome Native Messaging manifestだけを作成する。manifestには実行中の`tabcli`の絶対パスが保存されるため、バイナリを移動した場合は移動先から再度実行する。

Chromeでは`chrome://extensions`を開き、デベロッパーモードを有効にして、`dist/tabcli-extension-unpacked`を読み込む。

## GitHub Releaseを作成

`extension/manifest.json`と`extension/package.json`のversionを一致させ、cleanな`main`で次を実行する。

```bash
VERSION=0.3.0
COMMIT="$(git rev-parse HEAD)"
test -z "$(git status --porcelain)"
go run ./cmd/release --out dist --version "$VERSION" --commit "$COMMIT"
(cd dist && shasum -a 256 -c SHA256SUMS)
git tag -a "v$VERSION" "$COMMIT" -m "tabcli $VERSION"
git push origin "v$VERSION"
gh release create "v$VERSION" \
  dist/tabcli-*.zip \
  dist/tabcli-darwin-* \
  dist/SHA256SUMS \
  dist/version.json \
  dist/INSTALL.txt \
  dist/install-with-gh.sh \
  --title "tabcli $VERSION" \
  --generate-notes \
  --verify-tag
```

release entrypointはテスト、両CPUのビルド、アドホック署名、checksum、成果物検査に加え、作業ツリーがcleanであること、指定commitがHEADと一致すること、指定versionと拡張機能のmanifest・package versionが一致することを検証する。いずれかが失敗した場合はタグとReleaseを作成しない。
