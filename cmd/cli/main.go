package main

import (
	"flag"
	"progression1/cmd/pkg/cli"
)

func main() {
	method := flag.String("method", "GET", "HTTP method")
	endpoint := flag.String("endpoint", "/", "API endpoint")
	data := flag.String("data", "", "JSON payload")
	host := flag.String("host", "http://localhost:8080", "API host")
	flag.Parse()
	client := cli.NewClient(*host)
	client.Request(*method, *endpoint, *data)
}
