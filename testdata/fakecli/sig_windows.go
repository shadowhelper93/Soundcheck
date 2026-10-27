//go:build windows

package main

import "os"

func notifyResize(_ chan<- os.Signal) {}
