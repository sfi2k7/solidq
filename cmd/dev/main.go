package main

import (
	"fmt"
	"time"

	"github.com/sfi2k7/solidq/client"
)

func worker(ctx client.SolidContext) string {
	fmt.Println(ctx.Count("test_channel"))
	fmt.Println(ctx.ListChannels())

	id := ctx.CurrentWork()
	fmt.Println("Id is being worked on", id)
	return "next_chnanel"
}

func main() {
	c, err := client.NewClient("http://localhost:8080/")
	if err != nil {
		panic(err)
	}

	c.WorkLoop("test_channel", worker, time.Second*1)
	// if err = c.Push("test_channel", "212"); err != nil {
	// 	fmt.Println("Error pushing into queue")
	// }

	// channels, err := c.ListChannels()
	// if err != nil {
	// 	panic(err)
	// }

	// ids, err := c.Pop("test_channel")
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("ids", ids)

	// fmt.Println(channels)
}
