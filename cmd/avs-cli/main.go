package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mantle-lsp/mantle-avs-operator-CLI/cmd/avs-cli/eigenda"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/cmd/avs-cli/eoracle"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/cmd/avs-cli/hyperlane"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{

		// global flags
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "rpc-url",
				Usage:    "rpc url",
				Required: false,
			},
		},
		Commands: []*cli.Command{
			// AVS specific commands
			eigenda.EigenDACmd,
			eoracle.EOracleCmd,
			hyperlane.HyperlaneCmd,
		},
	}

	ctx := context.Background()
	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
