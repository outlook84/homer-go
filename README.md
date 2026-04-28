# homer-go

[简体中文](./docs/README.zh-CN.md)

homer-go is a lightweight, self-hosted dashboard inspired by [Homer](https://github.com/bastienwirtz/homer). It keeps your links, services, status badges, and quick actions in one browser page, configured with a simple `config.yml` file.

This project is aimed at people who want the Homer experience as a single Go server and server-side status collection for supported smart cards.

## What is different from Homer?

homer-go reads much of the same YAML shape as Homer, so existing Homer users can usually start from their current `config.yml`. The important differences are:

- homer-go runs as a Go HTTP server. Homer is a static Vue app served by a web server.
- Configuration is loaded from the data directory, usually `config.yml` beside the binary or inside `/data` in Docker. Homer normally uses `assets/config.yml`.
- Supported smart-card data is collected by homer-go on the server. This usually avoids browser CORS problems that affect Homer smart cards.
- The app includes its own embedded assets.
- Search is form-based rather than Homer's fuzzy keyboard search. The `/`, `Escape`, and `Enter` search shortcuts from Homer are not currently implemented.
- Some Homer configuration keys are currently ignored, including `hotkey`, `proxy.useCredentials`, per-item `useCredentials`, and per-item refresh interval keys.
- homer-go does not yet implement every Homer smart card. Unsupported service types still render as normal links, but no live status is collected.

## Features

- YAML-based dashboard configuration
- Groups, service cards, tags, icons, logos, quick links, and custom card colors
- Light, dark, and automatic theme preference
- Column and list layout preference
- Search across name, subtitle, tag, and keywords
- Multiple pages using `page-name.yml`
- Optional remote message banner
- PWA assets and offline cached shell
- Optional base path for reverse proxies, such as `/dashboard`
- Server-side status collection for supported integrations

## Quick Start

Run the binary from an empty directory:

```sh
homer-go
```

Open `http://localhost:8732`.

If `config.yml` does not exist, homer-go creates an example file for you. Edit that file and refresh the page.

Useful options:

```sh
homer-go -addr :8732 -data /path/to/config -base-path /homer-go
```

Environment variables are also supported:

- `HOMER_GO_ADDR`, default `:8732`
- `HOMER_GO_DATA_DIR`, default `.`
- `HOMER_GO_ASSETS_DIR`, default `assets`
- `HOMER_GO_BASE_PATH`, default empty

## Docker

```sh
docker run -d \
  --name homer-go \
  -p 8732:8732 \
  -v /path/to/homer-go-data:/data \
  --restart unless-stopped \
  ghcr.io/outlook84/homer-go:latest
```

The container stores `config.yml` in `/data`. If the mounted directory is empty and writable, homer-go writes an example configuration there on first start.

## Configuration

A small configuration looks like this:

```yaml
title: "Home"
subtitle: "Dashboard"
logo: "/assets/icons/homer-go-logo-v2.png"
header: true
columns: "3"
defaults:
  layout: columns
  colorTheme: auto

links:
  - name: "GitHub"
    icon: "fab fa-github"
    url: "https://github.com/outlook84/homer-go"
    target: "_blank"

services:
  - name: "Apps"
    icon: "fas fa-server"
    items:
      - name: "Home Assistant"
        type: "HomeAssistant"
        icon: "fas fa-house"
        url: "https://homeassistant.example.com"
        apikey: "your-token"
        tag: "home"
      - name: "Docs"
        icon: "fas fa-book"
        subtitle: "Local documentation"
        url: "https://docs.example.com"
        quick:
          - name: "Admin"
            icon: "fas fa-lock"
            url: "https://docs.example.com/admin"
```

For additional pages, create `media.yml`, `infra.yml`, or another page file in the same data directory. Link to it with `#media` in `links`, or open `/?page=media`.

Local images and custom stylesheets can be placed in the data directory and referenced with relative paths. Built-in assets are available under `/assets/`.

## Supported Smart Cards

homer-go currently collects status for:

AdGuardHome, Docuseal, DockerSocketProxy, Emby, FreshRSS, Gatus, Gitea, Glances, Gotify, Healthchecks, HomeAssistant, HyperHDR, Immich, Jellyfin, Lidarr, Matrix, Mealie, Medusa, Miniflux, Mylar, NetAlertx, Nextcloud, Olivetin, OpenHAB, PaperlessNG, PeaNUT, PiAlert, Ping, Portainer, Prometheus, Proxmox, Prowlarr, qBittorrent, Radarr, Readarr, SABnzbd, Scrutiny, Sonarr, SpeedtestTracker, Tautulli, Tdarr, Traefik, TruenasScale, UptimeKuma, Vaultwarden, Wallabag, and WUD.

Compared with the referenced Homer source, these Homer smart cards are not currently implemented in homer-go: CopyToClipboard, Jellystat, Linkding, OctoPrint/Moonraker, OpenWeather, PiHole, Plex, rTorrent, and Transmission.

## Security Notes

Service tokens in `config.yml` are used by homer-go to collect status. Treat the data directory as sensitive.

Unlike Homer installations that serve `assets/config.yml` directly as a static file, homer-go is designed to keep `config.yml` in the data directory.

## License

Apache-2.0.
