#!/usr/bin/env bash

# Copyright (c) 2016 Paul Jolly <paul@myitcv.org.uk>, all rights reserved.
# Use of this document is governed by a license found in the LICENSE document.

set -eu
set -o pipefail
set -o errtrace

cd $(git rev-parse --show-toplevel)

if [ ! -d _scripts/known_diffs ]
then
	exit 0
fi

goversion=$(go version | cut -d ' ' -f 3)

for i in $(ls _scripts/known_diffs)
do
	if [ "$goversion" == $i ]
	then
		for j in $(find _scripts/known_diffs/$i -type f)
		do
			git apply $j
		done
	fi
done

