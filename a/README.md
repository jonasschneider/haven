Strategy A is the most crude and hacky one, has no integration tests, but been in active use for pretty long.

Follow the below checklist on a fairly recent Ubuntu-ish Linux box (other platforms should work with adjustments) in order to get weekly incremental updates and a full update every three months, created using [Duplicity](http://duplicity.nongnu.org/).

- Install a recent version (>= 0.7.0) of [Duplicity](http://duplicity.nongnu.org/) -- the one that comes with Ubuntu 14.04 is too old and doesn't support the HubiC backend.
- Sign up for a [HubiC](https://hubic.com/en/offers/) account (I recommend the 10TB plan).
- In your HubiC account settings, create an application in the "Developer" panel. Put your hubic account data into `/root/.hubic_credentials`:

        root@qubit:~# cat .hubic_credentials
        [hubic]
        email = <your hubic email>
        password = <your hubic password>
        client_id = <client_id>
        client_secret = <secret>
        redirect_uri = http://localhost/

- For reporting, sign up for a [Postmark](https://postmarkapp.com/) account. They will give you an API token. Place that into `/etc/postmark-api-token` -- the contents should look like a UUID.
- Install `report` to `/usr/local/bin/report` and customize the sender and recipient addresses. This is a convenience script for people behind NATs who just want to send a bit of email. `$ echo Test report | report test_service` tests the email setup.
- Install `backup-tank` to `/root/bin/backup-tank`.
- Generate a passphrase (`pwgen` should do) and put it in `/.backupkey` (or change the path in `backup-tank`.)
- If your ZFS dataset name isn't `tank`, change it in the last line of `backup-tank`.
- Finally, add a cronjob to trigger the backup:

        # cat /etc/cron.weekly/backup

        #!/bin/bash
        declare -x PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
        /root/bin/backup-tank
