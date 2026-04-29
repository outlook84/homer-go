# Tips and Tricks

## Dashboard Icons

Useful icon sources:

- <https://fontawesome.com/search>
- <https://selfh.st/icons/>
- <https://github.com/homarr-labs/dashboard-icons>

Put custom images in the data directory and reference them with bare relative paths:

```yaml
services:
  - name: "Daily"
    items:
      - name: "Calendar"
        logo: "icons/calendar.png"
        url: "https://calendar.example.com"
```

These paths are resolved from the configured data directory, not relative to the browser URL. Use `http://` or `https://` for remote images.

## YAML Anchors

YAML anchors are resolved by the YAML parser before homer-go reads the configuration. They are useful for repeated tag styles or common card options.

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

`tagDefaults` is not a homer-go feature by itself; it is just a place to store YAML anchors. Unknown top-level keys are ignored by the renderer.

## Multiple Pages

Create page files beside `config.yml`:

```text
config.yml
media.yml
infra.yml
```

Then add links:

```yaml
links:
  - name: "Media"
    icon: "fas fa-film"
    url: "#media"
  - name: "Infra"
    icon: "fas fa-server"
    url: "#infra"
```

## Remote Message Mapping

The message banner can pull JSON from another endpoint:

```yaml
message:
  url: "https://status.example.com/message.json"
  mapping:
    title: "headline"
    content: "body"
    style: "severity"
```

If the remote JSON is:

```json
{
  "headline": "Maintenance",
  "body": "Storage maintenance starts at 22:00.",
  "severity": "is-warning"
}
```

homer-go maps it to the message fields `title`, `content`, and `style`.

## HTML in Messages and Footer

`message.content` and `footer` can contain HTML:

```yaml
message:
  style: "is-info"
  content: '<a href="https://status.example.com">Status page</a>'

footer: '<p>Powered by homer-go.</p>'
```

Treat this as trusted configuration. Do not put untrusted remote HTML into your dashboard.

## Reverse Proxy Base Path

When serving homer-go under a sub-path, set `HOMER_GO_BASE_PATH` or `-base-path`:

```sh
homer-go -base-path /dashboard
```

Then proxy `/dashboard` to homer-go. Built-in assets, user assets, page links, and preference routes will include the base path.
