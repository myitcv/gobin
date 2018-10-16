```
goget [-i|-r|-p] [-n] packages

goget resolves (and installs or runs) main packages.

The packages argument to goget is similar to that of go get (in module mode) but the list of packages must strictly
be main packages that are part of Go modules. The rule for default package version (and overrides via @version) is
the same as go get. Hence goget main_pkg@latest has the same version semantics as go get main_pkg@latest.

With no flags, goget installs the main packages to goget/pkg/$module/$main_pkg@$version under your user cache directory.
See the documentation for os.UserCacheDir for OS-specific details on how to configure its location.

The -i flag installs the main packages to $GOBIN (or $GOPATH/bin if GOBIN is unset, which defaults to $HOME/go/bin if
GOPATH is not set).

The -r flag requires exactly one main package argument and runs that package. It is similar therefore to go run. Any
arguments to goget following --- will be passed to the main package.

The -p flag prints the cache install path for each of the provided packages once versions have been resolved.

The -i, -r and -p flags are mutually exclusive.

The -n flag prevents network access and instead uses the GOPATH module download cache where required.

It is an error for a non-main package, a package pattern or a main package not part of a module to be provided as a
package argument.

To install a main package not part of a go module, use:

    GO111MODULE=off go get main_pkg
```
