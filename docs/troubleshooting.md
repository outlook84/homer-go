# Troubleshooting

## The App Starts Without My Configuration

homer-go reads `config.yml` from the data directory. Check the effective data directory:

```sh
homer-go -data /path/to/data
```

In Docker, mount the directory to `/data`:

```sh
docker run -d \
  --name homer-go \
  -p 8732:8732 \
  -v /path/to/homer-go-data:/data \
  ghcr.io/outlook84/homer-go:latest
```

The container writes an example `config.yml` only when the mounted directory is empty and writable.

## A Page Returns Not Found

Additional pages must be YAML files in the data directory. The page name may contain only letters, numbers, dashes, and underscores.

```text
media.yml      -> /?page=media or #media
home-lab.yml   -> /?page=home-lab or #home-lab
```

## A Smart Card Shows Offline

Smart-card requests are made from the homer-go server, not from the browser. Check these first:

- The `url` or `endpoint` is reachable from the machine or container running homer-go.
- API tokens, usernames, and passwords are valid.
- The target service path matches the service version you run.
- If homer-go runs in Docker, internal hostnames and ports must be reachable from inside the container.
- Per-item `headers` may be overriding `proxy.headers`.
- If the service returns a non-2xx success response, add that status code to `successCodes`.

For `Ping`, try `method: "GET"` if the service does not respond correctly to `HEAD`.

## Browser CORS Errors

Most Homer CORS guidance does not apply to homer-go smart cards because collection is server-side. If you see CORS errors, they are usually from links or custom HTML that your browser loads directly, not from homer-go's smart-card collectors.

## Assets Do Not Load Behind a Reverse Proxy

Set the base path to the public URL prefix:

```sh
homer-go -base-path /dashboard
```

or:

```sh
HOMER_GO_BASE_PATH=/dashboard homer-go
```

Make sure the reverse proxy forwards requests for `/dashboard` to homer-go without stripping paths in a way that conflicts with the configured base path.

## Custom Images or Stylesheets Do Not Load

Bare relative asset paths are resolved from the data directory, not from the browser URL:

```yaml
logo: "icons/home.png"
stylesheet:
  - "custom.css"
```

Confirm those files exist under the same directory as `config.yml`. Built-in files should use `/assets/...`.

If you want an image or stylesheet to load from a remote web server, use a full `http://` or `https://` URL.

## A Homer Configuration Key Does Nothing

homer-go intentionally ignores some Homer client-side options:

- `hotkey`
- `proxy.useCredentials`
- per-item `useCredentials`
- per-item refresh interval settings

Unsupported smart-card `type` values render as normal links without live status.
