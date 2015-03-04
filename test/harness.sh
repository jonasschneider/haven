#!/bin/bash

set -eux

#repo=$(mktemp -d /tmp/haven-testXXXXX)
repo=/tmp/test

# initialize the repo
#zbackup --non-encrypted init $repo

# do the backup
runner/main.sh /Users/jonas/Pictures/2014 $repo/backups/mybackup5 | tee $repo/buplog

# compare the restored data
rmdir $repo/bundles
du -sh $repo/archived_bundles
mv $repo/archived_bundles $repo/bundles
expected=$(tar c test/data | sha1sum)
zbackup restore $repo/backups/mybackup > $repo/rest
actual=$(cat $repo/rest | sha1sum)

cat $repo/buplog|grep du:

rm -fr $repo

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi



echo "all is well!"
