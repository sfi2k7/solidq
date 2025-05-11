package solidq

import (
	"strconv"

	"github.com/sfi2k7/blueweb"
)

type response[T any] struct {
	Success  bool           `json:"success"`
	Items    []*Work[T]     `json:"work,omitempty"`
	Error    string         `json:"error"`
	Count    int            `json:"count,omitempty"`
	Channels map[string]int `json:"channels,omitempty"`
}

func StartQueServer[T any](dbpath string, port int) error {

	que, err := OpenQue[T](dbpath)
	if err != nil {
		return err
	}

	api := blueweb.NewRouter()
	api.Post("/solidq/push", func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")
		workid := ctx.Query("id")
		var payload T
		err = ctx.ParseBody(&payload)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}

		err = que.Push(channel, Work[T]{
			Id:   workid,
			Data: payload,
		})
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}

		ctx.Json(response[T]{Success: true})
	})

	api.Get("/solidq/pop/:count", func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")
		count := ctx.Params("count")

		co, _ := strconv.Atoi(count)
		if co < 1 {
			co = 1
		}

		items, err := que.PopWithCount(channel, co)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}

		if items == nil {
			ctx.Json(response[T]{Success: false})
			return
		}

		ctx.Json(response[T]{Success: true, Items: items})
	})

	api.Get("/solidq/count", func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")
		count, err := que.Count(channel)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}
		ctx.Json(response[T]{Success: true, Count: count})
	})

	api.Get("/solidq/reset", func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")
		err := que.ResetChannel(channel)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}
		ctx.Json(response[T]{Success: true})
	})

	api.Get("/solidq/channels", func(ctx *blueweb.Context) {
		channels, err := que.ListChannelsWithCount()
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}
		ctx.Json(response[T]{Success: true, Channels: channels})
	})

	api.Config().SetDev(true).SetPort(port).StopOnInterrupt()

	api.StartServer()

	return nil
}
