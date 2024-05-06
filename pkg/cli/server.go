package cli

import (
	"fmt"
	"os"

	"github.com/gptscript-ai/clicky-serves/pkg/server"
	"github.com/gptscript-ai/clicky-serves/pkg/version"
	"github.com/spf13/cobra"
)

type Server struct {
	ServerPort string `usage:"Server port" default:"8080" env:"CLICKY_SERVES_SERVER_PORT"`
}

func (s *Server) Run(cmd *cobra.Command, _ []string) error {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable must be set")
	}

	return server.Start(cmd.Context(), server.Config{
		Port: s.ServerPort,
	})
}

func (s *Server) Customize(cmd *cobra.Command) {
	cmd.Use = version.ProgramName + " [flags]"
	cmd.Version = version.Get().String()
}
