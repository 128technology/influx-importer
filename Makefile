BINARY=influx-importer
BUILD=0.0.0-Development
LDFLAGS=-ldflags "-X main.build=${BUILD}"

.PHONY: build clean

build: clean
	go build ${LDFLAGS} -o ${BINARY}

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi