package main

import (
	"CSI-test/pkg/driver"
	"flag"
	"log"
	"os"
)

func init() {
	flag.Set("logtostderr", "true")
}

var (
	endpoint = flag.String("endpoint", "http://127.0.0.1:9000", "CSI endpoint")
	nodeID   = flag.String("nodeid", "", "node_id")
)

func main() {
	flag.Parse()

	driver, err := driver.New(*nodeID, *endpoint)
	if err != nil {
		log.Fatal(err)
	}
	driver.Run()
	os.Exit(0)
}
