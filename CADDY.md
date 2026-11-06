# Caddy + fail2ban

## Caddy
Modern http proxy server with easy configuration.

Install log formatter plugin:
```sh
caddy add-package github.com/caddyserver/transform-encoder
```

Add to `/etc/caddy/Caddyfile`:
```Caddyfile
files.example.com { # your url
    request_body {
        max_size 4GB
    }
    reverse_proxy 127.0.0.1:8080 { # serveg address
        transport http {
            versions 1.1
        }
    }
    # log response status and request authorization for fail2ban
    log {
		output file /var/log/caddy/files-auth.log
		format transform "{ts} {request>client_ip} {status} {request>headers>Authorization>[0]}" {
			time_format iso8601
			placeholder -
		}
	}
}
```

Restart caddy:
```sh
sudo systemctl restart caddy
```

### Fail2ban
Service that reads logs from various sources, finds password trials and bans ip if it tries to brute force password.

Copy [delpoy/caddy-auth.conf](./delpoy/caddy-auth.conf) to `/etc/fail2ban/filter.d/caddy-auth.conf`
Copy [delpoy/caddy-auth.local](./delpoy/caddy-auth.local) to `/etc/fail2ban/jail.d/caddy-auth.local`

Reload fail2ban:
```sh
sudo fail2ban-client reload
```

Test fail2ban:
```sh
curl -s -o /dev/null -u wrong:wrong https://files.example.com/
curl -s -o /dev/null https://files.example.com/
fail2ban-regex /var/log/caddy/files-auth.log /etc/fail2ban/filter.d/caddy-auth.conf
```
