---
title: Warp
description: Warp はアウトバウンド透過型 SMTP プロキシです。SMTP レベルのロギング、メッセージのフィルタリング、到達性のシグナルを提供します。
---

**WARP** はアウトバウンド透過型 SMTP プロキシです。

メールサーバーとインターネットのあいだに、どちらにも変更を加えずに入ります。
送信 SMTP 接続はネットワークレベルで Warp にリダイレクトされます。
Warp はセッション全体を記録し、フィルターフックが導入されていればメッセージをそれに渡し、宛先の MX にリレーします。

[![](/images/github-mark.svg) linyows/warp - GitHub](https://github.com/linyows/warp)

## なぜ Warp か

送信メールは、動いているあいだは見えず、動かなくなってから説明が難しくなります。

- 何が誰に送られ、宛先サーバーがどう応答したのかが見えない。
- スロットリングやブロックリストへの登録は静かに起こり、たいていはユーザーからの連絡で気付く。
- 侵害されたアカウントや設定を誤ったアプリケーションが1つあるだけで、IP レンジ全体のレピュテーションが損なわれる。
- 送信メールの監査には、MTA に外部ツールを継ぎ足すことになる。

Warp は、MTA の設定に手を入れずに、送信経路そのものを観測と制御の地点に変えます。

## 機能

### オブザーバビリティ

すべての SMTP コマンドとレスポンスを、双方向かつプロキシの両側で、発生した時点で記録します。
セッションのメタデータ（`MAIL FROM`、`RCPT TO`、HELO ホスト名、宛先がメッセージを受け付けるまでにかかった時間）も抽出し、構造化されたデータとしてフックに渡します。
ログは既定では標準出力に出ます。
プラグインを使えば、ファイル、MySQL、SQLite、Slack に書き出せます。

### フィルタリング

Warp は DATA フェーズでメッセージ全体をバッファし、リレーする前にフィルターフックに渡せます。
次に何が起きるかは、フックが決めます。

| アクション | 動作 |
|---|---|
| **Relay** | メッセージをそのまま宛先サーバーに中継する |
| **Reject** | 送信者に SMTP エラーを返す。メッセージは宛先に到達しない |
| **Add header** | メッセージを変更（たとえば `X-Spam-Score` ヘッダーの追加）してから中継する |

フィルタリングは、有効にすれば使える機能ではなく拡張ポイントです。
組み込みプラグインはいずれも `BeforeRelay` を実装していないため、判定ロジックは自分でプラグインとして書くことになります。
Warp はロードしたプラグインのうち、最初に `FilterHook` を実装しているものを使います。

### 到達性のシグナル

Warp は、宛先への接続から `354` レスポンスまでの時間を記録します。
`354` は、宛先がメッセージ本文を受け付ける準備ができた時点です。
この経過時間はログに出力され、セッション中のすべての SMTP ステータスコードとともに、接続メタデータとしてフックに渡されます。
Warp 自身はその結果を判定しません。
`354` を返すまでの時間が延び続ける宛先はスロットリングをかけていますが、そう結論づけるのはログを読む側です。

### 送信元アドレス

リレーする接続の送信元アドレスは `-outbound-ip` で指定し、プロセス全体に適用されます。
複数のアドレスから送信するには、アドレスごとにインスタンスを起動し、iptables のルールで振り分けます。
Warp がメッセージごとにアドレスを選ぶことはありません。

## 仕組み

Warp は MTA の代わりに話すのではなく、既存の接続の途中に立ちます。
`nat` テーブルの iptables ルールが接続を Warp に送り、Warp は `SO_ORIGINAL_DST` ソケットオプションで、その接続が本来どこへ向かっていたのかをカーネルに尋ねます。
そして自分でその宛先への接続を開き、双方向にトラフィックをコピーしながら、途中で中身を見ます。

`SO_ORIGINAL_DST` は NAT が作成したコネクション追跡エントリを読み取ります。
このエントリは NAT を行ったホストにしか存在しません。
したがって **Warp は iptables ルールを適用したホスト上で動作する必要があります**。
MTA 自体はどのホストにあっても構いません。
Netfilter に依存するため、実質的に Linux 専用でもあります。

![Warp のアーキテクチャ](/images/architecture.png)

### STARTTLS の扱い

Warp は、クライアントに返す EHLO レスポンスから `STARTTLS` を取り除きます。
クライアントは Warp とは平文で話し続けるため、そのコマンドは読める状態に保たれます。
Warp は宛先サーバーとの TLS を自分でネゴシエートします。
メールは暗号化されたままネットワークを出ていき、セッションは観測できる状態のままになります。

## 配置

### MTA と同一のホスト

接続はローカルで生成されるため、ルールは `OUTPUT` チェーンに入れます。

```shell
$ iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination <proxy-ip>:<proxy-port>
```

上の図はこの構成です。MTA と Warp が1つのホストを共有しています。

### 別のホスト

MTA の送信 SMTP トラフィックを Warp ホスト経由にルーティングし（たとえば MTA のネットワークのデフォルトゲートウェイにします）、`PREROUTING` チェーンでリダイレクトします。

```shell
$ iptables -t nat -A PREROUTING -p tcp --dport 25 -j DNAT --to-destination <proxy-ip>:<proxy-port>
```

NAT は Warp ホスト上で発生するため、元の宛先は Warp から参照できます。

## 使い方

Go でコマンドをインストールするか、[Releases](https://github.com/linyows/warp/releases) からビルド済みバイナリをダウンロードします。

```shell
$ go install github.com/linyows/warp/cmd/warp@latest
```

プロキシを起動します。
`-port` を指定しない場合は `127.0.0.1` のエフェメラルポートを使います。最初に動かしてみるには便利ですが、実運用の構成ではありません。

```shell
$ warp -ip 0.0.0.0 -port 10025 -verbose
2023/11/27 08:16:28.368892 warp listens to 0.0.0.0:10025
```

| フラグ | デフォルト | 説明 |
|---|---|---|
| `-ip` | `127.0.0.1` | リッスンアドレス |
| `-port` | エフェメラル | リッスンポート |
| `-outbound-ip` | `0.0.0.0` | リレーする接続に使う送信元アドレス |
| `-plugins` | | カンマ区切りのプラグイン名: `mysql`, `sqlite`, `file`, `slack` |
| `-message-size-limit` | `10240000` | メッセージの最大サイズ（バイト） |
| `-verbose` | `false` | すべての SMTP コマンドとレスポンスをログに出す |
| `-pprof` | | `net/http/pprof` を `host:port` で公開する。ループバックアドレスにバインドする |
| `-version` | `false` | ビルドバージョンを表示する |

## プラグイン

Warp は単体では標準出力に書き出します。
プラグインは、そのログの置き場所と、フィルタリングの判定そのものを与えます。
プラグインは Go のプラグインであり、起動時に `/opt/warp/plugins`、または `PLUGIN_PATH` が指すディレクトリから `.so` ファイルとしてロードされます。
ロードされるのは `-plugins` に挙げた名前のものだけです。

| プラグイン | 説明 | 環境変数 |
|---|---|---|
| **file** | すべての SMTP 通信を JSON としてファイルに追記する | `FILE_PATH` |
| **mysql** | 接続と通信を MySQL に保存する | `DSN` |
| **sqlite** | 接続と通信を SQLite に保存する | `DSN` |
| **slack** | 接続が閉じたときにチャンネルへ通知を送る | `SLACK_TOKEN`, `SLACK_CHANNEL` |

Go のプラグインは cgo を必要とするため、Warp 本体とプラグインの両方を `CGO_ENABLED=1` でビルドします。

```shell
$ make build-withcgo
$ make mysql-plugin file-plugin
$ export DSN="warp:PASSWORD@tcp(localhost:3306)/warp"
$ export FILE_PATH="/tmp/warp.log"
$ warp -ip 0.0.0.0 -port 10025 -plugins mysql,file
2023/11/27 08:12:04 plugin loaded: /opt/warp/plugins/file.so
2023/11/27 08:12:04 plugin loaded: /opt/warp/plugins/mysql.so
2023/11/27 08:12:04.451675 use file hook
2023/11/27 08:12:04.451677 use mysql hook
2023/11/27 08:12:04.451767 warp listens to 0.0.0.0:10025
```

`sqlite` プラグインは `DSN` にファイルパスを受け取ります（たとえば `/var/db/warp.sqlite3`）。

### データベーススキーマ

`mysql` プラグインと `sqlite` プラグインは2つのテーブルを使います。
`connections` はセッションごとに1行を持ち、`id`、`mail_from`、`mail_to`、ミリ秒単位の `elapse`、`occurred_at` を保存します。
`communications` は SMTP のコマンドとレスポンスごとに1行を持ち、`connection_id` でセッションを参照します。
MySQL のスキーマは [misc/setup.sql](https://github.com/linyows/warp/blob/main/misc/setup.sql) にあります。

```shell
$ mysql < misc/setup.sql
```

### プラグインを書く

プラグインは、ロギングのインターフェースを実装した `Hook` 変数をエクスポートします。
フィルタリングを行う場合は `BeforeRelay` も実装します。

```go
type Hook interface {
    Name() string
    AfterInit()
    AfterComm(*AfterCommData) // 各 SMTP コマンドまたはレスポンスの後に呼ばれる
    AfterConn(*AfterConnData) // 接続が閉じたときに呼ばれる
}

type FilterHook interface {
    Hook
    BeforeRelay(*BeforeRelayData) *FilterResult
}
```

Go のプラグインはバージョンに拘束されます。
プラグインは Warp 本体と同じ Go ツールチェーン、同じ依存バージョンでビルドしないと、`plugin.Open` がロードを拒否します。
両者は一緒にビルドしてください。

## ログ

各ログ行は、コネクション ID と、そのデータがセッションのどの区間のものかを示す2文字のマーカーを持ちます。
このマーカーは、Warp がそのまま通過させたトラフィックと、Warp 自身が処理したトラフィックを区別します。
STARTTLS を使うセッションの中身が読めるのは、この区別があるからです。

| マーカー | 意味 |
|---|---|
| `->` | クライアントから宛先へ |
| `<-` | 宛先からクライアントへ |
| `>\|` | クライアントから Warp へ |
| `\|>` | Warp から宛先へ |
| `\|<` | 宛先から Warp へ |
| `<\|` | Warp からクライアントへ |
| `--` | Warp 内部のイベント |

宛先が STARTTLS を提示するセッションは、次のように見えます。

```shell
2023/11/27 08:14:54.942921 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connected from 127.0.0.1:49172
2023/11/27 08:14:54.943046 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connecting to 127.0.0.1:11025
2023/11/27 08:14:54.947454 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 220 example.local ESMTP Server\r\n
2023/11/27 08:14:54.947569 01HG7XGYYZCEB3WSNXPAVG4ZZN -> EHLO localhost\r\n
2023/11/27 08:14:54.947752 01HG7XGYYZCEB3WSNXPAVG4ZZN |< 250-example.local\r\n250-STARTTLS\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951257 01HG7XGYYZCEB3WSNXPAVG4ZZN <| 250-example.local\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951335 01HG7XGYYZCEB3WSNXPAVG4ZZN -- pipe locked for tls connection
2023/11/27 08:14:54.951341 01HG7XGYYZCEB3WSNXPAVG4ZZN |> STARTTLS\r\n
2023/11/27 08:14:54.957949 01HG7XGYYZCEB3WSNXPAVG4ZZN -- tls connected, to pipe unlocked
2023/11/27 08:14:54.958262 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 250 2.1.0 Ok\r\n
2023/11/27 08:14:54.958316 01HG7XGYYZCEB3WSNXPAVG4ZZN -> RCPT TO:<bob@example.local>\r\n
2023/11/27 08:14:54.958900 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 354 End data with <CR><LF>.<CR><LF>\r\n
2023/11/27 08:14:54.959709 01HG7XGYYZCEB3WSNXPAVG4ZZN -- from:alice@example.test to:bob@example.local elapse:14 msec
2023/11/27 08:14:54.959710 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connections closed
```

Warp には、送信クライアントと受信サーバーを起動する統合テストが付属しています。
`make integration` を実行すれば、経路全体を手元で通せます。
