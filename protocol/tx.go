package protocol

import (
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/state"
	"github.com/doslink/doslink/protocol/validation"
)

// ErrBadTx is returned for transactions failing validation
var ErrBadTx = errors.New("invalid transaction")

// GetTransactionStatus return the transaction status of give block
func (c *Chain) GetTransactionStatus(hash *bc.Hash) (*bc.TransactionStatus, error) {
	return c.store.GetTransactionStatus(hash)
}

// GetTransactionsUtxo return all the utxos that related to the txs' inputs
func (c *Chain) GetTransactionsUtxo(view *state.UtxoViewpoint, txs []*bc.Tx) error {
	return c.store.GetTransactionsUtxo(view, txs)
}

// ValidateTx validates the given transaction. A cache holds
// per-transaction validation results and is consulted before
// performing full validation.
func (c *Chain) ValidateTx(tx *types.Tx) (acceptable bool, height uint64, gasStatus *validation.GasState, err error) {
	defer func() {
		if err != nil {
			c.txPool.AddErrCache(&tx.ID, err)
		}
	}()

	if c.txPool.IsTransactionInErrCache(&tx.ID) {
		return false, 0, nil, c.txPool.GetErrCache(&tx.ID)
	}
	if ok := c.txPool.IsTransactionInPool(&tx.ID); ok {
		return false, 0, nil, ErrTransactionIsInPool
	}

	bh := c.BestBlockHeader()
	block := types.MapBlock(&types.Block{BlockHeader: *bh})

	stateDB, err := NewState(&bh.StateRoot, c)
	if err != nil {
		return false, 0, nil, err
	}

	stateDB.Prepare(tx.ID.Byte32(), [32]byte{}, 0)
	vs, err := validation.ValidateTx(tx.Tx, block, c, stateDB)
	gasStatus = vs.GasState()
	if gasStatus.GasValid == false {
		c.txPool.AddErrCache(&tx.ID, err)
		return false, 0, gasStatus, err
	}

	return true, block.BlockHeader.Height, gasStatus, err
}
