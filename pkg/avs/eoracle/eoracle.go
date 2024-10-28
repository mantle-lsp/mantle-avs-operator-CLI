package eoracle

import (
	"context"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/consensys/gnark-crypto/ecc/bn254/fp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/contracts/safeglobal"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/config"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/eigenlayer"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/gnosis"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/types"
	"github.com/mantle-lsp/mantle-avs-operator-CLI/pkg/utils"

	"github.com/Layr-Labs/eigensdk-go/crypto/bls"

	registryCoordinator "github.com/Eoracle/core-go/contracts/bindings/EORegistryCoordinator"
)

type EOracle struct {
	Client                     *ethclient.Client
	RegistryCoordinatorAddress common.Address
	RegistryCoordinator        *registryCoordinator.ContractEORegistryCoordinator
	ServiceManagerAddress      common.Address

	EigenLayer *eigenlayer.EigenLayer
	SafeGlobal *gnosis.SafeGlobal
}

func New(cfg config.Config, rpcClient *ethclient.Client) *EOracle {

	registryCoordinator, _ := registryCoordinator.NewContractEORegistryCoordinator(cfg.EOracleRegistryCoordinatorAddress, rpcClient)

	return &EOracle{
		Client:                     rpcClient,
		RegistryCoordinator:        registryCoordinator,
		RegistryCoordinatorAddress: cfg.EOracleRegistryCoordinatorAddress,
		ServiceManagerAddress:      cfg.EOracleServiceManagerAddress,
		EigenLayer:                 eigenlayer.New(cfg, rpcClient),
		SafeGlobal:                 gnosis.New(cfg, rpcClient),
	}
}

// Info that node operator must supply to the mantle admin for registration
type RegistrationInfo struct {
	Operator                    common.Address
	BLSPubkeyRegistrationParams *types.BLSPubkeyRegistrationParams
	Quorums                     []int64
	AliasAddress                common.Address
}

func (a *EOracle) PrepareRegistration(operator common.Address, blsKey *bls.KeyPair, quorums []int64, alias common.Address) error {
	// compute hash to sign with bls key
	// the hash is converted to a G1 point on the curve before it is returned
	g1Point, err := a.RegistryCoordinator.PubkeyRegistrationMessageHash(nil, operator)
	if err != nil {
		return fmt.Errorf("fetching pubkeyRegistrationMessageHash: %w", err)
	}

	// map from contract type to type expected by signing algorithm
	g1MsgToSign := &bn254.G1Affine{
		X: *new(fp.Element).SetBigInt(g1Point.X),
		Y: *new(fp.Element).SetBigInt(g1Point.Y),
	}

	g1Sig := blsKey.SignHashedToCurveMessage(g1MsgToSign)

	signedParams := new(types.BLSPubkeyRegistrationParams)
	signedParams.Load(blsKey.GetPubKeyG1().G1Affine, blsKey.GetPubKeyG2().G2Affine, g1Sig.G1Affine)

	ri := RegistrationInfo{
		Operator:                    operator,
		BLSPubkeyRegistrationParams: signedParams,
		Quorums:                     quorums,
		AliasAddress:                alias,
	}
	return utils.ExportJSON("eoracle-prepare-registration", operator, ri)
}

func (a *EOracle) RegisterOperator(operator common.Address, info RegistrationInfo) error {

	// generate and sign registration hash to be signed by admin ecdsa key
	sigWithSaltAndExpiry, err := a.EigenLayer.GenerateAndSignRegistrationDigest(operator, a.ServiceManagerAddress)
	if err != nil {
		return fmt.Errorf("signing registration digest: %w", err)
	}

	// convert to types expected by contract call
	quorums := make([]byte, len(info.Quorums))
	for i, v := range info.Quorums {
		quorums[i] = byte(v)
	}

	sigParams := registryCoordinator.ISignatureUtilsSignatureWithSaltAndExpiry{
		Signature: nil, //signature is signed in safe wallet
		Salt:      sigWithSaltAndExpiry.Salt,
		Expiry:    sigWithSaltAndExpiry.Expiry,
	}
	pubkeyParams := registryCoordinator.IEOBLSApkRegistryPubkeyRegistrationParams{
		PubkeyRegistrationSignature: registryCoordinator.BN254G1Point(info.BLSPubkeyRegistrationParams.Signature),
		ChainValidatorSignature:     registryCoordinator.BN254G1Point{X: big.NewInt(0), Y: big.NewInt(0)}, // not currently used by protocol
		PubkeyG1:                    registryCoordinator.BN254G1Point(info.BLSPubkeyRegistrationParams.G1),
		PubkeyG2:                    registryCoordinator.BN254G2Point(info.BLSPubkeyRegistrationParams.G2),
	}

	// manually pack tx data since we are submitting via gnosis instead of directly
	coordinatorABI, err := registryCoordinator.ContractEORegistryCoordinatorMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("fetching abi: %w", err)
	}

	calldata, err := coordinatorABI.Pack("registerOperator", quorums, pubkeyParams, sigParams)
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
	batch := gnosis.NewSingleTxBatch(calldata, chainID, fmt.Sprintf("eoracle-register-operator-%s", operator))

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

	return utils.ExportJSON("eoracle-register-gnosis", operator, batch)
}
