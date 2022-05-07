PACKAGE = github.com/zjkmxy/go-ndn

gondn_tlv_gen: clean
	go build ${PACKAGE}/cmd/gondn_tlv_gen
	go install ${PACKAGE}/cmd/gondn_tlv_gen

clean:
	-rm -rf ./gondn_tlv_gen
