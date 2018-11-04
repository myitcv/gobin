package main

import (
	"fmt"
	"io"
	"text/template"
)

func mainUsage(f io.Writer) {
	t := template.Must(template.New("").Parse(mainHelpTemplate))
	if err := t.Execute(f, nil); err != nil {
		fmt.Fprintf(f, "cannot write usage output: %v", err)
	}
}

var mainHelpTemplate = `
The gobin command installs/runs main packages.

Usage:
	gobin [-m] [-run|-p|-d] [-u|-nonet] packages [run arguments...]

The gobin command builds, installs, and possibly runs an executable binary for
each of the named main packages.

The packages argument to gobin is similar to that of the go get command (in
module aware mode) with the additional constraint that the list of packages
must be main packages. Each argument takes the form $main_pkg[@$version].

By default, gobin will use the main package's module to resolve its
dependencies, unless the -m flag is specified, in which case dependencies will
be resolved using the main module (as given by go env GOMOD).

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

The -d flag instructs gobin to stop after installing the packages to the gobin
cache; that is, it instructs gobin not to install, run or print the packages.

The -run, -p and -d flags are mutually exclusive.

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

`[1:]
