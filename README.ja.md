<p align="right"><a href="https://github.com/linyows/warp/blob/main/README.md">English</a> | 日本語</p>

<br><br><br><br><br><br><br><br><p align="center">
  <a href="https://warp.linyo.ws">
    <img alt="WARP" src="https://github.com/linyows/warp/blob/main/misc/warp.svg" width="200">
  </a>
</p><br><br><br><br><br><br><br><br>

<strong>Warp</strong> はアウトバウンド透過型SMTPプロキシです。
送信セッションのすべての SMTP コマンドとレスポンスを記録し、宛先がメッセージを受け付けるまでの時間を計測します。
また、リレー前にメッセージをフィルターフックに渡すことができます。
https://warp.linyo.ws/ja

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

Warp は Linux の `SO_ORIGINAL_DST` を利用した透過型プロキシとしてデプロイされます。ネットワークレベルで SMTP 接続をインターセプトするため、**既存のメールクライアントやアプリケーションへの変更は一切不要**です。このソケットオプションは NAT が作成したコネクション追跡エントリを読み取るため、Warp は iptables ルールを適用したホスト上で動作する必要があります。MTA 自体はどのホストにあっても構いません。導入後、Warp は以下の機能を提供します:

### オブザーバビリティ

すべての SMTP コマンドとレスポンスをリアルタイムにキャプチャします。ファイル、MySQL、SQLite へのログ出力や、Slack への通知が可能です。すべてシンプルなプラグインシステムを通じて実現されます。接続メタデータ（送信者、受信者、HELO ホスト名、経過時間）は自動的に抽出・構造化されます。

### フィルタリング

Warp は DATA フェーズでメッセージ全体をバッファリングし、リレー前に **フィルターフック** に渡すことができます。フィルターロジックが次のアクションを決定します:

| アクション | 動作 |
|---|---|
| **Relay** | メッセージをそのまま宛先サーバーに中継する |
| **Reject** | 送信者に SMTP エラーを返す — メッセージは宛先に到達しない |
| **Add Header** | メッセージを変更（例: `X-Spam-Score` ヘッダーの追加）してから中継する |

フィルタリングは、有効にすれば使える機能ではなく拡張ポイントです。組み込みプラグインはいずれも `BeforeRelay` を実装していないため、判定ロジックは自分でプラグインとして書くことになります。Warp はロードしたプラグインのうち、最初に `FilterHook` を実装しているものを使います。

### 到達性のシグナル

Warp は、宛先への接続から `354` レスポンス — 宛先がメッセージ本文を受け付ける準備ができた時点 — までの時間を記録します。この経過時間はログに出力され、セッション中のすべての SMTP ステータスコードとともに、接続メタデータとしてフックに渡されます。Warp 自身はその結果を判定しません。宛先がスロットリングしているかどうかの判断は、ログを受け取る側に委ねられています。

### 送信元アドレス

リレーする接続の送信元アドレスは `-outbound-ip` で指定し、プロセス全体に適用されます。複数のアドレスから送信するには、アドレスごとにインスタンスを起動し、iptables のルールで振り分けます。Warp がメッセージごとにアドレスを選ぶことはありません。

## アーキテクチャ

<p align="center">
  <img src="https://github.com/linyows/warp/blob/main/misc/architecture.png" alt="Warp Architecture">
</p>

```
                    透過型プロキシ (iptables nat)
                    ┌─────────────────────────────┐
  SMTP Client ────▶ │           Warp              │ ────▶ 宛先 MX
                    │                             │
                    │  ┌─────────┐  ┌──────────┐  │
                    │  │Upstream │  │Downstream│  │
                    │  │Mediator │  │Mediator  │  │
                    │  └────┬────┘  └────┬─────┘  │
                    │       │            │        │
                    │  ┌────▼────────────▼────┐   │
                    │  │     Hook System      │   │
                    │  │  ┌─────┐  ┌────────┐ │   │
                    │  │  │ Log │  │ Filter │ │   │
                    │  │  └─────┘  └────────┘ │   │
                    │  └──────────────────────┘   │
                    └─────────────────────────────┘
```

**主要コンポーネント:**

- **Server** — TCP 接続を受け付け、`SO_ORIGINAL_DST` で元の宛先アドレスを取得する
- **Pipe** — Upstream/Downstream の Mediator を使って、クライアントとサーバー間の双方向データフローを管理する
- **Mediators** — SMTP トラフィックを双方向で検査・変換し、メタデータ（MAIL FROM, RCPT TO, HELO）を抽出。STARTTLS ネゴシエーションも処理する
- **Hook System** — ロギング（`AfterComm`, `AfterConn`）とフィルタリング（`BeforeRelay`）のための拡張可能なプラグインアーキテクチャ

### STARTTLS の処理

Warp は TLS ネゴシエーションを透過的に処理します。サーバーの EHLO レスポンスから `STARTTLS` を除去してクライアントに返し、Warp 自身が宛先サーバーとの TLS 接続を確立します。これにより、暗号化された配信を維持しつつ、SMTP トラフィックの検査が可能になります。

### ログの形式

各ログ行には、コネクション ID と、そのデータがセッションのどの区間のものかを示す 2 文字のマーカーが含まれます。Warp がそのまま通過させたトラフィックと、Warp 自身が処理したトラフィックを区別できるため、STARTTLS を使うセッションの中身も読み取れます:

| マーカー | 意味 |
|---|---|
| `->` | クライアント → 宛先 |
| `<-` | 宛先 → クライアント |
| `>\|` | クライアント → Warp |
| `\|>` | Warp → 宛先 |
| `\|<` | 宛先 → Warp |
| `<\|` | Warp → クライアント |
| `--` | Warp 内部のイベント |

```
2023/11/27 08:14:54.947569 01HG7XGYYZCEB3WSNXPAVG4ZZN -> EHLO localhost\r\n
2023/11/27 08:14:54.947752 01HG7XGYYZCEB3WSNXPAVG4ZZN |< 250-example.local\r\n250-STARTTLS\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951257 01HG7XGYYZCEB3WSNXPAVG4ZZN <| 250-example.local\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951335 01HG7XGYYZCEB3WSNXPAVG4ZZN -- pipe locked for tls connection
2023/11/27 08:14:54.951341 01HG7XGYYZCEB3WSNXPAVG4ZZN |> STARTTLS\r\n
2023/11/27 08:14:54.957949 01HG7XGYYZCEB3WSNXPAVG4ZZN -- tls connected, to pipe unlocked
2023/11/27 08:14:54.959709 01HG7XGYYZCEB3WSNXPAVG4ZZN -- from:alice@example.test to:bob@example.local elapse:14 msec
```

## はじめに

### 動作要件

- **Netfilter が使える Linux** — 透過型プロキシは `SO_ORIGINAL_DST` に依存しており、Warp は NAT を行うホスト上で動作する必要があります
- **Go 1.23 以降** — ソースからビルドする場合
- **cgo**（`CGO_ENABLED=1`）— プラグインを使う場合。Go のプラグインシステムは純 Go ビルドでは動作しません

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

送信 SMTP トラフィックを Warp にリダイレクトします。MTA が Warp と同一ホストにある場合、接続はローカルで生成されるためルールは `OUTPUT` チェーンに入れます:

```bash
iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination 127.0.0.1:10025
```

MTA が別のホストにある場合は、MTA の送信 SMTP トラフィックを Warp ホスト経由にルーティングし（例えば MTA のネットワークのデフォルトゲートウェイにする）、`PREROUTING` チェーンでリダイレクトします:

```bash
iptables -t nat -A PREROUTING -p tcp --dport 25 -j DNAT --to-destination 203.0.113.10:10025
```

いずれの場合も、NAT — つまり Warp が読み取るコネクション追跡エントリ — は Warp が動作するホスト上で発生する必要があります。

### コマンドラインオプション

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-ip` | `127.0.0.1` | リッスン IP アドレス |
| `-port` | *(エフェメラル)* | リッスンポート |
| `-outbound-ip` | `0.0.0.0` | 送信接続に使用するソース IP |
| `-plugins` | | カンマ区切りのプラグイン名: `mysql`, `sqlite`, `file`, `slack` |
| `-message-size-limit` | `10240000` | メッセージの最大サイズ（バイト、約10MB） |
| `-verbose` | `false` | 詳細ログを有効にする |
| `-pprof` | | `net/http/pprof` を `host:port` で公開する。ループバックアドレスにバインドすること |
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

Warp は Go のプラグインシステムを使って `/opt/warp/plugins/`（または `PLUGIN_PATH` 環境変数で指定したパス）から `.so` ファイルをロードします。ロードされるのは `-plugins` に指定した名前のものだけで、ディレクトリ内のその他のファイルは無視されます。各プラグインは `Hook` インターフェースを実装します:

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

Go のプラグインは cgo を必要とするため、Warp 本体とプラグインの両方を `CGO_ENABLED=1` でビルドします。`build-withcgo` ターゲットがそれを行います:

```bash
make build-withcgo
make file-plugin mysql-plugin      # plugins/*.so にビルドされる
PLUGIN_PATH=./plugins ./warp -ip 0.0.0.0 -port 10025 -plugins file,mysql
```

`.so` ファイルを `/opt/warp/plugins/` に配置すれば、`PLUGIN_PATH` なしで起動できます。

### MySQL セットアップ

付属のスキーマを使ってデータベースをセットアップします:

```bash
mysql < misc/setup.sql
```

Go のプラグインはバージョンに拘束されます。プラグインは Warp 本体と同じ Go ツールチェーン、同じ依存バージョンでビルドしないと、`plugin.Open` がロードを拒否します。両者は一緒にビルドしてください。

## ユースケース

- **メールゲートウェイ** — 組織全体の SMTP 検査ポイントを一元化
- **スパム・フィッシング防止** — 自作のフィルタープラグインで、レピュテーションを損なう前に送信メッセージをフィルタリング
- **コンプライアンス・監査** — すべての送信メールを SMTP レベルの詳細と共にログ記録
- **データ漏洩防止 (DLP)** — 配信前にメッセージ内容を検査し機密データの流出を防止
- **到達性モニタリング** — MX レスポンスタイムと SMTP ステータスコードを継続的に記録
- **IP レピュテーション管理** — アドレスごとにインスタンスを起動し、送信トラフィックを送信元アドレスで分離
- **開発・デバッグ** — 開発中の SMTP トラフィックをキャプチャして検査

## 開発

```bash
make build          # warp バイナリをビルドする
make test           # ユニットテスト
make integration    # 送信クライアントと受信サーバーを起動して Warp 経由で通す
make test-all       # レースディテクタとカバレッジ付きのフルスイート
```

`make integration` は送信側のメールクライアントと受信側の SMTP サーバーを Go で起動し、Warp を経由してメッセージを送ります。iptables ルールなしで、経路全体を手元で確認できます。

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
