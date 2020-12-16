PACKAGE=github.com/eric135/YaNFD

.PHONY: all clean test coverage

all: yanfd

yanfd: clean
	go build ${PACKAGE}/cmd/yanfd

clean:
	rm -f yanfd

test:
	go test ./... -coverprofile=coverage.out

coverage:
	go tool cover -func=coverage.out
