package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alecthomas/kong"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"golang.org/x/sync/semaphore"
)

var cli struct {
	NatsAddr              string `help:"Address of nats cluster to connect to." default:"[::]:4222" env:"NATS_ADDR"`
	Region                string `env:"REGION" default:"dev"`
	Count                 int    `help:"number of clients to start" env:"CLIENT_COUNT" default:"1"`
	ConcurrentConnections int64  `help:"Restrict the number of connections being established concurrently." default:"256"`
}

func main() {
	_ = kong.Parse(&cli)
	ctx := context.Background()
	errs := make(chan error)
	connSem := semaphore.NewWeighted(cli.ConcurrentConnections)

	for i := 0; i < cli.Count; i++ {
		go func() {
			if err := connSem.Acquire(ctx, 1); err != nil {
				log.Printf("Failed getting registration lock %v", err)
				return
			}
			nc, err := connect(ctx, cli.NatsAddr)
			if err != nil {
				errs <- err
				return
			}
			connSem.Release(1)
			defer nc.Close()
			id := uuid.NewString()[:8]
			log.Printf("client=%s connected", id)
			err = ensurePingSub(ctx, nc, id, cli.Region)
			if err != nil {
				errs <- err
				return
			}

			<-ctx.Done()
		}()
	}

	for err := range errs {
		log.Printf("connection err %v", err)
	}
}

func ensurePingSub(ctx context.Context, nc *nats.Conn, clientID, region string) error {
	subscribeAndFlush := func() (func() error, error) {
		sub, err := nc.Subscribe("ping", func(msg *nats.Msg) {
			response := fmt.Sprintf("client=%s region=%s now=%s pong", clientID, region, time.Now().Format(time.RFC3339))
			if err := msg.Respond([]byte(response)); err != nil {
				log.Printf("client=%s failed respond %v", clientID, err)
			}
		})
		if err != nil {
			return nil, err
		}
		return sub.Unsubscribe, nil
	}

	for {
		if ctx.Err() != nil {
			break
		}
		close, err := subscribeAndFlush()
		if close != nil {
			defer close()
		}
		if err == nil {
			break
		}
		log.Printf("err subscribing %v", err)
		time.Sleep(100 * time.Millisecond)
	}

	<-ctx.Done()
	return nil
}

func connect(ctx context.Context, addr string) (*nats.Conn, error) {
	nc, err := nats.Connect(addr,
		nats.Name("client"),
		nats.Timeout(10*time.Second),
		nats.ReconnectWait(10*time.Second),
		nats.ReconnectJitter(1*time.Second, 5*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.ErrorHandler(func(c *nats.Conn, s *nats.Subscription, err error) {
			log.Printf("client err handler %v", err)
		}),
	)
	return nc, err
}
