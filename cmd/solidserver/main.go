package main

import (
	"flag"
	"fmt"

	"github.com/sfi2k7/solidq"
)

func main() {
	appname := flag.String("app", "core", "Application name")
	port := flag.Int("port", 8080, "Port to listen on")
	version := flag.Bool("version", false, "Show version information")
	if *version {
		fmt.Println("SolidQ version 0.0.3")
		return
	}

	flag.Parse()

	fmt.Println("Starting SolidQ server...")
	options := &solidq.SeverOptions{
		Appname: *appname,
		Port:    *port,
	}

	err := solidq.StartQueServer(options)
	if err != nil {
		panic(err)
	}
}
