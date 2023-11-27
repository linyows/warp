<br><br><br><br><br><br><p align="center">
  <img alt="WARP" src="https://github.com/linyows/warp/blob/main/misc/warp.svg" width="200">
</p>
<p align="center">
  <strong>WARP</strong> is an outbound <b>transparent</b> SMTP proxy.
</p><br><br><br><br><br><br><br><br>
<p align="center">
  <a href="https://github.com/linyows/warp/actions" title="actions"><img src="https://img.shields.io/github/actions/workflow/status/linyows/warp/build.yml?branch=main&style=for-the-badge"></a>
  <a href="https://github.com/linyows/warp/releases"><img src="http://img.shields.io/github/release/linyows/warp.svg?style=for-the-badge" alt="GitHub Release"></a>
  <br />
  <a href="https://goreportcard.com/report/github.com/linyows/warp"> <img src="https://goreportcard.com/badge/github.com/linyows/warp" alt="Go Report Card"></a>
  <a href="https://pkg.go.dev/github.com/linyows/warp"><img src="https://pkg.go.dev/badge/github.com/linyows/warp.svg" alt="Go Reference"></a>
  <a href="https://github.com/linyows/warp/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="MIT License" /></a>
</p><br><br>

For redirect the port need by iptables rule:

```
iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination <proxy-ip>:<proxy-port>
```

Also, the MTA and Proxy must be on the same host to know the DST Address before NAT.

![Architecture](https://github.com/linyows/warp/blob/main/misc/architecture.png)

Usage
--

Start the proxy from the warp command; by default, it uses the ephemeral port 127.0.0.1.

```sh
# ./warp -h
Usage of ./warp:
  -ip string
        listen ip (default "127.0.0.1")
  -message-size-limit int
        The maximal size in bytes of a message (default 10240000)
  -outbound-ip string
        outbound ip (default "0.0.0.0")
  -plugins string
        use plugin names: mysql, sqlite, file, slack
  -port int
        listen port
  -verbose
        verbose logging
  -version
        show build version
# ./warp
2023/11/27 08:16:28.368892 warp listens to 127.0.0.1:0
```

Integration Test
--

You can start up an outgoing mail client and an incoming server with Go to verify Warp's operation. In this way:

```sh
# make integration
go test -v -run TestIntegration
-----
req warning: No value provided for subject name attribute "C", skipped
req warning: No value provided for subject name attribute "ST", skipped
req warning: No value provided for subject name attribute "L", skipped
req warning: No value provided for subject name attribute "O", skipped
req warning: No value provided for subject name attribute "OU", skipped
=== RUN   TestIntegration
Wait for port 10025 listen............
Wait for port 11025 listen...

Warp Server:
2023/11/27 08:14:54.942554 warp listens to 127.0.0.1:10025
2023/11/27 08:14:54.942921 01HG7XGYYYKJDB18Y043M8X9BN -- connected from 127.0.0.1:49172
2023/11/27 08:14:54.943046 01HG7XGYYYKJDB18Y043M8X9BN -- connecting to 127.0.0.1:11025
2023/11/27 08:14:54.943792 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connected from 127.0.0.1:49176
2023/11/27 08:14:54.943847 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connecting to 127.0.0.1:11025
2023/11/27 08:14:54.943851 01HG7XGYYYKJDB18Y043M8X9BN -- connected to 127.0.0.1:11025
2023/11/27 08:14:54.944391 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connected to 127.0.0.1:11025
2023/11/27 08:14:54.944762 01HG7XGYYYKJDB18Y043M8X9BN -- connections closed
2023/11/27 08:14:54.944766 01HG7XGYYYKJDB18Y043M8X9BN -- from:unknown to:unknown elapse:-2 msec
2023/11/27 08:14:54.947454 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 220 example.local ESMTP Server\r\n
2023/11/27 08:14:54.947569 01HG7XGYYZCEB3WSNXPAVG4ZZN -> EHLO localhost\r\n
2023/11/27 08:14:54.947752 01HG7XGYYZCEB3WSNXPAVG4ZZN |< 250-example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-STARTTLS\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951257 01HG7XGYYZCEB3WSNXPAVG4ZZN <| 250-example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951335 01HG7XGYYZCEB3WSNXPAVG4ZZN -- pipe locked for tls connection
2023/11/27 08:14:54.951341 01HG7XGYYZCEB3WSNXPAVG4ZZN |> STARTTLS\r\n
2023/11/27 08:14:54.951344 01HG7XGYYZCEB3WSNXPAVG4ZZN >| MAIL FROM:<alice@example.test> BODY=8BITMIME\r\n
2023/11/27 08:14:54.951451 01HG7XGYYZCEB3WSNXPAVG4ZZN |< 220 2.0.0 Ready to start TLS\r\n
2023/11/27 08:14:54.951591 01HG7XGYYZCEB3WSNXPAVG4ZZN |> EHLO localhost\r\n
2023/11/27 08:14:54.957870 01HG7XGYYZCEB3WSNXPAVG4ZZN |< 250-example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-STARTTLS\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.957949 01HG7XGYYZCEB3WSNXPAVG4ZZN -- tls connected, to pipe unlocked
2023/11/27 08:14:54.957952 01HG7XGYYZCEB3WSNXPAVG4ZZN |> MAIL FROM:<alice@example.test> BODY=8BITMIME\r\n
2023/11/27 08:14:54.958262 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 250 2.1.0 Ok\r\n
2023/11/27 08:14:54.958316 01HG7XGYYZCEB3WSNXPAVG4ZZN -> RCPT TO:<bob@example.local>\r\n
2023/11/27 08:14:54.958573 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 250 2.1.5 Ok\r\n
2023/11/27 08:14:54.958632 01HG7XGYYZCEB3WSNXPAVG4ZZN -> DATA\r\n
2023/11/27 08:14:54.958900 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 354 End data with <CR><LF>.<CR><LF>\r\n
2023/11/27 08:14:54.958949 01HG7XGYYZCEB3WSNXPAVG4ZZN -> To: bob@example.local\r\nFrom: alice@example.test\r\nSubject: Test
2023/11/27 08:14:54.959268 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 250 2.0.0 Ok: queued\r\n
2023/11/27 08:14:54.959709 01HG7XGYYZCEB3WSNXPAVG4ZZN -- from:alice@example.test to:bob@example.local elapse:14 msec
2023/11/27 08:14:54.959708 01HG7XGYYZCEB3WSNXPAVG4ZZN <- 221 2.0.0 Bye\r\n
2023/11/27 08:14:54.959710 01HG7XGYYZCEB3WSNXPAVG4ZZN -- connections closed

SMTP Server:
2023/11/27 08:14:54.941549 SMTP server is listening on :11025
2023/11/27 08:14:54.943854 01HG7XGYYZJW0TFYMEV3XNZ9ST -> 220 example.local ESMTP Server
2023/11/27 08:14:54.943860 01HG7XGYYZQ9AV9DKFRGS7HDCN -> 220 example.local ESMTP Server
2023/11/27 08:14:54.943909 01HG7XGYYZQ9AV9DKFRGS7HDCN -- conn ReadString error: &net.OpError{Op:"read", Net:"tcp", Source:(*net.TCPAddr)(0x4000118f00), Addr:(*net.TCPAddr)(0x4000118f30), Err:(*os.SyscallError)(0x400013e460)}
2023/11/27 08:14:54.944811 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 220 example.local ESMTP Server
2023/11/27 08:14:54.947574 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- EHLO localhost
2023/11/27 08:14:54.947577 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 250-example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-STARTTLS\r\n250 8BITMIME
2023/11/27 08:14:54.951349 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- STARTTLS
2023/11/27 08:14:54.951351 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 220 2.0.0 Ready to start TLS
2023/11/27 08:14:54.957825 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- EHLO localhost
2023/11/27 08:14:54.957828 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 250-example.local\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-STARTTLS\r\n250 8BITMIME
2023/11/27 08:14:54.957957 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- MAIL FROM:<alice@example.test> BODY=8BITMIME
2023/11/27 08:14:54.957989 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 250 2.1.0 Ok
2023/11/27 08:14:54.958330 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- RCPT TO:<bob@example.local>
2023/11/27 08:14:54.958334 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 250 2.1.5 Ok
2023/11/27 08:14:54.958651 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- DATA
2023/11/27 08:14:54.958654 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 354 End data with <CR><LF>.<CR><LF>
2023/11/27 08:14:54.959054 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- To: bob@example.local
2023/11/27 08:14:54.959056 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- From: alice@example.test
2023/11/27 08:14:54.959057 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- Subject: Test
2023/11/27 08:14:54.959058 01HG7XGYZ0QRFSY87EGCQ5KJRQ <-
2023/11/27 08:14:54.959058 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- TestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTestTest
2023/11/27 08:14:54.959065 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- .
2023/11/27 08:14:54.959066 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 250 2.0.0 Ok: queued
2023/11/27 08:14:54.959475 01HG7XGYZ0QRFSY87EGCQ5KJRQ <- QUIT
2023/11/27 08:14:54.959477 01HG7XGYZ0QRFSY87EGCQ5KJRQ -> 221 2.0.0 Bye

--- PASS: TestIntegration (1.02s)
PASS
ok      github.com/linyows/warp 1.040s
```

Plugins
--

Warp outputs logs as stdout, but plugins can save logs to a database or a specified file.

Native Plugins:

- MySQL
    ```sh
    export DSN="warp:PASSWORD@tcp(localhost:3306)/warp"
    ```
- SQLite
    ```sh
    export DSN="/var/db/warp.sqlite3"
    ```
- File
    ```sh
    export FILE_PATH="/tmp/warp.log"
    ```
- Slack

```sh
# make build-withcgo
env CGO_ENABLED=1 go build -o warp ./cmd/warp/main.go
# make mysql-plugin
go build -buildmode=plugin -o plugins/mysql.so plugins/mysql/main.go
# make file-plugin
go build -buildmode=plugin -o plugins/file.so plugins/file/main.go
# ./warp -plugins mysql,file
2023/11/27 08:12:04 plugin loaded: /go/src/app/plugins/file.so
2023/11/27 08:12:04 plugin loaded: /go/src/app/plugins/mysql.so
2023/11/27 08:12:04.451675 use file hook
2023/11/27 08:12:04.451677 use mysql hook
2023/11/27 08:12:04.451767 warp listens to 127.0.0.1:0
```

RDB schema:

```sql
# sudo mysql -uroot -D warp
mysql> select * from connections;
+----------------------------+-------------+---------------+----------------------------+
| id                         | mail_from   | mail_to       | occurred_at                |
+----------------------------+-------------+---------------+----------------------------+
| 01FR74VW574PVQ5WGYE5RQATTG | root@sender | root@receiver | 2021-12-31 02:24:56.009302 |
| 01FR755XZKA594WA8SACQB4HC3 | root@sender | root@receiver | 2021-12-31 02:30:25.557302 |
+----------------------------+-------------+---------------+----------------------------+
2 rows in set (0.00 sec)

mysql> select communications.occurred_at, direction as d, substring(data, 1, 40) as data from communications, connections where connections.id = communications.connection_id and connections.id = "01FR755XZKA594WA8SACQB4HC3" order by communications.occurred_at;
+----------------------------+----+------------------------------------------+
| occurred_at                | d  | data                                     |
+----------------------------+----+------------------------------------------+
| 2021-12-31 02:30:25.523678 | -- | connected to 192.168.30.50:25            |
| 2021-12-31 02:30:25.534128 | <- | 220 receiver ESMTP Postfix (Ubuntu)\r\n  |
| 2021-12-31 02:30:25.534692 | -> | EHLO sender\r\n                          |
| 2021-12-31 02:30:25.535251 | <- | 250-receiver\r\n250-PIPELINING\r\n250-SI |
| 2021-12-31 02:30:25.535399 | |< | 250-receiver\r\n250-PIPELINING\r\n250-SI |
| 2021-12-31 02:30:25.538790 | -- | pipe locked for tls connection           |
| 2021-12-31 02:30:25.538791 | |> | STARTTLS\r\n                             |
| 2021-12-31 02:30:25.538820 | >| | MAIL FROM:<root@sender> SIZE=327\r\nRCPT |
| 2021-12-31 02:30:25.539568 | |< | 220 2.0.0 Ready to start TLS\r\n         |
| 2021-12-31 02:30:25.539701 | |> | EHLO sender\r\n                          |
| 2021-12-31 02:30:25.547124 | |< | 250-receiver\r\n250-PIPELINING\r\n250-SI |
| 2021-12-31 02:30:25.547459 | -- | tls connected, to pipe unlocked          |
| 2021-12-31 02:30:25.547811 | -> | MAIL FROM:<root@sender> SIZE=327\r\nRCPT |
| 2021-12-31 02:30:25.554912 | <- | 250 2.1.0 Ok\r\n250 2.1.5 Ok\r\n354 End  |
| 2021-12-31 02:30:25.555126 | -> | Received: from sender (localhost [127.0. |
| 2021-12-31 02:30:25.556812 | <- | 250 2.0.0 Ok: queued as 1EA19412C8\r\n22 |
| 2021-12-31 02:30:25.559877 | -- | connections closed                       |
+----------------------------+----+------------------------------------------+
17 rows in set (0.00 sec)
```

Author
--

[linyows](https://github.com/linyows)
