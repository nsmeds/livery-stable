package main_test

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	main "github.com/nsmeds/livery-stable/cmd"
)

func TestRun(t *testing.T) {
	t.Run("start the service", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		args := []string{
			"livery-stable",
			"--host", "localhost",
			"--port", "8080",
		}
		var stdout bytes.Buffer
		var waitgroup sync.WaitGroup
		var err error
		waitgroup.Add(1)
		go func() {
			err = main.Run(ctx, cancel, args, &stdout, io.Discard)
			if err != nil {
				t.Error("unexpected err in main.Run: ", err)
			}
			waitgroup.Done()
		}()
		cancel()
		waitgroup.Wait()
		if err != nil {
			t.Error(err)
		}
	})
}
