package txbuilder

import (
	"context"

	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/protocol"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/validation"
)

var (
	// ErrRejected means the network rejected a tx (as a double-spend)
	ErrRejected = errors.New("transaction rejected")
	// ErrMissingRawTx means missing transaction
	ErrMissingRawTx = errors.New("missing raw tx")
	// ErrBadInstructionCount means too many signing instructions compare with inputs
	ErrBadInstructionCount = errors.New("too many signing instructions in template")
	// ErrOrphanTx means submit transaction is orphan
	ErrOrphanTx = errors.New("finalize can't find transaction input utxo")
	// ErrExtTxFee means transaction fee exceed max limit
	ErrExtTxFee = errors.New("transaction fee exceed max limit")
)

// FinalizeTx validates a transaction signature template,
// assembles a fully signed tx, and stores the effects of
// its changes on the UTXO set.
func FinalizeTx(ctx context.Context, c *protocol.Chain, tx *types.Tx, onlyValidate bool) (gasStatus *validation.GasState, err error) {
	// maxTxFee means max transaction fee, maxTxFee = 0.4NativeAsset * 25 = 10NativeAsset
	maxTxFee := consensus.MaxGasAmount * consensus.VMGasRate * 25
	if fee := calculateTxFee(tx); fee > uint64(maxTxFee) {
		return nil, ErrExtTxFee
	}

	// This part is use for prevent tx size  is 0
	data, err := tx.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	tx.TxData.SerializedSize = uint64(len(data))
	tx.Tx.SerializedSize = uint64(len(data))

	acceptable, height, gasStatus, err := c.ValidateTx(tx)

	var isOrphan = false
	if acceptable && !onlyValidate {
		isOrphan, err = c.ProcessTransaction(tx, err != nil, height, gasStatus.AssetValue)
	}

	if errors.Root(err) == protocol.ErrBadTx {
		return gasStatus, errors.Sub(ErrRejected, err)
	}
	if err != nil {
		return gasStatus, errors.WithDetail(err, "tx validate rejected: "+err.Error())
	}

	if isOrphan {
		return gasStatus, ErrOrphanTx
	}
	return gasStatus, nil
}

// calculateTxFee calculate transaction fee
func calculateTxFee(tx *types.Tx) (fee uint64) {
	totalInput := uint64(0)
	totalOutput := uint64(0)

	for _, input := range tx.Inputs {
		if input.InputType() != types.CoinbaseInputType && input.AssetID() == *consensus.NativeAssetID {
			totalInput += input.Amount()
		}
	}

	for _, output := range tx.Outputs {
		if *output.AssetId == *consensus.NativeAssetID {
			totalOutput += output.Amount
		}
	}

	fee = totalInput - totalOutput
	return
}
