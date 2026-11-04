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

