package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsmeds/livery-stable/server"
)

func Run(ctx context.Context, cancel context.CancelFunc, args []string, stdout, stderr io.Writer) error {
	defer cancel()

	defaultHost := "localhost"
	defaultPort := 8080

	flags := flag.NewFlagSet("livery-stable", flag.ContinueOnError)
	flags.SetOutput(stderr)
	host := flags.String("host", defaultHost, "hostname for server")
	port := flags.Int("port", defaultPort, "port for server")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	srv := server.New(*host, *port)
	go func() {
		fmt.Println("starting server ...")
		if err := srv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				fmt.Println("could not start server: ", err)
			}
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-ctx.Done():
		fmt.Println("terminating: context canceled")
	case s := <-signals:
		cancel()
		fmt.Println("terminating: signal received " + s.String())
	}
	if err := srv.Shutdown(ctx); err != nil {
		msg := fmt.Sprintf("could not close server: %v", err)
		return errors.New(msg)
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	if err := Run(ctx, cancel, os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
}
