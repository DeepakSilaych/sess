package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DeepakSilaych/sess/internal/api"
	"github.com/DeepakSilaych/sess/internal/backend"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Run(args []string) int {
	if len(args) == 1 && args[0] == "detach" {
		z, e := backend.Open()
		if e == nil {
			e = z.Detach(context.Background())
		}
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return 1
		}
		return 0
	}
	if len(args) != 2 || (args[0] != "rpc" && args[0] != "attach") {
		fmt.Fprintln(os.Stderr, "sess remote helper; use sess on your laptop (or sess detach here)")
		return 2
	}
	r, e := api.Decode(args[1])
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		return 40
	}
	res := api.Response{Protocol: api.Protocol, Version: api.Version}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGTERM)
	defer cancel()
	if args[0] == "rpc" {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeout(ctx, 15*time.Second)
		defer stop()
	}
	z, e := backend.Open()
	if e == nil {
		res.Backend, e = z.Version(ctx)
	}
	if e == nil {
		switch r.Action {
		case "ping":
		case "list":
			res.Sessions, e = z.List(ctx)
		case "create":
			var s api.Session
			s, e = z.Create(ctx, r.Name)
			if e == nil {
				res.Session = &s
			}
		case "find":
			var s api.Session
			s, e = z.Find(ctx, r.Name, r.ID)
			if e == nil {
				res.Session = &s
			}
		case "remove":
			e = z.Remove(ctx, r.Name, r.ID)
		case "attach":
			if args[0] != "attach" {
				e = fmt.Errorf("attachment requires a terminal request")
			} else {
				e = z.Attach(ctx, r.Name, r.ID)
			}
		default:
			e = fmt.Errorf("unknown operation %q", r.Action)
		}
	}
	if args[0] == "attach" {
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return 40
		}
		return 0
	}
	if e != nil {
		var ae *api.Error
		if errors.As(e, &ae) {
			res.Error = ae
		} else {
			res.Error = &api.Error{Code: "remote", Message: e.Error()}
		}
	}
	if e = json.NewEncoder(os.Stdout).Encode(res); e != nil {
		return 40
	}
	return 0
}
