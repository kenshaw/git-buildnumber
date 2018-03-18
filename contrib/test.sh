#!/bin/bash

set -e

SRC=$( cd -P "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )

NEW=$SRC/git-buildnumber
OLD=$SRC/git-buildnumber-old

REVS=$(git rev-list --all)
for rev in $REVS; do
  o=$($OLD -rev="$rev")
  n=$($NEW -rev="$rev")
  echo -e "$rev\t$o\t$n"
done
