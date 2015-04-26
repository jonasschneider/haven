# A safe haven for your data [![Build Status](https://travis-ci.org/jonasschneider/haven.svg?branch=master)](https://travis-ci.org/jonasschneider/haven)
Haven is a set of concrete procedures and tools for backing up a personal
storage server. It manages redundant offsite backups using independent
software and remote storage providers. Paranoid integration-test-style
restoration drills are automatically performed to ensure that your data is
actually backed up, no matter what.

Despite all our engineering efforts, technology occasionally fails. Don't put
all your eggs in one basket. Your stuff is too valuable for that.

## Philosophy

Haven assumes chaos. The goal is to provide a fully automated
backup solution with the following properties:
- Based on cheap-ish cloud storage products (~10$/mo for a 1TB backup)
- No single points of failure (cloud provider, software bugs) introduced
- No unencrypted data ever leaves the local network

In order to achieve no single points of failure, Haven also doesn't have a
coherent installation process (it's not a bug, it's a feature!) -- you'll have
to set up a set of backup strategy that you want to use.

Haven's backup strategies are designed so that they are completely independent
of each other, and also as *different* as possible. The only thing in common
for all the strategies is this document.

This ensures that a bug *anywhere*, be it within Haven, within `tar`, or
within Linux, cannot compromise your entire backup. Of course, this is a
theoretical goal. In practice, ZFS and therefore ZFS on Linux is the last
common point on the data path that is shared by all the strategies.

## Architecture

(fancy diagram here)

Haven assumes an environment roughly conforming to this description:
- There's a box on your local network (let's call it the **host**).
- That box is always-on and runs something like Ubuntu LTS and has at least
  modest hardware (a 1.6GHz Atom & 4GB RAM should do)
- Connected to the host are a bunch of hard drives sufficent to store all your
  data. I personally use super-cheap "external USB"-type drives that are
  super-slow and unreliable but cheap.
- These drives are configured on the host to form a zpool (hopefully as a
  raidz, if you use shitty drives)
- All your valuable data is stored in this zpool. (I use a single zfs dataset,
  you might have to shuffle around a bit to work with multiple datasets)
- The host is connected to the public internet via an unreliable, slow uplink,
  but potentially a fast downlink (like a typical connection provided by a
  residential ISP)
- Your dataset is small enough to fit through the uplink in non-astronomic
  time (read: a couple of weeks is probably OK)
- Planned: other devices in the local network (like your laptop) can back up
  to the host and have their data be included in the main backup as well

## Strategy overview

| Strategy | Trigger | Deduplicator | Uploader                    | Storage Provider | Storage costs                 | Realtime restore |
|----------|---------|--------------|-----------------------------|------------------|-------------------------------|------------------|
| **A**        | Cron    | Duplicity    | Duplicity               | hubiC            | 0.001€/GB/mo (10TB at 10€/mo) | ✓                |
| **B**        | Cron    | ZFS          | (custom)                    | Google Drive     | 0.01$/GB/mo (1TB at 10$/mo)   | ✓                |

## Backup strategy A (`a/`)
- Cronjob
- [Duplicity](http://duplicity.nongnu.org/)
- [HubiC](https://hubic.com/en/offers/) is a cloud storage product from
  European hosting company OVH offering 10TB of OpenStack Swift object storage
  for 10€/month

This is the most battle-tested strategy, and while it's somewhat janky to set
up with a couple of *short* intertwined bash scripts, it's been known to work,
and the backup runtime code is very easy to inspect. Besides Duplicity, there
are very few moving parts.

No automatic disaster recovery simulation is implemented yet, so you'll have to
trust Duplicity to do its work or occasionally try a restore yourself.

## Backup strategy B (`b/`)
- Cronjob
- `zfs send | xz | gpg | haven-b-upload`
- Upload to [Google Drive](https://drive.google.com/drive/u/0/my-drive) (via custom uploader)

Since ZFS is assumed to be the file system storing the live data, we can
exploit some of its unique features, like snapshots. We previously
experimented with sending ZFS snapshots to Amazon EC2 instances and using EBS
snapshots of the vdevs as long-term storage. However, EBS pricing makes this method
prohibitively expensive for large datasets. Also, sending data to EC2
unencrypted violates our threat model.

The alternative is to `zfs send` the snapshot itself but simply compress, encrypt and
store it as a single giant file. This makes it impossible to lose parts of the
file without us immediately noticing (unlike Duplicity's volume system). We
can verify the snapshot by taking a huge hash over the entire stream, which we
can calculate efficiently without using any disk. In addition, Google Drive
provides md5summed uploads, which cause us to immediately detect corruption
during the upload, without having to finish the upload first. They also take
an md5 over the entire uploaded file, which we can access and verify.
