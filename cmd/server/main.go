package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/alecthomas/kong"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
)

var cli struct {
	Addr     string `help:"Address to listen on" default:"localhost:9000" env:"ADDR"`
	NatsAddr string `help:"Address of nats cluster to connect to." default:"[::]:4222" env:"NATS_ADDR"`
}

func main() {
	cliCtx := kong.Parse(&cli)
	nc, err := nats.Connect(cli.NatsAddr,
		nats.Name("server"),
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(10*time.Second),
		nats.ReconnectJitter(1*time.Second, 5*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.ErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
			log.Printf("nats err %+v %+v", s, err)
		}),
	)
	cliCtx.FatalIfErrorf(err, "connecting to nats")

	http.HandleFunc("/", pingHandler(nc))
	http.ListenAndServe(cli.Addr, nil)
}

func pingHandler(nc *nats.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sub, err := nc.SubscribeSync(nats.NewInbox())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer sub.Unsubscribe()

		if err := sub.SetPendingLimits(-1, -1); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := nc.PublishRequest("ping", sub.Subject, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res := struct {
			Count     int      `json:"count"`
			Duration  string   `json:"duration"`
			Responses []string `json:"responses,omitempty"`
		}{}
		includeResponses := r.URL.Query().Has("with_responses")
		start := time.Now()
		for {
			msg, err := sub.NextMsg(time.Second)
			if err != nil {
				if !errors.Is(err, nats.ErrTimeout) {
					log.Printf("failed sub %v", err)
				}
				break
			}
			res.Count++
			if includeResponses {
				res.Responses = append(res.Responses, string(msg.Data))
			}
		}
		res.Duration = time.Since(start).String()
		resBytes, _ := json.MarshalIndent(res, "", "  ")
		w.Write(resBytes)
	}
}
