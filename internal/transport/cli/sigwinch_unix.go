//go:build !windows

package cli

import (
	"os"
	"os/signal"
	"syscall"
)

func notifySIGWINCH(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}

func stopSIGWINCH(ch chan<- os.Signal) {
	signal.Stop(ch)
}
