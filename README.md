# Caddy module for Lume

This is a module for Caddy to have a reverse proxy to Lume and LumeCMS.
Configure the Caddyfile like this:

```
example.com {
  reverse_proxy {
    dynamic lume {
      directory "/path/to/your/lume/site"
    }

    lb_retries 10
    lb_try_interval 2s
  }
}
```

- In the first request, it starts Lume using the first available port running the command `deno task lume --serve --hostname=localhost --port={port} --location={public_url}`.
- After 2 hours of inactiviy, the process is closed.

Code "inspired" by
[cweagans/caddy_ondemand_upstreams](https://github.com/cweagans/caddy_ondemand_upstreams).
Thanks Cameron Eagans!
