package integration_test

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/stretchr/testify/require"
)

func spawnProcess(t *testing.T, name string, args ...string) (io.ReadWriteCloser, *exec.Cmd) {
	cmd := exec.Command(name, args...)
	ptmx, err := pty.Start(cmd)
	require.NoError(t, err, "starting %s", name)
	return ptmx, cmd
}

func sendLines(t *testing.T, tty io.Writer, lines ...string) {
	for _, l := range lines {
		_, err := fmt.Fprintf(tty, "%s\n", l)
		require.NoError(t, err)
	}
}

func readUntil(t *testing.T, tty io.Reader, marker string, timeout time.Duration) string {
	r := bufio.NewReader(tty)
	deadline := time.Now().Add(timeout)
	var out string
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %q, got: %q", marker, out)
		}
		chunk, err := r.ReadString('\n')
		require.NoError(t, err)
		out += chunk
		if strings.Contains(out, marker) {
			return out
		}
	}
}

func Test_GameLog(t *testing.T) {
	// 1) Start the server
	serverTTY, serverCmd := spawnProcess(t, "go", "run", "./cmd/server/main.go")
	defer serverCmd.Process.Kill()
	// Wait for it to connect to RabbitMQ
	readUntil(t, serverTTY, "Connected to RabbitMQ", 5*time.Second)

	// 2) Start client #1
	cl1TTY, cl1Cmd := spawnProcess(t, "go", "run", "./cmd/client/main.go")
	defer cl1Cmd.Process.Kill()
	// Wait for the username prompt
	readUntil(t, cl1TTY, "Please enter your username:", 5*time.Second)
	sendLines(t, cl1TTY, "napoleon", "spawn europe cavalry")

	// 3) Start client #2
	cl2TTY, cl2Cmd := spawnProcess(t, "go", "run", "./cmd/client/main.go")
	defer cl2Cmd.Process.Kill()
	readUntil(t, cl2TTY, "Please enter your username:", 5*time.Second)
	sendLines(t, cl2TTY, "washington", "spawn americas infantry", "move europe 1")

	// 4) Observe the war message
	out := readUntil(t, cl2TTY, "has declared war on", 10*time.Second)
	t.Logf("cl2 output: %q", out)
	require.Contains(t, out, "washington has declared war on napoleon")

	srvout := readUntil(t, serverTTY, "Received Ack", 10*time.Second)
	t.Logf("server output: %q", srvout)
	require.Contains(t, srvout, "received game log")
}

func Test_spam(t *testing.T) {
	serverTTY, serverCmd := spawnProcess(t, "go", "run", "./cmd/server/main.go")
	defer serverCmd.Process.Kill()

	readUntil(t, serverTTY, "Connected to RabbitMQ", 5*time.Second)

	cl1TTY, cl1Cmd := spawnProcess(t, "go", "run", "./cmd/client/main.go")
	defer cl1Cmd.Process.Kill()

	readUntil(t, cl1TTY, "Please enter your username:", 5*time.Second)
	sendLines(t, cl1TTY, "napoleon", "spam 10")
	clout := readUntil(t, cl1TTY, "Spam was published succesfully", 10*time.Second)
	require.Contains(t, clout, "Spam was published succesfully")

	// Uncomment this part to see the server output (it will print every spam)
	//srvout := readUntil(t, serverTTY, "Last Message c9jsd", 10*time.Second)
	//require.Contains(t, srvout, "Last Message c9jsd")
	//t.Logf("server output: %q", srvout)
}
