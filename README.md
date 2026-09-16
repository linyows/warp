<p align="right">English | <a href="https://github.com/linyows/warp/blob/main/README.ja.md">日本語</a></p>

<br><br><br><br><br><br><br><br><p align="center">
  <a href="https://warp.linyo.ws">
    <img alt="WARP" src="https://github.com/linyows/warp/blob/main/misc/warp.svg" width="200">
  </a>
</p><br><br><br><br><br><br><br><br>

<strong>Warp</strong> is an outbound transparent SMTP proxy.
It records every SMTP command and response of an outgoing session, measures how long each destination takes before it accepts a message, and can hand the message to a filter hook before relaying it.
https://warp.linyo.ws

<br>
<p align="center">
  <a href="https://github.com/linyows/warp/actions" title="actions"><img src="https://img.shields.io/github/actions/workflow/status/linyows/warp/build.yml?branch=main&style=for-the-badge"></a>
  <a href="https://github.com/linyows/warp/releases"><img src="http://img.shields.io/github/release/linyows/warp.svg?style=for-the-badge" alt="GitHub Release"></a>
</p>

## The Problem

Operating email infrastructure at scale comes with significant challenges:

- **No visibility** — You can't see what's being sent, to whom, or how the remote server responds
- **Deliverability issues** — Throttling and blocklisting happen silently; you only find out when users complain
- **No filtering layer** — Compromised accounts or misconfigured applications send spam or phishing emails before anyone notices
- **IP reputation management** — A single bad sender can damage the reputation of your entire IP range
- **Compliance gaps** — Auditing outbound email requires bolting on external tools that don't integrate cleanly

## How Warp Solves It

Warp deploys as a transparent proxy using Linux's `SO_ORIGINAL_DST` — it intercepts SMTP connections at the network level, meaning **zero changes to your existing mail clients or applications**. That socket option reads the connection tracking entry created by NAT, so Warp has to run on the host where the iptables rule is applied; the MTA itself may run anywhere. Once in place, Warp provides:

### Observability

Every SMTP command and response is captured in real time. You can log to files, MySQL, SQLite, or send notifications to Slack — all through a simple plugin system. Connection metadata (sender, recipient, HELO hostname, elapsed time) is automatically extracted and structured for analysis.

### Filtering

Warp can buffer the entire message during the DATA phase and pass it to a **filter hook** before relaying. Your filter logic decides what happens next:

| Action | Behavior |
|---|---|
| **Relay** | Pass the message through to the destination server as-is |
| **Reject** | Return an SMTP error to the sender — the message never reaches the destination |
| **Add Header** | Modify the message (e.g., add `X-Spam-Score` headers) and then relay |

Filtering is an extension point, not a feature you can switch on: none of the built-in plugins implement `BeforeRelay`, so the decision logic is a plugin you write yourself. Warp uses the first loaded plugin that implements `FilterHook`.

### Deliverability Signals

Warp records the time between connecting to the destination and its `354` response, the point at which the destination is ready to accept the message body. That elapsed time goes into the log and into the connection metadata passed to hooks, next to every SMTP status code of the session. Warp does not classify the result: concluding that a destination is throttling you is left to whatever consumes the logs.

### Outbound Address

The source address used for relayed connections is set with `-outbound-ip` and applies to the whole process. Sending through more than one address means running one instance per address and choosing between them in the iptables rules; Warp does not pick an address per message.

## Architecture

<p align="center">
  <img src="https://github.com/linyows/warp/blob/main/misc/architecture.png" alt="Warp Architecture">
</p>

```
                    Transparent Proxy (iptables nat)
                    ┌─────────────────────────────┐
  SMTP Client ────▶ │           Warp              │ ────▶ Destination MX
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

**Key components:**

- **Server** — Listens for incoming TCP connections and extracts the original destination using `SO_ORIGINAL_DST`
- **Pipe** — Manages bidirectional data flow between client and server with upstream/downstream mediators
- **Mediators** — Inspect and transform SMTP traffic in each direction, extracting metadata (MAIL FROM, RCPT TO, HELO) and handling STARTTLS negotiation
- **Hook System** — Extensible plugin architecture for logging (`AfterComm`, `AfterConn`) and filtering (`BeforeRelay`)

### STARTTLS Handling

Warp transparently handles TLS negotiation: it strips `STARTTLS` from the server's EHLO response to the client, then initiates its own TLS connection to the destination server. This allows Warp to inspect SMTP traffic while maintaining encrypted delivery.

### Log Format

Every log line carries the connection id and a two-character marker telling which leg of the session the data belongs to. The markers separate traffic that Warp passed through from traffic it handled itself, which is what makes a STARTTLS session readable:

| Marker | Meaning |
|---|---|
| `->` | Client to destination |
| `<-` | Destination to client |
| `>\|` | Client to Warp |
| `\|>` | Warp to destination |
| `\|<` | Destination to Warp |
| `<\|` | Warp to client |
| `--` | An event inside Warp |

```
2023/11/27 08:14:54.947569 01HG7XGYYZCEB3WSNXPAVG4ZZN -> EHLO localhost\r\n
2023/11/27 08:14:54.947752 01HG7XGYYZCEB3WSNXPAVG4ZZN |< 250-example.local\r\n250-STARTTLS\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951257 01HG7XGYYZCEB3WSNXPAVG4ZZN <| 250-example.local\r\n250 8BITMIME\r\n
2023/11/27 08:14:54.951335 01HG7XGYYZCEB3WSNXPAVG4ZZN -- pipe locked for tls connection
2023/11/27 08:14:54.951341 01HG7XGYYZCEB3WSNXPAVG4ZZN |> STARTTLS\r\n
2023/11/27 08:14:54.957949 01HG7XGYYZCEB3WSNXPAVG4ZZN -- tls connected, to pipe unlocked
2023/11/27 08:14:54.959709 01HG7XGYYZCEB3WSNXPAVG4ZZN -- from:alice@example.test to:bob@example.local elapse:14 msec
```

## Getting Started

### Requirements

- **Linux with Netfilter** — the transparent proxy relies on `SO_ORIGINAL_DST`, and Warp must run on the host that performs the NAT
- **Go 1.23 or later** to build from source
- **cgo** (`CGO_ENABLED=1`) if you want to use plugins; Go's plugin system does not work in a pure-Go build

### Installation

```bash
go install github.com/linyows/warp/cmd/warp@latest
```

Or download a pre-built binary from [Releases](https://github.com/linyows/warp/releases).

### Basic Usage

```bash
warp -ip 0.0.0.0 -port 10025 -verbose
```

### iptables Setup (Transparent Proxy)

Redirect outgoing SMTP traffic to Warp. When the MTA runs on the same host as Warp, the connection is created locally, so the rule goes in the `OUTPUT` chain:

```bash
iptables -t nat -A OUTPUT -p tcp --dport 25 -j DNAT --to-destination 127.0.0.1:10025
```

When the MTA runs on another host, route its outgoing SMTP traffic through the Warp host — for example by making it the default gateway of the network of the MTA — and redirect the traffic in the `PREROUTING` chain instead:

```bash
iptables -t nat -A PREROUTING -p tcp --dport 25 -j DNAT --to-destination 203.0.113.10:10025
```

Either way the NAT, and therefore the connection tracking entry Warp reads, has to happen on the host running Warp.

### Command Line Options

| Flag | Default | Description |
|---|---|---|
| `-ip` | `127.0.0.1` | Listen IP address |
| `-port` | *(ephemeral)* | Listen port |
| `-outbound-ip` | `0.0.0.0` | Source IP for outgoing connections |
| `-plugins` | | Comma-separated plugin names: `mysql`, `sqlite`, `file`, `slack` |
| `-message-size-limit` | `10240000` | Maximum message size in bytes (~10MB) |
| `-verbose` | `false` | Enable detailed logging |
| `-pprof` | | Expose `net/http/pprof` on `host:port`; bind it to a loopback address |
| `-version` | `false` | Show version information |

### Example: Full Setup

```bash
# Start Warp with MySQL logging and Slack notifications
warp -ip 0.0.0.0 -port 10025 \
     -outbound-ip 203.0.113.10 \
     -plugins mysql,slack \
     -message-size-limit 20480000 \
     -verbose
```

## Plugin System

Warp uses Go's plugin system to load `.so` files from `/opt/warp/plugins/` (or the path specified by the `PLUGIN_PATH` environment variable). Only the plugins named in `-plugins` are loaded; the rest of the directory is ignored. Each plugin implements the `Hook` interface:

```go
type Hook interface {
    Name() string
    AfterInit()
    AfterComm(*AfterCommData)  // Called after each SMTP command/response
    AfterConn(*AfterConnData)  // Called when a connection closes
}
```

For message filtering, implement the `FilterHook` interface:

```go
type FilterHook interface {
    Hook
    BeforeRelay(*BeforeRelayData) *FilterResult
}
```

### Built-in Plugins

| Plugin | Description | Environment Variables |
|---|---|---|
| **file** | Logs all SMTP communications to a JSON file | `FILE_PATH` |
| **mysql** | Stores communications and connections in MySQL | `DSN` |
| **sqlite** | Stores communications and connections in SQLite | `DSN` |
| **slack** | Sends connection notifications to a Slack channel | `SLACK_TOKEN`, `SLACK_CHANNEL` |

### Building Plugins

Go plugins require cgo, so Warp and the plugins have to be built with `CGO_ENABLED=1` — the `build-withcgo` target does that:

```bash
make build-withcgo
make file-plugin mysql-plugin      # builds into plugins/*.so
PLUGIN_PATH=./plugins ./warp -ip 0.0.0.0 -port 10025 -plugins file,mysql
```

Install the `.so` files into `/opt/warp/plugins/` to run without `PLUGIN_PATH`.

### MySQL Setup

Use the provided schema to set up the database:

```bash
mysql < misc/setup.sql
```

Go plugins are version-locked: a plugin has to be built with the same Go toolchain and the same dependency versions as Warp itself, or `plugin.Open` refuses to load it. Build them together.

## Use Cases

- **Email Gateway** — Centralized SMTP inspection point for your organization
- **Spam & Phishing Prevention** — Filter outbound messages before they damage your reputation, with a filter plugin of your own
- **Compliance & Auditing** — Log every outbound email with full SMTP-level detail
- **Data Loss Prevention (DLP)** — Inspect message content for sensitive data before delivery
- **Deliverability Monitoring** — Track MX response times and SMTP status codes over time
- **IP Reputation Management** — Separate outgoing traffic onto different source addresses by running an instance per address
- **Development & Debugging** — Capture and inspect SMTP traffic during development

## Development

```bash
make build          # build the warp binary
make test           # unit tests
make integration    # run a sending client and a receiving server against Warp
make test-all       # full suite with race detector and coverage
```

`make integration` starts an outgoing mail client and an incoming SMTP server in Go and sends a message through Warp, so the whole path can be exercised on a workstation without any iptables rule.

## Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create a new Pull Request

## Author

[linyows](https://github.com/linyows)

## License

MIT
