package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Kook-Dohyun/To-Do-Connect/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := app.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
