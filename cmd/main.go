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
	"github.com/nsmeds/livery-stable/storage"
	"github.com/nsmeds/livery-stable/store"
)

// newStorage selects a Cloudflare R2-backed store when its credentials are
// present in the environment, and otherwise falls back to a local
// filesystem store for dev and CI.
func newStorage(ctx context.Context) (storage.Store, error) {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")

	if accountID != "" && accessKeyID != "" && secretAccessKey != "" && bucket != "" {
		return storage.NewR2Store(ctx, storage.R2Config{
			AccountID:       accountID,
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
			Bucket:          bucket,
		})
	}

	dir := os.Getenv("STORAGE_DIR")
	if dir == "" {
		dir = "./data/uploads"
	}
	return storage.NewFilesystemStore(dir), nil
}

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

	storageStore, err := newStorage(ctx)
	if err != nil {
		return fmt.Errorf("could not initialize storage: %w", err)
	}
	deleter := server.NewDeleter(storageStore)

	cfg := server.Config{JWTSecret: []byte(jwtSecret), Storage: storageStore}
	srv := server.New(*host, *port, st, deleter, cfg)

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
	deleter.Close()

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	if err := Run(ctx, cancel, os.Args, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
