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
	cmd              *exec.Cmd
	directory        string
	command          string
	port             int
	startupDelay     time.Duration
	terminationDelay time.Duration
	idleTimeout      time.Duration
	lastActivity     time.Time
	mu               sync.Mutex
}

func NewUpstreamProcess(directory string, task string) *UpstreamProcess {
	return &UpstreamProcess{
		directory:        directory,
		startupDelay:     time.Duration(time.Second * 5),
		terminationDelay: time.Duration(time.Second * 2),
		idleTimeout:      time.Duration(time.Hour * 2),
		lastActivity:     time.Now(),
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

	// Run `deno install` to download the dependencies
	cmd := exec.Command("sh", "-c", "deno install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = u.directory
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
	u.port = port

	// Start the command
	u.cmd = u.createCommand()
	err = u.cmd.Start()
	if err != nil {
		return err
	}
	time.Sleep(u.startupDelay)
	u.LogActivity()

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
	time.Sleep(u.terminationDelay)

	if u.IsRunning() {
		u.cmd.Process.Kill()
	}
	u.cmd = nil
}

func (u *UpstreamProcess) createCommand() *exec.Cmd {
	command := fmt.Sprintf("deno task lume -s --port=%d", u.port)
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = u.directory
	return cmd
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
