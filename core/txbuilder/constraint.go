package txbuilder

import (
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/vm"
	"github.com/doslink/doslink/protocol/vmutil"
	"github.com/doslink/doslink/basis/crypto/sha3pool"
)

// Constraint types express a constraint on an input of a proposed
// transaction, and know how to turn that constraint into part of a
// signature program in that input's witness.
type constraint interface {
	// Code produces bytecode expressing the constraint. The code, when
	// executed, must consume nothing from the stack and leave a new
	// boolean value on top of it.
	code() []byte
}

// outpointConstraint requires the outputID (and therefore, the outpoint) being spent to equal the
// given value.
type outputIDConstraint bc.Hash

func (o outputIDConstraint) code() []byte {
	builder := vmutil.NewBuilder()
	builder.AddData(bc.Hash(o).Bytes())
	builder.AddOp(vm.OP_OUTPUTID)
	builder.AddOp(vm.OP_EQUAL)
	prog, _ := builder.Build() // error is impossible
	return prog
}

// refdataConstraint requires the refdatahash of the transaction (if
// tx is true) or the input (if tx is false) to match that of the
// given data.
type refdataConstraint struct {
	data []byte
	tx   bool
}

func (r refdataConstraint) code() []byte {
	var h [32]byte
	sha3pool.Sum256(h[:], r.data)
	builder := vmutil.NewBuilder()
	builder.AddData(h[:])
	if r.tx {
		builder.AddOp(vm.OP_TXDATA)
	} else {
		builder.AddOp(vm.OP_ENTRYDATA)
	}
	builder.AddOp(vm.OP_EQUAL)
	prog, _ := builder.Build() // error is impossible
	return prog
}

// PayConstraint requires the transaction to include a given output
// at the given index, optionally with the given refdatahash.
type payConstraint struct {
	Index int
	bc.AssetAmount
	Program []byte
}

func (p payConstraint) code() []byte {
	builder := vmutil.NewBuilder()
	builder.AddInt64(int64(p.Index))
	builder.AddInt64(int64(p.Amount)).AddData(p.AssetId.Bytes()).AddInt64(1).AddData(p.Program)
	builder.AddOp(vm.OP_CHECKOUTPUT)
	prog, _ := builder.Build() // error is impossible
	return prog
}