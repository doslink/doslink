package validation

import (
	"bytes"

	"github.com/doslink/doslink/basis/crypto/sha3pool"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/consensus/segwit"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/vm"
)

// NewTxVMContext generates the vm.Context for BVM
func NewTxVMContext(vs *ValidationState, entry bc.Entry, prog *bc.Program, args [][]byte) *vm.Context {
	var (
		tx          = vs.tx
		blockHeight = vs.block.BlockHeader.GetHeight()
		numResults  = uint64(len(tx.ResultIds))
		txData      = tx.Data.Bytes()
		entryID     = bc.EntryID(entry) // TODO(bobg): pass this in, don't recompute it

		assetID       *[]byte
		amount        *uint64
		entryData     *[]byte
		destPos       *uint64
		spentOutputID *[]byte
		code          *[]byte
	)

	switch e := entry.(type) {
	case *bc.Issuance:
		a1 := e.Value.AssetId.Bytes()
		assetID = &a1
		amount = &e.Value.Amount
		destPos = &e.WitnessDestination.Position
		c1 := witnessProgram(prog.Code)
		code = &c1

	case *bc.Spend:
		spentOutput := tx.Entries[*e.SpentOutputId].(*bc.Output)
		a1 := spentOutput.Source.Value.AssetId.Bytes()
		assetID = &a1
		amount = &spentOutput.Source.Value.Amount
		destPos = &e.WitnessDestination.Position
		s := e.SpentOutputId.Bytes()
		spentOutputID = &s
		c1 := witnessProgram(prog.Code)
		code = &c1

	case *bc.Creation:
		a1 := consensus.NativeAssetID.Bytes()
		assetID = &a1
		a2 := uint64(0)
		amount = &a2
		c1 := witnessProgram(prog.Code)
		code = &c1

	case *bc.Call:
		a1 := consensus.NativeAssetID.Bytes()
		assetID = &a1
		a2 := uint64(0)
		amount = &a2
		c1 := witnessProgram(prog.Code)
		code = &c1

	case *bc.Contract:
		a1 := consensus.NativeAssetID.Bytes()
		assetID = &a1
		a2 := uint64(0)
		amount = &a2
		c1 := witnessProgram(prog.Code)
		code = &c1

	case *bc.Deposit:
		a1 := e.Source.Value.AssetId.Bytes()
		assetID = &a1
		a2 := e.Source.Value.Amount
		amount = &a2
		c1 := witnessProgram(prog.Code)
		code = &c1

	case *bc.Withdrawal:
		a1 := e.Value.AssetId.Bytes()
		assetID = &a1
		amount = &e.Value.Amount
		destPos = &e.WitnessDestination.Position
		c1 := witnessProgram(prog.Code)
		code = &c1

	}

	var txSigHash *[]byte
	txSigHashFn := func() []byte {
		if txSigHash == nil {
			hasher := sha3pool.Get256()
			defer sha3pool.Put256(hasher)

			entryID.WriteTo(hasher)
			tx.ID.WriteTo(hasher)

			var hash bc.Hash
			hash.ReadFrom(hasher)
			hashBytes := hash.Bytes()
			txSigHash = &hashBytes
		}
		return *txSigHash
	}

	ec := &entryContext{
		entry:   entry,
		entries: tx.Entries,
	}

	result := &vm.Context{
		Chain:     vs.chain,
		StateDB:   vs.stateDB,
		VMVersion: prog.VmVersion,
		Code:      *code,
		Arguments: args,

		EntryID: entryID.Bytes(),

		TxVersion:   &tx.Version,
		BlockHeight: &blockHeight,

		TxSigHash:     txSigHashFn,
		NumResults:    &numResults,
		AssetID:       assetID,
		Amount:        amount,
		EntryData:     entryData,
		TxData:        &txData,
		DestPos:       destPos,
		SpentOutputID: spentOutputID,
		CheckOutput:   ec.checkOutput,
	}

	return result
}

func witnessProgram(prog []byte) []byte {
	if segwit.IsP2WSHScript(prog) {
		if witnessProg, err := segwit.ConvertP2SHProgram([]byte(prog)); err == nil {
			return witnessProg
		}
	}
	return prog
}

type entryContext struct {
	entry   bc.Entry
	entries map[bc.Hash]bc.Entry
}

func (ec *entryContext) checkOutput(index uint64, amount uint64, assetID []byte, vmVersion uint64, code []byte, expansion bool) (bool, error) {
	checkEntry := func(e bc.Entry) (bool, error) {
		check := func(prog *bc.Program, value *bc.AssetAmount) bool {
			return prog.VmVersion == vmVersion &&
				bytes.Equal(prog.Code, code) &&
				bytes.Equal(value.AssetId.Bytes(), assetID) &&
				value.Amount == amount
		}

		switch e := e.(type) {
		case *bc.Output:
			return check(e.ControlProgram, e.Source.Value), nil

		case *bc.Retirement:
			var prog bc.Program
			if expansion {
				// The spec requires prog.Code to be the empty string only
				// when !expansion. When expansion is true, we prepopulate
				// prog.Code to give check() a freebie match.
				//
				// (The spec always requires prog.VmVersion to be zero.)
				prog.Code = code
			}
			return check(&prog, e.Source.Value), nil

		case *bc.Deposit:
			return check(e.ControlProgram, e.Source.Value), nil

		}

		return false, vm.ErrContext
	}

	checkMux := func(m *bc.Mux) (bool, error) {
		if index >= uint64(len(m.WitnessDestinations)) {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= %d", index, len(m.WitnessDestinations))
		}
		eID := m.WitnessDestinations[index].Ref
		e, ok := ec.entries[*eID]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for mux destination %d, id %x, not found", index, eID.Bytes())
		}
		return checkEntry(e)
	}

	switch e := ec.entry.(type) {
	case *bc.Mux:
		return checkMux(e)

	case *bc.Issuance:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for issuance destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)

	case *bc.Spend:
		d, ok := ec.entries[*e.WitnessDestination.Ref]
		if !ok {
			return false, errors.Wrapf(bc.ErrMissingEntry, "entry for spend destination %x not found", e.WitnessDestination.Ref.Bytes())
		}
		if m, ok := d.(*bc.Mux); ok {
			return checkMux(m)
		}
		if index != 0 {
			return false, errors.Wrapf(vm.ErrBadValue, "index %d >= 1", index)
		}
		return checkEntry(d)
	}

	return false, vm.ErrContext
}
