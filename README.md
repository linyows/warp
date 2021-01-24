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
<strong><i>WARP</i></strong> : This is an outbound <b>transparent</b> SMTP proxy.
</p>
<p align="center">
<a href="https://github.com/linyows/warp/actions" title="actions"><img src="https://img.shields.io/github/workflow/status/linyows/warp/Go?style=for-the-badge"></a>
<a href="https://github.com/linyows/warp/releases"><img src="http://img.shields.io/github/release/linyows/warp.svg?style=for-the-badge" alt="GitHub Release"></a>
<a href="https://github.com/linyows/warp/blob/master/LICENSE"><img src="http://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge" alt="MIT License"></a>
<a href="http://godoc.org/github.com/linyows/warp"><img src="http://img.shields.io/badge/go-documentation-blue.svg?style=for-the-badge" alt="Go Docs"></a>
<a href="https://codecov.io/gh/linyows/warp"> <img src="https://img.shields.io/codecov/c/github/linyows/warp.svg?style=for-the-badge" alt="codecov"></a>
</p><br><br>

For redirect the port need by iptables rule:

```
iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination <proxy-ip>:<proxy-port>
```

Also, the MTA and Proxy must be on the same host to know the DST Address before NAT.

Usage
--

To check the operation, use the sandbox environment with the Vagrantfile in the repository.

```sh
warp master 🏄 vagrant up
...
warp master 🏄 vagrant status
Current machine states:

sender                    running (virtualbox)
receiver                  running (virtualbox)
```

Start proxy on sender:

```sh
warp master 🏄 vagrant ssh sender
vagrant@sender:~$ /vagrant/warp -ip 192.168.30.30 -port 10025
2021/01/24 03:46:38 warp listens to 192.168.30.30:10025
2021/01/24 03:46:40 new connection
2021/01/24 03:46:40 remote addr: 192.168.30.40:42163 origin addr: 192.168.30.50:25
<===
220 receiver ESMTP Postfix (Ubuntu)

===>
EHLO sender

<===
250-receiver
250-PIPELINING
250-SIZE 10240000
250-VRFY
250-ETRN
250-ENHANCEDSTATUSCODES
250-8BITMIME
250-DSN
250-SMTPUTF8
250 CHUNKING

===>
MAIL FROM:<root@sender> SIZE=309
RCPT TO:<root@receiver> ORCPT=rfc822;root@receiver
DATA

<===
250 2.1.0 Ok
250 2.1.5 Ok
354 End data with <CR><LF>.<CR><LF>
RN
250-ENHANCEDSTATUSCODES
250-8BITMIME
250-DSN
250-SMTPUTF8
250 CHUNKING

===>
Received: from sender (localhost [127.0.0.1])
        by sender (Postfix) with SMTP id 298633E8E0
        for <root@receiver>; Sun, 24 Jan 2021 03:46:40 +0000 (UTC)
From: <root@sender>
To: <root@receiver>
Date: Sun, 24 Jan 2021 03:46:40 +0000 (UTC)
Message-Id: <12065.0003.0000@sender>
Subject: warp!

XXXXXXXXXX
.
QUIT

<===
250 2.0.0 Ok: queued as 326BB410E2
221 2.0.0 Bye
LF>.<CR><LF>
RN
250-ENHANCEDSTATUSCODES
250-8BITMIME
250-DSN
250-SMTPUTF8
250 CHUNKING

2021/01/24 03:46:50 connection closed
```

Send mail on sender:

```sh
warp master 🏄 vagrant ssh sender
vagrant@sender:~$ smtp-source -m 1 -s 1 -l 10 -S 'warp!' -f root@sender -t root@receiver localhost:25
```

Received mail on receiver:

```sh
warp master 🏄 vagrant ssh receiver
vagrant@receiver:~$ sudo cat /var/spool/mail/root
From root@sender  Sun Jan 24 03:46:50 2021
Return-Path: <root@sender>
X-Original-To: root@receiver
Delivered-To: root@receiver
Received: from sender (proxy [192.168.30.30])
        by receiver (Postfix) with ESMTP id 326BB410E2
        for <root@receiver>; Sun, 24 Jan 2021 03:46:50 +0000 (UTC)
Received: from sender (localhost [127.0.0.1])
        by sender (Postfix) with SMTP id 298633E8E0
        for <root@receiver>; Sun, 24 Jan 2021 03:46:40 +0000 (UTC)
From: <root@sender>
To: <root@receiver>
Date: Sun, 24 Jan 2021 03:46:40 +0000 (UTC)
Message-Id: <12065.0003.0000@sender>
Subject: warp!

XXXXXXXXXX

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
