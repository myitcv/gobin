<!-- __JSON: go list -json .
### `{{ filepathBase .Out.ImportPath}}`

{{.Out.Doc}}

See the [FAQ](https://github.com/myitcv/gobin/wiki/FAQ) for more details.

### `gobin` is deprecated as of Go 1.16

Go 1.16 supports [`go install $pkg@$version`](https://golang.org/doc/go1.16#go-command) to install commands without
affecting the main module. This is the default and most popular mode of operation for `gobin`.

A proposal to support [`go run $pkg@$version`](https://github.com/golang/go/issues/42088) was accepted in January
2021, and should hopefully land in Go 1.17. This will cover the `gobin -run` use case.

Hence we have decided to archive this project.

-->
### `gobin`

The gobin command installs/runs main packages.

See the [FAQ](https://github.com/myitcv/gobin/wiki/FAQ) for more details.

### `gobin` is deprecated as of Go 1.16

Go 1.16 supports [`go install $pkg@$version`](https://golang.org/doc/go1.16#go-command) to install commands without
affecting the main module. This is the default and most popular mode of operation for `gobin`.

A proposal to support [`go run $pkg@$version`](https://github.com/golang/go/issues/42088) was accepted in January
2021, and should hopefully land in Go 1.17. This will cover the `gobin -run` use case.

Hence we have decided to archive this project.

<!-- END -->

<!-- __JSON: sh -c "go run github.com/myitcv/gobin -m -run myitcv.io/cmd/egrunner -df=-v=$DOLLAR{GOPATH%%:*}:/gopath -df=-v=${DOLLAR}PWD:/self .readme.sh" # LONG ONLINE

### Installation

```
{{PrintBlock "get" -}}
```

or download a binary from [the latest release](https://github.com/myitcv/gobin/releases).

Update your `PATH` and verify we can find `gobin` in our new `PATH`:

```
{{PrintBlock "fix path" -}}
```

### Examples

Install `gohack`:

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

### Examples: using `-m`

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

### Examples

Install `gohack`:

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
$ gobin -run github.com/rogpeppe/gohack@v1.0.0 -help
The gohack command checks out Go module dependencies
into a directory where they can be edited, and adjusts
the go.mod file appropriately.
...
```

### Examples: using `-m`

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
$ gobin -m -run golang.org/x/tools/cmd/stringer -help
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

//go:generate gobin -m -run golang.org/x/tools/cmd/stringer -type=Pill

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
	gobin [-m] [-run|-p|-v|-d] [-u|-nonet] [-tags 'tag list'] packages [run arguments...]

The gobin command builds, installs, and possibly runs an executable binary for
each of the named main packages.

The packages argument to gobin is similar to that of the go get command (in
module aware mode) with the additional constraint that the list of packages
must be main packages. Each argument takes the form $main_pkg[@$version].

By default, gobin will use the main package's module to resolve its
dependencies, unless the -m flag is specified, in which case dependencies will
be resolved using the main module (as given by go env GOMOD).

The -mod flag provides additional control over updating and use of go.mod when
using the main module to resolve dependencies. If the -mod flag is provided it
implies -m. With -mod=readonly, gobin is disallowed from any implicit updating
of go.mod. Instead, it fails when any changes to go.mod are needed. With
-mod=vendor, gobin assumes that the vendor directory holds the correct copies
of dependencies and ignores the dependency descriptions in go.mod

This means that gobin $package@v1.2.3 is a repeatable way to install an exact
version of a binary (assuming it has an associated go.mod file).

The version "latest" matches the latest available tagged version for the module
containing the main package. If gobin is able to resolve "latest" within the
module download cache it will use that version. Otherwise, gobin will make a
network request to resolve "latest". The -u flag forces gobin to check the
network for the latest tagged version. If the -nonet flag is provided, gobin
will only check the module download cache. Hence, the -u and -nonet flags are
mutually exclusive.

Versions that take the form of a revision identifier (a branch name, for
example) can only be resolved with a network request and hence are incompatible
with -nonet.

If no version is specified for a main package, gobin behaves differently
depending on whether the -m flag is provided. If the -m flag is not provided,
gobin $module is equivalent to gobin $module@latest. If the -m flag is
provided, gobin attempts to resolve the current version via the main module's
go.mod; if this resolution fails, "latest" is assumed as the version.

By default, gobin installs the main packages to $GOBIN (or $GOPATH/bin if GOBIN
is not set, which defaults to $HOME/go/bin if GOPATH is not set).

The -run flag takes exactly one main package argument and runs that package.
It is similar therefore to go run. Any arguments after the single main package
will be passed to the main package as command line arguments.

The -p flag prints the gobin cache path for each of the packages' executables
once versions have been resolved.

The -v flag prints the module path and version for each of the packages. Each
line in the output has two space-separated fields: a module path and a version.

The -d flag instructs gobin to stop after installing the packages to the gobin
cache; that is, it instructs gobin not to install, run or print the packages.

The -run, -p, -v and -d flags are mutually exclusive.

The -tags flag is identical to the cmd/go build flag (see go help build). It is
a space-separated list of build tags to consider satisfied during the build.
Alternatively, GOFLAGS can be set to include a value for -tags (see go help
environment).

It is an error for a non-main package to be provided as a package argument.


Cache directories
=================

gobin maintains a cache of executables, separate from any executables that may
be installed to $GOBIN.

By default, gobin uses the directories gobin/$module@$version/$main_pkg under
your user cache directory. See the documentation for os.UserCacheDir for
OS-specific details on how to configure its location.

When the -m flag is provided, gobin uses the directories
.gobincache/$module@$version/$main_pkg under the directory containing the main
module's go.mod.

```
<!-- END -->


### Credits

* [mvdan](https://github.com/mvdan)
* [rogpeppe](https://github.com/rogpeppe)

