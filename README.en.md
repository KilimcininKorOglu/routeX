# RouteX

DNS-based selective traffic routing application. Designed for OpenWrt and Entware (Keenetic) routers.

## What It Does

RouteX intercepts DNS queries on your router and routes traffic to specific network interfaces based on domain name rules. For example, you can send certain services through a VPN tunnel while keeping the rest on your regular connection.

How it works:

1. An intermediary layer is placed in front of the existing DNS server
2. Incoming DNS queries are intercepted and responses are cached
3. IP addresses are mapped to domain names
4. Matched traffic is routed to the target interface via iptables/ipset rules

No DNS cache clearing is needed on the client side. A brief warm-up period occurs only when the service is restarted, until the cache is populated.

## Supported Platforms

| Platform           | Package Manager | Package Format |
|:-------------------|:----------------|:---------------|
| OpenWrt >= 25.12.X | apk             | .apk           |
| OpenWrt <= 24.10.X | opkg            | .ipk           |
| Entware (Keenetic) | opkg            | .ipk           |

## Installation

The following command automatically detects your platform and architecture, then adds the package repository:

```shell
wget -qO- https://raw.githubusercontent.com/KilimcininKorOglu/routeX/main/scripts/add_repo.sh | sh
```

Then install with your package manager:

**Entware (Keenetic):**

```shell
opkg update && opkg install routex
/opt/etc/init.d/S99routex start
```

**OpenWrt (opkg):**

```shell
opkg update && opkg install routex
service routex start
```

**OpenWrt (apk):**

```shell
apk update && apk add routex
service routex start
```

To update, simply re-run `opkg update && opkg install routex` or `apk update && apk add routex`.

## Rule Types

Rules defined within groups determine which interface DNS queries are routed to. Four rule types are supported:

### Namespace

Covers the specified domain and all its subdomains.

With the rule `example.com`:

```
example.com             matched
sub.example.com         matched
sub.sub.example.com     matched
anotherexample.com      not matched
example.net             not matched
```

### Wildcard

Flexible matching using `*` (any number of characters) and `?` (single character).

With the rule `*example.com`:

```
example.com             matched
sub.example.com         matched
anotherexample.com      matched
example.net             not matched
```

### Domain

Applies only to the exact specified domain. Subdomains are not included.

With the rule `sub.example.com`:

```
sub.example.com         matched
example.com             not matched
sub.sub.example.com     not matched
```

### Regular Expression

For advanced users. Uses the [dlclark/regexp2](https://github.com/dlclark/regexp2) engine.

With the rule `^[a-z]*example\.com$`:

```
example.com             matched
anotherexample.com      matched
sub.example.com         not matched
```

## Web Interface

After installation, the web interface is available at `http://<router-ip>:7080` by default. Authentication is enabled by default; log in using your router's system credentials:

| Platform           | Username | Password Source   |
|:-------------------|:---------|:------------------|
| OpenWrt            | `root`   | `/etc/shadow`     |
| Entware (Keenetic) | `root`   | `/opt/etc/shadow` |

Entware users must set a root password with the `passwd` command if one is not already configured.

The web interface supports Turkish and English. Use the TR/EN toggle in the top-right corner to switch languages. The preference is saved in a browser cookie.

Through the web interface you can:

- Create, edit, and delete groups
- Add, edit, and reorder rules
- URL-based rule subscription lists (auto-updating)
- Real-time statistics dashboard (live updates via WebSocket)
- Rule test tool (enter a domain to see which rules match)
- Import and export configuration
- Search and filter
- View system settings
- Switch between Turkish and English

## Technical Details

| Property        | Value                                     |
|:----------------|:------------------------------------------|
| Language        | Go 1.25                                   |
| Web Interface   | templ + htmx + Alpine.js + WebSocket      |
| DNS Engine      | MITM proxy with miekg/dns                 |
| DNS Upstream    | Plain DNS, DoT (RFC 7858), DoH (RFC 8484) |
| Network Control | iptables, ipset, netlink                  |
| Configuration   | YAML                                      |
| Authentication  | JWT (enabled by default)                  |
| Localization    | Turkish, English (JSON locale files)      |
| Package Format  | .ipk (opkg) and .apk (Alpine, signed)     |
| License         | GPL-3.0-or-later                          |

## Encrypted DNS (DoH / DoT)

To send DNS queries over an encrypted channel, change the upstream protocol in the configuration file:

**DNS-over-TLS (Cloudflare):**

```yaml
dnsProxy:
  upstream:
    address: 1.1.1.1
    port: 853
  protocol: dot
  tlsServerName: cloudflare-dns.com
```

**DNS-over-HTTPS (Google):**

```yaml
dnsProxy:
  protocol: doh
  url: https://dns.google/dns-query
```

Supported protocols: `plain` (default), `dot`, `doh`. TLS certificate verification can be disabled with `tlsSkipVerify: true`.

## Rule Subscription Lists

Groups can subscribe to URL-based rule lists. Lists are updated automatically:

```yaml
groups:
  - name: Ad Blocking
    interface: nwg0
    subscriptionUrl: https://example.com/block-list.txt
    subscriptionInterval: 1440
```

Supported formats: plain text (one domain per line), hosts file (`0.0.0.0 domain`), AdGuard basic (`||domain^`). Update interval is in minutes (1440 = 24 hours).

## Building from Source

```shell
cp config/openwrt/aarch64_generic.config .config
make
```

Output is written to the `.build/` directory. Building requires Go 1.25, templ, upx, and fakeroot.

## License

This project is licensed under [GPL-3.0-or-later](LICENSE).

## Attribution

This project is a fork of [MagiTrickle](https://gitlab.com/magitrickle/magitrickle), originally developed by Ponywka and contributors under the GPL-3.0-or-later license. See [CONTRIBUTORS.md](CONTRIBUTORS.md) for the full list of contributors.
