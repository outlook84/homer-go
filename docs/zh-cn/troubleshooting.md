# 故障排查

## 应用启动后没有加载我的配置

homer-go 从数据目录读取 `config.yml`。先确认实际数据目录：

```sh
homer-go -data /path/to/data
```

Docker 中请将目录挂载到 `/data`：

```sh
docker run -d \
  --name homer-go \
  -p 8732:8732 \
  -v /path/to/homer-go-data:/data \
  ghcr.io/outlook84/homer-go:latest
```

只有当挂载目录为空且可写时，容器才会写入示例 `config.yml`。

## 页面返回 Not Found

额外页面必须是数据目录中的 YAML 文件。页面名只能包含字母、数字、短横线和下划线。

```text
media.yml      -> /?page=media or #media
home-lab.yml   -> /?page=home-lab or #home-lab
```

## 智能卡片显示离线

智能卡片请求从 homer-go 服务器发出，而不是从浏览器发出。先检查：

- 运行 homer-go 的机器或容器能访问 `url` 或 `endpoint`。
- API token、用户名和密码有效。
- 目标服务路径与正在运行的服务版本匹配。
- 如果 homer-go 运行在 Docker 中，内部主机名和端口必须能从容器内访问。
- 单项 `headers` 可能覆盖了 `proxy.headers`。
- 如果服务用非 2xx 状态码表示成功，请将该状态码加入 `successCodes`。

对于 `Ping`，如果服务不能正确响应 `HEAD`，可以尝试 `method: "GET"`。

## 浏览器 CORS 错误

大多数 Homer 的 CORS 建议不适用于 homer-go 智能卡片，因为数据收集在服务端完成。如果看到 CORS 错误，通常来自浏览器直接加载的链接或自定义 HTML，而不是 homer-go 的智能卡片 collector。

## 反向代理后资源无法加载

将 base path 设置为公开访问的 URL 前缀：

```sh
homer-go -base-path /dashboard
```

或：

```sh
HOMER_GO_BASE_PATH=/dashboard homer-go
```

确保反向代理将 `/dashboard` 转发到 homer-go，且路径剥离方式不要和配置的 base path 冲突。

## 自定义图片或样式表无法加载

裸相对资源路径会从数据目录解析，而不是从浏览器 URL 解析：

```yaml
logo: "icons/home.png"
stylesheet:
  - "custom.css"
```

确认这些文件存在于 `config.yml` 所在目录下。内置文件应使用 `/assets/...`。

如果希望图片或样式表从远程 Web 服务器加载，请使用完整的 `http://` 或 `https://` URL。

## 某个 Homer 配置项没有效果

homer-go 有意忽略部分 Homer 客户端选项：

- `hotkey`
- `proxy.useCredentials`
- 单项 `useCredentials`
- 单项刷新间隔设置

不支持的智能卡片 `type` 会渲染为普通链接，但不会显示实时状态。
