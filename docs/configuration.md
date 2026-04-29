# Configuration

homer-go reads a YAML configuration file named `config.yml` from the data directory. When running the binary directly, the data directory defaults to the current working directory. In Docker, it defaults to `/data`.

If `config.yml` is missing and the data directory is writable, homer-go writes an example configuration on first start.

homer-go is mostly compatible with Homer YAML, but local path handling is intentionally different. `http://` and `https://` values remain remote URLs. Bare relative paths such as `icons/home.png` or `custom.css` are resolved as files under the configured data directory, not relative to the browser URL.

```sh
homer-go -addr :8732 -data /path/to/data -base-path /dashboard
```

The same settings can be provided with environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `HOMER_GO_ADDR` | `:8732` | Listen address. |
| `HOMER_GO_DATA_DIR` | `.` | Directory containing `config.yml` and page files. |
| `HOMER_GO_ASSETS_DIR` | `assets` | Optional external asset directory. Embedded assets are used by default. |
| `HOMER_GO_BASE_PATH` | empty | URL prefix when served behind a reverse proxy, for example `/dashboard`. |

## Minimal Example

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
```

## Top-Level Options

| Key | Description |
| --- | --- |
| `title` | Main dashboard title. |
| `subtitle` | Header subtitle. |
| `documentTitle` | Browser tab title. When omitted, homer-go uses `title` and `subtitle`. |
| `logo` | Header logo path. Use `/assets/...` for built-in assets or a relative path for data-directory files. |
| `icon` | Font Awesome icon for the header. If both `logo` and `icon` are set, both are rendered. |
| `header` | Show or hide the header. Defaults to `true`. |
| `footer` | Footer HTML. Set to an empty string to hide content. |
| `columns` | Card columns: `auto`, `1`, `2`, `3`, `4`, `6`, or `12`. |
| `connectivityCheck` | Enables browser-side connectivity refresh behavior. Defaults to `true`. |
| `defaults.layout` | Initial layout preference, `columns` or `list`. |
| `defaults.colorTheme` | Initial color preference, `auto`, `light`, or `dark`. |
| `theme` | CSS theme name. Built-in values include `default`, `walkxcode`, and `neon`. |
| `colors` | Optional per-theme color overrides. See [theming](./theming.md). |
| `stylesheet` | One stylesheet path or a list of stylesheet paths. |
| `message` | Optional banner configuration. |
| `links` | Navigation links. |
| `services` | Service groups and service items. |
| `proxy.headers` | Headers applied to smart-card HTTP requests unless overridden on an item. |
| `updateIntervalMs` | Global auto-refresh interval for service status fragments. Set to `0` or omit to disable. |
| `externalConfig` | Replace the current YAML file with another YAML file or `http(s)` URL. Local paths must stay inside the data directory. |

The Homer keys `hotkey`, `proxy.useCredentials`, per-item `useCredentials`, and per-item refresh interval options are accepted if present but are not currently used by homer-go.

## Links

```yaml
links:
  - name: "GitHub"
    icon: "fab fa-github"
    url: "https://github.com/outlook84/homer-go"
    target: "_blank"
  - name: "Media"
    icon: "fas fa-film"
    url: "#media"
```

Use `#page-name` to link to another page file in the same data directory. For example, `#media` loads `media.yml`.

## Services

The first level of `services` is a group. Each group contains service `items`.

```yaml
services:
  - name: "Infrastructure"
    icon: "fas fa-network-wired"
    tagstyle: "is-info"
    items:
      - name: "Router"
        icon: "fas fa-wifi"
        subtitle: "Gateway"
        tag: "network"
        tagstyle: "is-success"
        url: "https://router.example.com"
        target: "_blank"
        keywords: "gateway lan wifi"
```

Group options:

| Key | Description |
| --- | --- |
| `name` | Group title. |
| `icon` | Font Awesome icon. For group headings, `icon` is rendered before `logo` when both are set. |
| `logo` | Image path. Used for group headings when `icon` is not set. |
| `class` | CSS class added to the group. |
| `tagstyle` | Default tag style inherited by cards in this group. |
| `items` | Service cards in the group. |

Item options:

| Key | Description |
| --- | --- |
| `name` | Card title. |
| `type` | Smart-card collector type. Omit or set `Generic` for a normal link card. |
| `logo` | Card image path. For service cards, `logo` is rendered before `icon` when both are set. |
| `icon` | Font Awesome icon. |
| `subtitle` | Static subtitle. If omitted, supported smart cards can use this line for live status. |
| `tag` | Tag text. |
| `keywords` | Extra text used by search. |
| `url` | Card link and default smart-card API base URL. |
| `endpoint` | Optional API base URL used by smart cards when it differs from `url`. |
| `target` | Link target, such as `_blank`. |
| `tagstyle` | Bulma tag modifier classes, such as `is-success`. |
| `class` | CSS class added to the card. |
| `background` | Inline card background value. |
| `headers` | Per-item HTTP headers for smart-card requests. Overrides `proxy.headers`. |
| `quick` | Quick links rendered on the card. |

Quick link options:

```yaml
quick:
  - name: "Admin"
    icon: "fas fa-lock"
    url: "https://service.example.com/admin"
    target: "_blank"
    color: "#e11d48"
```

## Pages

Additional pages are separate YAML files in the data directory:

```text
config.yml
media.yml
infra.yml
```

Open them with `/?page=media`, or link to them with `#media`. Page names may contain letters, numbers, dashes, and underscores.

Page files replace top-level keys from `config.yml`. For example, if `media.yml` defines `services`, it replaces the base `services` list for that page.

## Message Banner

```yaml
message:
  style: "is-warning"
  title: "Maintenance"
  icon: "fas fa-triangle-exclamation"
  content: "NAS maintenance tonight."
```

Remote message JSON is also supported. The first page render shows the local message values, then the browser refreshes the message fragment from `message.url`.

```yaml
message:
  url: "https://status.example.com/message.json"
  mapping:
    title: "headline"
    content: "body"
    style: "level"
    icon: "icon"
```

The remote response can provide `title`, `style`, `content`, and `icon`. Missing or empty remote values leave the local values in place.

## Local Assets and Paths

Files in the data directory can be referenced with bare relative paths:

```yaml
logo: "icons/home.png"
stylesheet:
  - "custom.css"
```

Those values point to files under the data directory. They are not resolved relative to the current page URL, which differs from Homer deployments that serve assets directly from a URL path.

Remote URLs still work as URLs:

```yaml
logo: "https://example.com/icons/home.png"
```

Built-in assets remain available under `/assets/`, for example `/assets/icons/homer-go-logo-v2.png`.

YAML files are not served through the user-asset route.
