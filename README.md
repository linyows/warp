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

To check the operation, use the sandbox environment with the Vagrantfile in the repository.

```sh
warp main 🏄 make
env GOOS=linux GOARCH=amd64 go build -o warp ./cmd/warp/main.go
warp main 🏄 vagrant up
...
warp main 🏄 vagrant status
Current machine states:

sender                    running (virtualbox)
receiver                  running (virtualbox)
```

Start proxy on sender:

```sh
warp main 🏄 vagrant ssh sender
vagrant@sender:~$ /vagrant/warp -ip 192.168.30.30 -port 10025
2021/02/06 14:50:44 warp listens to 192.168.30.30:10025
```

Send mail on sender:

```sh
warp main 🏄 vagrant ssh sender
vagrant@sender:~$ smtp-source -m 1 -s 1 -l 10 -S 'Hi, Receiver from Sender' -f root@sender -t root@receiver localhost:25
```

Output by proxy on sender:

```sh
2021/02/06 14:50:48 connected from 192.168.30.40:57493
2021/02/06 14:50:48 connected to 192.168.30.50:25
2021/02/06 14:50:48 <- 220 receiver ESMTP Postfix (Ubuntu)\r\n
2021/02/06 14:50:48 -> EHLO sender\r\n
2021/02/06 14:50:48 |< 250-receiver\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-VRFY\r\n250-ETRN\r\n250-STARTTLS\r\n250-ENHANCEDSTATUSCODES\r\n250-8BITMIME\r\n250-DSN\r\n250-SMTPUTF8\r\n250 CHUNKING\r\n
2021/02/06 14:50:48 <- 250-receiver\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-VRFY\r\n250-ETRN\r\n250-ENHANCEDSTATUSCODES\r\n250-8BITMIME\r\n250-DSN\r\n250-SMTPUTF8\r\n250 CHUNKING\r\n
2021/02/06 14:50:48 |> STARTTLS\r\n
2021/02/06 14:50:48 >| MAIL FROM:<root@sender> SIZE=327\r\nRCPT TO:<root@receiver> ORCPT=rfc822;root@receiver\r\nDATA\r\n
2021/02/06 14:50:48 |< 220 2.0.0 Ready to start TLS\r\n
2021/02/06 14:50:48 |> EHLO sender\r\n
2021/02/06 14:50:48 pipe locked for tls connection
2021/02/06 14:50:48 |< 250-receiver\r\n250-PIPELINING\r\n250-SIZE 10240000\r\n250-VRFY\r\n250-ETRN\r\n250-ENHANCEDSTATUSCODES\r\n250-8BITMIME\r\n250-DSN\r\n250-SMTPUTF8\r\n250 CHUNKING\r\n
2021/02/06 14:50:48 tls connected, to pipe unlocked
2021/02/06 14:50:48 -> MAIL FROM:<root@sender> SIZE=327\r\nRCPT TO:<root@receiver> ORCPT=rfc822;root@receiver\r\nDATA\r\n
2021/02/06 14:50:48 <- 250 2.1.0 Ok\r\n250 2.1.5 Ok\r\n354 End data with <CR><LF>.<CR><LF>\r\n
2021/02/06 14:50:48 -> Received: from sender (localhost [127.0.0.1])\r\n        by sender (Postfix) with SMTP id 45B113EA9B\r\n for <root@receiver>; Sat,  6 Feb 2021 14:50:48 +0000 (UTC)\r\nFrom: <root@sender>\r\nTo: <root@receiver>\r\nDate: Sat,  6 Feb 2021 14:50:48 +0000 (UTC)\r\nMessage-Id: <a77e.0003.0000@sender>\r\nSubject: Hi, Receiver from Sender\r\n\r\nXXXXXXXXXX\r\n.\r\nQUIT\r\n
2021/02/06 14:50:48 <- 250 2.0.0 Ok: queued as 76DAD4113D\r\n221 2.0.0 Bye\r\n
2021/02/06 14:50:48 connections closed
```

Received mail on receiver:

```sh
warp main 🏄 vagrant ssh receiver
vagrant@receiver:~$ sudo cat /var/spool/mail/root
From root@sender  Fri Feb  5 16:00:41 2021
Return-Path: <root@sender>
X-Original-To: root@receiver
Delivered-To: root@receiver
Received: from receiver (proxy [192.168.30.30])
        by receiver (Postfix) with ESMTPS id 3B9874160A
        for <root@receiver>; Fri,  5 Feb 2021 16:00:41 +0000 (UTC)
Received: from sender (localhost [127.0.0.1])
        by sender (Postfix) with SMTP id C08023E8E0
        for <root@receiver>; Fri,  5 Feb 2021 16:00:01 +0000 (UTC)
From: <root@sender>
To: <root@receiver>
Date: Fri,  5 Feb 2021 16:00:01 +0000 (UTC)
Message-Id: <c2f05.0003.0000@sender>
Subject: Hi, Receiver from Sender

XXXXXXXXXX

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

```sh
warp main 🏄 vagrant ssh sender
vagrant@sender:~$ cd /vagrant
vagrant@sender:/vagrant$ make plugin && make
env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildmode=plugin -o .dist/mysql.so plugin/mysql/main.go
env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildmode=plugin -o .dist/file.so plugin/file/main.go
env GOOS=linux GOARCH=amd64 go build -o warp ./cmd/warp/main.go
vagrant@sender:/vagrant$ /vagrant/warp -ip 192.168.30.30 -port 10025 -plugins mysql
```

Run on vagrant:

```sql
vagrant@sender:/vagrant$ sudo mysql -uroot -D warp
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
