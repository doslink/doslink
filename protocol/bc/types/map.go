package types

import (
	"github.com/doslink/doslink/basis/crypto/sha3pool"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/vm"
	"github.com/doslink/doslink/protocol/vmutil"
)

// MapTx converts a types TxData object into its entries-based
// representation.
func MapTx(oldTx *TxData) *bc.Tx {
	txID, txHeader, entries := mapTx(oldTx)
	tx := &bc.Tx{
		TxHeader: txHeader,
		ID:       txID,
		Entries:  entries,
		InputIDs: make([]bc.Hash, len(oldTx.Inputs)),
	}

	spentOutputIDs := make(map[bc.Hash]bool)
	for id, e := range entries {
		var ord uint64
		switch e := e.(type) {
		case *bc.Issuance:
			ord = e.Ordinal

		case *bc.Spend:
			ord = e.Ordinal
			spentOutputIDs[*e.SpentOutputId] = true
			if *e.WitnessDestination.Value.AssetId == *consensus.NativeAssetID {
				tx.GasInputIDs = append(tx.GasInputIDs, id)
			}

		case *bc.Coinbase:
			ord = 0

		case *bc.Creation:
			ord = e.Ordinal

		case *bc.Call:
			ord = e.Ordinal

		case *bc.Contract:
			ord = e.Ordinal

		case *bc.Deposit:
			ord = e.Ordinal

		case *bc.Withdrawal:
			ord = e.Ordinal
			if *e.Value.AssetId == *consensus.NativeAssetID {
				tx.GasInputIDs = append(tx.GasInputIDs, id)
			}

		default:
			continue
		}

		if ord >= uint64(len(oldTx.Inputs)) {
			continue
		}
		tx.InputIDs[ord] = id
	}

	for id := range spentOutputIDs {
		tx.SpentOutputIDs = append(tx.SpentOutputIDs, id)
	}
	return tx
}

func mapTx(tx *TxData) (headerID bc.Hash, hdr *bc.TxHeader, entryMap map[bc.Hash]bc.Entry) {
	entryMap = make(map[bc.Hash]bc.Entry)
	addEntry := func(e bc.Entry) bc.Hash {
		id := bc.EntryID(e)
		entryMap[id] = e
		return id
	}

	var (
		spends      []*bc.Spend
		issuances   []*bc.Issuance
		coinbase    *bc.Coinbase
		creations   []*bc.Creation
		calls       []*bc.Call
		contracts   []*bc.Contract
		withdrawals []*bc.Withdrawal
	)

	muxSources := make([]*bc.ValueSource, len(tx.Inputs))
	for i, input := range tx.Inputs {
		switch inp := input.TypedInput.(type) {
		case *IssuanceInput:
			nonceHash := inp.NonceHash()
			assetDefHash := inp.AssetDefinitionHash()
			value := input.AssetAmount()

			issuance := bc.NewIssuance(&nonceHash, &value, uint64(i))
			issuance.WitnessAssetDefinition = &bc.AssetDefinition{
				Data: &assetDefHash,
				IssuanceProgram: &bc.Program{
					VmVersion: inp.VMVersion,
					Code:      inp.IssuanceProgram,
				},
			}
			issuance.WitnessArguments = inp.Arguments
			issuanceID := addEntry(issuance)

			muxSources[i] = &bc.ValueSource{
				Ref:   &issuanceID,
				Value: &value,
			}
			issuances = append(issuances, issuance)

		case *SpendInput:
			// create entry for prevout
			prog := &bc.Program{VmVersion: inp.VMVersion, Code: inp.ControlProgram}
			src := &bc.ValueSource{
				Ref:      &inp.SourceID,
				Value:    &inp.AssetAmount,
				Position: inp.SourcePosition,
			}
			prevout := bc.NewOutput(src, prog, 0) // ordinal doesn't matter for prevouts, only for result outputs
			prevoutID := addEntry(prevout)
			// create entry for spend
			spend := bc.NewSpend(&prevoutID, uint64(i))
			spend.WitnessArguments = inp.Arguments
			spendID := addEntry(spend)
			// setup mux
			muxSources[i] = &bc.ValueSource{
				Ref:   &spendID,
				Value: &inp.AssetAmount,
			}
			spends = append(spends, spend)

		case *CoinbaseInput:
			coinbase = bc.NewCoinbase(inp.Arbitrary)
			coinbaseID := addEntry(coinbase)

			out := tx.Outputs[0]
			muxSources[i] = &bc.ValueSource{
				Ref:   &coinbaseID,
				Value: &out.AssetAmount,
			}

		case *CreationInput:
			creator := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.ControlProgram,
			}
			data := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.Data,
			}
			creation := bc.NewCreation(inp.Nonce, creator, data, inp.Arguments, uint64(i))
			creationID := addEntry(creation)

			muxSources[i] = &bc.ValueSource{
				Ref:   &creationID,
				Value: &bc.AssetAmount{AssetId: &bc.AssetID{}, Amount: 0},
			}
			creations = append(creations, creation)

		case *CallInput:
			from := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.ControlProgram,
			}
			to := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.To,
			}
			data := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.Data,
			}
			call := bc.NewCall(inp.Nonce, from, to, data, inp.Arguments, uint64(i))
			callID := addEntry(call)

			muxSources[i] = &bc.ValueSource{
				Ref:   &callID,
				Value: &bc.AssetAmount{AssetId: &bc.AssetID{}, Amount: 0},
			}
			calls = append(calls, call)

		case *ContractInput:
			from := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.ControlProgram,
			}
			to := inp.To
			data := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.Data,
			}
			contract := bc.NewContract(inp.Nonce, from, to, data, inp.Arguments, uint64(i))
			contractID := addEntry(contract)

			muxSources[i] = &bc.ValueSource{
				Ref:   &contractID,
				Value: &bc.AssetAmount{AssetId: &bc.AssetID{}, Amount: 0},
			}
			contracts = append(contracts, contract)

		case *WithdrawalInput:
			controlProgram := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.ControlProgram,
			}
			withdrawProgram := &bc.Program{
				VmVersion: inp.VMVersion,
				Code:      inp.WithdrawProgram,
			}
			value := input.AssetAmount()
			withdrawal := bc.NewWithdrawal(controlProgram, &value, withdrawProgram, inp.Arguments, uint64(i))
			withdrawalID := addEntry(withdrawal)

			muxSources[i] = &bc.ValueSource{
				Ref:   &withdrawalID,
				Value: &value,
			}
			withdrawals = append(withdrawals, withdrawal)

		}
	}

	mux := bc.NewMux(muxSources, &bc.Program{VmVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	muxID := addEntry(mux)

	// connect the inputs to the mux
	for _, spend := range spends {
		spentOutput := entryMap[*spend.SpentOutputId].(*bc.Output)
		spend.SetDestination(&muxID, spentOutput.Source.Value, spend.Ordinal)
	}
	for _, issuance := range issuances {
		issuance.SetDestination(&muxID, issuance.Value, issuance.Ordinal)
	}

	for _, creation := range creations {
		creation.SetDestination(&muxID, mux.Sources[creation.Ordinal].Value, creation.Ordinal)
	}

	for _, call := range calls {
		call.SetDestination(&muxID, mux.Sources[call.Ordinal].Value, call.Ordinal)
	}

	for _, contract := range contracts {
		contract.SetDestination(&muxID, mux.Sources[contract.Ordinal].Value, contract.Ordinal)
	}

	for _, withdrawal := range withdrawals {
		withdrawal.SetDestination(&muxID, withdrawal.Value, withdrawal.Ordinal)
	}

	if coinbase != nil {
		coinbase.SetDestination(&muxID, mux.Sources[0].Value, 0)
	}

	// convert types.outputs to the bc.output
	var resultIDs []*bc.Hash
	for i, out := range tx.Outputs {
		src := &bc.ValueSource{
			Ref:      &muxID,
			Value:    &out.AssetAmount,
			Position: uint64(i),
		}
		var resultID bc.Hash
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			r := bc.NewRetirement(src, uint64(i))
			resultID = addEntry(r)
		} else if vm.IsOpDeposit(out.ControlProgram) {
			// deposit
			prog := &bc.Program{VmVersion: out.VMVersion, Code: out.ControlProgram}
			o := bc.NewDeposit(src, prog, uint64(i))
			resultID = addEntry(o)
		} else {
			// non-retirement
			prog := &bc.Program{VmVersion: out.VMVersion, Code: out.ControlProgram}
			o := bc.NewOutput(src, prog, uint64(i))
			resultID = addEntry(o)
		}

		dest := &bc.ValueDestination{
			Value:    src.Value,
			Ref:      &resultID,
			Position: 0,
		}
		resultIDs = append(resultIDs, &resultID)
		mux.WitnessDestinations = append(mux.WitnessDestinations, dest)
	}

	refdatahash := hashData(tx.ReferenceData)
	h := bc.NewTxHeader(tx.Version, tx.SerializedSize, tx.TimeRange, resultIDs, &muxID, &refdatahash)
	return addEntry(h), h, entryMap
}

func mapBlockHeader(old *BlockHeader) (bc.Hash, *bc.BlockHeader) {
	bh := bc.NewBlockHeader(old.Version, old.Height, &old.PreviousBlockHash, old.Timestamp, &old.TransactionsMerkleRoot, &old.TransactionStatusHash, &old.StateRoot, old.Nonce, old.Bits)
	return bc.EntryID(bh), bh
}

// MapBlock converts a types block to bc block
func MapBlock(old *Block) *bc.Block {
	if old == nil {
		return nil
	}

	b := new(bc.Block)
	b.ID, b.BlockHeader = mapBlockHeader(&old.BlockHeader)
	for _, oldTx := range old.Transactions {
		b.Transactions = append(b.Transactions, oldTx.Tx)
	}
	return b
}

func hashData(data []byte) bc.Hash {
	var b32 [32]byte
	sha3pool.Sum256(b32[:], data)
	return bc.NewHash(b32)
}
