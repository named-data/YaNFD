PACKAGE = github.com/named-data/YaNFD
VERSION = 0.0.1
# COMMIT = git rev-parse --short HEAD
# DATE != date

.PHONY: all install clean test coverage

all: yanfd

yanfd: clean
	go build -ldflags "-X 'main.Version=${VERSION}'" ${PACKAGE}/cmd/yanfd

install:
	install -m 755 yanfd /usr/local/bin
	mkdir -p /usr/local/etc/ndn
	install -m 644 yanfd.toml.sample /usr/local/etc/ndn

clean:
	rm -f yanfd coverage.out

test:
	go test ./... -coverprofile=coverage.out

coverage:
	go tool cover -html=coverage.out

cleanui:
	rm -f yanfdui

yanfdui: cleanui
	go build -ldflags "-X 'main.Version=${VERSION}'" ${PACKAGE}/cmd/yanfdui