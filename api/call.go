package api

import (
	"context"
	"encoding/hex"
	"math"
	"math/big"
	"strings"

	chainjson "github.com/doslink/doslink/basis/encoding/json"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/protocol"
	"github.com/doslink/doslink/protocol/vm"
	"github.com/doslink/doslink/protocol/vm/evm"
	"github.com/doslink/doslink/protocol/vm/state"

	evm_common "github.com/ethereum/go-ethereum/common"
	evm_types "github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

// POST /call-contract
func (a *API) callContract(ctx context.Context, ins struct {
	Sender          chainjson.HexBytes `json:"from"`
	ContractAddress chainjson.HexBytes `json:"to"`
	AssetId         string             `json:"asset_id"`
	AssertAlias     string             `json:"asset_alias"`
	AssetAmount     uint64             `json:"value"`
	Data            chainjson.HexBytes `json:"input"`
}) Response {

	assetID := ins.AssetId
	assertAlias := ins.AssertAlias
	if assetID == "" && assertAlias != "" {
		assertAlias = strings.ToUpper(assertAlias)
		switch assertAlias {
		case consensus.NativeAssetAlias:
			assetID = consensus.NativeAssetID.String()
		default:
			asset, err := a.wallet.AssetReg.FindByAlias(assertAlias)
			if err != nil {
				return NewErrorResponse(errors.WithDetailf(err, "invalid asset alias %s", assertAlias))
			}
			assetID = asset.AssetID.String()
		}
	}

	res, gas, _, err := doCall(a.chain, ins.Sender, ins.ContractAddress, assetID, ins.AssetAmount, ins.Data)
	if err != nil {
		return NewErrorResponse(err)
	}
	resMap := map[string]interface{}{"ret": hex.EncodeToString(res)}
	resMap["gas"] = gas
	return NewSuccessResponse(resMap)
}

func doCall(
	chain *protocol.Chain,
	sender []byte,
	contractAddress []byte,
	assetID string,
	assetAmount uint64,
	data []byte,
) (res []byte, gas uint64, failed bool, err error) {
	var (
		from     evm_common.Address
		to       = new(evm_common.Address)
		nonce    = uint64(0)
		amount   = evm_common.Big0
		gasLimit = uint64(math.MaxUint64 - 1)
		gasPrice = evm_common.Big0

		msg      evm_types.Message
		author   *evm_common.Address
		header   = chain.BestBlockHeader()
		stateDB  evm.StateDB
		vmConfig = evm.Config{}
	)

	stateDB, err = protocol.NewState(&header.StateRoot, chain)
	if err != nil {
		return nil, 0, false, err
	}

	to.SetBytes(contractAddress)

	from = evm_common.BytesToAddress(sender)
	if assetID == consensus.NativeAssetID.String() {
		amount = new(big.Int).SetUint64(assetAmount)
		stateDB.AddBalance(from, amount)
	}

	author = &from

	log.WithField("data", hex.EncodeToString(data)).WithField("from", from.Hex()).WithField("to", to.Hex()).Println()
	msg = evm_types.NewMessage(from, to, nonce, amount, gasLimit, gasPrice, data, false)
	evmContext := vm.NewEVMContext(msg, header.Height, header.Timestamp, header.Bits, chain, author)
	evmEnv := evm.NewEVM(evmContext, stateDB, vmConfig)
	gp := new(state.GasPool).AddGas(math.MaxUint64)

	res, gas, failed, err = state.ApplyMessage(evmEnv, msg, gp)

	return res, gas, failed, err
}

func (a *API) balanceOf(ctx context.Context, ins struct {
	Sender          chainjson.HexBytes `json:"sender"`
	ContractAddress chainjson.HexBytes `json:"contract"`
}) Response {

	if ins.ContractAddress == nil {
		chain := a.chain
		header := chain.BestBlockHeader()
		stateDB, err := protocol.NewState(&header.StateRoot, chain)
		if err != nil {
			return NewErrorResponse(err)
		}
		addr := evm_common.BytesToAddress(ins.Sender)
		balance := stateDB.GetBalance(addr)
		resMap := map[string]interface{}{"balance": balance}
		return NewSuccessResponse(resMap)
	}

	dataHex := "70a08231" + "000000000000000000000000" + hex.EncodeToString(ins.Sender)
	data, _ := hex.DecodeString(dataHex)
	res, gas, _, err := doCall(a.chain, ins.Sender, ins.ContractAddress, "", 0, data)
	if err != nil {
		return NewErrorResponse(err)
	}
	resMap := map[string]interface{}{"balance": new(big.Int).SetBytes(res).String()}
	resMap["gas"] = gas
	return NewSuccessResponse(resMap)
}
