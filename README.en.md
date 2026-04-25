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

| Platform              | Package Manager | Package Format |
| :-------------------- | :-------------- | :------------- |
| OpenWrt >= 25.12.X    | apk             | .apk           |
| OpenWrt <= 24.10.X    | opkg            | .ipk           |
| Entware (Keenetic)    | opkg            | .ipk           |

## Installation

The following command automatically detects your platform and architecture, then adds the package repository:

```shell
wget -qO- https://raw.githubusercontent.com/KilimcininKorOglu/routex/develop/scripts/add_repo.sh | sh
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
apk update && apk add --allow-untrusted routex
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

After installation, the web interface is available at `http://<router-ip>:8080` by default. Through the web interface you can:

- Create, edit, and delete groups
- Add, edit, and reorder rules
- Import and export configuration
- Search and filter
- View system settings

## Technical Details

| Property         | Value                                              |
| :--------------- | :------------------------------------------------- |
| Language         | Go 1.23                                            |
| Web Interface    | templ + htmx + Alpine.js                           |
| DNS Engine       | MITM proxy with miekg/dns                          |
| Network Control  | iptables, ipset, netlink                           |
| Configuration    | YAML                                               |
| Authentication   | JWT (optional)                                     |
| Package Format   | .ipk (opkg) and .apk (Alpine)                     |
| License          | GPL-3.0-or-later                                   |

## Building from Source

```shell
cp config/openwrt/aarch64_generic.config .config
make
```

Output is written to the `.build/` directory. Building requires Go 1.23, templ, upx, and fakeroot.

## License

This project is licensed under [GPL-3.0-or-later](LICENSE).

---

This project was reorganized using [https://gitlab.com/magitrickle/magitrickle](https://gitlab.com/magitrickle/magitrickle).
