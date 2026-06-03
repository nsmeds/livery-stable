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
	"time"

	"github.com/nsmeds/livery-stable/db"
	"github.com/nsmeds/livery-stable/server"
	"github.com/nsmeds/livery-stable/store"
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

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("JWT_SECRET environment variable is required")
	}

	if err := db.Migrate(dbURL); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	connectCtx, connectCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connectCancel()
	pool, err := db.Connect(connectCtx, dbURL)
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer pool.Close()

	st := store.New(pool)
	cfg := server.Config{JWTSecret: []byte(jwtSecret)}
	srv := server.New(*host, *port, st, cfg)

	go func() {
		fmt.Fprintln(stdout, "starting server ...")
		if err := srv.ListenAndServe(); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				fmt.Fprintln(stderr, "could not start server: ", err)
			}
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-ctx.Done():
		fmt.Fprintln(stdout, "terminating: context canceled")
	case s := <-signals:
		cancel()
		fmt.Fprintln(stdout, "terminating: signal received "+s.String())
	}
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("could not close server: %w", err)
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	if err := Run(ctx, cancel, os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
