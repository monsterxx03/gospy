package main

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"gospy/pkg/proc"
)

type data struct {
	goversion string
}

const testbin = "testdata/test_bin" // created by github actions

func assert(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Logf("%s != %s", a, b)
		t.Fail()
	}
}

func testgo(t *testing.T, f string) error {
	done := make(chan int)
	errCh := make(chan error)

	cmd := exec.Command(f)
	err := cmd.Start()
	if err != nil {
		return err
	}
	go func() {
		err := cmd.Wait()
		status := cmd.ProcessState.Sys().(syscall.WaitStatus)
		if !status.Signaled() && err != nil {
			t.Log("Failed on wait", err)
			errCh <- err
		} else {
			done <- 1
		}
	}()
	p, err := proc.New(cmd.Process.Pid, "")
	if err != nil {
		return err
	}
	sum, err := p.Summary(false)
	if err != nil {
		return err
	}
	// env var set in github actions
	if !strings.HasPrefix(sum.GoVersion, os.Getenv("E2E_GO_VERSION")) {
		t.Fatalf("remote process built with go%s, but parsed %s", os.Getenv("E2E_GO_VERSION"), sum.GoVersion)
	}
	if err := p.DumpHeap(false); err != nil {
		t.Fatal("Failed to dump heap:", err)
	}

	if err := cmd.Process.Kill(); err != nil {
		return err
	}

	select {
	case err := <-errCh:
		return err
	case <-done:
	default:
	}
	return nil
}

func TestCompatibility(t *testing.T) {
	if err := testgo(t, testbin); err != nil {
		t.Log(err)
		t.Fail()
	}
}
