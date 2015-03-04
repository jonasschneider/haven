install:
	go build -o /tmp/haven-b-zbackup-splitter zbackup-splitter/*.go
	go build -o /tmp/haven-b-gdrivesync gdrivesync/*.go
	go build -o /tmp/haven-b-gdriverestore gdriverestore/*.go

	mv /tmp/haven-b-zbackup-splitter /tmp/haven-b-gdrivesync /tmp/haven-b-gdriverestore /usr/local/bin/
	cp bin/haven-b-archive-bundles bin/haven-b-backup bin/haven-b-init bin/haven-b-restore /usr/local/bin/
