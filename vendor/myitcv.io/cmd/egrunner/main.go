// egrunner runs bash scripts in a Docker container to help with creating reproducible examples.
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
	"regexp"
	"strings"

	"mvdan.cc/sh/syntax"
	"myitcv.io/cmd/internal/bindmnt"
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

	fDebug      = flag.Bool("debug", false, "Print debug information for egrunner")
	fOut        = flag.String("out", "json", "output format; json(default)|debug|std")
	fGoRoot     = flag.String("goroot", "", "path to GOROOT to use")
	fGoProxy    = flag.String("goproxy", "", "path to GOPROXY to use")
	fGithubCLI  = flag.String("githubcli", "", "path to githubcli program")
	fEnvSubVars = flag.String("envsubst", "HOME,GITHUB_ORG,GITHUB_USERNAME", "comma-separated list of env vars to expand in commands")
)

const (
	debug = false

	scriptName      = "script.sh"
	blockPrefix     = "block:"
	outputSeparator = "============================================="
	commentStart    = "**START**"

	commentEnvSubstAdj = "egrunner_envsubst:"
	commentRewrite     = "egrunner_rewrite:"

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

	type rewrite struct {
		p *regexp.Regexp
		r string
	}

	var rewrites []rewrite
	envsubvars := strings.Split(*fEnvSubVars, ",")

	applyRewrite := func(s string) string {
		for _, r := range rewrites {
			s = r.p.ReplaceAllString(s, r.r)
		}

		return s
	}

	switch *fOut {
	case outJson, outStd, outDebug:
	default:
		return errorf("unknown option to -out: %v", *fOut)
	}

	debugOut = *fOut == outDebug || debug || *fDebug
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

catfile()
{
	echo "\$ cat $1"
	cat "$1"
}

comment()
{
	if [ "$#" -eq 0 ] || [ "$1" == "" ]
	then
		echo ""
	else
		echo "$1" | fold -w 100 -s | sed -e 's/^$/#/' | sed -e 's/^\([^#]\)/# \1/'
	fi
}

`)

	var ghcli string
	if *fGithubCLI != "" {
		if abs, err := filepath.Abs(*fGithubCLI); err == nil {
			ghcli = abs
		}
	} else {
		// this is a fallback in case any lookups via gobin fail
		ghcli, _ = exec.LookPath(commgithubcli)

		gobin, err := exec.LookPath("gobin")
		if err != nil {
			goto FinishedLookupGithubCLI
		}

		mbin := exec.Command(gobin, "-mod=readonly", "-p", "myitcv.io/cmd/githubcli")
		if mout, err := mbin.Output(); err == nil {
			ghcli = string(mout)
			goto FinishedLookupGithubCLI
		}

		gbin := exec.Command(gobin, "-nonet", "-p", "myitcv.io/cmd/githubcli")
		if gout, err := gbin.Output(); err == nil {
			ghcli = string(gout)
		}
	}

FinishedLookupGithubCLI:

	ghcli = strings.TrimSpace(ghcli)

	if ghcli != "" {
		// ghcli could still be empty at this point. We do nothing
		// because it's not guaranteed that it is required in the script.
		// Hence we let that error happen if and when it does and the user
		// will be able to work it out (hopefully)
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

	// TODO it would be significantly cleaner if we grouped, tidied etc all the statements
	// and comments into a custom data structure in one phase, then processed it in another.
	// The mixing of logic below is hard to read. Not to mention much more efficient.

	// process handles comment blocks and any special instructions within them
	process := func(cb []syntax.Comment) error {
		for _, c := range cb {
			l := strings.TrimSpace(c.Text)
			switch {
			case strings.HasPrefix(l, commentEnvSubstAdj):
				l := strings.TrimPrefix(l, commentEnvSubstAdj)
				for _, d := range strings.Fields(l) {
					a, d := d[0], d[1:]
					if len(d) == 0 {
						return errorf("envsubst adjustment invalid: %q", l)
					}

					switch a {
					case '+':
						envsubvars = append(envsubvars, d)
					case '-':
						nv := envsubvars[:0]
						for _, v := range envsubvars {
							if v != d {
								nv = append(nv, v)
							}
						}
						envsubvars = nv
					default:
						return errorf("envsubst adjustment invalid: %q", l)
					}
				}
			case strings.HasPrefix(l, commentRewrite):
				l := strings.TrimPrefix(l, commentRewrite)
				fs, err := splitQuotedFields(l)
				if err != nil {
					return errorf("failed to handle arguments for rewrite %q: %v", l, err)
				}
				if len(fs) != 2 {
					return errorf("rewrite expects exactly 2 (quoted) arguments; got %v from %q", len(fs), l)
				}
				p, err := regexp.Compile(fs[0])
				if err != nil {
					return errorf("failed to compile rewrite regexp %q: %v", fs[0], err)
				}
				rewrites = append(rewrites, rewrite{p, fs[1]})
			}
		}

		return nil
	}

	for si, s := range f.Stmts {

		lastNonBlank := uint(0)
		if last != nil {
			lastNonBlank = last.Line()
		}
		var commBlock []syntax.Comment
		for _, c := range s.Comments {
			if start == nil {
				if s.Pos().After(c.End()) {
					if strings.TrimSpace(c.Text) == commentStart {
						start = &c
					}
				}
			}

			// commBlock != nil indicates we have started adding comments to a block
			// The end of the block is makred by a blank line.

			// Work out whether we have passed a blank line.
			if c.Pos().Line() > lastNonBlank+1 {
				if err := process(commBlock); err != nil {
					return err
				}
				commBlock = make([]syntax.Comment, 0)
			}

			if commBlock != nil {
				// this comment is contiguous with last in existing comment
				commBlock = append(commBlock, c)
			}
			lastNonBlank = c.End().Line()
		}
		if s.Pos().Line() > lastNonBlank+1 {
			if err := process(commBlock); err != nil {
				return err
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
			isAssert = len(ce.Args) > 0 && ce.Args[0].Lit() == "assert"
		}
		nextIsAssert := false
		if si < len(f.Stmts)-1 {
			s := f.Stmts[si+1]
			if ce, ok := s.Cmd.(*syntax.CallExpr); ok {
				nextIsAssert = len(ce.Args) > 0 && ce.Args[0].Lit() == "assert"
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
			var envsubvarsstr string
			if len(envsubvars) > 0 {
				envsubvarsstr = "$" + strings.Join(envsubvars, ",$")
			}
			if !stdOut {
				fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH' | envsubst '%v' \n%v\nTHISWILLNEVERMATCH\n", envsubvarsstr, stmtString(s))
				fmt.Fprintf(toRun, "echo \"%v\"\n", outputSeparator)
			}
			stmts = append(stmts, co)
			if debugOut || (stdOut && b != nil) {
				fmt.Fprintf(toRun, "cat <<'THISWILLNEVERMATCH' | envsubst '%v' \n$ %v\nTHISWILLNEVERMATCH\n", envsubvarsstr, stmtString(s))
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

	debugf("finished compiling script: \ns%v\n", toRun.String())

	// docker requires the file/directory we are mapping to be within our
	// home directory because of "security"
	tf, err := ioutil.TempFile("", ".go_modules_by_example")
	if err != nil {
		return errorf("failed to create temp file: %v", err)
	}

	tfn := tf.Name()

	defer func() {
		debugf("Removing temp script %v\n", tf.Name())
		os.Remove(tf.Name())
	}()

	if err := ioutil.WriteFile(tfn, toRun.Bytes(), 0644); err != nil {
		return errorf("failed to write to temp file %v: %v", tfn, err)
	}

	debugf("wrote script to %v\n", tfn)

	if etfn, err := bindmnt.Resolve(tfn); err == nil {
		tfn = etfn
	}

	debugf("script will map from %v to %v\n", tfn, scriptName)

	args := []string{"docker", "run", "--rm", "-w", "/home/gopher", "-e", "GITHUB_PAT", "-e", "GITHUB_USERNAME", "-e", "GO_VERSION", "-e", "GITHUB_ORG", "-e", "GITHUB_ORG_ARCHIVE", "--entrypoint", "bash", "-v", fmt.Sprintf("%v:/%v", tfn, scriptName)}

	if ghcli != "" {
		if eghcli, err := bindmnt.Resolve(ghcli); err == nil {
			args = append(args, "-v", fmt.Sprintf("%v:/go/bin/%v", eghcli, commgithubcli))
		}
	}

	for _, df := range fDockerFlags {
		parts := strings.SplitN(df, "=", 2)
		switch len(parts) {
		case 1:
			args = append(args, parts[0])
		case 2:
			flag, value := parts[0], parts[1]
			if flag == "-v" {
				vparts := strings.Split(value, ":")
				if len(vparts) != 2 {
					return errorf("-v flag had unexpected format: %q", value)
				}
				src := vparts[0]
				if esrc, err := bindmnt.Resolve(src); err == nil {
					value = esrc + ":" + vparts[1]
				}
			}
			args = append(args, flag, value)
		default:
			panic("invariant fail")
		}
	}

	if *fGoRoot != "" {
		if egr, err := bindmnt.Resolve(*fGoRoot); err == nil {
			args = append(args, "-v", fmt.Sprintf("%v:/go", egr))
		}
	}

	if *fGoProxy != "" {
		if egp, err := bindmnt.Resolve(*fGoProxy); err == nil {
			args = append(args, "-v", fmt.Sprintf("%v:/goproxy", egp), "-e", "GOPROXY=file:///goproxy")
		}
	}

	// build docker image
	{
		td, err := ioutil.TempDir("", "egrunner-docker-build")
		if err != nil {
			return errorf("failed to create temp dir for docker build: %v", err)
		}
		defer func() {
			debugf("Removing temp dir %v\n", td)
			os.RemoveAll(td)
		}()
		df := filepath.Join(td, "Dockerfile")
		udf := fmt.Sprintf(userDockerfile, os.Getuid(), os.Getgid())
		if err := ioutil.WriteFile(df, []byte(udf), 0644); err != nil {
			return errorf("failed to write temp Dockerfile %v: %v", df, err)
		}

		var stdout, stderr bytes.Buffer
		dbcmd := exec.Command("docker", "build", "-q", td)
		dbcmd.Stdout = &stdout
		dbcmd.Stderr = &stderr
		debugf("building docker image with %v\n", strings.Join(dbcmd.Args, " "))
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
		cmdout, err := cmd.StdoutPipe()
		if err != nil {
			return errorf("failed to create cmd stdout pipe: %v", err)
		}

		var scanerr error
		done := make(chan bool)

		scanner := bufio.NewScanner(cmdout)
		go func() {
			for scanner.Scan() {
				fmt.Println(applyRewrite(scanner.Text()))
			}
			if err := scanner.Err(); err != nil {
				scanerr = err
			}
			close(done)
		}()

		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return errorf("failed to run %v: %v", strings.Join(cmd.Args, " "), err)
		}
		<-done
		if scanerr != nil {
			return errorf("failed to rewrite output: %v", scanerr)
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
		cur.WriteString(applyRewrite(l))
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

func splitQuotedFields(s string) ([]string, error) {
	// Split fields allowing '' or "" around elements.
	// Quotes further inside the string do not count.
	var f []string
	for len(s) > 0 {
		for len(s) > 0 && isSpaceByte(s[0]) {
			s = s[1:]
		}
		if len(s) == 0 {
			break
		}
		// Accepted quoted string. No unescaping inside.
		if s[0] == '"' || s[0] == '\'' {
			quote := s[0]
			s = s[1:]
			i := 0
			for i < len(s) && s[i] != quote {
				i++
			}
			if i >= len(s) {
				return nil, errorf("unterminated %c string", quote)
			}
			f = append(f, s[:i])
			s = s[i+1:]
			continue
		}
		i := 0
		for i < len(s) && !isSpaceByte(s[i]) {
			i++
		}
		f = append(f, s[:i])
		s = s[i:]
	}
	return f, nil
}

func isSpaceByte(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func errorf(format string, args ...interface{}) error {
	if debugOut {
		panic(fmt.Errorf(format, args...))
	}
	return fmt.Errorf(format, args...)
}

func debugf(format string, args ...interface{}) {
	if debugOut {
		fmt.Fprintf(os.Stderr, "+ "+format, args...)
	}
}

const userDockerfile = `
FROM golang

ENV PATH=/vbash/bin:/home/gopher/.local/bin:/home/gopher/gopath/bin:$PATH
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
RUN apt-get install -y graphviz

RUN usermod -aG docker gopher

USER gopher

RUN mkdir -p $GOPATH/bin
RUN curl -fsSLo $GOPATH/bin/gobin https://github.com/myitcv/gobin/releases/download/v0.0.3/linux-amd64
RUN chmod 755 $GOPATH/bin/gobin
`
