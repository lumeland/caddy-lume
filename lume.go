package lume

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
)

const CHANNEL = "lume"

func init() {
	caddy.RegisterModule(Lume{})
}

type Lume struct {
	// Required. The working directory of the Lume site
	Directory string `json:"directory,omitempty"`

	// Optional. The deno binary path.
	// If this is not set, search for deno in the PATH
	Deno string `json:"deno,omitempty"`

	// Optional. The duration that the process should continue running if no traffic is received
	// By default is 2h
	IdleTimeout caddy.Duration `json:"idle_timeout,omitempty"`

	// The managed upstream process.
	process *UpstreamProcess
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
		case "deno":
			if !d.NextArg() {
				return d.ArgErr()
			}
			lume.Deno = d.Val()
		case "idle_timeout":
			if !d.NextArg() {
				return d.ArgErr()
			}
			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return d.Errf("Invalid duration: %v", err)
			}
			lume.IdleTimeout = caddy.Duration(dur)
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

	if lume.Deno == "" {
		deno, err := exec.LookPath("deno")

		if err != nil {
			return err
		}
		lume.Deno = deno
	}

	if lume.IdleTimeout == caddy.Duration(0) {
		lume.IdleTimeout = caddy.Duration(time.Hour * 2)
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
		location := fmt.Sprintf("%s://%s", r.Header.Get("X-Forwarded-Proto"), r.Header.Get("X-Forwarded-Host"))
		caddy.Log().Named(CHANNEL).Info("New Lume process for " + location)
		lume.process = NewUpstreamProcess(lume.Deno, lume.Directory, location, time.Duration(lume.IdleTimeout))
	}

	err := lume.process.Start()

	if err != nil {
		return nil, err
	}

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
