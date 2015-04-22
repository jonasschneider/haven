set -eux
set -o pipefail
shopt -s dotglob

testfolder=0B-39XlY-_MIBfklocThEV0Vqamk5YncwUUstWlNNMWpHQVBHNGxoSXdvcGoxSzJVenVNU0E

pool=$1

sudo zfs create $pool/data
sudo chown `whoami` /$pool/data
report1=$(mktemp /tmp/reportXXXXXX)
report2=$(mktemp /tmp/reportXXXXXX)
export GNUPG_HOME=$(mktemp -d /tmp/gpgXXXXXX)

dd if=/dev/urandom of=/$pool/data/x bs=512 count=8

echo "Key-Type: RSA
Key-Length: 1024
Subkey-Type: ELG-E
Subkey-Length: 1024
Name-Real: Joe Tester
Name-Comment: with stupid passphrase
Name-Email: joe@foo.bar
Expire-Date: 0
Passphrase: abc
%commit" | gpg --gen-key --batch
recipient=joe@foo.bar

sudo zfs snapshot $pool/data@1
sudo env snapshot=$pool/data@1 name=firstbackup gdrive_folder=$testfolder recipient=$recipient haven-b-backup > $report1
echo zwei > /$pool/data/x
sudo zfs snapshot $pool/data@2
expected=$(cd /$pool/data; tar c * | sha1sum)
#sudo env snapshot=$pool/data@2 name=secondbackup gdrive_folder=$testfolder recipient=mail@jonasschneider.com haven-b-backup > $report2

# now, a crash happens
sleep 2
sudo zfs destroy -fr $pool/data

# now attempt to restore
filename=$(cat $report1|grep "Completed Backup" | cut -d ':' -f 2)
echo $filename
#unxz | sudo zfs recv $pool/data@2

actual=$(cd /$pool/data/.zfs/snapshot/2; tar c * | sha1sum)

if [ "$expected" != "$actual" ]; then
  echo expected $expected, but got $actual
  exit 1
fi

sleep 2
sudo zfs destroy -fr $pool/data
