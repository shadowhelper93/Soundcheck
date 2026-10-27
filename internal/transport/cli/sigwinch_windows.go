//go:build windows

package cli

import (
	"os"
)

func notifySIGWINCH(_ chan<- os.Signal) {}

func stopSIGWINCH(_ chan<- os.Signal) {}
