package main

import (
	"flag"
	"CSI-test/pkg/driver"
)

func init() {
	flag.Set("logtostderr", "true")
}

var (
	endpoint = flag.String("endpoint", "http://127.0.0.1:9000", "CSI endpoint")
	nodeID = flag.String("nodeid", "", "node_id")
)

func main() {
	flag.Parse()

	driver.New(*)
}
