package lume

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
)

type UpstreamProcess struct {
	cmd          *exec.Cmd
	directory    string
	location     string
	port         int
	idleTimeout  time.Duration
	lastActivity time.Time
	mu           sync.Mutex
	running      bool
}

func NewUpstreamProcess(directory string, location string, idle_timeout time.Duration) *UpstreamProcess {
	return &UpstreamProcess{
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

	packageManager := getPackageManager(u.directory)
	caddy.Log().Named(CHANNEL).Info("Install packages with " + packageManager)
	var cmd *exec.Cmd
	switch packageManager {
	case "pnpm":
		cmd = exec.Command("pnpm", "ci")
	case "yarn":
		cmd = exec.Command("yarn", "install", "--frozen-lockfile")
	default:
		cmd = exec.Command("npm", "ci")
	}
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
	switch packageManager {
	case "pnpm":
		u.cmd = exec.Command("pnpm", "lume", "build", "--serve", "--hostname", "localhost", fmt.Sprintf("--port=%d", u.port), fmt.Sprintf("--location=%s", u.location))
	case "yarn":
		u.cmd = exec.Command("yarn", "lume", "build", "--serve", "--hostname", "localhost", fmt.Sprintf("--port=%d", u.port), fmt.Sprintf("--location=%s", u.location))
	default:
		u.cmd = exec.Command("npx", "lume", "build", "--serve", "--hostname", "localhost", fmt.Sprintf("--port=%d", u.port), fmt.Sprintf("--location=%s", u.location))
	}
	u.cmd.Stdout = os.Stdout
	u.cmd.Stderr = os.Stderr
	u.cmd.Dir = u.directory
	u.cmd.Env = append(os.Environ(), "LUME_PROXIED=true", "LUME_LOGS=WARNING")

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

func getPackageManager(directory string) string {
	if fileExists(path.Join(directory, "pnpm-lock.yaml")) {
		return "pnpm"
	}
	if fileExists(path.Join(directory, "yarn.lock")) {
		return "yarn"
	}
	return "npm"
}

func fileExists(path string) bool {
	_, err := os.Stat(path);
	return err == nil
}
