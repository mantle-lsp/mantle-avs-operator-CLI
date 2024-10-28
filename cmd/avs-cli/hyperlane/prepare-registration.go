package hyperlane

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v3"
)

var PrepareRegistrationCmd = &cli.Command{
	Name:   "prepare-registration",
	Usage:  "(Node Operator) gather all inputs required to register for avs",
	Action: handlePrepareRegistration,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "operator-address",
			Usage:    "Operator address(0x...)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "avs-signer",
			Usage:    "Address of hyperlane specific ecdsa signing key",
			Required: true,
		},
	},
}

func handlePrepareRegistration(ctx context.Context, cli *cli.Command) error {

	// parse cli input
	operatorAddress := cli.String("operator-address")
	avsSignerAddress := common.HexToAddress(cli.String("avs-signer"))

	operator := common.HexToAddress(operatorAddress)
	if (operator == common.Address{}) {
		return fmt.Errorf("invalid operator address")
	}

	return hyperlaneAPI.PrepareRegistration(operator, avsSignerAddress)
}
