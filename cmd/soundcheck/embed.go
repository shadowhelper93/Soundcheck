package main

import (
	"github.com/soundcheck/soundcheck/internal/transport/cli"
	"github.com/soundcheck/soundcheck/ui"
)

var version string

func init() {
	if version != "" {
		cli.Version = version
	}
	if assets, err := ui.Assets(); err == nil {
		cli.SetDefaultAssets(assets)
	}
}
