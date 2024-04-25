package main

import (
	"log/slog"
	_ "net/http/pprof"
	"os"

	"github.com/acorn-io/cmd"
	"github.com/thedadams/clicky-serves/pkg/cli"
)

func main() {
	if os.Getenv("CLICKY_SERVES_DEBUG") != "" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}
	cmd.Main(cmd.Command(new(cli.Server)))
}
