# A safe haven for your data [![Build Status](https://travis-ci.org/jonasschneider/haven.svg?branch=master)](https://travis-ci.org/jonasschneider/haven)
Haven is a set of procedures and tools for backing up a personal storage
server. It manages redundant offsite backups using independent software and
remote storage providers. Paranoid integration-test-style restoration drills
are automatically performed to ensure that your data is actually backed up, no
matter what.

Despite all our engineering efforts, technology occasionally fails. Don't put
all your eggs in one basket. Your stuff is too valuable for that.

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

Under these specific conditions, Haven will provide you with an automatic
backup solution that provides the following properties:
- Based on cheap-ish cloud storage products (~10$/mo for a 1TB backup)
- No additional single points of failure (cloud provider, software bugs)
  besides ZFS
- No unencrypted data ever leaves the local network

In order to achieve no single points of failure, Haven also doesn't have a
coherent installation process (it's not a bug, it's a feature!) -- you'll have
to set up each backup strategy that you want to use.

Strategies are designed so that they are completely independent of each other,
and also as *different* as possible. The only thing in common for all the strategies
is this document.

This ensures that a bug *anywhere*, be it within Haven, within `tar`, or
within Linux, cannot compromise your entire backup. Of course, this is a
theoretical goal. In practice, ZFS and therefore ZFS on Linux is the last
common point on the data path that is shared by all the strategies.


## Backup strategy A:
- Cronjob
- Duplicity
- [HubiC](https://hubic.com/en/offers/) is a cloud storage product from
  European hosting company OVH offering 10TB of OpenStack Swift object storage
  for 10€/month

## Backup strategy B:
- Cronjob (TODO: should we treat cron as a SPOF? Maybe do something like
  https://addons.heroku.com/deadmanssnitch)
- ZBackup
- Google Drive

## Backup strategy C:
- Cronjob
- `zfs send [-i]` -> s3cmd to S3 -> Glacier
