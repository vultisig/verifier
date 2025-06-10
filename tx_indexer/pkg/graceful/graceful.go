package graceful

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func HandleSignals(stopFunc ...func()) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	<-signals
	wg := sync.WaitGroup{}
	wg.Add(len(stopFunc))
	for _, _f := range stopFunc {
		f := _f
		go func() {
			defer wg.Done()
			f()
		}()
	}
	wg.Wait()
}
