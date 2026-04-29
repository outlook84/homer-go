# 主题

homer-go 自带嵌入式 CSS、Font Awesome、Bulma 和几个仪表盘主题。可以在 `config.yml` 中选择主题，并从数据目录加载自定义样式表。

```yaml
theme: default
defaults:
  colorTheme: auto
```

用户可以在导航栏中切换 `auto`、`light` 和 `dark`。偏好会保存在浏览器 cookie 中。

## 内置主题

内置主题名包括：

| 主题 | 说明 |
| --- | --- |
| `default` | 标准 homer-go 主题。 |
| `walkxcode` | 使用内置 Walkxcode 壁纸资源的主题。 |
| `neon` | 高对比霓虹主题。 |

## 颜色覆盖

可以在 YAML 中分别覆盖浅色和深色模式的颜色：

```yaml
colors:
  light:
    highlight-primary: "#3367d6"
    highlight-secondary: "#2f855a"
    background: "#f7fafc"
    card-background: "#ffffff"
    text: "#1f2937"
    text-header: "#ffffff"
    text-title: "#111827"
    text-subtitle: "#4b5563"
    link: "#2563eb"
    link-hover: "#1d4ed8"
  dark:
    highlight-primary: "#60a5fa"
    highlight-secondary: "#34d399"
    background: "#111827"
    card-background: "#1f2937"
    text: "#e5e7eb"
    text-header: "#ffffff"
    text-title: "#ffffff"
    text-subtitle: "#d1d5db"
    link: "#93c5fd"
    link-hover: "#bfdbfe"
```

支持的键会映射到加上 `--` 前缀的同名 CSS 自定义属性。

## 自定义样式表

将自定义 CSS 放在数据目录中，并通过 `stylesheet` 引用：

```yaml
stylesheet:
  - "custom.css"
```

裸相对样式表路径会从数据目录解析，而不是相对于浏览器 URL。`http://` 和 `https://` 样式表值仍然按远程 URL 处理。

支持多个文件，且它们会在内置 CSS 之后加载：

```yaml
stylesheet:
  - "icons.css"
  - "overrides.css"
```

这适合用于自定义图标、小范围视觉覆盖或服务卡片专用 class。

## Logo 和图标

使用 `icon` 配置 Font Awesome class：

```yaml
icon: "fas fa-server"
```

使用 `logo` 配置图片路径：

```yaml
logo: "icons/nas.png"
```

裸相对 Logo 路径会从数据目录解析。

内置资源使用 `/assets/...`：

```yaml
logo: "/assets/icons/homer-go-logo-v2.png"
```

分组标题中同时设置 `icon` 和 `logo` 时，优先渲染 `icon`。服务卡片中同时设置二者时，优先渲染 `logo`。
