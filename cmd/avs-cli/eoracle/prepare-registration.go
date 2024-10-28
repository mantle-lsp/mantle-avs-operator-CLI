package eoracle

import (
	"context"
	"fmt"

	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
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
			Name:     "bls-keystore",
			Usage:    "path to bls keystore file",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "bls-password",
			Usage:    "password for encrypted keystore file",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "alias-address",
			Usage:    "address associated with alias ECDSA key",
			Required: true,
		},
		&cli.IntSliceFlag{
			Name:     "quorums",
			Usage:    "which quorums to register for i.e. 0,1",
			Required: false,
			Value:    []int64{0},
		},
	},
}

func handlePrepareRegistration(ctx context.Context, cli *cli.Command) error {

	// parse cli input
	operatorAddress := cli.String("operator-address")
	blsKeyFile := cli.String("bls-keystore")
	blsKeyPassword := cli.String("bls-password")
	quorums := cli.IntSlice("quorums")
	aliasAddress := common.HexToAddress(cli.String("alias-address"))

	keyPair, err := bls.ReadPrivateKeyFromFile(blsKeyFile, blsKeyPassword)
	if err != nil {
		return fmt.Errorf("loading bls keystore: %w", err)
	}

	operator := common.HexToAddress(operatorAddress)
	if (operator == common.Address{}) {
		return fmt.Errorf("invalid operator address")
	}

	return eoracleAPI.PrepareRegistration(operator, keyPair, quorums, aliasAddress)
}
