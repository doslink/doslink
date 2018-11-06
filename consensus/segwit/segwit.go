package segwit

import (
	"errors"

	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/protocol/vm"
	"github.com/doslink/doslink/protocol/vmutil"
)

func IsP2WScript(prog []byte) bool {
	return IsP2WSHScript(prog) || IsStraightforward(prog) || IsP2ContractProgram(prog)
}

func IsStraightforward(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 1 {
		return false
	}
	return insts[0].Op == vm.OP_TRUE || insts[0].Op == vm.OP_FAIL
}

func IsP2WSHScript(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 2 {
		return false
	}
	if insts[0].Op > vm.OP_16 {
		return false
	}
	return insts[1].Op == vm.OP_DATA_20 && len(insts[1].Data) == consensus.PayToWitnessScriptHashDataSize
}

func ConvertP2SHProgram(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}
	if insts[0].Op == vm.OP_0 {
		return vmutil.P2SHProgram(insts[1].Data)
	}
	return nil, errors.New("unknown P2SHP version number")
}

func GetHashFromStandardProg(prog []byte) ([]byte, error) {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return nil, err
	}

	return insts[len(insts)-1].Data, nil
}

func IsP2ContractProgram(prog []byte) bool {
	insts, err := vm.ParseProgram(prog)
	if err != nil {
		return false
	}
	if len(insts) != 3 {
		return false
	}

	version := insts[0]
	vmType := insts[1]

	if version.Op > vm.OP_16 {
		return false
	}

	switch vmType.Op {
	case vm.VM_EVM:
	default:
		return false
	}

	return insts[len(insts)-1].Op == vm.OP_DATA_20 && len(insts[len(insts)-1].Data) == 20
}
