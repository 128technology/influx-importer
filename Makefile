BINARY=dist/influx-importer
PACKAGE=./cmd/main.go
BUILD=0.0.0-Development
LDFLAGS=-ldflags "-X main.build=${BUILD}"

.PHONY: build clean

build: clean
	go build ${LDFLAGS} -o ${BINARY} ${PACKAGE}

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi