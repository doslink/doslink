package vm

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/protocol/vm/evm"

	evm_common "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
)

func IsOpDeposit(prog []byte) bool {
	insts, err := ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 4 {
		return false
	}

	version := insts[0]
	vmType := insts[1]
	address := insts[2]

	if version.Op > OP_16 {
		return false
	}

	if vmType.Op > OP_16 {
		return false
	}

	if address.Op != OP_DATA_20 || len(address.Data) != 20 {
		return false
	}

	return insts[len(insts)-1].Op == OP_DEPOSIT
}

func GetAddressFromOpDeposit(prog []byte) ([]byte, error) {
	insts, err := ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[2].Data, nil
}

func opDeposit(vm *virtualMachine) error {
	var (
		assetID     = *vm.context.AssetID
		assetAmount = *vm.context.Amount

		caller evm_common.Address

		stateDB = vm.context.StateDB
	)

	// get params from dataStack
	address, err := vm.pop(false)
	if err != nil {
		return err
	}

	vmTypeBytes, err := vm.pop(false)
	if err != nil {
		return err
	}
	vmType := new(big.Int).SetBytes(vmTypeBytes).Uint64()
	if vmType > 0 {
		return errors.New("unknown vmType")
	}

	versionBytes, err := vm.pop(false)
	if err != nil {
		return err
	}
	version := new(big.Int).SetBytes(versionBytes).Uint64()
	if version > 0 {
		return errors.New("unknown version number")
	}

	caller = evm_common.BytesToAddress(address)
	log.WithField("caller", caller.Hex()).
		WithField("vmType", vmType).
		WithField("asset", hex.EncodeToString(assetID)).
		WithField("amount", assetAmount).
		Infoln("Deposit")

	if bytes.Compare(assetID, consensus.NativeAssetID.Bytes()) == 0 {
		amount := new(big.Int).SetUint64(assetAmount)
		stateDB.AddBalance(caller, amount)
	}

	return vm.pushBool(true, false)
}

func GetAddressFromOpWithdraw(prog []byte) ([]byte, error) {
	insts, err := ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[2].Data, nil
}

func IsOpWithdraw(prog []byte) bool {
	insts, err := ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 4 {
		return false
	}

	version := insts[0]
	vmType := insts[1]
	address := insts[2]

	if version.Op > OP_16 {
		return false
	}

	if vmType.Op > OP_16 {
		return false
	}

	if address.Op != OP_DATA_20 || len(address.Data) != 20 {
		return false
	}

	return insts[len(insts)-1].Op == OP_WITHDRAW
}

func opWithdraw(vm *virtualMachine) error {
	var (
		assetID     = *vm.context.AssetID
		assetAmount = *vm.context.Amount

		caller evm_common.Address

		stateDB = vm.context.StateDB
	)

	// get params from dataStack
	address, err := vm.pop(false)
	if err != nil {
		return err
	}

	vmTypeBytes, err := vm.pop(false)
	if err != nil {
		return err
	}
	vmType := new(big.Int).SetBytes(vmTypeBytes).Uint64()
	if vmType > 0 {
		return errors.New("unknown vmType")
	}

	versionBytes, err := vm.pop(false)
	if err != nil {
		return err
	}
	version := new(big.Int).SetBytes(versionBytes).Uint64()
	if version > 0 {
		return errors.New("unknown version number")
	}

	caller = evm_common.BytesToAddress(address)
	log.WithField("caller", caller.Hex()).
		WithField("vmType", vmType).
		WithField("asset", hex.EncodeToString(assetID)).
		WithField("amount", assetAmount).
		Infoln("Withdraw")

	if bytes.Compare(assetID, consensus.NativeAssetID.Bytes()) == 0 {
		amount := new(big.Int).SetUint64(assetAmount)
		// Fail if we're trying to transfer more than the available balance
		if !CanTransfer(stateDB, caller, amount) {
			return evm.ErrInsufficientBalance
		}
		stateDB.SubBalance(caller, amount)
	}

	return vm.pushBool(true, false)
}
