package main

import (
	"flag"
	"fmt"

	"github.com/sfi2k7/solidq"
)

func main() {
	path := flag.String("db", "solidq.db", "Path to the database file")
	port := flag.Int("port", 8080, "Port to listen on")
	version := flag.Bool("version", false, "Show version information")
	if *version {
		fmt.Println("SolidQ version 0.0.2")
		return
	}

	flag.Parse()

	fmt.Println("Starting SolidQ server...")
	err := solidq.StartQueServer[solidq.Payload](*path, *port)
	if err != nil {
		panic(err)
	}
}
