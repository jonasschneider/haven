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

| **Strategy** | Trigger | Deduplicator | Uploader                    | Storage Provider | Storage costs                 | Realtime restore |
|----------|---------|--------------|-----------------------------|------------------|-------------------------------|------------------|
| **A**        | Cron    | Duplicity    | Duplicity (OpenStack Swift) | hubiC            | 0.001€/GB/mo (10TB at 10€/mo) | ✓                |
| **B**        | Cron    | ZBackup      | (custom)                    | Google Drive     | 0.01$/GB/mo (1TB at 10$/mo)   | ✓                |
| **C        | Cron    | ZFS          | s3cmd                       | Amazon S3        | 0.01$/GB/mo (flexible)        | very slow        |

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
- [ZBackup](https://github.com/zbackup/zbackup)
- (custom sync to) Google Drive

ZBackup is a deduplicating backup tool. It provides a very simple interface
which you can just pipe huge files into -- we feed it a tar archive of all our
data. ZBackup calculates a rolling checksum of the data and references any
sections already seen. This results in global deduplication. There is a bit of
tooling built around zbackup in order to integrate it with Google Drive and to
split a huge backup into partial backups so file transfer can begin before
ZBackup has committed the entire backup to its local repository.

## Backup strategy C (not yet implemented)
- Cronjob
- `zfs send [-i]` -> s3cmd to S3 -> Glacier

This might be a good idea to implement in the future. We previously
experimented with sending ZFS snapshots to Amazon EC2 instances and using EBS
snapshots as long-term storage. However, EBS pricing prohibits this method
from working on very large datasets. Also, sending data to EC2 unencrypted
violates our threat model. The alternative is to `zfs send` the (maybe-
incremental) snapshot but simply compress and pipe it into `s3cmd`, a tool to
upload files to Amazon S3. It can accept large uploads from stdin without
having the entire file on disk. That way, we can get our snapshots onto S3 and
then use Amazon's Lifecycle Policies to immediately migrate all the data to
Glacier, where storage is only $0.01/GB/mo.
