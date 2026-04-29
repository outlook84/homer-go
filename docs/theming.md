# Theming

homer-go ships with embedded CSS, Font Awesome, Bulma, and a few dashboard themes. You can select a theme in `config.yml` and add your own stylesheet from the data directory.

```yaml
theme: default
defaults:
  colorTheme: auto
```

The user can switch between `auto`, `light`, and `dark` from the navigation bar. The preference is stored in a browser cookie.

## Built-In Themes

Built-in theme names include:

| Theme | Description |
| --- | --- |
| `default` | The standard homer-go theme. |
| `walkxcode` | Theme with the bundled Walkxcode wallpaper assets. |
| `neon` | High-contrast neon theme. |

## Color Overrides

Colors can be overridden in YAML for light and dark modes:

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

Supported keys map to CSS custom properties with the same names prefixed by `--`.

## Custom Stylesheets

Place custom CSS in the data directory and reference it with `stylesheet`:

```yaml
stylesheet:
  - "custom.css"
```

Bare relative stylesheet paths are resolved from the data directory, not relative to the browser URL. `http://` and `https://` stylesheet values remain remote URLs.

Multiple files are supported and are loaded after the built-in CSS:

```yaml
stylesheet:
  - "icons.css"
  - "overrides.css"
```

Use this for custom icons, small visual overrides, or service-specific card classes.

## Logos and Icons

Use Font Awesome classes with `icon`:

```yaml
icon: "fas fa-server"
```

Use image paths with `logo`:

```yaml
logo: "icons/nas.png"
```

Bare relative logo paths are resolved from the data directory.

For built-in assets, use `/assets/...`:

```yaml
logo: "/assets/icons/homer-go-logo-v2.png"
```

For group headings, `icon` is rendered before `logo` when both are set. For service cards, `logo` is rendered before `icon`.
