PACKAGE = github.com/eric135/YaNFD
VERSION = 0.0.1
COMMIT != git rev-parse --short HEAD
DATE != date

.PHONY: all clean test coverage

all: yanfd

yanfd: clean
	go build -ldflags "-X 'main.Version=${VERSION}-${COMMIT}' -X 'main.BuildTime=${DATE}'" ${PACKAGE}/cmd/yanfd

clean:
	rm -f yanfd coverage.out

test:
	go test ./... -coverprofile=coverage.out

coverage:
	go tool cover -html=coverage.out
