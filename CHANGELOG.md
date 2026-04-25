# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/).

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
