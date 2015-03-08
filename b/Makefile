gobins := haven-b-zbackup-splitter haven-b-gdrivesync haven-b-gdriverestore
shbins := haven-b-archive-bundles haven-b-backup haven-b-init haven-b-restore
outgo := $(foreach bin,$(gobins),out/$(bin))
outsh := $(foreach bin,$(shbins),out/$(bin))

outvendor := out/haven-b-jq out/haven-b-zbackup out/haven-b-gdrive

executables := $(outgo) $(outsh) $(outvendor)
version = $(shell cat VERSION)

pkgbase = haven-v$(version)-$(shell eval '$(shell go env); echo $$GOOS')-$(shell eval '$(shell go env); echo $$GOARCH')
pkgtgz=pkg/$(pkgbase).tar.gz
pkgtxz=pkg/$(pkgbase).tar.xz
pkgs=$(pkgtgz) $(pkgtxz)

all: $(executables)

pkg: $(pkgs)

$(pkgtgz): $(executables) pkg
	tar --transform "s,^out,/$(pkgbase)/," -cxf $@ $^

$(pkgtxz): $(executables) pkg
	tar --transform "s,^out,/$(pkgbase)/," -cJf $@ $^

pkg:
	mkdir $@

out/haven-b-jq:
	cp `which jq` $@

out/haven-b-zbackup:
	cp `which zbackup` $@

out/haven-b-gdrive:
	godep go build -o $@ github.com/prasmussen/gdrive

$(outgo): out/haven-b-%: %/main.go
	godep go build -o $@ $<

$(outsh): out/%: bin/%
	cp $< $@

install: $(executables)
	cp $^ /usr/local/bin

clean:
	rm -fr out
