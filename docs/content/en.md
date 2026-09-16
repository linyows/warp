---
title: Warp
description: Warp is an outbound transparent SMTP proxy with SMTP-level logging, message filtering and deliverability insight.
---

**WARP** is an outbound **transparent** SMTP proxy.

It sits between your mail servers and the internet without any change to them. Outgoing SMTP connections are redirected to Warp at the network level, Warp records the whole session, hands the message to a filter hook if one is installed, and relays it to the destination MX.

[![](/images/github-mark.svg) linyows/warp - GitHub](https://github.com/linyows/warp)

## Why

Outbound email is mostly invisible while it is working, and hard to explain once it is not:

- You cannot see what was sent, to whom, or how the destination server replied.
- Throttling and blocklisting happen silently — you usually learn about them from your users.
- A single compromised account or misconfigured application can damage the reputation of an entire IP range.
- Auditing outbound mail means bolting external tools onto the MTA.

Warp turns the outbound path itself into an observation and control point, without touching the MTA configuration.

## Features

### Observability

Every SMTP command and reply is captured as it happens, in both directions and on both sides of the proxy. Warp also extracts the metadata of the session — `MAIL FROM`, `RCPT TO`, the HELO hostname and the time the destination took to accept the message — and passes it to hooks as structured data. Logs go to stdout by default, and plugins can write them to a file, MySQL, SQLite or Slack.

### Filtering

Warp can buffer the entire message during the DATA phase and hand it to a filter hook before relaying it. The hook decides what happens next:

| Action | Behaviour |
|---|---|
| **Relay** | Pass the message to the destination server as-is |
| **Reject** | Return an SMTP error to the sender; the message never reaches the destination |
| **Add header** | Modify the message, for example to add an `X-Spam-Score` header, and then relay it |

Filtering is an extension point rather than something to switch on: none of the built-in plugins implement `BeforeRelay`, so the decision logic is a plugin you write yourself. Warp uses the first loaded plugin that implements `FilterHook`.

### Deliverability signals

Warp records the time between connecting to the destination and its `354` reply, the point at which the destination is ready to accept the message body. That elapsed time goes into the log and into the connection metadata handed to hooks, next to every SMTP status code of the session. Warp does not judge the result itself — a destination that keeps delaying that reply is throttling you, but concluding so is left to whatever reads the logs.

### Outbound address

The source address used for relayed connections is set with `-outbound-ip` and applies to the whole process. Sending through more than one address means running one instance per address and choosing between them in the iptables rules; Warp does not pick an address per message.

## How It Works

Warp does not speak for the MTA; it stands in the middle of an existing connection. An iptables rule in the `nat` table sends the connection to Warp, and Warp asks the kernel where the connection was originally headed through the `SO_ORIGINAL_DST` socket option. It then opens its own connection to that destination and copies traffic in both directions, inspecting it on the way.

`SO_ORIGINAL_DST` reads the connection tracking entry created by NAT, and that entry only exists on the host that performed the NAT. **Warp therefore has to run on the host where the iptables rule is applied.** The MTA itself may run anywhere. This also makes Warp Linux-only in practice, since it relies on Netfilter.

![Warp architecture](/images/architecture.png)

### STARTTLS

Warp removes `STARTTLS` from the EHLO reply it passes back to the client, so the client keeps talking in plain text to Warp and its commands stay readable. Warp then negotiates TLS with the destination server on its own. Mail still leaves the network encrypted, and the session is still observable.

## Deploy

### Same host as the MTA

The connection is created locally, so the rule goes in the `OUTPUT` chain:

```shell
$ iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination <proxy-ip>:<proxy-port>
```

This is the layout in the diagram above, where the MTA and Warp share a single host.

### Separate hosts

Route the outgoing SMTP traffic of the MTA through the Warp host, for example by making it the default gateway of the network of the MTA, and redirect the traffic in the `PREROUTING` chain instead:

```shell
$ iptables -t nat -A PREROUTING -p tcp --dport 25 -j DNAT --to-destination <proxy-ip>:<proxy-port>
```

NAT then happens on the Warp host, so the original destination stays available to Warp.

## Usage

Install the command with Go, or download a pre-built binary from [Releases](https://github.com/linyows/warp/releases):

```shell
$ go install github.com/linyows/warp/cmd/warp@latest
```

Start the proxy. Without `-port` it listens on an ephemeral port of `127.0.0.1`, which is useful for a first look but not for a real deployment:

```shell
$ warp -ip 0.0.0.0 -port 10025 -verbose
2023/11/27 08:16:28.368892 warp listens to 0.0.0.0:10025
```

| Flag | Default | Description |
|---|---|---|
| `-ip` | `127.0.0.1` | Listen address |
| `-port` | ephemeral | Listen port |
| `-outbound-ip` | `0.0.0.0` | Source address used for relayed connections |
| `-plugins` | | Comma-separated plugin names: `mysql`, `sqlite`, `file`, `slack` |
| `-message-size-limit` | `10240000` | Maximum size of a message in bytes |
| `-verbose` | `false` | Log every SMTP command and reply |
| `-pprof` | | Expose `net/http/pprof` on `host:port`; bind it to a loopback address |
| `-version` | `false` | Show the build version |

## Plugins

Warp writes to stdout on its own. Plugins extend it with somewhere to put those logs, and with the filtering decision itself. They are Go plugins: `.so` files loaded at startup from `/opt/warp/plugins`, or from the directory in `PLUGIN_PATH`. Only the names listed in `-plugins` are loaded.

| Plugin | Description | Environment |
|---|---|---|
| **file** | Appends every SMTP communication to a file as JSON | `FILE_PATH` |
| **mysql** | Stores connections and communications in MySQL | `DSN` |
| **sqlite** | Stores connections and communications in SQLite | `DSN` |
| **slack** | Posts a notification to a channel when a connection closes | `SLACK_TOKEN`, `SLACK_CHANNEL` |

Go plugins require cgo, so both Warp and the plugins have to be built with `CGO_ENABLED=1`:

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

The `sqlite` plugin takes a file path in `DSN`, such as `/var/db/warp.sqlite3`.

### Database schema

The `mysql` and `sqlite` plugins use two tables. `connections` holds one row per session — `id`, `mail_from`, `mail_to`, `elapse` in milliseconds and `occurred_at` — and `communications` holds one row per SMTP command or reply, referring back to the session through `connection_id`. The MySQL schema is in [misc/setup.sql](https://github.com/linyows/warp/blob/main/misc/setup.sql):

```shell
$ mysql < misc/setup.sql
```

### Writing a plugin

A plugin exports a `Hook` variable implementing the logging interface, and optionally `BeforeRelay` for filtering:

```go
type Hook interface {
    Name() string
    AfterInit()
    AfterComm(*AfterCommData) // after each SMTP command or reply
    AfterConn(*AfterConnData) // when the connection closes
}

type FilterHook interface {
    Hook
    BeforeRelay(*BeforeRelayData) *FilterResult
}
```

Go plugins are version-locked: a plugin has to be built with the same Go toolchain and the same dependency versions as Warp itself, or `plugin.Open` refuses to load it. Build them together.

## Logs

Each log line carries the connection id and a two-character marker saying which leg of the session the data belongs to. The markers distinguish traffic that Warp passed through from traffic it handled itself, which is what makes a STARTTLS session readable:

| Marker | Meaning |
|---|---|
| `->` | Client to destination |
| `<-` | Destination to client |
| `>\|` | Client to Warp |
| `\|>` | Warp to destination |
| `\|<` | Destination to Warp |
| `<\|` | Warp to client |
| `--` | An event inside Warp |

A session where the destination offers STARTTLS looks like this:

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

Warp ships with an integration test that starts a sending client and a receiving server, so the whole path can be exercised locally with `make integration`.
