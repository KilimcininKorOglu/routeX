# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/).

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
