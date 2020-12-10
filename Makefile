PACKAGE=github.com/eric135/YaNFD

.PHONY: all clean test

all: yanfd

yanfd: clean
	go build ${PACKAGE}/cmd/yanfd

clean:
	rm -f yanfd

test:
	go test ./...
