VERSION = 1.3.0.0

.PHONY: all install clean test coverage

all: ndnd

ndnd: clean
	CGO_ENABLED=0 go build -o ndnd cmd/ndnd/main.go

clean:
	rm -f ndnd coverage.out

test:
	go test ./... -coverprofile=coverage.out

coverage:
	go tool cover -html=coverage.out
