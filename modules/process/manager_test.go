package process

import (
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestManager_Add(t *testing.T) {
	pm := Manager{Processes: make(map[int64]*Process)}

	pid := pm.Add("foo", exec.Command("foo"))
	assert.Equal(t, int64(1), pid, "expected to get pid 1 got %d", pid)

	pid = pm.Add("bar", exec.Command("bar"))
	assert.Equal(t, int64(2), pid, "expected to get pid 2 got %d", pid)
}

func TestManager_Remove(t *testing.T) {
	pm := Manager{Processes: make(map[int64]*Process)}

	pid1 := pm.Add("foo", exec.Command("foo"))
	assert.Equal(t, int64(1), pid1, "expected to get pid 1 got %d", pid1)

	pid2 := pm.Add("bar", exec.Command("bar"))
	assert.Equal(t, int64(2), pid2, "expected to get pid 2 got %d", pid2)

	pm.Remove(pid2)

	_, exists := pm.Processes[pid2]
	assert.False(t, exists, "PID %d is in the list but shouldn't", pid2)
}

func TestExecTimeoutNever(t *testing.T) {

	// TODO Investigate how to improve the time elapsed per round.
	maxLoops := 10
	for i := 1; i < maxLoops; i++ {
		_, stderr, err := GetManager().ExecTimeout(5*time.Second, "ExecTimeout", git.GitExecutable, "--version")
		if err != nil {
			t.Fatalf("git --version: %v(%s)", err, stderr)
		}
	}
}

func TestExecTimeoutAlways(t *testing.T) {

	maxLoops := 100
	for i := 1; i < maxLoops; i++ {
		_, stderr, err := GetManager().ExecTimeout(100*time.Microsecond, "ExecTimeout", "sleep", "5")
		// TODO Simplify logging and errors to get precise error type. E.g. checking "if err != context.DeadlineExceeded".
		if err == nil {
			t.Fatalf("sleep 5 secs: %v(%s)", err, stderr)
		}
	}
}
