package openapi

import (
	"fmt"
	"os"

	"github.com/naralabs/naralabs-atlas/cmd/server"
	"github.com/naralabs/naralabs-atlas/config"
	"github.com/naralabs/naralabs-atlas/internal/module/explore/handler"
	"github.com/spf13/cobra"
)

// Cmd prints the OpenAPI specification to stdout.
var Cmd = &cobra.Command{
	Use:   "openapi",
	Short: "Print the OpenAPI spec",
	Long:  "Print the OpenAPI specification for the Atlas read API in YAML format.",
	RunE:  run,
}

func run(_ *cobra.Command, _ []string) error {
	if err := config.Reload(); err != nil {
		return err
	}

	cfg := config.Get()
	exploreHandler := handler.NewExploreHandler(nil, cfg.Stellar.Network)
	api := server.RegisterRoutes(cfg, exploreHandler, nil)

	output, err := api.OpenAPI().YAML()
	if err != nil {
		return fmt.Errorf("generate OpenAPI spec: %w", err)
	}
	if _, err := os.Stdout.Write(output); err != nil {
		return fmt.Errorf("write OpenAPI spec: %w", err)
	}
	return nil
}
