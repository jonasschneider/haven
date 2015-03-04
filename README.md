A set of procedures and tools for backing up personal storage servers

- Multiple redundant offsite backups using different technologies & remote storage providers
  (Don't put all your eggs in one basket)
- Full round-trip integration-test-style backup verification after every backup
- Email notifications when things go wrong...

Assumptions about your environment:
- Old commodity hardware
- Very slow uplink (like a residential ISP)
- Dataset < ~10TB
- Recommended: Source data is a ZFS dataset (for snapshotting)

Backup strategy 1:
- Cronjob
- Duplicity
- node-hubic-swiftauth
- [HubiC](https://hubic.com/en/offers/) is a cloud storage product from European hosting company OVH offering 10TB of OpenStack Swift object storage for 10€/month

Backup strategy 2:
- Cronjob (TODO: should we treat cron as a SPOF? Maybe do something like https://addons.heroku.com/deadmanssnitch)
- ZBackup
- Google Drive (or maybe Glacier)
