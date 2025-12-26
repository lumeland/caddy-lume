package lume

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

type UpstreamProcess struct {
	cmd          *exec.Cmd
	directory    string
	location     string
	port         int
	idleTimeout  time.Duration
	lastActivity time.Time
	mu           sync.Mutex
}

func NewUpstreamProcess(directory string, location string) *UpstreamProcess {
	return &UpstreamProcess{
		directory:    directory,
		location:     location,
		idleTimeout:  time.Duration(time.Hour * 2),
		lastActivity: time.Now(),
	}
}

func (u *UpstreamProcess) GetDial() string {
	port := u.port
	return fmt.Sprintf("localhost:%d", port)
}

func (u *UpstreamProcess) IsRunning() bool {
	return u.cmd != nil
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

	deno, err := exec.LookPath("deno")

	if err != nil {
		return err
	}

	// Run `deno install` to download the dependencies
	cmd := exec.Command(deno, "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = u.directory
	cmd.Env = os.Environ()
	err = cmd.Start()
	if err != nil {
		return err
	}
	cmd.Wait()

	// Assign a port
	port, err := getAvailablePort()
	if err != nil {
		return err
	}
	u.port = port

	// Start the command
	u.cmd = exec.Command(
		deno,
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

	err = u.cmd.Start()
	if err != nil {
		return err
	}
	u.LogActivity()

	// Wait to finish the process
	go func() {
		u.cmd.Wait()
		u.cmd = nil
	}()

	// Watch for idle timeout.
	go func() {
		for {
			time.Sleep(time.Second)

			if u.lastActivity.Add(u.idleTimeout).After(time.Now()) {
				continue
			}

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
		u.cmd.Process.Release()
		u.cmd.Process.Kill()
	}
	u.cmd = nil
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
