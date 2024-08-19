package quit

import (
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func WaitForQuitSignal() {
	// Wait for interrupt signal to gracefully shut down the server.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
}

type waitForQuitGoroutine struct {
	name string
	c    chan struct{}
}

var waitForQuitGoroutines = make([]waitForQuitGoroutine, 0)
var mutex = sync.Mutex{}

type Goroutine interface {
	Done()
}

type done struct {
	*waitForQuitGoroutine
	once sync.Once
}

func (d *done) Done() {
	d.once.Do(func() {
		mutex.Lock()
		defer mutex.Unlock()
		close(d.c)
		var i = -1
		for j, wg := range waitForQuitGoroutines {
			if wg == *d.waitForQuitGoroutine {
				i = j
				break
			}
		}
		if i != -1 {
			waitForQuitGoroutines = append(waitForQuitGoroutines[:i], waitForQuitGoroutines[i+1:]...)
		}
		goRoutineCount := len(waitForQuitGoroutines)
		if goRoutineCount == 0 {
			logrus.Debugf("all goroutines ended")
		} else {
			logrus.Debugf("goroutine \"%s\" ended. %d goroutines left", d.name, goRoutineCount)
		}
	})
}

func ReportGoroutine(name string) Goroutine {
	mutex.Lock()
	defer mutex.Unlock()
	c := make(chan struct{}, 1)
	wg := waitForQuitGoroutine{
		name: name,
		c:    c,
	}

	waitForQuitGoroutines = append(waitForQuitGoroutines, wg)
	logrus.Debugf("goroutine \"%s\" started, %d goroutines running", name, len(waitForQuitGoroutines))
	return &done{
		waitForQuitGoroutine: &wg,
	}
}

func WaitForAllGoroutineEnd(finalizeTimeout time.Duration) {
	mutex.Lock()
	defer mutex.Unlock()
	//	iterate all chan in reverse order
	for i := len(waitForQuitGoroutines) - 1; i >= 0; i-- {
		select {
		case <-waitForQuitGoroutines[i].c:
			logrus.Infof("goroutine \"%s\" ended", waitForQuitGoroutines[i].name)
		case <-time.After(finalizeTimeout):
			logrus.Errorf("goroutine \"%s\" didn't end in time (%s)", waitForQuitGoroutines[i].name, finalizeTimeout.String())
		}
	}
}
