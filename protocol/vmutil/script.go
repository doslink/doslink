package vmutil

import (
	"github.com/doslink/doslink/basis/crypto/ed25519"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/protocol/vm"
)

// pre-define errors
var (
	ErrBadValue       = errors.New("bad value")
	ErrMultisigFormat = errors.New("bad multisig program format")
)

// IsUnspendable checks if a contorl program is absolute failed
func IsUnspendable(prog []byte) bool {
	return len(prog) > 0 && prog[0] == byte(vm.OP_FAIL)
}

func (b *Builder) addP2SPMultiSig(pubkeys []ed25519.PublicKey, nrequired int) error {
	if err := checkMultiSigParams(int64(nrequired), int64(len(pubkeys))); err != nil {
		return err
	}

	b.AddOp(vm.OP_TXSIGHASH) // stack is now [... NARGS SIG SIG SIG PREDICATEHASH]
	for _, p := range pubkeys {
		b.AddData(p)
	}
	b.AddInt64(int64(nrequired))    // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M]
	b.AddInt64(int64(len(pubkeys))) // stack is now [... SIG SIG SIG PREDICATEHASH PUB PUB PUB M N]
	b.AddOp(vm.OP_CHECKMULTISIG)    // stack is now [... NARGS]
	return nil
}

// DefaultCoinbaseProgram generates the script for contorl coinbase output
func DefaultCoinbaseProgram() ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_TRUE)
	return builder.Build()
}

// P2WSHProgram return the segwit pay to script hash
func P2WSHProgram(hash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddInt64(0)
	builder.AddData(hash)
	return builder.Build()
}

// RetireProgram generates the script for retire output
func RetireProgram(comment []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_FAIL)
	if len(comment) != 0 {
		builder.AddData(comment)
	}
	return builder.Build()
}

// P2SHProgram generates the script for control with script hash
func P2SHProgram(scriptHash []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_DUP)
	builder.AddOp(vm.OP_HASH160)
	builder.AddData(scriptHash)
	builder.AddOp(vm.OP_EQUALVERIFY)
	builder.AddInt64(-1)
	builder.AddOp(vm.OP_SWAP)
	builder.AddInt64(0)
	builder.AddOp(vm.OP_CHECKPREDICATE)
	return builder.Build()
}

// P2SPMultiSigProgram generates the script for contorl transaction output
func P2SPMultiSigProgram(pubkeys []ed25519.PublicKey, nrequired int) ([]byte, error) {
	builder := NewBuilder()
	if err := builder.addP2SPMultiSig(pubkeys, nrequired); err != nil {
		return nil, err
	}
	return builder.Build()
}

// ParseP2SPMultiSigProgram is unknow for us yet
func ParseP2SPMultiSigProgram(program []byte) ([]ed25519.PublicKey, int, error) {
	pops, err := vm.ParseProgram(program)
	if err != nil {
		return nil, 0, err
	}
	if len(pops) < 11 {
		return nil, 0, vm.ErrShortProgram
	}

	// Count all instructions backwards from the end in case there are
	// extra instructions at the beginning of the program (like a
	// <pushdata> DROP).

	npubkeys, err := vm.AsInt64(pops[len(pops)-6].Data)
	if err != nil {
		return nil, 0, err
	}
	if int(npubkeys) > len(pops)-10 {
		return nil, 0, vm.ErrShortProgram
	}
	nrequired, err := vm.AsInt64(pops[len(pops)-7].Data)
	if err != nil {
		return nil, 0, err
	}
	err = checkMultiSigParams(nrequired, npubkeys)
	if err != nil {
		return nil, 0, err
	}

	firstPubkeyIndex := len(pops) - 7 - int(npubkeys)

	pubkeys := make([]ed25519.PublicKey, 0, npubkeys)
	for i := firstPubkeyIndex; i < firstPubkeyIndex+int(npubkeys); i++ {
		if len(pops[i].Data) != ed25519.PublicKeySize {
			return nil, 0, err
		}
		pubkeys = append(pubkeys, ed25519.PublicKey(pops[i].Data))
	}
	return pubkeys, int(nrequired), nil
}

func checkMultiSigParams(nrequired, npubkeys int64) error {
	if nrequired < 0 {
		return errors.WithDetail(ErrBadValue, "negative quorum")
	}
	if npubkeys < 0 {
		return errors.WithDetail(ErrBadValue, "negative pubkey count")
	}
	if nrequired > npubkeys {
		return errors.WithDetail(ErrBadValue, "quorum too big")
	}
	if nrequired == 0 && npubkeys > 0 {
		return errors.WithDetail(ErrBadValue, "quorum empty with non-empty pubkey list")
	}
	return nil
}

func CreateContractProgram(code []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_0)
	builder.AddData(code)
	builder.AddOp(vm.OP_CREATE)
	return builder.Build()
}

func CallContractProgram(input []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_0)
	builder.AddData(input)
	builder.AddOp(vm.OP_CALL)
	return builder.Build()
}

func ContractProgram(input []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_0)
	builder.AddData(input)
	builder.AddOp(vm.OP_CONTRACT)
	return builder.Build()
}

func P2ContractProgram(vmType int64, address []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_FAIL)
	builder.AddInt64(vmType)
	builder.AddData(address)
	return builder.Build()
}
