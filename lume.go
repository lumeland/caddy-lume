package lume

import (
	"fmt"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
)

func init() {
	caddy.RegisterModule(Lume{})
}

type Lume struct {
	Directory string `json:"directory,omitempty"`
	Task      string `json:"task,omitempty"`
	process   *UpstreamProcess
}

// CaddyModule returns the Caddy module information.
func (Lume) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.reverse_proxy.upstreams.lume",
		New: func() caddy.Module { return new(Lume) },
	}
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (lume *Lume) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	d.Next()

	for d.NextBlock(0) {
		switch d.Val() {
		case "directory":
			if !d.NextArg() {
				return d.ArgErr()
			}
			lume.Directory = d.Val()
		case "task":
			if !d.NextArg() {
				return d.ArgErr()
			}
			lume.Task = d.Val()
		default:
			return d.Errf("Unknown subdirective: %s", d.Val())
		}
	}

	return nil
}

// Validate implements caddy.Validator.
func (lume *Lume) Validate() error {
	if lume.Directory == "" {
		return fmt.Errorf("directory is required")
	}

	if lume.Task == "" {
		lume.Task = "serve"
	}

	return nil
}

// Provision implements caddy.Provisioner.
func (lume *Lume) Provision(ctx caddy.Context) error {
	return nil
}

// GetUpstreams implements reverseproxy.UpstreamSource.
func (lume *Lume) GetUpstreams(r *http.Request) ([]*reverseproxy.Upstream, error) {
	if lume.process == nil {
		lume.process = NewUpstreamProcess(lume.Directory, lume.Task)
	}
	lume.process.Start()

	if lume.process.IsRunning() {
		lume.process.LogActivity()

		return []*reverseproxy.Upstream{
			{
				Dial: lume.process.GetDial(),
			},
		}, nil
	}

	return nil, fmt.Errorf("no upstream available")
}

// Cleanup implements caddy.CleanerUpper.
func (lume *Lume) Cleanup() error {
	if lume.process != nil && lume.process.IsRunning() {
		lume.process.Stop()
	}

	return nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*Lume)(nil)
	_ caddy.Validator             = (*Lume)(nil)
	_ caddyfile.Unmarshaler       = (*Lume)(nil)
	_ reverseproxy.UpstreamSource = (*Lume)(nil)
)
