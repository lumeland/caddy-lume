package lume

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
)

type UpstreamProcess struct {
	cmd          *exec.Cmd
	deno         string
	directory    string
	location     string
	port         int
	idleTimeout  time.Duration
	lastActivity time.Time
	mu           sync.Mutex
	running      bool
}

func NewUpstreamProcess(deno string, directory string, location string, idle_timeout time.Duration) *UpstreamProcess {
	return &UpstreamProcess{
		deno:         deno,
		directory:    directory,
		location:     location,
		idleTimeout:  idle_timeout,
		lastActivity: time.Now(),
		running:      false,
	}
}

func (u *UpstreamProcess) GetDial() string {
	port := u.port
	return fmt.Sprintf("localhost:%d", port)
}

func (u *UpstreamProcess) IsRunning() bool {
	return u.cmd != nil && u.running == true
}

func (u *UpstreamProcess) LogActivity() {
	u.lastActivity = time.Now()
}

func (u *UpstreamProcess) Start() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.IsRunning() {
		return nil
	}

	// Run `deno install` to download the dependencies
	caddy.Log().Named(CHANNEL).Info("Run 'deno install' to download dependencies")
	cmd := exec.Command(u.deno, "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = u.directory
	cmd.Env = os.Environ()
	err := cmd.Start()
	if err != nil {
		return err
	}
	cmd.Wait()

	// Assign a port
	port, err := getAvailablePort()
	if err != nil {
		return err
	}
	caddy.Log().Named(CHANNEL).Info("Lume server assigned to port " + strconv.Itoa(port))
	u.port = port

	// Start the command
	u.cmd = exec.Command(
		u.deno,
		"task",
		"lume",
		"--serve",
		"--hostname=localhost",
		fmt.Sprintf("--port=%d", u.port),
		fmt.Sprintf("--location=%s", u.location),
	)
	u.cmd.Stdout = os.Stdout
	u.cmd.Stderr = os.Stderr
	u.cmd.Dir = u.directory
	u.cmd.Env = os.Environ()
	u.cmd.Env = append(cmd.Env, "LUME_PROXIED=true")
	u.cmd.Env = append(cmd.Env, "LUME_LOGS=WARNING")

	caddy.Log().Named(CHANNEL).Info("Start Lume for " + u.location)
	err = u.cmd.Start()
	if err != nil {
		return err
	}
	u.running = true
	u.LogActivity()

	// Wait to finish the process
	go func() {
		u.cmd.Wait()
		caddy.Log().Named(CHANNEL).Info("Lume process finished for " + u.location)
		u.running = false
	}()

	// Watch for idle timeout.
	go func() {
		for {
			wait := time.Until(u.lastActivity.Add(u.idleTimeout))
			time.Sleep(wait)
			if u.lastActivity.Add(u.idleTimeout).After(time.Now()) {
				continue
			}
			caddy.Log().Named(CHANNEL).Info("Idle timeout. Stop Lume process for " + u.location)
			u.Stop()
			break
		}
	}()

	return nil
}

func (u *UpstreamProcess) Stop() {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !u.IsRunning() {
		return
	}

	u.cmd.Process.Signal(os.Interrupt)
	time.Sleep(time.Duration(time.Second))

	if u.IsRunning() {
		u.cmd.Process.Kill()
	}
	u.cmd.Process.Release()
	u.running = false
	caddy.Log().Named(CHANNEL).Info("Stopped Lume process for " + u.location)
}

func getAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")

	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)

	if err != nil {
		return 0, err
	}

	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
