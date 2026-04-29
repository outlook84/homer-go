# 智能卡片

智能卡片是带有 `type` 值的服务项。homer-go 在服务端收集数据，并将结果渲染到仪表盘中。不支持的类型仍会渲染为普通链接卡片，但不会显示实时状态。

```yaml
services:
  - name: "Status"
    items:
      - name: "Homepage"
        type: "Ping"
        icon: "fas fa-heart-pulse"
        url: "https://example.com"
```

## 通用选项

| 键 | 说明 |
| --- | --- |
| `type` | Collector 类型。匹配时不区分大小写。 |
| `url` | 卡片链接，也是默认 API 基础 URL。 |
| `endpoint` | 当可点击 URL 和 API URL 不同时，可用它指定 API 基础 URL。 |
| `subtitle` | 静态副标题。设置后会替代实时副标题行。 |
| `headers` | 单卡片请求头。会覆盖 `proxy.headers`。 |
| `successCodes` | collector 请求接受的 HTTP 状态码。默认接受任何 `2xx`。 |

可以配置全局请求头：

```yaml
proxy:
  headers:
    Authorization: "Bearer token"
```

单卡片请求头会覆盖全局请求头：

```yaml
items:
  - name: "Private API"
    type: "Ping"
    url: "https://api.example.com/health"
    headers:
      Authorization: "Bearer service-token"
```

由于收集逻辑在 homer-go 服务端运行，目标服务的浏览器 CORS 设置通常无关。运行 homer-go 的服务器必须能够访问目标 URL 或 socket。

## 支持的类型

| 类型 | 常用额外选项 |
| --- | --- |
| `AdGuardHome` | `username`、`password` 用于 basic auth。 |
| `Docuseal` | 无额外选项。 |
| `DockerSocketProxy` | `endpoint`、`socket`。支持 HTTP Docker Socket Proxy 和 `unix://` socket。 |
| `Emby` | `apikey`、`libraryType`（`movies`、`series` 或省略）。 |
| `FreshRSS` | `username`、`password`。 |
| `Gatus` | `groups` 列表用于筛选 Gatus 分组，可选 `hideaverages: true`。 |
| `Gitea` | 无额外选项。也可用于 Forgejo 兼容端点。 |
| `Glances` | 可选 `stats` 列表。 |
| `Gotify` | `apikey`。 |
| `Healthchecks` | `apikey`。 |
| `HomeAssistant` | `apikey`，可选 `items` 列表和 `separator`。 |
| `HyperHDR` | 无额外选项。 |
| `Immich` | `apikey`。 |
| `Jellyfin` | `apikey`。 |
| `Lidarr` | `apikey`。 |
| `Matrix` | 无额外选项。 |
| `Mealie` | `apikey`。 |
| `Medusa` | `apikey`。 |
| `Miniflux` | `apikey`，可选 `style: counter`。 |
| `Mylar` | `apikey`。 |
| `NetAlertx` | `apikey`。 |
| `Nextcloud` | 无额外选项。 |
| `Olivetin` | 无额外选项。 |
| `OpenHAB` | `apikey`，可选 `things: true`、`items: true`。 |
| `PaperlessNG` | `apikey`。 |
| `PeaNUT` | `device`。 |
| `PiAlert` | 无额外选项。 |
| `Ping` | `endpoint`、`method`、`timeout`、`successCodes`。 |
| `Portainer` | `apikey`，可选 `environments` 列表。 |
| `Prometheus` | 无额外选项。 |
| `Proxmox` | `node`，以及 `api_token` 或 `api_token_id`/`api_token_secret`。可选 `token_id`、`token_secret`、`warning_value`、`danger_value`、`hide`、`hide_decimals`。 |
| `Prowlarr` | `apikey`。 |
| `qBittorrent` | `username`、`password`。 |
| `Radarr` | `apikey`，可选 `legacyApi: true`。 |
| `Readarr` | `apikey`。 |
| `SABnzbd` | `apikey`。 |
| `Scrutiny` | 无额外选项。 |
| `Sonarr` | `apikey`，可选 `legacyApi: true`。 |
| `SpeedtestTracker` | `apikey`。 |
| `Tautulli` | `apikey`。 |
| `Tdarr` | `apikey`。 |
| `Traefik` | `basic_auth`，格式为 `username:password`。 |
| `TruenasScale` | `api_token`。 |
| `UptimeKuma` | `slug`，用于指定状态页。 |
| `Vaultwarden` | 无额外选项。 |
| `Wallabag` | 无额外选项。 |
| `WUD` | 无额外选项。 |

当前未实现的 Homer 智能卡片：`CopyToClipboard`、`Jellystat`、`Linkding`、`OctoPrint`/`Moonraker`、`OpenWeather`、`PiHole`、`Plex`、`rTorrent` 和 `Transmission`。

## 示例

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

`Ping` 默认使用 `HEAD`。如果服务不能正确处理 `HEAD`，请设置 `method: GET`。

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

也可以提供完整的 `api_token` 值，包括 `PVEAPIToken=` 前缀。

### Docker Socket Proxy

```yaml
- name: "Docker"
  type: "DockerSocketProxy"
  icon: "fab fa-docker"
  endpoint: "http://docker-socket-proxy:2375"
  url: "https://docker.example.com"
```

仅本地部署时，也可以使用 Unix socket：

```yaml
- name: "Docker"
  type: "DockerSocketProxy"
  socket: "/var/run/docker.sock"
```

挂载 Docker socket 会让 homer-go 获得对 Docker 主机的广泛访问权限。尽可能优先使用受限的 Docker Socket Proxy。
