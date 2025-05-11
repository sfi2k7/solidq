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

type SeverOptions struct {
	Path        string
	Port        int
	CrossOrigin bool
	Auth        []blueweb.Middleware
	Secret      string
}

var defaultOptions = SeverOptions{
	Path:        "solidq.db",
	Port:        8080,
	CrossOrigin: true,
	Auth:        nil,
}

func StartQueServer[T any](options *SeverOptions) error {

	if options == nil {
		options = &defaultOptions
	}

	middle := func(fn func(ctx *blueweb.Context)) blueweb.Handler {
		return func(ctx *blueweb.Context) {
			//Cross-Origin Resource Sharing (CORS)
			if options.CrossOrigin {
				ctx.SetHeader("Content-Type", "application/json")
				ctx.SetHeader("Access-Control-Allow-Origin", "*")
				ctx.SetHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
				if ctx.Method() == "OPTIONS" {
					ctx.Status(200)
					return
				}
			}

			//Simple Authentication
			if options.Secret != "" {
				var token = ctx.Query("secret")
				if token == "" {
					token = ctx.Query("api_key")
					if token == "" {
						token = ctx.Query("key")
						if token == "" {
							token = ctx.Query("access_token")
							if token == "" {
								token = ctx.Header("Authorization")
							}
						}
					}
				}

				if len(token) == 0 || token != options.Secret {
					ctx.Json(response[T]{Error: "Unauthorized"})
					return
				}
			}

			//Custom Authentication
			if len(options.Auth) > 0 {
				success := options.Auth[0](ctx)
				if !success {
					ctx.Json(response[T]{Error: "Unauthorized"})
					return
				}
			}

			//Call the handler
			fn(ctx)
		}
	}

	que, err := OpenQue[T](options.Path)
	if err != nil {
		return err
	}

	api := blueweb.NewRouter()

	api.Post("/solidq/push", middle(func(ctx *blueweb.Context) {
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
	}))

	api.Get("/solidq/pop/:count", middle(func(ctx *blueweb.Context) {
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
	}))

	api.Get("/solidq/count", middle(func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")
		count, err := que.Count(channel)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}
		ctx.Json(response[T]{Success: true, Count: count})
	}))

	api.Get("/solidq/reset", middle(func(ctx *blueweb.Context) {
		channel := ctx.Query("channel")
		err := que.ResetChannel(channel)
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}
		ctx.Json(response[T]{Success: true})
	}))

	api.Get("/solidq/channels", middle(func(ctx *blueweb.Context) {
		channels, err := que.ListChannelsWithCount()
		if err != nil {
			ctx.Json(response[T]{Error: err.Error()})
			return
		}
		ctx.Json(response[T]{Success: true, Channels: channels})
	}))

	api.Config().SetDev(true).SetPort(options.Port).StopOnInterrupt()

	api.StartServer()

	return nil
}
