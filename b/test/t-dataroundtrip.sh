set -eux
set -o pipefail
shopt -s dotglob

testfolder=0B-39XlY-_MIBfklocThEV0Vqamk5YncwUUstWlNNMWpHQVBHNGxoSXdvcGoxSzJVenVNU0E

pool=$1
report1=$(mktemp /tmp/reportXXXXXX)
report2=$(mktemp /tmp/reportXXXXXX)

sudo zfs create $pool/data
sudo dd if=/dev/urandom of=/$pool/data/x bs=512 count=8
recipient=joe@foo.bar # see test/harness

sudo zfs snapshot $pool/data@1
snapshot=$pool/data@1 name=firstbackup gdrive_folder=$testfolder recipient=$recipient haven-b-backup > $report1
expected1=$(sudo bash -c "cd /$pool/data; tar c * | sha1sum")
sudo bash -c "echo zwei > /$pool/data/x"
sudo zfs snapshot $pool/data@2
expected2=$(sudo bash -c "cd /$pool/data; tar c * | sha1sum")
snapshot=$pool/data@2 name=secondbackup gdrive_folder=$testfolder recipient=$recipient haven-b-backup > $report2

# now, a crash happens
sleep 2
sudo zfs destroy -fr $pool/data

cat $report1
# now attempt to restore
filename=$(cat $report1|grep "In GDrive as" | cut -d ':' -f 2 | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
filename=$filename dest_snapshot=$pool/rest1data@restored1 haven-b-restore

cat $report2
filename=$(cat $report2|grep "In GDrive as" | cut -d ':' -f 2 | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
filename=$filename dest_snapshot=$pool/rest2data@restored2 haven-b-restore

checkout1=/$pool/rest1data/.zfs/snapshot/restored1
checkout2=/$pool/rest2data/.zfs/snapshot/restored2

actual1=$(sudo bash -c "cd $checkout1; tar c * | sha1sum")
actual2=$(sudo bash -c "cd $checkout2; tar c * | sha1sum")

if [ "$expected1" != "$actual1" ]; then
  echo expected $expected1, but got $actual1
  exit 1
fi

if [ "$expected2" != "$actual2" ]; then
  echo expected $expected2, but got $actual2
  exit 1
fi
