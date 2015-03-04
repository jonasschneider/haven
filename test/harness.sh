#!/bin/bash

set -eux
shopt -s dotglob

tmp=$(mktemp -d /tmp/haven-testXXXXX)
#repo=/tmp/test
repo=$tmp/repo

# initialize the repo
zbackup --non-encrypted init $repo

gdrive_folder_id=$(gdrive folder -t "haven-test $(date)"|grep Id:|cut -d ' ' -f 2)

cat > $tmp/backupspec.json <<END
{
  "source": "test/data",
  "repo": "$repo",
  "gdrive_folder_id": "$gdrive_folder_id"
}
END

# TEST: backup preserves file contents & metadata
# runner/main.sh $tmp/backupspec.json firstbackup
# runner/restore $tmp/backupspec.json firstbackup $tmp/rest

# expected=$(cd test/data; tar c * | sha1sum)
# actual=$(cd $tmp/rest; tar c * | sha1sum)

# if [ "$expected" != "$actual" ]; then
#   echo expected $expected, but got $actual
#   exit 1
# fi

# TEST: subsequent backups deduplicate
runner/main.sh $tmp/backupspec.json firstbackup
runner/main.sh $tmp/backupspec.json secondbackup
runner/restore $tmp/backupspec.json secondbackup $tmp/rest

expected=$(cd test/data; tar c * | sha1sum)
actual=$(cd $tmp/rest; tar c * | sha1sum)

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

rm -fr $tmp

echo "all is well!"
