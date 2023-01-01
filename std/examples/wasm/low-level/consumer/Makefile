all: main.wasm

main.wasm: main.go
	GOOS=js GOARCH=wasm go build -o main.wasm

clean:
	rm main.wasm

serve:
	gondn_wasm_server -path . -port 9090
