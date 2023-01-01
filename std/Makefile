PACKAGE = github.com/zjkmxy/go-ndn

gondn_tlv_gen: clean
	go build ${PACKAGE}/cmd/gondn_tlv_gen
	go install ${PACKAGE}/cmd/gondn_tlv_gen

gondn_wasm_server: clean
	go build ${PACKAGE}/cmd/gondn_wasm_server
	go install ${PACKAGE}/cmd/gondn_wasm_server

generate: clean gondn_tlv_gen
	go generate ./...

clean:
	-rm -rf ./gondn_tlv_gen
