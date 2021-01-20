<p align="center">
(⌒▽　　▽⌒)<br>
＼ ▽　▽ ／<br>
|＼▽▽／|<br>
|ｏ＼／ｏ|<br>
ヽｏoｏoノ<br>
￣TT￣<br>
(⌒||⌒)<br>
|￣￣￣￣￣|<br>
|＿＿＿＿＿|<br>
|　　　　|<br>
|＿＿＿＿|<br>
</p>

<p align="center">
<strong>WARP</strong>: This is an outbound transparent SMTP proxy.
</p>
<p align="center">
<a href="https://github.com/linyows/warp/actions" title="actions"><img src="https://img.shields.io/github/workflow/status/linyows/warp/Go?style=for-the-badge"></a>
<a href="https://github.com/linyows/warp/releases"><img src="http://img.shields.io/github/release/linyows/warp.svg?style=for-the-badge" alt="GitHub Release"></a>
<a href="https://github.com/linyows/warp/blob/master/LICENSE"><img src="http://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge" alt="MIT License"></a>
<a href="http://godoc.org/github.com/linyows/warp"><img src="http://img.shields.io/badge/go-documentation-blue.svg?style=for-the-badge" alt="Go Documentation"></a>
<a href="https://codecov.io/gh/linyows/warp"> <img src="https://img.shields.io/codecov/c/github/linyows/warp.svg?style=for-the-badge" alt="codecov"></a>
</p><br><br>

Usage
--

Proxy:

```sh
vagrant@sender:~$ /vagrant/warp -ip 192.168.30.30 -port 10025
2021/01/19 16:17:20 new connection
2021/01/19 16:17:20 remote addr: 192.168.30.40:42516 origin addr: 192.168.30.50:25
2021/01/19 16:17:20 start proxy
2021/01/19 16:17:20 end proxy
2021/01/19 16:17:20 connection closed
```

Send mail:

```sh
vagrant@sender:~$ telnet localhost 25
Trying 127.0.0.1...
Connected to localhost.
Escape character is '^]'.
220 sender ESMTP Postfix (Ubuntu)
HELO localhost
250 sender
MAIL FROM: root@sender
250 2.1.0 Ok
RCPT TO: root@receiver
250 2.1.5 Ok
DATA
354 End data with <CR><LF>.<CR><LF>
Subject: Yo
From: root@sender
To: root@receiver
This is from proxy.
.
250 2.0.0 Ok: queued as 345FE3E8F8
quit
221 2.0.0 Bye
Connection closed by foreign host.
```

Received mail:

```sh
vagrant@receiver:~$ sudo cat /var/spool/mail/root
From root@sender  Tue Jan 19 16:17:20 2021
Return-Path: <root@sender>
X-Original-To: root@receiver
Delivered-To: root@receiver
Received: from sender (proxy [192.168.30.30])
        by receiver (Postfix) with ESMTPS id 978363E8E4
        for <root@receiver>; Tue, 19 Jan 2021 16:17:20 +0000 (UTC)
Received: from localhost (localhost [127.0.0.1])
        by sender (Postfix) with SMTP id 345FE3E8F8
        for <root@receiver>; Tue, 19 Jan 2021 16:16:21 +0000 (UTC)
Subject: Yo
From: root@sender
To: root@receiver
Message-Id: <20210119161632.345FE3E8F8@sender>
Date: Tue, 19 Jan 2021 16:16:21 +0000 (UTC)

This is from proxy.

```

Contribution
--

1. Fork ([https://github.com/linyows/warp/fork](https://github.com/linyows/warp/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create a new Pull Request

Author
--

[linyows](https://github.com/linyows)
