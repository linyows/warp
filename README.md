<br><br><br><br><br><br><p align="center">
  <img alt="WARP" src="https://github.com/linyows/warp/blob/master/misc/warp.svg" width="200">
</p>
<p align="center">
  <strong>WARP</strong> is an outbound <b>transparent</b> SMTP proxy.
</p><br><br><br><br><br><br><br><br>
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

![Architecture](https://github.com/linyows/warp/blob/master/misc/architecture.png)

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
2021/02/06 14:50:44 warp listens to 192.168.30.30:10025
```

Send mail on sender:

```sh
warp master 🏄 vagrant ssh sender
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
2021/02/06 14:50:48 |> STARTTLS
2021/02/06 14:50:48 >| MAIL FROM:<root@sender> SIZE=327\r\nRCPT TO:<root@receiver> ORCPT=rfc822;root@receiver\r\nDATA\r\n
2021/02/06 14:50:48 |< 220 2.0.0 Ready to start TLS\r\n
2021/02/06 14:50:48 |> EHLO receiver
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
warp master 🏄 vagrant ssh receiver
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
