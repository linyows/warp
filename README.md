<p align="right">English | <a href="https://github.com/linyows/warp/blob/main/README.ja.md">日本語</a></p>

<br><br><br><br><br><br><br><br><p align="center">
  <a href="https://warp.linyo.ws">
    <img alt="WARP" src="https://github.com/linyows/warp/blob/main/misc/warp.svg" width="200">
  </a>
</p><br><br><br><br><br><br><br><br>

<strong>Warp</strong> is an outbound transparent SMTP proxy.
SMTP level logging is possible, and throttling is detected from MX response time and other comprehensive response status.
Additionally, it is possible to use different IP addresses for outgoing communications based on internal reputation.
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

Warp deploys as a transparent proxy using Linux's `SO_ORIGINAL_DST` — it intercepts SMTP connections at the network level, meaning **zero changes to your existing mail clients or applications**. Once in place, Warp provides:

### Observability

Every SMTP command and response is captured in real time. You can log to files, MySQL, SQLite, or send notifications to Slack — all through a simple plugin system. Connection metadata (sender, recipient, HELO hostname, elapsed time) is automatically extracted and structured for analysis.

### Filtering

Warp can buffer the entire message during the DATA phase and pass it to a **filter hook** before relaying. Your filter logic decides what happens next:

| Action | Behavior |
|---|---|
| **Relay** | Pass the message through to the destination server as-is |
| **Reject** | Return an SMTP error to the sender — the message never reaches the destination |
| **Add Header** | Modify the message (e.g., add `X-Spam-Score` headers) and then relay |

### Deliverability Intelligence

By measuring the time between connection and the server's `354` response, Warp detects **MX throttling patterns**. Combined with SMTP response codes, you get a comprehensive view of how remote servers are treating your mail.

### IP Routing

Warp supports configurable **outbound IP addresses**, enabling you to route mail through different IPs based on sender reputation, recipient domain, or any custom logic.

## Architecture

<p align="center">
  <img src="https://github.com/linyows/warp/blob/main/misc/architecture.png" alt="Warp Architecture">
</p>

```
                    Transparent Proxy (iptables REDIRECT)
                    ┌─────────────────────────────┐
  SMTP Client ────▶ │           Warp              │ ────▶ Destination MX
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

**Key components:**

- **Server** — Listens for incoming TCP connections and extracts the original destination using `SO_ORIGINAL_DST`
- **Pipe** — Manages bidirectional data flow between client and server with upstream/downstream mediators
- **Mediators** — Inspect and transform SMTP traffic in each direction, extracting metadata (MAIL FROM, RCPT TO, HELO) and handling STARTTLS negotiation
- **Hook System** — Extensible plugin architecture for logging (`AfterComm`, `AfterConn`) and filtering (`BeforeRelay`)

### STARTTLS Handling

Warp transparently handles TLS negotiation: it strips `STARTTLS` from the server's EHLO response to the client, then initiates its own TLS connection to the destination server. This allows Warp to inspect SMTP traffic while maintaining encrypted delivery.

## Getting Started

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

Redirect outgoing SMTP traffic to Warp:

```bash
iptables -t nat -A OUTPUT -p tcp --dport 25 -j REDIRECT --to-port 10025
```

### Command Line Options

| Flag | Default | Description |
|---|---|---|
| `-ip` | `127.0.0.1` | Listen IP address |
| `-port` | *(required)* | Listen port |
| `-outbound-ip` | `0.0.0.0` | Source IP for outgoing connections |
| `-plugins` | | Comma-separated plugin names: `mysql`, `sqlite`, `file`, `slack` |
| `-message-size-limit` | `10240000` | Maximum message size in bytes (~10MB) |
| `-verbose` | `false` | Enable detailed logging |
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

Warp uses Go's plugin system to load `.so` files from `/opt/warp/plugins/` (or the path specified by `PLUGIN_PATH` environment variable). Each plugin implements the `Hook` interface:

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

```bash
cd plugins/file
go build -buildmode=plugin -o /opt/warp/plugins/file.so
```

### MySQL Setup

Use the provided schema to set up the database:

```bash
mysql < misc/setup.sql
```

## Use Cases

- **Email Gateway** — Centralized SMTP inspection point for your organization
- **Spam & Phishing Prevention** — Filter outbound messages before they damage your reputation
- **Compliance & Auditing** — Log every outbound email with full SMTP-level detail
- **Data Loss Prevention (DLP)** — Inspect message content for sensitive data before delivery
- **Deliverability Monitoring** — Track MX response times and detect throttling in real time
- **IP Reputation Management** — Route mail through different IPs based on sender behavior
- **Development & Debugging** — Capture and inspect SMTP traffic during development

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
