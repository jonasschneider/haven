#!/bin/bash

set -eux
set -o pipefail
shopt -s dotglob

tmp=$(mktemp -d /tmp/haven-testXXXXX)
repo=$tmp/repo

gdrive_folder_name="haven-test $(date)"
haven-b-init $repo $gdrive_folder_name > $tmp/backupspec.json

haven-b-backup $tmp/backupspec.json test/data firstbackup
haven-b-backup $tmp/backupspec.json test/data secondbackup

# now, a crash happens
rm -fr $tmp/repo

# now attempt to restore
haven-b-restore $tmp/backupspec.json secondbackup $tmp/restored

expected=$(cd test/data; tar c * | sha1sum)
actual=$(cd $tmp/restored; tar c * | sha1sum)

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

rm -fr $tmp
