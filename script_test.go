package main

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/rogpeppe/go-internal/goproxytest"
	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

var (
	proxyURL string
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(gobinMain{m}, map[string]func() int{
		"gobin": main1,
	}))
}

type gobinMain struct {
	m *testing.M
}

func (m gobinMain) Run() int {
	// Start the Go proxy server running for all tests.
	srv, err := goproxytest.NewServer("testdata/mod", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot start proxy: %v", err)
		return 1
	}
	proxyURL = srv.URL

	return m.m.Run()
}

func TestScripts(t *testing.T) {
	var (
		pathToMod     string // local path to this module
		modTestGOPATH string // GOPATH set when running tests in this module
	)

	// set pathToMod
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory for test: %v", err)
	}
	pathToMod = wd

	// set modTestGOPATH
	cmd := exec.Command("go", "env", "GOPATH")
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if err, ok := err.(*exec.ExitError); ok {
			stderr = err.Stderr
		}
		t.Fatalf("failed to get GOPATH for test: %v\n%s", err, stderr)
	}
	modTestGOPATH = string(out)

	p := testscript.Params{
		Dir: "testdata",
		Setup: func(e *testscript.Env) error {
			e.Vars = append(e.Vars,
				"TESTGOPATH="+modTestGOPATH,
				"GOBINMODPATH="+pathToMod,
				"GOPROXY="+proxyURL,
			)
			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}
