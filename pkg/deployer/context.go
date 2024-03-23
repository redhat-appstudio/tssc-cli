package deployer

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func backgroundContext(fn func()) context.Context {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	signalCh := make(chan os.Signal, 2)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalCh
		fn()
		cancel()
	}()

	return ctx
}
