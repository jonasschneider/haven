gobins := haven-b-upload
shbins := haven-b-backup haven-b-restore
outgo := $(foreach bin,$(gobins),out/$(bin))
outsh := $(foreach bin,$(shbins),out/$(bin))

executables := $(outgo) $(outsh)
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

.SECONDEXPANSION:
$(outgo): out/%: $$(notdir $$*)/*.go
	godep go build -o $@ $*/*.go

$(outsh): out/%: bin/%
	cp $< $@

install: $(executables)
	cp $^ /usr/local/bin

clean:
	rm -fr out

savedeps:
	rm -fr /vagrant/_go/src/github.com/jonasschneider/haven/b
	mkdir -p /vagrant/_go/src/github.com/jonasschneider/haven/b
	bash -c "cp -r /vagrant/haven-* /vagrant/_go/src/github.com/jonasschneider/haven/b"
	cp -r /vagrant/Godeps /vagrant/_go/src/github.com/jonasschneider/haven/b || true

	cd /vagrant/_go/src/github.com/jonasschneider/haven/b && git init && godep save github.com/jonasschneider/haven/b/...
	mv /vagrant/Godeps /vagrant/Godeps_old || true
	mv /vagrant/_go/src/github.com/jonasschneider/haven/b/Godeps /vagrant
	rm -fr /vagrant/Godeps_old
