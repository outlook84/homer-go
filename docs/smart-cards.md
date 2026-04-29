# Smart Cards

Smart cards are service items with a `type` value. homer-go collects their data on the server and renders the result into the dashboard. Unsupported types still render as normal link cards, but they do not show live status.

```yaml
services:
  - name: "Status"
    items:
      - name: "Homepage"
        type: "Ping"
        icon: "fas fa-heart-pulse"
        url: "https://example.com"
```

## Common Options

| Key | Description |
| --- | --- |
| `type` | Collector type. Matching is case-insensitive. |
| `url` | Card link and default API base URL. |
| `endpoint` | Optional API base URL when the clickable URL differs from the API URL. |
| `subtitle` | Static subtitle. If set, it replaces the live subtitle line. |
| `headers` | Per-card request headers. Overrides `proxy.headers`. |
| `successCodes` | Accepted HTTP status codes for collector requests. Defaults to any `2xx`. |

Global request headers can be configured once:

```yaml
proxy:
  headers:
    Authorization: "Bearer token"
```

Per-card headers override the global header map:

```yaml
items:
  - name: "Private API"
    type: "Ping"
    url: "https://api.example.com/health"
    headers:
      Authorization: "Bearer service-token"
```

Because collection runs on the homer-go server, browser CORS settings on the target service are usually not relevant. The server running homer-go must be able to reach the target URL or socket.

## Supported Types

| Type | Common extra options |
| --- | --- |
| `AdGuardHome` | `username`, `password` for basic auth. |
| `Docuseal` | None beyond common options. |
| `DockerSocketProxy` | `endpoint`, `socket`. Supports HTTP Docker Socket Proxy endpoints and `unix://` sockets. |
| `Emby` | `apikey`, `libraryType` (`movies`, `series`, or omitted). |
| `FreshRSS` | `username`, `password`. |
| `Gatus` | `groups` list to include selected Gatus groups, optional `hideaverages: true`. |
| `Gitea` | None beyond common options. This can also be used against Forgejo-compatible endpoints. |
| `Glances` | Optional `stats` list. |
| `Gotify` | `apikey`. |
| `Healthchecks` | `apikey`. |
| `HomeAssistant` | `apikey`, optional `items` list and `separator`. |
| `HyperHDR` | None beyond common options. |
| `Immich` | `apikey`. |
| `Jellyfin` | `apikey`. |
| `Lidarr` | `apikey`. |
| `Matrix` | None beyond common options. |
| `Mealie` | `apikey`. |
| `Medusa` | `apikey`. |
| `Miniflux` | `apikey`, optional `style: counter`. |
| `Mylar` | `apikey`. |
| `NetAlertx` | `apikey`. |
| `Nextcloud` | None beyond common options. |
| `Olivetin` | None beyond common options. |
| `OpenHAB` | `apikey`, optional `things: true`, `items: true`. |
| `PaperlessNG` | `apikey`. |
| `PeaNUT` | `device`. |
| `PiAlert` | None beyond common options. |
| `Ping` | `endpoint`, `method`, `timeout`, `successCodes`. |
| `Portainer` | `apikey`, optional `environments` list. |
| `Prometheus` | None beyond common options. |
| `Proxmox` | `node`, plus `api_token` or `api_token_id`/`api_token_secret`. Optional `token_id`, `token_secret`, `warning_value`, `danger_value`, `hide`, `hide_decimals`. |
| `Prowlarr` | `apikey`. |
| `qBittorrent` | `username`, `password`. |
| `Radarr` | `apikey`, optional `legacyApi: true`. |
| `Readarr` | `apikey`. |
| `SABnzbd` | `apikey`. |
| `Scrutiny` | None beyond common options. |
| `Sonarr` | `apikey`, optional `legacyApi: true`. |
| `SpeedtestTracker` | `apikey`. |
| `Tautulli` | `apikey`. |
| `Tdarr` | `apikey`. |
| `Traefik` | `basic_auth` in `username:password` form. |
| `TruenasScale` | `api_token`. |
| `UptimeKuma` | `slug` for a named status page. |
| `Vaultwarden` | None beyond common options. |
| `Wallabag` | None beyond common options. |
| `WUD` | None beyond common options. |

Homer smart cards that are not currently implemented in homer-go: `CopyToClipboard`, `Jellystat`, `Linkding`, `OctoPrint`/`Moonraker`, `OpenWeather`, `PiHole`, `Plex`, `rTorrent`, and `Transmission`.

## Examples

### Ping

```yaml
- name: "Public site"
  type: "Ping"
  icon: "fas fa-heart-pulse"
  url: "https://example.com"
  method: "GET"
  timeout: 2500
  successCodes: [200, 204, 301, 302]
```

`Ping` uses `HEAD` by default. Set `method: GET` for services that do not handle `HEAD` correctly.

### Home Assistant

```yaml
- name: "Home Assistant"
  type: "HomeAssistant"
  icon: "fas fa-house"
  url: "https://homeassistant.example.com"
  apikey: "long-lived-access-token"
  items: ["name", "version", "entities"]
  separator: " | "
```

### Portainer

```yaml
- name: "Portainer"
  type: "Portainer"
  icon: "fab fa-docker"
  url: "https://portainer.example.com"
  apikey: "portainer-api-key"
  environments:
    - "local"
    - "prod"
```

### Proxmox

```yaml
- name: "Proxmox"
  type: "Proxmox"
  icon: "fas fa-server"
  url: "https://pve.example.com:8006"
  node: "pve1"
  api_token_id: "root@pam!homer"
  api_token_secret: "secret"
  warning_value: 75
  danger_value: 90
  hide: ["lxcs_total"]
  hide_decimals: false
```

You can also provide a complete `api_token` value, including the `PVEAPIToken=` prefix.

### Docker Socket Proxy

```yaml
- name: "Docker"
  type: "DockerSocketProxy"
  icon: "fab fa-docker"
  endpoint: "http://docker-socket-proxy:2375"
  url: "https://docker.example.com"
```

For local-only deployments, a Unix socket can be used:

```yaml
- name: "Docker"
  type: "DockerSocketProxy"
  socket: "/var/run/docker.sock"
```

Mounting the Docker socket gives homer-go broad access to the Docker host. Prefer a restricted Docker Socket Proxy when possible.
