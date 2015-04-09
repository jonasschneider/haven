set -eux
set -o pipefail
shopt -s dotglob

pool=$1

zfs create $pool/data

dd if=/dev/urandom of=/$pool/data/x bs=512 count=8

gdrive_folder_name="haven-test $(date)"

zfs snapshot $pool/data@1
haven-b-backup $pool/data@1 firstbackup
echo zwei > /$pool/data/x
zfs snapshot $pool/data@2
expected=$(cd /$pool/data; tar c * | sha1sum)
haven-b-backup $pool/data@2 secondbackup > /tmp/bupname

sleep 4
# now, a crash happens
zfs destroy -fr $pool/data

# now attempt to restore
haven-b-gdrive download --stdout -i $(cat /tmp/bupname) | unxz | zfs recv $pool/data@2

actual=$(cd /$pool/data/.zfs/snapshots/2; tar c * | sha1sum)

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

rm -fr $tmp
