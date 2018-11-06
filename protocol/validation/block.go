package validation

import (
	"time"

	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/consensus/difficulty"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/state"
	evm_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/doslink/doslink/protocol/vm"
)

var (
	errBadTimestamp          = errors.New("block timestamp is not in the valid range")
	errBadBits               = errors.New("block bits is invalid")
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errOverBlockLimit        = errors.New("block's gas is over the limit")
	errWorkProof             = errors.New("invalid difficulty proof of work")
	errVersionRegression     = errors.New("version regression")
)

func checkBlockTime(b *bc.Block, parent *state.BlockNode) error {
	if b.Timestamp > uint64(time.Now().Unix())+consensus.MaxTimeOffsetSeconds {
		return errBadTimestamp
	}

	if b.Timestamp <= parent.CalcPastMedianTime() {
		return errBadTimestamp
	}
	return nil
}

func checkCoinbaseAmount(b *bc.Block, amount uint64) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	output, err := tx.Output(*tx.TxHeader.ResultIds[0])
	if err != nil {
		return err
	}

	if output.Source.Value.Amount != amount {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output amount")
	}
	return nil
}

// ValidateBlockHeader check the block's header
func ValidateBlockHeader(b *bc.Block, parent *state.BlockNode) error {
	if b.Version < parent.Version {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", parent.Version, b.Version)
	}
	if b.Height != parent.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", parent.Height, b.Height)
	}
	if b.Bits != parent.CalcNextBits() {
		return errBadBits
	}
	if parent.Hash != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", parent.Hash.Bytes(), b.PreviousBlockId.Bytes())
	}
	if err := checkBlockTime(b, parent); err != nil {
		return err
	}
	if !difficulty.CheckProofOfWork(&b.ID, parent.CalcNextSeed(), b.BlockHeader.Bits) {
		return errWorkProof
	}
	return nil
}

// ValidateBlock validates a block and the transactions within.
func ValidateBlock(b *bc.Block, parent *state.BlockNode, chain vm.ChainContext, stateDB *evm_state.StateDB) error {
	if err := ValidateBlockHeader(b, parent); err != nil {
		return err
	}

	blockGasSum := uint64(0)
	coinbaseAmount := consensus.BlockSubsidy(b.BlockHeader.Height)
	b.TransactionStatus = bc.NewTransactionStatus()

	for i, tx := range b.Transactions {
		gasOnlyTx := false
		revision := stateDB.Snapshot()
		stateDB.Prepare(tx.ID.Byte32(), b.ID.Byte32(), i)
		vs, err := ValidateTx(tx, b, chain, stateDB)
		gasStatus := vs.gasStatus
		if !gasStatus.GasValid {
			return errors.Wrapf(err, "validate of transaction %d of %d", i, len(b.Transactions))
		}
		if err != nil {
			gasOnlyTx = true
		}

		if gasOnlyTx {
			stateDB.RevertToSnapshot(revision)
		}
		stateDB.Finalise(true)

		var txLogs []*bc.TxLog
		for _, log := range stateDB.GetLogs(tx.ID.Byte32()) {
			var topics [][]byte
			for _, topic := range log.Topics {
				topics = append(topics, topic.Bytes())
			}
			txLogs = append(txLogs,
				&bc.TxLog{
					Address: log.Address.Bytes(),
					Topics:  topics,
					Data:    log.Data,
				},
			)
		}

		b.TransactionStatus.SetLogs(i, txLogs)
		b.TransactionStatus.SetStatus(i, gasOnlyTx)
		coinbaseAmount += gasStatus.AssetValue
		if blockGasSum += uint64(gasStatus.GasUsed); blockGasSum > consensus.MaxBlockGas {
			return errOverBlockLimit
		}
	}

	if err := checkCoinbaseAmount(b, coinbaseAmount); err != nil {
		return err
	}

	txMerkleRoot, err := types.TxMerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction id merkle root")
	}
	if txMerkleRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction id merkle root")
	}

	txStatusHash, err := types.TxStatusMerkleRoot(b.TransactionStatus.VerifyStatus)
	if err != nil {
		return errors.Wrap(err, "computing transaction status merkle root")
	}
	if txStatusHash != *b.TransactionStatusHash {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction status merkle root")
	}

	stateRoot := bc.NewHash(stateDB.IntermediateRoot(true))
	if stateRoot != *b.StateRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "state root")
	}

	return nil
}
