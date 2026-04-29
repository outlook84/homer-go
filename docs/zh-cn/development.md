# 开发

homer-go 是一个 Go 服务，使用服务端渲染 HTML、嵌入式资源和服务端智能卡片 collector。它不是 Vue 应用，因此 Homer 的 Vue component 和 SCSS 开发说明不适用于这里。

## 项目结构

| 路径 | 用途 |
| --- | --- |
| `main.go` | 服务初始化、路由、collector 注册、嵌入资源。 |
| `internal/config` | YAML 加载、配置归一化、页面覆盖、消息解析。 |
| `internal/collectors` | 智能卡片 collector 和 HTTP 辅助函数。 |
| `internal/views` | HTML 渲染辅助函数。 |
| `assets` | 嵌入式 CSS、JS、图标、主题、vendor 资源和示例配置。 |
| `docs` | 用户文档和截图。 |

## 本地运行

```sh
go run . -addr :8732 -data ./tmp
```

请先创建数据目录。如果 `./tmp/config.yml` 不存在，homer-go 可以从嵌入的示例配置初始化它。

## 测试

```sh
go test ./...
```

## 添加智能卡片

1. 在 `internal/collectors` 中添加 collector。
2. 实现：

```go
type MyService struct{}

func (MyService) Type() string { return "MyService" }

func (MyService) Collect(ctx context.Context, item config.Item, proxy config.Proxy) Status {
    // 收集数据并返回 Status。
}
```

3. 在 `main.go` 中注册它。
4. 在 `internal/collectors/collectors_test.go` 或专门的测试文件中添加测试。
5. 在[智能卡片文档](./smart-cards.md)中记录新类型。

为保持一致，优先使用 `collectJSON`、`requestOptions`、`stringField` 和 `effectiveHeaders`。

## 配置兼容性

解析器会保留原始 YAML map：`config.Config.Raw`、group `Raw` 和 item `Raw`。这样 collector 可以读取服务特定键，而无需把每个选项都加入共享的 `Item` struct。

在合理时，优先保留 Homer 兼容的键名，例如 `apikey`、`endpoint`、`username` 和 `password`。
