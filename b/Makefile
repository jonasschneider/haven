gobins := haven-b-zbackup-splitter haven-b-gdrivesync haven-b-gdriverestore
shbins := haven-b-archive-bundles haven-b-backup haven-b-init haven-b-restore
outgo := $(foreach bin,$(gobins),out/$(bin))
outsh := $(foreach bin,$(shbins),out/$(bin))

executables := $(outgo) $(outsh)

all: $(executables)

$(outgo): out/haven-b-%: %/main.go
	go build -o $@ $<

$(outsh): out/%: bin/%
	cp $< $@

install: $(executables)
	cp $^ /usr/local/bin

clean:
	rm -fr out
