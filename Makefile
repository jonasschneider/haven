install:
	go build -o /tmp/haven-b-zbackup-splitter zbackup-splitter/*.go
	go build -o /tmp/haven-b-gdrivesync gdrivesync/*.go
	go build -o /tmp/haven-b-gdriverestore gdriverestore/*.go

	mv /tmp/{haven-b-zbackup-splitter,haven-b-gdrivesync,haven-b-gdriverestore} /usr/local/bin
	cp bin/{haven-b-archive-bundles,haven-b-backup,haven-b-init,haven-b-restore} /usr/local/bin/
