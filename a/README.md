Strategy A is the most crude and hacky one, but has also been in use for longest. (Also, there are no integration tests.)

Follow the following checklist on a fairly recent Ubuntu-ish Linux box in order to get weekly incremental updates and a full update every three months, created using [Duplicity](http://duplicity.nongnu.org/).

- Sign up for a [HubiC](https://hubic.com/en/offers/) account (I recommend the 10TB plan).
- Following the [HubiC API docs](https://api.hubic.com/sandbox/), create an OAuth client application. Use whatever trickery necessary (i.e. the Ruby oauth client libraries, or just cURL) to get the OAuth tokens. You don't need to do this ever again.
- Configure `swiftauth-server` at the places marked with `XXX`. These parameters should also be kept in a safe place.
- Install `swiftauth-server` to `/root/bin/swiftauth-server`.
- Ensure `swiftauth-server` is always running. You can do this with an Upstart job like this:

        # cat /etc/init/haven-swiftauth-server.conf
        description "haven swiftauth server"

        respawn
        respawn limit 1000 60

        exec /root/bin/swiftauth-server

- For reporting, sign up for a [Postmark](https://postmarkapp.com/) account. They will give you an API token. Place that into `/etc/postmark-api-token` -- the contents should look like a UUID.
- Install `report` to `/usr/local/bin/report` and customize the sender and recipient addresses. This is a convenience script for people behind NATs who just want to send a bit of email. `$ echo Test report | report test_service` tests the email setup.
- Install `backup-tank` to `/root/bin/backup-tank`, and configure the `swift://` destination address.
- Generate a passphrase (`pwgen` should do) and put it in `/.backupkey` (or change the path in `backup-tank`.)
- If your ZFS dataset name isn't `tank`, change it in the last line of `backup-tank`.
- Finally, add a cronjob to trigger the backup:

        # cat /etc/cron.weekly/backup

        #!/bin/bash
        declare -x PATH="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
        /root/bin/backup-tank
