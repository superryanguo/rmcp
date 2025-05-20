package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/superryanguo/rmcp/ollama"
)

var (
	logger   *slog.Logger
	loglevel *slog.LevelVar
)

// Ryai own the runtime instance
type Ryai struct {
	ctx       context.Context
	slog      *slog.Logger   // slog output to use
	slogLevel *slog.LevelVar // slog level, for changing as needed
	http      *http.Client   // http client to use
	addr      string         // address to serve HTTP on
}

func InitClient() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger.Info("Srv is running...***...***...***...***...***...")

	g := &Ryai{
		ctx:       context.Background(),
		slog:      logger,
		slogLevel: loglevel,
		http:      http.DefaultClient,
		addr:      "localhost:4229",
	}

	input := "you are a good reporter, pls think this question carefully, how about Donald Trump?"

	var osrv string
	gai, err := ollama.NewClient(g.slog, g.http, osrv, ollama.DefaultGenModel2)
	if err != nil {
		log.Fatal(err)
	}

	rsp, err := gai.Prompt(g.ctx, input)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Gai get rsp: %s\n", string(rsp))
	s, err := ollama.AssembleRsp(rsp)
	if err != nil {
		//log.Fatal(err)
	}

	fmt.Printf("AssembleRsp: %s\n", s)
}

func main() {
	InitClient()

}
