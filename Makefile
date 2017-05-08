BINARY=influx-importer

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

LDFLAGS=-ldflags "-X main.build=${VERSION}"

.PHONY: build install clean

build: clean
	go build ${LDFLAGS} -o ${BINARY}

install:
	go install

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi