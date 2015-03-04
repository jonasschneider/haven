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

# do the backup
runner/main.sh $tmp/backupspec.json firstbackup | tee $tmp/buplog

# compare the restored data
runner/restore $tmp/backupspec.json firstbackup $tmp/rest

expected=$(cd test/data; tar c * | sha1sum)
actual=$(cd $tmp/rest; tar c * | sha1sum)

cat $tmp/buplog|grep du:

rm -fr $repo

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

echo "all is well!"
