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
# - we can't use gdb since travis doesn't allow setting the yama ptrace scope
# - we can't use an external thing calling ptsname on /proc/X/fd/Y since that will alloc a new pty
# soo.. sigh and just use the most recent one... i wonder if this even works
sleep 1
linuxtty=$(ls /dev/pts/*|grep -v ptmx|sort -n|tail -1)

# spawn this to keep the console open.. sigh
python -c "import serial,os,sys,time
ser=serial.Serial()
ser.port='$linuxtty'
ser.open()
while True:
  time.sleep(1)" &
watcherpid=$!

# so that exec works
export HAVEN_B_TEST_UML_PTS=$linuxtty

test/scripts/uml-exec mount -t proc proc /proc
test/scripts/uml-exec zpool status |& grep "no pools available"

# could do it with an alias, but then it doesn't work in subprocesses that use shell
lepath=$(mktemp -d /tmp/tempXXXXXX)
ln -s `pwd`/test/scripts/uml-exec $lepath/sudo
export PATH="$lepath:$PATH"

pool=diving
vdev=/dev/ubdb # conform to the inner file system
kill_me_1=$linuxpid
kill_me_2=$watcherpid
