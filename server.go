package solidq

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sfi2k7/blueweb"
)

type response struct {
	Success  bool           `json:"success"`
	Ids      []string       `json:"ids,omitempty"`
	Error    string         `json:"error"`
	Count    int            `json:"count,omitempty"`
	Channels map[string]int `json:"channels,omitempty"`
	Apps     []string       `json:"apps,omitempty"`
	IsPaused bool           `json:"isPaused"`
	Took     string         `json:"took"`
}

type SeverOptions struct {
	Appname     string
	RootPath    string
	Port        int
	CrossOrigin bool
	Auth        blueweb.Middleware
	Secret      string
}

var defaultOptions = SeverOptions{
	Appname:     "core",
	Port:        8080,
	CrossOrigin: true,
	Auth:        func(c *blueweb.Context) bool { return true }, //default auth always returns true
	Secret:      "secret",
}

func channeltoappchannel(channel string) (app string, ch string) {
	//example: "core:channel1" -> app="core", ch="channel1"
	parts := strings.Split(channel, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "core", channel
}

//app:channel:id

func extractaci(str string) (app string, channel string, id string) {
	//example: "core:channel:id" -> app="core", channel="channel", id="id"

	parts := strings.Split(str, ":")
	if len(parts) == 1 {
		return "core", "default", parts[0]
	}

	if len(parts) == 2 {
		return "core", parts[0], parts[1]
	}

	if len(parts) == 3 {
		return parts[0], parts[1], parts[2]
	}

	return "core", "default", str
}

func StartQueServer(options *SeverOptions) error {
	isPaused := false
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
					ctx.Json(response{Error: "Unauthorized"})
					return
				}
			}

			//Custom Authentication
			if options.Auth != nil {
				success := options.Auth(ctx)
				if !success {
					ctx.Json(response{Error: "Unauthorized"})
					return
				}
			}

			ctx.State = time.Now()
			//Call the handler
			fn(ctx)
		}
	}

	pauserfunc := func(c *blueweb.Context) {
		c.Json(response{IsPaused: true})
	}

	inttotimesince := func(t interface{}) string {
		return time.Since(t.(time.Time)).String()
	}

	api := blueweb.NewRouter()

	api.Get("/solidq/pause", middle(func(ctx *blueweb.Context) {
		isPaused = true
		ctx.Json(response{Success: true, Took: inttotimesince(ctx.State)})
	}))

	api.Get("/solidq/unpause", middle(func(ctx *blueweb.Context) {
		isPaused = false
		ctx.Json(response{Success: true, Took: inttotimesince(ctx.State)})
	}))

	api.Post("/solidq/push/:item", middle(func(ctx *blueweb.Context) {
		if isPaused {
			pauserfunc(ctx)
			return
		}

		app, channel, workid := extractaci(ctx.Params("item"))

		localqueue, err := enusureQ(app)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		err = localqueue.Push(channel, workid)

		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		ctx.Json(response{Success: true, Took: inttotimesince(ctx.State)})
	}))

	api.Get("/solidq/pop/:channel/:count", middle(func(ctx *blueweb.Context) {
		if isPaused {
			pauserfunc(ctx)
			return
		}

		channel := ctx.Params("channel")
		count := ctx.Params("count")

		fmt.Println("Pop request for channel:", channel, "with count:", count)
		var app string
		app, channel = channeltoappchannel(channel)

		localqueue, err := enusureQ(app)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		co, _ := strconv.Atoi(count)
		if co < 1 {
			co = 1
		}

		ids, err := localqueue.PopWithCount(channel, co)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		if ids == nil {
			ctx.Json(response{Success: true, Took: inttotimesince(ctx.State)})
			return
		}

		ctx.Json(response{Success: true, Ids: ids, Took: inttotimesince(ctx.State)})
	}))

	api.Get("/solidq/listapps/:physical", middle(func(ctx *blueweb.Context) {
		if isPaused {
			pauserfunc(ctx)
			return
		}

		isPhysical := ctx.Params("physical") == "true"
		apps, err := listapps(isPhysical)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		ctx.Json(response{Success: true, Apps: apps})
	}))

	api.Get("/solidq/count/:channel/:count", middle(func(ctx *blueweb.Context) {
		if isPaused {
			pauserfunc(ctx)
			return
		}

		channel := ctx.Params("channel")
		var app string
		app, channel = channeltoappchannel(channel)

		localqueue, err := enusureQ(app)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		count, err := localqueue.Count(channel)
		if err != nil {
			ctx.Json(response{Error: err.Error()})
			return
		}
		ctx.Json(response{Success: true, Count: count, Took: inttotimesince(ctx.State)})
	}))

	api.Get("/solidq/reset/:channel", middle(func(ctx *blueweb.Context) {
		if isPaused {
			pauserfunc(ctx)
			return
		}

		channel := ctx.Query("channel")
		var app string
		app, channel = channeltoappchannel(channel)

		localqueue, err := enusureQ(app)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		err = localqueue.ResetChannel(channel)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}
		ctx.Json(response{Success: true, Took: inttotimesince(ctx.State)})
	}))

	api.Get("/solidq/channels/:appname", middle(func(ctx *blueweb.Context) {
		if isPaused {
			pauserfunc(ctx)
			return
		}

		app := ctx.Params("appname")

		localqueue, err := enusureQ(app)
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}

		channels, err := localqueue.ListChannelsWithCount()
		if err != nil {
			ctx.Json(response{Error: err.Error(), Took: inttotimesince(ctx.State)})
			return
		}
		ctx.Json(response{Success: true, Channels: channels, Took: inttotimesince(ctx.State)})
	}))

	api.Config().SetDev(true).SetPort(options.Port).StopOnInterrupt()

	api.StartServer()

	return nil
}
