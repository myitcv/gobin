package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/goproxytest"
	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

var (
	proxyURL string

	goVersionCondRegex = regexp.MustCompile(`\Ago\d\.\d\d*\z`)
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
				"GOPROXY="+proxyURL,
				"USERCACHEDIR="+ucd,
			)

			if runtime.GOOS == "windows" {
				e.Vars = append(e.Vars,
					"USERPROFILE="+wd+"\\home",
					"LOCALAPPDATA="+wd+"\\appdata",
					"HOME="+wd+"\\home", // match USERPROFILE
				)
			} else {
				e.Vars = append(e.Vars,
					"HOME="+wd+"/home",
				)
			}
			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}
