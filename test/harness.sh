#!/bin/sh

set -eux

repo=$(mktemp -d /tmp/haven-testXXXXX)

# initialize the repo
zbackup --non-encrypted init $repo

# do the backup
runner/main.sh test/data $repo/backups/mybackup

# compare the restored data
expected=$(tar c test/data | sha1sum)
actual=$(zbackup restore $repo/backups/mybackup | sha1sum)

rm -fr $repo

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

echo "all is well!"
