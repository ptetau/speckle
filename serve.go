package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/ptetau/speckle/internal/render"
	"github.com/ptetau/speckle/internal/server"
	"github.com/ptetau/speckle/internal/spec"
)

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", "127.0.0.1:0", "address to listen on (default: random localhost port)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: speckle serve [--addr=ADDR] <file.speckle>")
	}

	srv, err := server.New(server.Config{
		Path:     fs.Arg(0),
		Addr:     *addr,
		Parser:   spec.NewParser(),
		Renderer: render.NewRenderer(),
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return srv.Run(ctx)
}
