package main_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"sync"
	"testing"

	main "github.com/nsmeds/livery-stable/cmd"
)

func TestRun(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5433/livery_stable_test?sslmode=disable"
	}
	t.Setenv("DATABASE_URL", dbURL)
	t.Setenv("JWT_SECRET", "test-jwt-secret-for-testing-only")

	t.Run("start the service", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		args := []string{
			"livery-stable",
			"--host", "localhost",
			"--port", "8081",
		}
		var stdout bytes.Buffer
		var waitgroup sync.WaitGroup
		var err error
		waitgroup.Add(1)
		go func() {
			err = main.Run(ctx, cancel, args, &stdout, io.Discard)
			waitgroup.Done()
		}()
		cancel()
		waitgroup.Wait()
		if err != nil {
			t.Error("unexpected err in main.Run: ", err)
		}
	})
}
