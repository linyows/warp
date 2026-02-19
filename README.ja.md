<p align="right"><a href="https://github.com/linyows/warp/blob/main/README.md">English</a> | 日本語</p>

<br><br><br><br><br><br><br><br><p align="center">
  <a href="https://warp.linyo.ws">
    <img alt="WARP" src="https://github.com/linyows/warp/blob/main/misc/warp.svg" width="200">
  </a>
</p><br><br><br><br><br><br><br><br>

<strong>Warp</strong> はアウトバウンド透過型SMTPプロキシです。
SMTPレベルのロギングが可能で、MXのレスポンスタイムやステータスコードからスロットリングを検知します。
また、内部レピュテーションに基づいて送信IPアドレスを使い分けることができます。
https://warp.linyo.ws

<br>
<p align="center">
  <a href="https://github.com/linyows/warp/actions" title="actions"><img src="https://img.shields.io/github/actions/workflow/status/linyows/warp/build.yml?branch=main&style=for-the-badge"></a>
  <a href="https://github.com/linyows/warp/releases"><img src="http://img.shields.io/github/release/linyows/warp.svg?style=for-the-badge" alt="GitHub Release"></a>
</p>

## 課題

メールインフラを大規模に運用するには、多くの課題が伴います:

- **可視性がない** — 何が、誰に、どのように送信され、リモートサーバーがどう応答しているか見えない
- **到達性の問題** — スロットリングやブロックリスト登録はサイレントに発生し、ユーザーからの苦情で初めて気づく
- **フィルタリング層がない** — 侵害されたアカウントや設定ミスのアプリケーションが、誰にも気づかれずにスパムやフィッシングメールを送信してしまう
- **IPレピュテーション管理** — たった1つの不正な送信者が、IP範囲全体のレピュテーションを損なう可能性がある
- **コンプライアンスの空白** — 送信メールの監査には、きれいに統合できない外部ツールの導入が必要になる

## Warp が解決すること

Warp は Linux の `SO_ORIGINAL_DST` を利用した透過型プロキシとしてデプロイされます。ネットワークレベルで SMTP 接続をインターセプトするため、**既存のメールクライアントやアプリケーションへの変更は一切不要**です。導入後、Warp は以下の機能を提供します:

### オブザーバビリティ

すべての SMTP コマンドとレスポンスをリアルタイムにキャプチャします。ファイル、MySQL、SQLite へのログ出力や、Slack への通知が可能です。すべてシンプルなプラグインシステムを通じて実現されます。接続メタデータ（送信者、受信者、HELO ホスト名、経過時間）は自動的に抽出・構造化されます。

### フィルタリング

Warp は DATA フェーズでメッセージ全体をバッファリングし、リレー前に **フィルターフック** に渡すことができます。フィルターロジックが次のアクションを決定します:

| アクション | 動作 |
|---|---|
| **Relay** | メッセージをそのまま宛先サーバーに中継する |
| **Reject** | 送信者に SMTP エラーを返す — メッセージは宛先に到達しない |
| **Add Header** | メッセージを変更（例: `X-Spam-Score` ヘッダーの追加）してから中継する |

### 到達性インテリジェンス

接続からサーバーの `354` レスポンスまでの時間を計測することで、Warp は **MX スロットリングパターン** を検知します。SMTP レスポンスコードと組み合わせることで、リモートサーバーがあなたのメールをどのように扱っているかを包括的に把握できます。

### IP ルーティング

Warp は **送信元 IP アドレス** を設定可能で、送信者レピュテーション、受信者ドメイン、その他のカスタムロジックに基づいて、異なる IP 経由でメールをルーティングできます。

## アーキテクチャ

<p align="center">
  <img src="https://github.com/linyows/warp/blob/main/misc/architecture.png" alt="Warp Architecture">
</p>

```
                    透過型プロキシ (iptables REDIRECT)
                    ┌─────────────────────────────┐
  SMTP Client ────▶ │           Warp              │ ────▶ 宛先 MX
                    │                             │
                    │  ┌─────────┐  ┌──────────┐  │
                    │  │Upstream │  │Downstream│  │
                    │  │Mediator │  │Mediator  │  │
                    │  └────┬────┘  └────┬─────┘  │
                    │       │            │        │
                    │  ┌────▼────────────▼─────┐  │
                    │  │    Hook System         │  │
                    │  │  ┌──────┐ ┌────────┐  │  │
                    │  │  │ Log  │ │ Filter │  │  │
                    │  │  └──────┘ └────────┘  │  │
                    │  └───────────────────────┘  │
                    └─────────────────────────────┘
```

**主要コンポーネント:**

- **Server** — TCP 接続を受け付け、`SO_ORIGINAL_DST` で元の宛先アドレスを取得する
- **Pipe** — Upstream/Downstream の Mediator を使って、クライアントとサーバー間の双方向データフローを管理する
- **Mediators** — SMTP トラフィックを双方向で検査・変換し、メタデータ（MAIL FROM, RCPT TO, HELO）を抽出。STARTTLS ネゴシエーションも処理する
- **Hook System** — ロギング（`AfterComm`, `AfterConn`）とフィルタリング（`BeforeRelay`）のための拡張可能なプラグインアーキテクチャ

### STARTTLS の処理

Warp は TLS ネゴシエーションを透過的に処理します。サーバーの EHLO レスポンスから `STARTTLS` を除去してクライアントに返し、Warp 自身が宛先サーバーとの TLS 接続を確立します。これにより、暗号化された配信を維持しつつ、SMTP トラフィックの検査が可能になります。

## はじめに

### インストール

```bash
go install github.com/linyows/warp/cmd/warp@latest
```

または [Releases](https://github.com/linyows/warp/releases) からビルド済みバイナリをダウンロードできます。

### 基本的な使い方

```bash
warp -ip 0.0.0.0 -port 10025 -verbose
```

### iptables の設定（透過型プロキシ）

送信 SMTP トラフィックを Warp にリダイレクトします:

```bash
iptables -t nat -A OUTPUT -p tcp --dport 25 -j REDIRECT --to-port 10025
```

### コマンドラインオプション

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-ip` | `127.0.0.1` | リッスン IP アドレス |
| `-port` | *(必須)* | リッスンポート |
| `-outbound-ip` | `0.0.0.0` | 送信接続に使用するソース IP |
| `-plugins` | | カンマ区切りのプラグイン名: `mysql`, `sqlite`, `file`, `slack` |
| `-message-size-limit` | `10240000` | メッセージの最大サイズ（バイト、約10MB） |
| `-verbose` | `false` | 詳細ログを有効にする |
| `-version` | `false` | バージョン情報を表示する |

### 構成例: フルセットアップ

```bash
# MySQL ログと Slack 通知を有効にして Warp を起動
warp -ip 0.0.0.0 -port 10025 \
     -outbound-ip 203.0.113.10 \
     -plugins mysql,slack \
     -message-size-limit 20480000 \
     -verbose
```

## プラグインシステム

Warp は Go のプラグインシステムを使って `/opt/warp/plugins/`（または `PLUGIN_PATH` 環境変数で指定したパス）から `.so` ファイルをロードします。各プラグインは `Hook` インターフェースを実装します:

```go
type Hook interface {
    Name() string
    AfterInit()
    AfterComm(*AfterCommData)  // 各 SMTP コマンド/レスポンス後に呼ばれる
    AfterConn(*AfterConnData)  // 接続クローズ時に呼ばれる
}
```

メッセージフィルタリングには `FilterHook` インターフェースを実装します:

```go
type FilterHook interface {
    Hook
    BeforeRelay(*BeforeRelayData) *FilterResult
}
```

### 組み込みプラグイン

| プラグイン | 説明 | 環境変数 |
|---|---|---|
| **file** | すべての SMTP 通信を JSON ファイルにログ出力 | `FILE_PATH` |
| **mysql** | 通信と接続を MySQL に保存 | `DSN` |
| **sqlite** | 通信と接続を SQLite に保存 | `DSN` |
| **slack** | 接続通知を Slack チャンネルに送信 | `SLACK_TOKEN`, `SLACK_CHANNEL` |

### プラグインのビルド

```bash
cd plugins/file
go build -buildmode=plugin -o /opt/warp/plugins/file.so
```

### MySQL セットアップ

付属のスキーマを使ってデータベースをセットアップします:

```bash
mysql < misc/setup.sql
```

## ユースケース

- **メールゲートウェイ** — 組織全体の SMTP 検査ポイントを一元化
- **スパム・フィッシング防止** — レピュテーションを損なう前に送信メッセージをフィルタリング
- **コンプライアンス・監査** — すべての送信メールを SMTP レベルの詳細と共にログ記録
- **データ漏洩防止 (DLP)** — 配信前にメッセージ内容を検査し機密データの流出を防止
- **到達性モニタリング** — MX レスポンスタイムを追跡し、スロットリングをリアルタイムに検知
- **IP レピュテーション管理** — 送信者の行動に基づいて異なる IP 経由でメールをルーティング
- **開発・デバッグ** — 開発中の SMTP トラフィックをキャプチャして検査

## コントリビューション

1. フォークする
2. フィーチャーブランチを作成する (`git checkout -b my-new-feature`)
3. 変更をコミットする (`git commit -am 'Add some feature'`)
4. ブランチにプッシュする (`git push origin my-new-feature`)
5. プルリクエストを作成する

## 作者

[linyows](https://github.com/linyows)

## ライセンス

MIT
