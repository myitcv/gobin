#!/usr/bin/env bash

# **START**

echo "machine github.com login $GITHUB_USERNAME password $GITHUB_PAT" >> $HOME/.netrc
echo "" >> $HOME/.netrc
echo "machine api.github.com login $GITHUB_USERNAME password $GITHUB_PAT" >> $HOME/.netrc
git config --global user.email "$GITHUB_USERNAME@example.com"
git config --global user.name "$GITHUB_USERNAME"
git config --global advice.detachedHead false
git config --global push.default current

# block: clone
export GO111MODULE=on
git clone https://github.com/myitcv/gobin /tmp/gobin
cd /tmp/gobin

# block: install
go install

# block: fix path
export PATH=$(go env GOPATH)/bin:$PATH
which gobin

# block: use
gobin -r github.com/rogpeppe/gohack@master -help
assert "$? -eq 2" $LINENO
