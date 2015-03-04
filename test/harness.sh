#!/bin/bash


set -eux

repo=$(mktemp -d /tmp/haven-testXXXXX)
#repo=/tmp/test

# initialize the repo
zbackup --non-encrypted init $repo

gdrive_folder_id=$(gdrive folder -t "haven-test $(date)"|grep Id:|cut -d ' ' -f 2)

cat > $repo/backupspec.json <<END
{
  "source": "test/data",
  "z_backupfile": "$repo/backups/mybackup",
  "gdrive_folder_id": "$gdrive_folder_id"
}
END

# do the backup
runner/main.sh $repo/backupspec.json | tee $repo/buplog

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
