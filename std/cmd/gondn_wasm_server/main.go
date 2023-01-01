package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var (
	path = flag.String("path", ".", "the directory containing html, wasm and js files")
	port = flag.Int("port", 9090, "port number to serve")
)

func main() {
	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil || len(*path) == 0 || *port == 0 {
		fmt.Println("Bad argument", err)
		return
	}

	err = http.ListenAndServe(fmt.Sprintf(":%d", *port), http.FileServer(http.Dir(*path)))
	if err != nil {
		fmt.Println("Failed to start server", err)
		return
	}
}
