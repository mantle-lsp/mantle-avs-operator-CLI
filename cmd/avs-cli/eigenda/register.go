package eigenda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/a41-official/mantle-eigenlayer-operator-cli/pkg/avs/eigenda"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/urfave/cli/v3"
)

var RegisterCmd = &cli.Command{
	Name:   "register",
	Usage:  "(Admin) Register target operator to the AVS",
	Action: handleRegister,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "registration-input",
			Usage:    "path to registration file created by prepare-registration command",
			Required: true,
		},
	},
}

func handleRegister(ctx context.Context, cli *cli.Command) error {

	// parse cli params
	inputFilepath := cli.String("registration-input")

	// read input file with required registration data
	var input eigenda.RegistrationInfo
	buf, err := os.ReadFile(inputFilepath)
	if err != nil {
		return fmt.Errorf("reading input file: %w", err)
	}
	if err := json.Unmarshal(buf, &input); err != nil {
		return fmt.Errorf("parsing registration input: %w", err)
	}
	if input.Operator == (common.Address{}) {
		return fmt.Errorf("invalid registration input, missing operatorID")
	}
	if input.Socket == "" {
		return fmt.Errorf("invalid registration input, missing socket")
	}

	operator := input.Operator

	// load eip-1271 admin signing key
	signingKey, err := crypto.HexToECDSA(os.Getenv("ADMIN_1271_SIGNING_KEY"))
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	return eigendaAPI.RegisterOperator(operator, input, signingKey)
}
