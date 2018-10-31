<!-- __JSON: go list -json .
### `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

See the [FAQ](https://github.com/myitcv/gobin/wiki/FAQ) for more details.

-->
### `gobin`

The gobin command installs/runs main packages.

See the [FAQ](https://github.com/myitcv/gobin/wiki/FAQ) for more details.

<!-- END -->

<!-- __JSON: go run github.com/myitcv/gobin -m -r myitcv.io/cmd/egrunner .readme.sh # LONG ONLINE

### Installation

```
{{PrintBlock "get" -}}
```

or download a binary from [the latest release](https://github.com/myitcv/gobin/releases).

Update your `PATH` and verify we can find `gobin` in our new `PATH`:

```
{{PrintBlock "fix path" -}}
```

### Examples: global mode

Globally install `gohack`:

```
{{PrintBlock "gohack" -}}
```

Install a specific version of `gohack`:

```
{{PrintBlock "gohack v1.0.0" -}}
```

Print the `gobin` cache location of a specific `gohack` version:

```
{{PrintBlock "gohack print" -}}
```

Run a specific `gohack` version:

```
{{PrintBlock "gohack run" | lineEllipsis 4 -}}
```

### Examples: main-module mode

Define a module:

```
{{PrintBlock "module" -}}
```

Add a [tool dependency](https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md):

```go
{{PrintBlock "tools" -}}
```

Review the version of `stringer` being used:

```
{{PrintBlock "tools version" -}}
```

Check the help for `stringer`:

```
{{PrintBlock "stringer help" | lineEllipsis 5 -}}
```

Use `stringer` via a `go:generate` directive:

```go
{{PrintBlock "use in go generate" -}}
```

Generate and run as a "test":

```
{{PrintBlock "go generate and run" -}}
```


-->

### Installation

```
$ GO111MODULE=off go get -u github.com/myitcv/gobin
```

or download a binary from [the latest release](https://github.com/myitcv/gobin/releases).

Update your `PATH` and verify we can find `gobin` in our new `PATH`:

```
$ export PATH=$(go env GOPATH)/bin:$PATH
$ which gobin
/home/gopher/gopath/bin/gobin
```

### Examples: global mode

Globally install `gohack`:

```
$ gobin github.com/rogpeppe/gohack
Installed github.com/rogpeppe/gohack@v1.0.0 to /home/gopher/gopath/bin/gohack
```

Install a specific version of `gohack`:

```
$ gobin github.com/rogpeppe/gohack@v1.0.0
Installed github.com/rogpeppe/gohack@v1.0.0 to /home/gopher/gopath/bin/gohack
```

Print the `gobin` cache location of a specific `gohack` version:

```
$ gobin -p github.com/rogpeppe/gohack@v1.0.0
/home/gopher/.cache/gobin/github.com/rogpeppe/gohack/@v/v1.0.0/github.com/rogpeppe/gohack/gohack
```

Run a specific `gohack` version:

```
$ gobin -r github.com/rogpeppe/gohack@v1.0.0 -help
The gohack command checks out Go module dependencies
into a directory where they can be edited, and adjusts
the go.mod file appropriately.
...
```

### Examples: main-module mode

Define a module:

```
$ cat go.mod
module example.com/hello
```

Add a [tool dependency](https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md):

```go
$ cat tools.go
// +build tools

package tools

import (
	_ "golang.org/x/tools/cmd/stringer"
)
```

Review the version of `stringer` being used:

```
$ gobin -m -p golang.org/x/tools/cmd/stringer
/home/gopher/hello/.gobincache/golang.org/x/tools/@v/v0.0.0-20181102223251-96e9e165b75e/golang.org/x/tools/cmd/stringer/stringer
```

Check the help for `stringer`:

```
$ gobin -m -r golang.org/x/tools/cmd/stringer -help
Usage of stringer:
	stringer [flags] -type T [directory]
	stringer [flags] -type T files... # Must be a single package
For more information, see:
...
```

Use `stringer` via a `go:generate` directive:

```go
$ cat main.go
package main

import "fmt"

//go:generate gobin -m -r golang.org/x/tools/cmd/stringer -type=Pill

type Pill int

const (
	Placebo Pill = iota
	Aspirin
	Ibuprofen
	Paracetamol
	Acetaminophen = Paracetamol
)

func main() {
	fmt.Printf("For headaches, take %v\n", Ibuprofen)
}
```

Generate and run as a "test":

```
$ go generate
$ go run .
For headaches, take Ibuprofen
```


<!-- END -->

### Usage

<!-- __TEMPLATE: sh -c "go run ${DOLLAR}(go list -f '{{.ImportPath}}') -h 2>&1 | head -n -1 || true"

```
{{.Out -}}
```
-->

```
The gobin command installs/runs main packages.

Usage:
	gobin [-m] [-r|-p] [-n|-g] packages [run arguments...]

The packages argument to gobin is similar to that of the go tool (in module
mode) with the additional constraint that the list of packages must be main
packages.

By default, gobin is said to operate in global mode. If the -m flag is provided
then it is said to operate in main-module mode, where the path to the main
module's go.mod is given by go env GOMOD.

The version "latest" matches the latest available tagged version. If no version
is specified, gobin behaves differently in global and main-module modes. In
global mode, gobin attempts to resolve the latest version available in the
module download cache. In main-module module, gobin attempts to resolve the
current version via the main module's go.mod. If this resolution fails in
either mode, "latest" is assumed and gobin resolves via the network.

In global mode, gobin installs the main packages to the directories
gobin/$module@$version/$main_pkg under your user cache directory. See the
documentation for os.UserCacheDir for OS-specific details on how to configure
its location.

In main-module mode, gobin installs the main packages to the directories
.gobincache/$module@$version/$main_pkg under the directory containing the main
module's go.mod.

By default, gobin installs the main packages to $GOBIN (or $GOPATH/bin if GOBIN
is not set, which defaults to $HOME/go/bin if GOPATH is not set).

The -r flag takes exactly one main package argument and runs that package.  It
is similar therefore to go run. Any arguments after the single main package
will be passed to the main package as command line arguments.

The -p flag prints the cache install path for each of the provided packages
once versions have been resolved.

The -r and -p flags are mutually exclusive.

The -g flag forces gobin to perform a network fetch for the provided main
packages.

The -n flag prevents network access and instead uses the GOPATH module download
cache where required.

The -g and -n flags are mutually exclusive.

The -m flag causes gobin to use the main module (the module containing the
directory where the gobin command is run). The main module is given by go env
GOMOD. Without this flag gobin effectively runs as a "global" tool.

It is an error for a non-main package to be provided as a package argument.

```
<!-- END -->


### Credits

* [mvdan](https://github.com/mvdan)
* [rogpeppe](https://github.com/rogpeppe)

### Notes

In the context of https://github.com/golang/go/issues/24250 and https://github.com/golang/go/issues/27653, this is a WIP
experiment. This project may die, move, etc at any time, until further notice.

