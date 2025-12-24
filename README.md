# Caddy module for Lume

This is a module for Caddy to have a reverse proxy to Lume and LumeCMS.
Configure the Caddyfile to this:

```
example.com {
  reverse_proxy {
    dynamic lume {
      directory "/path/to/your/lumecms/site"
    }
  }
}
```

- In the first request, it starts Lume using the first available port running the command `deno task lume -s --port={port}`.
- After 2 hours of inactiviy, the process is closed.

## TODO

This project is W.I.P. I really appreciate any help if you are familiarized with
Go and Caddy. Specially if you know:

- A better way to wait until the Lume server started. For now, it waits 5
  seconds.
- Figure out a way to detect responses with the header `X-Lume-CMS: reload` to
  close the process and (restart again).

Code "inspired" by [cweagans/caddy_ondemand_upstreams](https://github.com/cweagans/caddy_ondemand_upstreams). Thanks Cameron Eagans!
