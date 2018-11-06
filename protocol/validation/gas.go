package validation

import (
	"errors"
	"math"
	"math/big"

	"github.com/doslink/doslink/consensus/segwit"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/vm"

	evm_state "github.com/ethereum/go-ethereum/core/state"
	log "github.com/sirupsen/logrus"
)

func EstimateContractGas(e bc.Entry, tx *bc.Tx, block *bc.Block, chain vm.ChainContext, stateDB *evm_state.StateDB) (gasStatus *GasState, err error) {

	stateDB.Prepare(tx.ID.Byte32(), [32]byte{}, 0)

	gasStatus = &GasState{GasValid: true, GasLeft: math.MaxInt64}

	vs := &ValidationState{
		chain:     chain,
		stateDB:   stateDB,
		block:     block,
		tx:        tx,
		entryID:   tx.ID,
		gasStatus: gasStatus,
		cache:     make(map[bc.Hash]error),
	}

	gasLeft := int64(0)
	var args [][]byte

	switch e := e.(type) {
	case *bc.Creation:
		if vm.IsOpCreate(e.Input.Code) {
			from, err := segwit.GetHashFromStandardProg(e.From.Code)
			if err != nil {
				return nil, err
			}
			args = append(args, from)
			args = append(args, new(big.Int).SetUint64(e.Nonce).Bytes())
			_, gasLeft, err = vm.Verify(NewTxVMContext(vs, e, e.Input, args), vs.gasStatus.GasLeft)
		}
	case *bc.Call:
		if vm.IsOpCall(e.Input.Code) {
			from, err := segwit.GetHashFromStandardProg(e.From.Code)
			if err != nil {
				return nil, err
			}
			args = append(args, from)
			args = append(args, new(big.Int).SetUint64(e.Nonce).Bytes())
			args = append(args, e.To.Code)
			_, gasLeft, err = vm.Verify(NewTxVMContext(vs, e, e.Input, args), vs.gasStatus.GasLeft)
		}
	case *bc.Contract:
		if vm.IsOpContract(e.Input.Code) {
			from, err := segwit.GetHashFromStandardProg(e.From.Code)
			if err != nil {
				return nil, err
			}
			args = append(args, from)
			args = append(args, new(big.Int).SetUint64(e.Nonce).Bytes())
			args = append(args, e.To)
			_, gasLeft, err = vm.Verify(NewTxVMContext(vs, e, e.Input, args), vs.gasStatus.GasLeft)
		}
	case *bc.Deposit:
		if vm.IsOpDeposit(e.ControlProgram.Code) {
			var args [][]byte
			_, gasLeft, err = vm.Verify(NewTxVMContext(vs, e, e.ControlProgram, args), vs.gasStatus.GasLeft)
		}
	case *bc.Withdrawal:
		if vm.IsOpWithdraw(e.WithdrawProgram.Code) {
			var args [][]byte
			_, gasLeft, err = vm.Verify(NewTxVMContext(vs, e, e.WithdrawProgram, args), vs.gasStatus.GasLeft)
		}
	default:
		return nil, errors.New("unknown program")
	}

	if err != nil {
		return gasStatus, err
	}

	log.WithField("gasUsed", gasStatus.GasLeft-gasLeft).Println("EstimateContractGas")
	err = gasStatus.updateUsage(gasLeft)
	if err != nil {
		return gasStatus, err
	}

	log.WithField("gasUsed", gasStatus.GasUsed).Println("EstimateContractGas")
	return gasStatus, nil

}
