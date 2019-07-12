package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
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

func TestExitCode(t *testing.T) {
	var err error
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to determine os.Executable: %v", err)
	}

	temp, err := ioutil.TempDir("", "gobin-exitcode-test")
	if err != nil {
		t.Fatalf("failed to create temp directory for home: %v", err)
	}
	defer os.RemoveAll(temp)

	cmd := exec.Command(self, "-run", "example.com/fail")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, homeEnv(temp)...)
	cmd.Env = append(cmd.Env,
		"GONOSUMDB=*",
		"GOPROXY="+proxyURL,
		"TESTSCRIPT_COMMAND=gobin",
	)

	err = cmd.Run()
	if err == nil {
		t.Fatalf("unexpected success")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError; got %T: %v", err, err)
	}
	want := 42
	if got := ExitCode(ee.ProcessState); want != got {
		t.Fatalf("expected exit code %v; got %v", want, got)
	}
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
	modTestGOPATH = strings.TrimSpace(string(out))

	ucd, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("failed to get os.UserCacheDir: %v", err)
	}

	p := testscript.Params{
		Dir: "testdata",
		Setup: func(e *testscript.Env) error {
			// TODO feels like this could be a method on testscript.Env?
			getEnv := func(s string) string {
				cmp := s + "="
				for i := len(e.Vars) - 1; i >= 0; i-- {
					v := e.Vars[i]
					if strings.HasPrefix(v, cmp) {
						return strings.TrimPrefix(v, cmp)
					}
				}
				return ""
			}

			wd := getEnv("WORK")

			e.Vars = append(e.Vars,
				"TESTGOPATH="+modTestGOPATH,
				"GOBINMODPATH="+pathToMod,
				"GONOSUMDB=*",
				"GOPROXY="+proxyURL,
				"USERCACHEDIR="+ucd,
			)

			e.Vars = append(e.Vars, homeEnv(wd)...)

			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}

func homeEnv(base string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			"USERPROFILE=" + base + "\\home",
			"LOCALAPPDATA=" + base + "\\appdata",
			"HOME=" + base + "\\home", // match USERPROFILE
		}
	} else {
		return []string{"HOME=" + base + "/home"}
	}
}
