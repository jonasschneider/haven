set -eux
set -o pipefail
shopt -s dotglob

tmp=$(mktemp -d /tmp/haven-testXXXXX)
repo=$tmp/repo
data=$tmp/data
mkdir $data
dd if=/dev/urandom of=$data/x bs=512 count=20

haven-b-init $repo "haven-test $(date)" > $tmp/backupspec.json
haven-b-backup $tmp/backupspec.json $data firstbackup

# try a resume that crashes during sync
! HAVEN_B_CRASHAT=onerestore haven-b-restore $tmp/backupspec.json firstbackup $tmp/restored

# resume
haven-b-restore $tmp/backupspec.json firstbackup $tmp/restored

expected=$(cd $data; tar c * | sha1sum)
actual=$(cd $tmp/restored; tar c * | sha1sum)

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

rm -fr $tmp
