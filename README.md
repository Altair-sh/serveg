# serveg
Small HTTP/1.1 file server. Meant to run behind a reverse proxy that handles TLS (see [CADDY.md](./CADDY.md)).

## Features:
- upload and download files
- per-directory passwords (read more in example config)
- generates html pages for directories
- own http implementation, not "net/http"

## Dependencies
- [BurntSushi/toml](https://github.com/BurntSushi/toml) - toml 1.1.0 parser
- [daviddengcn/go-colortext](https://github.com/daviddengcn/go-colortext) - print colored text (linux/windows)

## Build
```sh
make build
```

## Run

```sh
cp serveg.example.toml serveg.toml # edit shares and passwords
bin/servegp         # reads ./serveg.toml
# or
bin/servegp -config /etc/serveg/serveg.toml
```

## HTTP API

| Request | Result |
|---|---|
| `GET /share/dir/` | directory listing with an upload form if the share has `write_password` |
| `GET /share/file` | download file, supports `If-Modified-Since`, `Range` (single range) and `If-Range` (date) |
| `PUT /share/file` | upload: 201 created, 200 replaced |
| `OPTIONS /share/file` with `X-Upload-Size: <size>` | upload check, same as PUT but without body |

Passwords use HTTP Basic auth, the user name is ignored:

```sh
curl -O http://localhost:8080/public/file.zip
curl -C - -O http://localhost:8080/public/file.zip # continue interrupted download
curl -u :password -T file.zip http://localhost:8080/public/file.zip
```

Error responses explain the reason in a plain text body.

