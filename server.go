package solidq

import (
	"github.com/sfi2k7/blueweb"
)

type response[T any] struct {
	Success  bool           `json:"success"`
	Work     *Work[T]       `json:"work,omitempty"`
	Error    string         `json:"error"`
	Count    int            `json:"count,omitempty"`
	Channels map[string]int `json:"channels,omitempty"`
}

func StartQueServer[T any](dbpath string, port int) error {
	// exitch := make(chan os.Signal, 2)
	// signal.Notify(exitch, os.Interrupt, syscall.SIGTERM)

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

	api.Get("/solidq/pop", func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")

		work, err := que.Pop(channel)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}

		if work == nil {
			ctx.Json(response[T]{Success: false})
			return
		}

		if len(work.Id) == 0 {
			ctx.Json(response[T]{Success: false})
			return
		}

		ctx.Json(response[T]{Success: true, Work: work})
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
