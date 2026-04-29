# 配置

homer-go 从数据目录读取名为 `config.yml` 的 YAML 配置文件。直接运行二进制时，数据目录默认为当前工作目录；Docker 中默认为 `/data`。

如果 `config.yml` 不存在且数据目录可写，homer-go 会在首次启动时写入示例配置。

homer-go 基本兼容 Homer 的 YAML 格式，但本地路径处理有意不同。`http://` 和 `https://` 仍然表示远程 URL。`icons/home.png` 或 `custom.css` 这类裸相对路径会作为数据目录下的文件解析，而不是相对于浏览器 URL 解析。

```sh
homer-go -addr :8732 -data /path/to/data -base-path /dashboard
```

同样可以使用环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `HOMER_GO_ADDR` | `:8732` | 监听地址。 |
| `HOMER_GO_DATA_DIR` | `.` | 包含 `config.yml` 和页面文件的数据目录。 |
| `HOMER_GO_ASSETS_DIR` | `assets` | 可选的外部资源目录。默认使用嵌入资源。 |
| `HOMER_GO_BASE_PATH` | 空 | 通过反向代理挂载到子路径时使用，例如 `/dashboard`。 |

## 最小示例

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

## 顶层配置项

| 键 | 说明 |
| --- | --- |
| `title` | 仪表盘主标题。 |
| `subtitle` | 头部副标题。 |
| `documentTitle` | 浏览器标签页标题。省略时，homer-go 使用 `title` 和 `subtitle`。 |
| `logo` | 头部 Logo 路径。内置资源使用 `/assets/...`，数据目录文件使用裸相对路径。 |
| `icon` | 头部 Font Awesome 图标。如果同时设置 `logo` 和 `icon`，两者都会渲染。 |
| `header` | 是否显示头部。默认 `true`。 |
| `footer` | 页脚 HTML。设为空字符串可隐藏内容。 |
| `columns` | 卡片列数：`auto`、`1`、`2`、`3`、`4`、`6` 或 `12`。 |
| `connectivityCheck` | 启用浏览器端连通性刷新行为。默认 `true`。 |
| `defaults.layout` | 初始布局偏好，`columns` 或 `list`。 |
| `defaults.colorTheme` | 初始颜色偏好，`auto`、`light` 或 `dark`。 |
| `theme` | CSS 主题名。内置值包括 `default`、`walkxcode` 和 `neon`。 |
| `colors` | 可选的主题颜色覆盖。见[主题](./theming.md)。 |
| `stylesheet` | 一个样式表路径或路径列表。 |
| `message` | 可选的消息横幅配置。 |
| `links` | 导航链接。 |
| `services` | 服务分组和服务卡片。 |
| `proxy.headers` | 智能卡片 HTTP 请求使用的全局请求头，除非被单项覆盖。 |
| `updateIntervalMs` | 服务状态片段的全局自动刷新间隔。设为 `0` 或省略表示禁用。 |
| `externalConfig` | 用另一个 YAML 文件或 `http(s)` URL 替换当前 YAML 文件。本地路径必须位于数据目录内。 |

Homer 配置项 `hotkey`、`proxy.useCredentials`、单项 `useCredentials` 和单项刷新间隔选项可以存在，但 homer-go 目前不会使用它们。

## 导航链接

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

使用 `#page-name` 链接到同一数据目录下的另一个页面文件。例如，`#media` 会加载 `media.yml`。

## 服务

`services` 的第一层是分组。每个分组包含服务 `items`。

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

分组选项：

| 键 | 说明 |
| --- | --- |
| `name` | 分组标题。 |
| `icon` | Font Awesome 图标。分组标题中同时设置 `icon` 和 `logo` 时，优先渲染 `icon`。 |
| `logo` | 图片路径。分组标题中未设置 `icon` 时使用。 |
| `class` | 添加到分组的 CSS class。 |
| `tagstyle` | 该分组中卡片继承的默认标签样式。 |
| `items` | 分组中的服务卡片。 |

服务项选项：

| 键 | 说明 |
| --- | --- |
| `name` | 卡片标题。 |
| `type` | 智能卡片 collector 类型。省略或设为 `Generic` 表示普通链接卡片。 |
| `logo` | 卡片图片路径。服务卡片中同时设置 `logo` 和 `icon` 时，优先渲染 `logo`。 |
| `icon` | Font Awesome 图标。 |
| `subtitle` | 静态副标题。省略时，支持的智能卡片可使用这一行显示实时状态。 |
| `tag` | 标签文本。 |
| `keywords` | 搜索使用的额外文本。 |
| `url` | 卡片链接，也是默认的智能卡片 API 基础 URL。 |
| `endpoint` | 当可点击链接和 API 地址不同时，可用它指定 API 基础 URL。 |
| `target` | 链接目标，例如 `_blank`。 |
| `tagstyle` | Bulma 标签修饰类，例如 `is-success`。 |
| `class` | 添加到卡片的 CSS class。 |
| `background` | 卡片背景色值。 |
| `headers` | 单项 HTTP 请求头。会覆盖 `proxy.headers`。 |
| `quick` | 渲染在卡片上的快速链接。 |

快速链接选项：

```yaml
quick:
  - name: "Admin"
    icon: "fas fa-lock"
    url: "https://service.example.com/admin"
    target: "_blank"
    color: "#e11d48"
```

## 页面

额外页面是数据目录中的独立 YAML 文件：

```text
config.yml
media.yml
infra.yml
```

通过 `/?page=media` 打开，或用 `#media` 链接。页面名只能包含字母、数字、短横线和下划线。

页面文件会替换 `config.yml` 中的顶层键。例如，`media.yml` 定义了 `services` 时，该页面会使用自己的 `services` 列表。

## 消息横幅

```yaml
message:
  style: "is-warning"
  title: "Maintenance"
  icon: "fas fa-triangle-exclamation"
  content: "NAS maintenance tonight."
```

也支持远程消息 JSON。首次页面渲染会显示本地 message 值，随后浏览器从 `message.url` 刷新消息片段。

```yaml
message:
  url: "https://status.example.com/message.json"
  mapping:
    title: "headline"
    content: "body"
    style: "level"
    icon: "icon"
```

远程响应可以提供 `title`、`style`、`content` 和 `icon`。缺失或空的远程值会保留本地值。

## 本地资源和路径

数据目录中的文件可以用裸相对路径引用：

```yaml
logo: "icons/home.png"
stylesheet:
  - "custom.css"
```

这些值指向数据目录下的文件。它们不会按当前页面 URL 解析，这一点不同于直接从 URL 路径提供静态资源的 Homer 部署。

远程 URL 仍然按 URL 使用：

```yaml
logo: "https://example.com/icons/home.png"
```

内置资源仍然位于 `/assets/`，例如 `/assets/icons/homer-go-logo-v2.png`。

YAML 文件不会通过用户资源路由提供。
