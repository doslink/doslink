package vm

import (
	evm_common "github.com/ethereum/go-ethereum/common"
	evm_types "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"github.com/doslink/doslink/protocol/vm/evm"
	"github.com/doslink/doslink/protocol/vm/state"
	log "github.com/sirupsen/logrus"
	"bytes"
	"github.com/doslink/doslink/consensus"
	"encoding/hex"
	"math"
	"errors"
)

func IsOpCall(prog []byte) bool {
	insts, err := ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 3 {
		return false
	}

	version := insts[0]
	input := insts[1]

	if version.Op > OP_16 {
		return false
	}

	if input.Op < OP_DATA_1 || input.Op > OP_PUSHDATA4 {
		return false
	}

	return insts[len(insts)-1].Op == OP_CALL
}

func opCall(vm *virtualMachine) error {
	var (
		chain       = vm.context.Chain
		assetID     = *vm.context.AssetID
		assetAmount = *vm.context.Amount

		from     evm_common.Address
		to       = new(evm_common.Address)
		nonce    uint64
		amount   = evm_common.Big0
		gasLimit = uint64(vm.runLimit)
		gasPrice = evm_common.Big0

		msg      evm_types.Message
		author   *evm_common.Address
		stateDB  = vm.context.StateDB
		vmConfig = evm.Config{}

		height, timestamp, difficulty = vm.context.Chain.BestBlockInfo()
	)

	// get params from dataStack
	data, err := vm.pop(false)
	if err != nil {
		return err
	}

	versionBytes, err := vm.pop(false)
	if err != nil {
		return err
	}
	version := new(big.Int).SetBytes(versionBytes).Uint64()
	if version > 0 {
		return errors.New("unknown version number")
	}

	contractAddress, err := vm.pop(false)
	if err != nil {
		return err
	}
	to.SetBytes(contractAddress)
	log.WithField("contractAddress", to.Hex()).Infoln("")

	nonceBytes, err := vm.pop(false)
	if err != nil {
		return err
	}
	nonce = new(big.Int).SetBytes(nonceBytes).Uint64()
	sender, err := vm.pop(false)
	if err != nil {
		return err
	}

	from = evm_common.BytesToAddress(sender)
	if bytes.Compare(assetID, consensus.NativeAssetID.Bytes()) == 0 {
		amount = new(big.Int).SetUint64(assetAmount)
		stateDB.AddBalance(from, amount)
	}

	gasLimit = uint64(vm.runLimit)
	log.WithFields(log.Fields{"amount": amount, "gasLimit": gasLimit, "execBalance": stateDB.GetBalance(from)}).Infoln("check balance")

	author = &from

	log.WithFields(log.Fields{"sender": from.Hex(), "nonce": nonce, "stateNonce": stateDB.GetNonce(from)}).Infoln("check nonce")
	msg = evm_types.NewMessage(from, to, nonce, amount, gasLimit, gasPrice, data, true)
	//fmt.Printf("msg=%v\n", msg)
	//fmt.Printf("header=%v\n", header)
	evmContext := NewEVMContext(msg, height, timestamp, difficulty, chain, author)
	//fmt.Printf("evmContext=%v\n", evmContext)
	evmEnv := evm.NewEVM(evmContext, stateDB, vmConfig)
	//fmt.Printf("evmEnv=%v\n", evmEnv)
	gp := new(state.GasPool).AddGas(math.MaxUint64)
	//fmt.Printf("GasPool=%v\n", gp)

	ret, gas, _, err := state.ApplyMessage(evmEnv, msg, gp)

	if err != nil {
		log.WithField("error", err).Error("ApplyMessage to evm failed")
		return err
	}

	err = vm.applyCost(int64(gas))
	if err != nil {
		return err
	}

	if ret != nil {
		log.WithField("ret", hex.EncodeToString(ret)).Infoln("call contract success")
		return vm.push(ret, false)
	}
	return vm.pushBool(true, false)
}
