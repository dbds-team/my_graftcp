package main

import (
	"io"
	"os"

	"github.com/hmgle/graftcp/local"
)

var appLogger = local.NewStderrLogger()

func openDebugLog(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	appLogger = local.NewLogger(io.MultiWriter(os.Stderr, f))
	local.SetLogger(appLogger)
	return f, nil
}
