# install the PPA here explicitly so we don't always install it in `test/harness`
sudo apt-add-repository --yes ppa:zfs-native/stable
sudo apt-get update
sudo apt-get -y install buffer gnupg-agent spl-dkms
sudo apt-get -y install ubuntu-zfs

export pool=$(basename $vdev)
