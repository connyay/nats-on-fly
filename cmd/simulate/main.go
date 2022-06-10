package main

import (
	"bytes"
	"context"
	"log"
	"math/rand"
	"os/exec"
	"time"

	"github.com/alecthomas/kong"
	"github.com/superfly/flyctl/api"
)

var cli struct {
	BaseURL   string `default:"https://api.fly.io"`
	Token     string
	Launch    launchCmd    `cmd:""`
	List      listCmd      `cmd:""`
	RemoveAll removeAllCmd `cmd:""`
}

type Context struct {
	context.Context
	client *api.Client
}

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {
	cliCtx := kong.Parse(&cli)

	accessToken := cli.Token
	if accessToken == "" {
		tokenBytes, err := exec.Command("fly", "auth", "token").Output()
		cliCtx.FatalIfErrorf(err, "token from cli")

		accessToken = string(bytes.TrimSpace(tokenBytes))
	}
	name, version := "simulate", "0.0.0"

	api.SetBaseURL(cli.BaseURL)
	err := cliCtx.Run(Context{
		Context: context.Background(),
		client:  api.NewClient(accessToken, name, version, logger{}),
	})
	cliCtx.FatalIfErrorf(err, "running cmd")
}

type launchCmd struct {
	AppName string `name:"app" required:""`
	Count   int    `default:"1"`
	Region  string
	Image   string `default:"connyay/nof-client:latest"`
	Env     map[string]string
}

func (c launchCmd) Run(ctx Context) error {
	for i := 0; i < c.Count; i++ {
		region := c.Region
		if region == "" {
			var err error
			region, err = randomRegion(ctx, ctx.client)
			if err != nil {
				return err
			}
		}
		env := c.Env
		env["REGION"] = region
		_, _, err := ctx.client.LaunchMachine(ctx, api.LaunchMachineInput{
			AppID:  c.AppName,
			Region: region,
			Config: &api.MachineConfig{
				Image: c.Image,
				Env:   env,
				Guest: &api.MachineGuest{
					CPUKind:  "shared",
					CPUs:     1,
					MemoryMB: 256,
				},
			},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type listCmd struct {
	AppName string `name:"app" required:""`
}

func (c listCmd) Run(ctx Context) error {
	state := ""
	machines, err := ctx.client.ListMachines(ctx, c.AppName, state)
	if err != nil {
		return err
	}
	for _, machine := range machines {
		log.Printf("machine %v", machine)
	}
	return nil
}

type removeAllCmd struct {
	AppName string `name:"app" required:""`
}

func (c removeAllCmd) Run(ctx Context) error {
	state := ""
	machines, err := ctx.client.ListMachines(ctx, c.AppName, state)
	if err != nil {
		return err
	}
	for _, machine := range machines {
		_, err = ctx.client.RemoveMachine(ctx, api.RemoveMachineInput{
			AppID: c.AppName,
			ID:    machine.ID,
			Kill:  true,
		})
		if err != nil {
			log.Printf("warn: failed removing machine %v %v", machine.ID, err)
		} else {
			log.Printf("removed machine %v", machine.ID)
		}
	}
	return nil
}

type logger struct{}

func (l logger) Debug(v ...interface{}) {
	log.Printf("debug: %v", v)
}
func (l logger) Debugf(format string, v ...interface{}) {
	log.Printf("debug: "+format, v...)
}

func randomRegion(ctx context.Context, client *api.Client) (string, error) {
	regions, _, err := client.PlatformRegions(ctx)
	if err != nil {
		return "", err
	}
	idx := rand.Intn(len(regions))
	return regions[idx].Code, nil
}
