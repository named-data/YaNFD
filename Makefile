VERSION = 1.3.0.0
STD_PACKAGE = github.com/named-data/ndnd/std

.PHONY: all install clean test coverage

all: ndnd

ndnd: clean
	CGO_ENABLED=0 go build -o ndnd cmd/ndnd/main.go

generate:
	go generate ./...

install:
	install -m 755 ndnd /usr/local/bin

clean:
	rm -f ndnd coverage.out

clean-gen:
	rm -f gondn_tlv_gen

test:
	go test ./... -coverprofile=coverage.out

coverage:
	go tool cover -html=coverage.out

gondn_tlv_gen: clean-gen
	go build ${STD_PACKAGE}/cmd/gondn_tlv_gen
	go install ${STD_PACKAGE}/cmd/gondn_tlv_gen
