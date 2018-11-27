// +build linux

package bindmnt

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	targetRegexp = regexp.MustCompile(`^.*\[(.*)\]$`)
)

func resolve(p string) (string, error) {
	findmnt, err := exec.LookPath("findmnt")
	if err != nil {
		return "", fmt.Errorf("failed to PATH-resolve findmnt: %v", err)
	}

	var res string

	// findmnt -n --raw -T /home/myitcv/.mountpoints/github_myitcv_neovim/src/github.com/myitcv/neovim
	//
	// results in a space-separated (quoted?) output:
	//
	// TARGET SOURCE FSTYPE OPTIONS
	//
	// e.g.
	//
	// /home/myitcv/gostuff /dev/sda1[/home/myitcv/.gostuff/1.11.1] ext4 rw,relatime,errors=remount-ro,data=ordered
	for {
		cmd := exec.Command(findmnt, "-n", "--raw", "-T", p)
		outb, err := cmd.Output()
		if err != nil {
			var stderr []byte
			if ee, ok := err.(*exec.ExitError); ok {
				stderr = append([]byte("\n"), ee.Stderr...)
			}
			return "", fmt.Errorf("failed to run %v: %v%s", strings.Join(cmd.Args, " "), err, stderr)
		}

		out := string(outb)
		if out[len(out)-1] == '\n' {
			out = out[:len(out)-1]
		}

		// there should be a single line
		lines := strings.Split(string(out), "\n")
		if len(lines) != 1 {
			return "", fmt.Errorf("command %v gave multiple lines: %q", strings.Join(cmd.Args, " "), out)
		}

		line := lines[0]

		fs := strings.Fields(line)
		if len(fs) != 4 {
			return "", fmt.Errorf("line %q did not have 4 fields", line)
		}

		target, disksource := fs[0], fs[1]

		// if target == / we are done (because there will not be a source)
		if target == "/" {
			res = filepath.Join(p, res)
			break
		}

		sms := targetRegexp.FindStringSubmatch(disksource)
		if sms == nil || len(sms) != 2 {
			return "", fmt.Errorf("source %q did not match as expected: %v", disksource, sms)
		}
		source := sms[1]

		// This happens when we have used a bind mount in, for example, a docker
		// container where the target, e.g. /tmp, is the same as the source, /tmp
		// on the host.
		if target == source {
			res = filepath.Join(p, res)
			break
		}

		// calculate p relative to target
		rel, err := filepath.Rel(target, p)
		if err != nil {
			return "", fmt.Errorf("failed to calculate %v relative to %v", p, target)
		}
		if rel != "." {
			res = filepath.Join(rel, res)
		}

		p = source
	}

	return res, nil
}
