stat test/haven-b-test-linux-v1.xz > /dev/null || curl -L -o test/haven-b-test-linux-v1.xz https://github.com/jonasschneider/haven/releases/download/v0.0.1/haven-b-test-linux-v1.xz
stat test/haven-b-test-rootfs-v1.ext4.xz > /dev/null || curl -L -o test/haven-b-test-rootfs-v1.ext4.xz https://github.com/jonasschneider/haven/releases/download/v0.0.1/haven-b-test-rootfs-v1.ext4.xz

sha256sum -c <<EOF
6ed94a556b055429d78f24d892ef21f3db6612cfa35d31a0fe644ec0883853e3  test/haven-b-test-linux-v1.xz
f046560c74c94dd3522f4838c00ebbada24500a017ac6cb9c5973a4995c528bf  test/haven-b-test-rootfs-v1.ext4.xz
EOF

stat test/haven-b-test-linux-v1 > /dev/null || unxz -k test/haven-b-test-linux-v1.xz
stat test/haven-b-test-rootfs-v1.ext4 > /dev/null || unxz -k test/haven-b-test-rootfs-v1.ext4.xz

chmod +x test/haven-b-test-linux-v1

rootfs_cow=$(mktemp /tmp/testrootfscowXXXXXX)
rm $rootfs_cow # need it to be absent, else COW gets confused

cmd="zpool create diving /dev/ubdb"

script=$(mktemp /tmp/testscriptXXXXXX)
echo "date" > $script

test/haven-b-test-linux-v1 \
  ubd0=$rootfs_cow,test/haven-b-test-rootfs-v1.ext4 \
  ubd1=$vdev \
  rw mem=256M \
  init=/bin/bash \
  con=pty &

linuxpid=$!

# find out which pty we spawned it on... ouch
sleep 1
linuxtty=$(sudo gdb --batch --pid $linuxpid -ex "print /s ptsname(12)" 2> /dev/null|grep buffer|sed 's/^[^\"]*\"//' | sed 's/\".*//')

# spawn this to keep the console open.. sigh
python -c "import serial,os,sys,time
ser=serial.Serial()
ser.port='$linuxtty'
ser.open()
while True:
  time.sleep(1)" &

# so that exec works
export HAVEN_B_TEST_UML_PTS=$linuxtty

test/uml-exec mount -t proc proc /proc
test/uml-exec zpool status |& grep "no pools available"

# enable aliases, and replace sudo with running things in the UML
shopt -s expand_aliases
alias sudo="`pwd`/test/uml-exec"

pool=diving
vdev=/dev/ubdb # conform to the inner file system
kill_me=$linuxpid
