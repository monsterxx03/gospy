package main

import (
	"os/exec"
	"runtime"
	"syscall"
	"testing"

	"gospy/pkg/proc"
)

type data struct {
	goversion string
}

var testdata = map[string]*data{
	"testdata/test_1_10":    &data{"1.10"},
	"testdata/test_1_10_8":  &data{"1.10.8"},
	"testdata/test_1_11_13": &data{"1.11.13"},
	"testdata/test_1_12_9":  &data{"1.12.9"},
	"testdata/test_1_13_1":  &data{"1.13.1"},
}

func assert(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Logf("%s != %s", a, b)
		t.Fail()
	}
}

func testgo(t *testing.T, f string, d *data) error {
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
	assert(t, sum.GoVersion, d.goversion)
	assert(t, sum.Gomaxprocs, runtime.NumCPU())

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
	for f, d := range testdata {
		if err := testgo(t, f, d); err != nil {
			t.Log(err)
			t.Fail()
		}
	}
}
