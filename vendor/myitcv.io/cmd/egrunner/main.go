package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/syntax"
)

var _ flag.Value = (*dockerFlags)(nil)

type dockerFlags []string

func (d *dockerFlags) String() string {
	return strings.Join(*d, " ")
}

func (d *dockerFlags) Set(v string) error {
	*d = append(*d, v)
	return nil
}

var (
	debugOut     = false
	stdOut       = false
	fDockerFlags dockerFlags

	fOut       = flag.String("out", "json", "output format; json(default)|debug|std")
	fGoRoot    = flag.String("goroot", "", "path to GOROOT to use")
	fGoProxy   = flag.String("goproxy", "", "path to GOPROXY to use")
	fGithubCLI = flag.String("githubcli", "", "path to githubcli program")
)

const (
	scriptName      = "script.sh"
	blockPrefix     = "block:"
	outputSeparator = "============================================="
	commentStart    = "**START**"

	commgithubcli = "githubcli"

	outJson  = "json"
	outStd   = "std"
	outDebug = "debug"
)

type block string

func (b *block) String() string {
	if b == nil {
		return "nil"
	}

	return string(*b)
}

func main() {
	flag.Var(&fDockerFlags, "df", "flag to pass to docker")
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if *fGoRoot == "" {
		*fGoRoot = os.Getenv("EGRUNNER_GOROOT")
	}

	if *fGoProxy == "" {
		*fGoProxy = os.Getenv("EGRUNNER_GOPROXY")
	}

	switch *fOut {
	case outJson:
	case outStd:
	case outDebug:
	default:
		return errorf("unknown option to -out: %v", *fOut)
	}

	debugOut = *fOut == outDebug
	if !debugOut {
		stdOut = *fOut == outStd
	}

	toRun := new(bytes.Buffer)
	toRun.WriteString(`#!/usr/bin/env bash
set -u
set -o pipefail

assert()
{
  E_PARAM_ERR=98
  E_ASSERT_FAILED=99

  if [ -z "$2" ]
  then
    exit $E_PARAM_ERR
  fi

  lineno=$2

  if [ ! $1 ]
  then
    echo "Assertion failed:  \"$1\""
    echo "File \"$0\", line $lineno"
    exit $E_ASSERT_FAILED
  fi
}

`)

	var ghcli string
	if *fGithubCLI != "" {
		ghcli = *fGithubCLI
	} else {
		ghcli, _ = exec.LookPath(commgithubcli)
	}
	if ghcli != "" {
		if abs, err := filepath.Abs(ghcli); err == nil {
			ghcli = abs
		}
	}

	if len(flag.Args()) != 1 {
		return errorf("expected a single argument script file to run")
	}

	fn := flag.Arg(0)

	fi, err := os.Open(fn)
	if err != nil {
		return errorf("failed to open %v: %v", fn, err)
	}

	f, err := syntax.NewParser(syntax.KeepComments).Parse(fi, fn)
	if err != nil {
		return errorf("failed to parse %v: %v", fn, err)
	}

	var last *syntax.Pos
	var b *block

	// blocks is a mapping from statement index to *block this allows us to take
	// the output from each statement and not only assign it to the
	// corresponding index but also add to the block slice too (if the block is
	// defined)
	seenBlocks := make(map[block]bool)

	p := syntax.NewPrinter()

	stmtString := func(s *syntax.Stmt) string {
		// temporarily "blank" the comments associated with the stmt
		cs := s.Comments
		s.Comments = nil
		var b bytes.Buffer
		p.Print(&b, s)
		s.Comments = cs
		return b.String()
	}

	type cmdOutput struct {
		Cmd string
		Out string
	}

	var stmts []*cmdOutput
	blocks := make(map[block][]*cmdOutput)

	pendingSep := false

	// find the # START comment
	var start *syntax.Comment

	for si, s := range f.Stmts {
		if start == nil {
			for _, c := range s.Comments {
				if s.Pos().After(c.End()) {
					if strings.TrimSpace(c.Text) == commentStart {
						start = &c
					}
				}
			}
		}
		if start == nil || start.Pos().After(s.Pos()) {
			continue
		}
		setBlock := false
		for _, c := range s.Comments {
			if s.Pos().After(c.End()) && s.Pos().Line()-1 == c.End().Line() {
				l := strings.TrimSpace(c.Text)
				if strings.HasPrefix(l, blockPrefix) {
					v := block(strings.TrimSpace(strings.TrimPrefix(l, blockPrefix)))
					if seenBlocks[v] {
						return errorf("block %q used multiple times", v)
					}
					seenBlocks[v] = true
					b = &v
					setBlock = true
				}
			}
		}
		if !setBlock {
			if last != nil && last.Line()+1 != s.Pos().Line() {
				// end of contiguous block
				b = nil
			}
		}
		isAssert := false
		if ce, ok := s.Cmd.(*syntax.CallExpr); ok {
			if ce.Args != nil && ce.Args[0].Parts != nil {
				if li, ok := ce.Args[0].Parts[0].(*syntax.Lit); ok {
					if li.Value == "assert" {
						isAssert = true
					}
				}
			}
		}
		nextIsAssert := false
		if si < len(f.Stmts)-1 {
			s := f.Stmts[si+1]
			if ce, ok := s.Cmd.(*syntax.CallExpr); ok {
				if ce.Args != nil && ce.Args[0].Parts != nil {
					if li, ok := ce.Args[0].Parts[0].(*syntax.Lit); ok {
						if li.Value == "assert" {
							nextIsAssert = true
						}
					}
				}
			}
		}

		if isAssert {
			// TODO improve this by actually inspecting the second argument
			// to assert
			l := stmtString(s)
			l = strings.Replace(l, "$LINENO", fmt.Sprintf("%v", s.Pos().Line()), -1)
			fmt.Fprintf(toRun, "%v\n", l)
		} else {
			co := &cmdOutput{
				Cmd: stmtString(s),
			}

			if pendingSep && !stdOut {
				fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
			}
			if !stdOut {
				fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH' | envsubst '$HOME,$GITHUB_ORG,$GITHUB_USERNAME' \n%v\nTHISWILLNEVERMATCH\n", stmtString(s))
				fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
			}
			stmts = append(stmts, co)
			if debugOut || (stdOut && b != nil) {
				fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH' | envsubst '$HOME,$GITHUB_ORG,$GITHUB_USERNAME' \n$ %v\nTHISWILLNEVERMATCH\n", stmtString(s))
			}
			fmt.Fprintf(toRun, "%v\n", stmtString(s))

			// if this statement is not an assert, and the next statement is
			// not an assert, then we need to inject an assert that corresponds
			// to asserting a zero exit code
			if !nextIsAssert {
				fmt.Fprintf(toRun, "assert \"$? -eq 0\" %v\n", s.Pos().Line())
			}

			pendingSep = true

			if b != nil {
				blocks[*b] = append(blocks[*b], co)
			}
		}

		// now calculate the last line of this statement, including heredocs etc

		// TODO this might need to be better
		end := s.End()
		for _, r := range s.Redirs {
			if r.End().After(end) {
				end = r.End()
			}
			if r.Hdoc != nil {
				if r.Hdoc.End().After(end) {
					end = r.Hdoc.End()
				}
			}
		}
		last = &end
	}

	if pendingSep {
		fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
	}

	// docker requires the file/directory we are mapping to be within our
	// home directory because of "security"
	tf, err := ioutil.TempFile("", ".go_modules_by_example")
	if err != nil {
		return errorf("failed to create temp file: %v", err)
	}

	tfn := tf.Name()

	defer os.Remove(tf.Name())

	if err := ioutil.WriteFile(tfn, toRun.Bytes(), 0644); err != nil {
		return errorf("failed to write to temp file %v: %v", tfn, err)
	}

	args := []string{"docker", "run", "--rm", "-w", "/home/gopher", "-e", "GITHUB_PAT", "-e", "GITHUB_USERNAME", "-e", "GO_VERSION", "-e", "GITHUB_ORG", "-e", "GITHUB_ORG_ARCHIVE", "--entrypoint", "bash", "-v", fmt.Sprintf("%v:/%v", tfn, scriptName)}

	if ghcli != "" {
		args = append(args, "-v", fmt.Sprintf("%v:/go/bin/%v", ghcli, commgithubcli))
	}

	for _, df := range fDockerFlags {
		parts := strings.SplitN(df, "=", 2)
		args = append(args, parts...)
	}

	if *fGoRoot != "" {
		args = append(args, "-v", fmt.Sprintf("%v:/go", *fGoRoot))
	}

	if *fGoProxy != "" {
		args = append(args, "-v", fmt.Sprintf("%v:/goproxy", *fGoProxy), "-e", "GOPROXY=file:///goproxy")
	}

	// build docker image
	{
		td, err := ioutil.TempDir("", "egrunner-docker-build")
		if err != nil {
			return errorf("failed to create temp dir for docker build: %v", err)
		}
		defer os.RemoveAll(td)
		df := filepath.Join(td, "Dockerfile")
		udf := fmt.Sprintf(userDockerfile, os.Getuid(), os.Getgid())
		if err := ioutil.WriteFile(df, []byte(udf), 0644); err != nil {
			return errorf("failed to write temp Dockerfile %v: %v", df, err)
		}

		var stdout, stderr bytes.Buffer
		dbcmd := exec.Command("docker", "build", "-q", td)
		dbcmd.Stdout = &stdout
		dbcmd.Stderr = &stderr
		if err := dbcmd.Run(); err != nil {
			return errorf("failed to run %v: %v\n%s", strings.Join(dbcmd.Args, " "), err, stderr.String())
		}

		iid := strings.TrimSpace(stdout.String())

		args = append(args, iid)
	}

	args = append(args, fmt.Sprintf("/%v", scriptName))

	cmd := exec.Command(args[0], args[1:]...)
	debugf("now running %v via %v\n", tfn, strings.Join(cmd.Args, " "))

	if debugOut || stdOut {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return errorf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
		}
		return nil
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to run %v: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	cur := new(strings.Builder)
	for scanner.Scan() {
		l := scanner.Text()
		if l == outputSeparator {
			lines = append(lines, cur.String())
			cur = new(strings.Builder)
			continue
		}
		cur.WriteString(l)
		cur.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return errorf("error scanning cmd output: %v", err)
	}

	if len(lines) != 2*len(stmts) {
		return errorf("had %v statements but %v lines of output", len(stmts), len(lines))
	}

	j := 0
	for i := 0; i < len(lines); {
		// strip the last \n off the cmd
		stmts[j].Cmd = lines[i][:len(lines[i])-1]
		i += 1
		stmts[j].Out = lines[i]
		i += 1
		j += 1
	}

	tmpl := struct {
		Stmts  []*cmdOutput
		Blocks map[block][]*cmdOutput
	}{
		Stmts:  stmts,
		Blocks: blocks,
	}

	byts, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return errorf("error marshaling JSON: %v", err)
	}

	fmt.Printf("%s\n", byts)

	return nil
}

func errorf(format string, args ...interface{}) error {
	if debugOut {
		panic(fmt.Errorf(format, args...))
	}
	return fmt.Errorf(format+"\n", args...)
}

func debugf(format string, args ...interface{}) {
	if debugOut {
		fmt.Printf(format, args...)
	}
}

const userDockerfile = `
FROM golang

ENV PATH=/vbash/bin:/home/gopher/.local/bin:$PATH
ENV GOPATH=/home/gopher/gopath

RUN groupadd -g %[2]v gopher && \
    adduser --uid %[1]v --gid %[2]v --disabled-password --gecos "" gopher

# install sudo
RUN apt-get update
RUN apt-get install -y sudo tree gettext-base

# enable sudo
RUN usermod -aG sudo gopher
RUN echo "gopher ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/gopher

RUN apt-get update
RUN apt-get install -y apt-transport-https ca-certificates curl gnupg2 software-properties-common
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"
RUN apt-get update
RUN apt-get install -y docker-ce

RUN usermod -aG docker gopher

USER gopher
`
