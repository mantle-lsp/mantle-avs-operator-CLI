package hyperlane

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/contracts/safeglobal"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/config"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/eigenlayer"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/gnosis"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/utils"

	registryCoordinator "github.com/mantle-lsp/mantle-avs-operator-CLI/contracts/hyperlane"
)

type Hyperlane struct {
	Client                     *ethclient.Client
	RegistryCoordinatorAddress common.Address
	RegistryCoordinator        *registryCoordinator.ECDSAStakeRegistry
	ServiceManagerAddress      common.Address

	EigenLayer *eigenlayer.EigenLayer
	SafeGlobal *gnosis.SafeGlobal
}

func New(cfg config.Config, rpcClient *ethclient.Client) *Hyperlane {

	registryCoordinator, _ := registryCoordinator.NewECDSAStakeRegistry(cfg.HyperlaneStakeRegistryAddress, rpcClient)

	return &Hyperlane{
		Client:                     rpcClient,
		RegistryCoordinator:        registryCoordinator,
		RegistryCoordinatorAddress: cfg.HyperlaneStakeRegistryAddress,
		ServiceManagerAddress:      cfg.HyperlaneServiceManagerAddress,
		EigenLayer:                 eigenlayer.New(cfg, rpcClient),
		SafeGlobal:                 gnosis.New(cfg, rpcClient),
	}
}

// Info that node operator must supply to the mantle admin for registration
type RegistrationInfo struct {
	Operator  common.Address
	AvsSigner common.Address
}

func (a *Hyperlane) PrepareRegistration(operator common.Address, avsSigner common.Address) error {
	ri := RegistrationInfo{
		Operator:  operator,
		AvsSigner: avsSigner,
	}

	return utils.ExportJSON("hyperlane-prepare-registration", operator, ri)
}

func (a *Hyperlane) RegisterOperator(operator common.Address, info RegistrationInfo) error {

	// generate and sign registration hash to be signed by admin ecdsa key
	sigWithSaltAndExpiry, err := a.EigenLayer.GenerateAndSignRegistrationDigest(operator, a.ServiceManagerAddress)
	if err != nil {
		return fmt.Errorf("signing registration digest: %w", err)
	}

	sigParams := registryCoordinator.ISignatureUtilsSignatureWithSaltAndExpiry{
		Signature: nil, //signature is signed in safe wallet
		Salt:      sigWithSaltAndExpiry.Salt,
		Expiry:    sigWithSaltAndExpiry.Expiry,
	}

	// manually pack tx data since we are submitting via gnosis instead of directly
	coordinatorABI, err := registryCoordinator.ECDSAStakeRegistryMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("fetching abi: %w", err)
	}

	calldata, err := coordinatorABI.Pack("registerOperatorWithSignature", sigParams, operator)
	if err != nil {
		return fmt.Errorf("packing input: %w", err)
	}

	signMessageLibABI, err := safeglobal.SignMessageLibMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("fetching abi: %w", err)
	}

	signCalldata, err := signMessageLibABI.Pack("signMessage", sigWithSaltAndExpiry.Signature)
	if err != nil {
		return fmt.Errorf("packing input: %w", err)
	}

	chainID, _ := a.Client.ChainID(context.Background())

	// output in gnosis compatible format
	batch := gnosis.NewSingleTxBatch(calldata, chainID, fmt.Sprintf("hyperlane-register-operator-%s", operator))

	batch.AddTransaction(gnosis.SubTransaction{
		Target: a.SafeGlobal.SignMessageLibAddress,
		Value:  big.NewInt(0),
		Data:   signCalldata,
	})

	batch.AddTransaction(gnosis.SubTransaction{
		Target: a.RegistryCoordinatorAddress,
		Value:  big.NewInt(0),
		Data:   calldata,
	})

	return utils.ExportJSON("hyperlane-register-gnosis", operator, batch)
}
