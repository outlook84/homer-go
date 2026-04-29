# 技巧

## 仪表盘图标

常用图标来源：

- <https://fontawesome.com/search>
- <https://selfh.st/icons/>
- <https://github.com/homarr-labs/dashboard-icons>

将自定义图片放在数据目录中，并使用裸相对路径引用：

```yaml
services:
  - name: "Daily"
    items:
      - name: "Calendar"
        logo: "icons/calendar.png"
        url: "https://calendar.example.com"
```

这些路径会从配置的数据目录解析，而不是相对于浏览器 URL。远程图片请使用 `http://` 或 `https://`。

## YAML Anchors

YAML anchors 会在 homer-go 读取配置之前由 YAML 解析器处理。它适合用于重复的标签样式或通用卡片选项。

```yaml
tagDefaults:
  apps: &apps
    tag: "app"
    tagstyle: "is-info"

services:
  - name: "Apps"
    items:
      - name: "Code"
        <<: *apps
        icon: "fas fa-code"
        url: "https://code.example.com"
```

`tagDefaults` 本身不是 homer-go 功能；它只是一个存放 YAML anchors 的位置。渲染器会忽略未知顶层键。

## 多页面

在 `config.yml` 旁边创建页面文件：

```text
config.yml
media.yml
infra.yml
```

然后添加链接：

```yaml
links:
  - name: "Media"
    icon: "fas fa-film"
    url: "#media"
  - name: "Infra"
    icon: "fas fa-server"
    url: "#infra"
```

## 远程消息映射

消息横幅可以从其他端点拉取 JSON：

```yaml
message:
  url: "https://status.example.com/message.json"
  mapping:
    title: "headline"
    content: "body"
    style: "severity"
```

如果远程 JSON 为：

```json
{
  "headline": "Maintenance",
  "body": "Storage maintenance starts at 22:00.",
  "severity": "is-warning"
}
```

homer-go 会将其映射到 `title`、`content` 和 `style`。

## 在消息和页脚中使用 HTML

`message.content` 和 `footer` 可以包含 HTML：

```yaml
message:
  style: "is-info"
  content: '<a href="https://status.example.com">Status page</a>'

footer: '<p>Powered by homer-go.</p>'
```

请将其视为可信配置。不要把不可信远程 HTML 放进仪表盘。

## 反向代理基路径

当 homer-go 挂载在子路径下时，设置 `HOMER_GO_BASE_PATH` 或 `-base-path`：

```sh
homer-go -base-path /dashboard
```

然后将 `/dashboard` 代理到 homer-go。内置资源、用户资源、页面链接和偏好设置路由都会包含该基路径。
