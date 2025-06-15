package solidq

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"strings"

// 	"github.com/go-redis/redis/v8"
// )

// const redisfpset = "solidq:fpset"

// const (
// 	FPCmdPush                  = "push"
// 	FPCmdPop                   = "pop"
// 	FPCmdList                  = "list"
// 	FPCmdReset                 = "reset"
// 	FPCmdCount                 = "count"
// 	FPCmdPopWithCount          = "pop_with_count"
// 	FPCmdListChannels          = "list_channels"
// 	FPCmdListChannelsWithCount = "list_channels_with_count"
// 	FPCmdResetChannel          = "reset_channel"
// 	FPCmdVersion               = "version"
// 	FPCmdPing                  = "ping"
// 	FPCmdPong                  = "pong"
// 	FPCmdSet                   = "set"
// 	FPCmdGet                   = "get"
// 	FPCmdDel                   = "del"
// )

// // foreign packet
// type FP struct {
// 	Cmd        string          `json:"cmd"`
// 	Appname    string          `json:"appname"`
// 	Channel    string          `json:"channel"`
// 	WorkId     string          `json:"work_id"`
// 	TargetList string          `json:"target_list,omitempty"` // optional, used for some commands
// 	Data       json.RawMessage `json:"data"`
// }

// func (fp *FP) String() string {
// 	app, channel := channeltoappchannel(fp.Channel)

// 	if len(fp.WorkId) == 0 {
// 		fp.WorkId = "-"
// 	}

// 	if len(fp.Data) == 0 {
// 		fp.Data = json.RawMessage("{}")
// 	}

// 	return fp.Cmd + "|" + app + "|" + channel + "|" + fp.WorkId + "|" + string(fp.Data)
// }

// // cmd|appname|channel|work_id|targetlist|data

// func ToFP(str string) *FP {
// 	parts := strings.SplitN(str, "|", 6)
// 	if len(parts) < 5 {
// 		return nil
// 	}

// 	fp := &FP{
// 		Cmd:        parts[0],
// 		Appname:    parts[1],
// 		Channel:    parts[2],
// 		WorkId:     parts[3],
// 		TargetList: parts[4],
// 		Data:       json.RawMessage(parts[5]),
// 	}

// 	if len(fp.Data) == 0 {
// 		fp.Data = json.RawMessage("{}")
// 	}

// 	return fp
// }

// func fpbackground() {
// 	r := redis.NewClient(&redis.Options{
// 		Network:  "tcp",
// 		Addr:     "localhost:6379",
// 		Password: "passme",
// 		DB:       0,
// 	})

// 	for {
// 		zs, err := r.ZPopMax(context.Background(), redisfpset, 1).Result()
// 		if err != nil {
// 			continue
// 		}

// 		for _, z := range zs {
// 			p := ToFP(z.Member.(string))

// 			que, err := enusureQ(p.Appname)
// 			if err != nil {
// 				continue
// 			}

// 			var payload Payload
// 			if err := json.Unmarshal(p.Data, &payload); err != nil {
// 				fmt.Println("Error unmarshalling payload:", err)
// 			}

// 			switch p.Cmd {
// 			case FPCmdPush:
// 				work := Work{
// 					Id:   p.WorkId,
// 					Data: payload,
// 				}
// 				if err := que.Push(p.Channel, work); err != nil {
// 					fmt.Println("Error pushing work:", err)
// 				}
// 			case FPCmdPop:
// 				work, err := que.Pop(p.Channel)
// 			}
// 		}
// 	}
// }
