PACKAGE=github.com/eric135/YaNFD

.PHONY: all clean test coverage

all: yanfd

yanfd: clean
	go build ${PACKAGE}/cmd/yanfd

clean:
	rm -f yanfd coverage.out

test:
	go test ./... -coverprofile=coverage.out

coverage:
	go tool cover -html=coverage.out
