#!/usr/bin/env bash

# **START**

echo "machine github.com login $GITHUB_USERNAME password $GITHUB_PAT" >> $HOME/.netrc
echo "" >> $HOME/.netrc
echo "machine api.github.com login $GITHUB_USERNAME password $GITHUB_PAT" >> $HOME/.netrc
git config --global user.email "$GITHUB_USERNAME@example.com"
git config --global user.name "$GITHUB_USERNAME"
git config --global advice.detachedHead false
git config --global push.default current

unset GOPATH

# these steps are effectively copied from .installation.sh
export GO111MODULE=on
git clone https://github.com/myitcv/gobin /tmp/gobin
cd /tmp/gobin
go install
export PATH=$(go env GOPATH)/bin:$PATH
which gobin

# block: gohack
gobin github.com/rogpeppe/gohack

# block: gohack latest
gobin github.com/rogpeppe/gohack@latest

# block: gohack v1.0.0-alpha.1
gobin github.com/rogpeppe/gohack@v1.0.0-alpha.1

# block: gohack print
gobin -p github.com/rogpeppe/gohack@v1.0.0-alpha.1

# block: gohack run
gobin -r github.com/rogpeppe/gohack@v1.0.0-alpha.1 -help
assert "$? -eq 2" $LINENO

