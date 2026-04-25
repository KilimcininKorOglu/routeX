# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [1.7.0] - 2026-04-26

### Added
- WebSocket live stats with 1-second push updates on the statistics page
- Event hub with client registry and broadcast (max 10 concurrent clients)
- Per-client stats pump goroutine with JSON snapshot delivery
- Connection status indicator (Connecting / Live / Offline)
- Targeted DOM updates via data-stat attributes (no full page re-render)
- Graceful fallback to HTMX 5-second polling on WebSocket disconnect

## [1.6.0] - 2026-04-26

### Added
- DNS-over-HTTPS (DoH) upstream support (RFC 8484) with HTTP/2 multiplexing
- DNS-over-TLS (DoT) upstream support (RFC 7858) with TLS connection pooling
- Upstream interface abstraction for protocol-agnostic DNS proxy core
- TLS configuration options: skip verification, custom server name (SNI)
- Protocol selection in config: plain, dot, doh
- Settings page displays upstream protocol, DoH URL, and TLS status

### Changed
- DNS MITM proxy refactored: connection pools extracted into Upstream implementations
- Sample config updated with DoH/DoT examples for Google and Cloudflare

## [1.5.0] - 2026-04-26

### Added
- Rule test tool for domain-to-rule matching debug on dashboard
- Inline domain test input with debounced live results (500ms)
- Displays matched groups, rules, cached IPs, and CNAME aliases
- REST API endpoint GET /api/v1/test?domain=... for programmatic testing

## [1.4.0] - 2026-04-26

### Added
- Real-time statistics dashboard with 5-second auto-refresh
- DNS query counters by type (A, AAAA, PTR, other)
- Response, fake PTR, and dropped AAAA record counters
- Per-group matched domains and ipset entry count tracking
- DNS cache statistics (domain count, address count)
- Uptime display with days/hours/minutes formatting
- Stats tab in navigation bar (Groups | Settings | Stats)
- REST API endpoint GET /api/v1/stats for JSON statistics
- Responsive card grid layout for statistics display

## [1.3.0] - 2026-04-25

### Added
- Rule subscription lists for auto-updating groups from remote URLs
- Multi-format parser supporting plain text, hosts file, and AdGuard basic formats
- Background refresh with configurable interval (default 24 hours)
- Conditional GET (ETag/Last-Modified) for bandwidth-efficient updates
- Subscription status UI with refresh button and auto-polling
- REST API and HTMX endpoints for subscription refresh and status
- Disk-based rule cache to prevent flash wear on routers
- Deterministic rule IDs via SHA-256 to avoid collisions at scale
- RSS badge indicator for subscription groups in collapsed view

## [1.2.0] - 2026-04-25

### Added
- Internationalization (i18n) system with Turkish and English support
- JSON locale files (201 keys per language) embedded in binary
- Language detection middleware (cookie, Accept-Language header, config default)
- TR/EN language switcher toggle in header bar
- Language preference stored in browser cookie

### Changed
- All backend error messages translated to English (internal consistency)
- All templ components accept `loc *i18n.Localizer` parameter for dynamic strings
- Language field added to AppConfig for default language setting
- Fork maintainer added to CONTRIBUTORS.md

## [1.1.0] - 2026-04-25

### Added
- IP-based login rate limiting (5 attempts per 15 minutes per IP)
- Request body size limit (1 MB) on all HTTP endpoints
- Auto-backup of config.yaml before import (config.yaml.bak)
- Rule type validation on API and config import paths
- Authentication credentials guide in README (OpenWrt/Entware)

### Changed
- Authentication enabled by default (uses router's /etc/shadow credentials)
- JWT token expiry reduced from 20 years to 7 days
- Go upgraded from 1.23 to 1.25
- golang.org/x/net upgraded from v0.42.0 to v0.53.0
- Third-party GitHub Actions pinned to commit SHAs
- GitHub Actions workflows restricted to read-only permissions

### Fixed
- iptables-restore stdin injection via group Interface field (CRITICAL)
- iptables-restore stdin injection via ChainPrefix/IpsetPrefix in config import (CRITICAL)
- ReDoS via unbounded regexp2 match timeout (CRITICAL)
- Unix socket exposed without file permission restrictions (CRITICAL)
- Unauthenticated network-wide access on fresh installations (CRITICAL)
- Auth bypass via config import manipulating auth.enabled flag
- MD5 password hashes accepted for authentication
- Verbose error messages exposing internal implementation details
- Clickjacking and MIME sniffing via missing security headers
- Session cookie attribute inconsistency on logout and error paths
- golang.org/x/net infinite parsing loop and quadratic complexity DoS

## [1.0.0] - 2026-04-25

### Added
- DNS-based selective traffic routing for OpenWrt and Entware routers
- Web UI with templ + htmx + Alpine.js (single Go binary, embedded static files)
- Group and rule management (namespace, wildcard, domain, regexp matching)
- Config import/export (YAML/JSON)
- Server-side search with debounced filtering
- Group and rule reordering via up/down controls
- Cookie-based JWT session authentication for web UI
- Bearer token authentication for REST API
- GitHub Pages package repository with automated publishing
- Cross-platform packaging for 40 architectures (ipk and apk formats)
- Xbox-inspired dark theme with Fluent Design elements

### Changed
- Fork and full rename from MagiTrickle to RouteX
- Migrated frontend from Svelte 5 SPA to server-rendered Go templates
- All UI strings localized to Turkish

### Fixed
- CI build failure caused by go mod tidy pruning templ dependency
- Remaining magitrickle references in package scripts, init files, and state directories
