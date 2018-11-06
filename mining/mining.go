package mining

import (
	"math/rand"
	"sort"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/config"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/core/account"
	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/protocol"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/state"
	"github.com/doslink/doslink/protocol/validation"
	"github.com/doslink/doslink/protocol/vmutil"
)

// createCoinbaseTx returns a coinbase transaction paying an appropriate subsidy
// based on the passed block height to the provided address.  When the address
// is nil, the coinbase transaction will instead be redeemable by anyone.
func createCoinbaseTx(accountManager *account.Manager, amount uint64, blockHeight uint64) (tx *types.Tx, err error) {
	amount += consensus.BlockSubsidy(blockHeight)
	arbitrary := append([]byte{0x00}, []byte(strconv.FormatUint(blockHeight, 10))...)

	var script []byte
	if accountManager == nil {
		script, err = vmutil.DefaultCoinbaseProgram()
	} else {
		script, err = accountManager.GetCoinbaseControlProgram()
		arbitrary = append(arbitrary, accountManager.GetCoinbaseArbitrary()...)
	}
	if err != nil {
		return nil, err
	}

	if len(arbitrary) > consensus.CoinbaseArbitrarySizeLimit {
		return nil, validation.ErrCoinbaseArbitraryOversize
	}

	builder := txbuilder.NewBuilder(time.Now())
	if err = builder.AddInput(types.NewCoinbaseInput(arbitrary), &txbuilder.SigningInstruction{}); err != nil {
		return nil, err
	}
	if err = builder.AddOutput(types.NewTxOutput(*consensus.NativeAssetID, amount, script)); err != nil {
		return nil, err
	}
	_, txData, err := builder.Build()
	if err != nil {
		return nil, err
	}

	byteData, err := txData.MarshalText()
	if err != nil {
		return nil, err
	}
	txData.SerializedSize = uint64(len(byteData))

	tx = &types.Tx{
		TxData: *txData,
		Tx:     types.MapTx(txData),
	}
	return tx, nil
}

// NewBlockTemplate returns a new block template that is ready to be solved
func NewBlockTemplate(c *protocol.Chain, txPool *protocol.TxPool, accountManager *account.Manager) (b *types.Block, err error) {
	view := state.NewUtxoViewpoint()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txEntries := []*bc.Tx{nil}
	gasUsed := uint64(0)
	txFee := uint64(0)

	// get preblock info for generate next block
	preBlockHeader := c.BestBlockHeader()
	preBlockHash := preBlockHeader.Hash()
	nextBlockHeight := preBlockHeader.Height + 1
	nextBits, err := c.CalcNextBits(&preBlockHash)
	if err != nil {
		return nil, err
	}

	blockTime := uint64(time.Now().Unix())
	blockTime = blockTime + (rand.Uint64() % consensus.TargetSecondsPerBlock)
	if blockTime < preBlockHeader.Timestamp {
		blockTime = preBlockHeader.Timestamp
	}

	b = &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            nextBlockHeight,
			PreviousBlockHash: preBlockHash,
			Timestamp:         blockTime,
			BlockCommitment:   types.BlockCommitment{},
			Bits:              nextBits,
		},
	}
	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: nextBlockHeight}}
	b.Transactions = []*types.Tx{nil}

	stateDB, err := protocol.NewState(&preBlockHeader.StateRoot, c)
	if err != nil {
		return nil, err
	}

	txs := txPool.GetTransactions()
	sort.Sort(byTime(txs))
	for _, txDesc := range txs {
		tx := txDesc.Tx.Tx
		gasOnlyTx := false

		if err := c.GetTransactionsUtxo(view, []*bc.Tx{tx}); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			continue
		}

		revision := stateDB.Snapshot()
		stateDB.Prepare(tx.ID.Byte32(), [32]byte{}, len(b.Transactions))
		vs, err := validation.ValidateTx(tx, bcBlock, c, stateDB)
		gasStatus := vs.GasState()
		if err != nil {
			if !gasStatus.GasValid {
				log.WithField("error", err).Error("mining block generate skip tx due to")
				txPool.RemoveTransaction(&tx.ID)
				stateDB.RevertToSnapshot(revision)
				continue
			}
			gasOnlyTx = true
		}

		if gasUsed+uint64(gasStatus.GasUsed) > consensus.MaxBlockGas {
			break
		}

		if err := view.ApplyTransaction(bcBlock, tx, gasOnlyTx); err != nil {
			log.WithField("error", err).Error("mining block generate skip tx due to")
			txPool.RemoveTransaction(&tx.ID)
			stateDB.RevertToSnapshot(revision)
			continue
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

		txStatus.SetLogs(len(b.Transactions), txLogs)
		txStatus.SetStatus(len(b.Transactions), gasOnlyTx)
		b.Transactions = append(b.Transactions, txDesc.Tx)
		txEntries = append(txEntries, tx)
		gasUsed += uint64(gasStatus.GasUsed)
		txFee += txDesc.Fee

		if gasUsed == consensus.MaxBlockGas {
			break
		}
	}

	// creater coinbase transaction
	b.Transactions[0], err = createCoinbaseTx(accountManager, txFee, nextBlockHeight)
	if err != nil {
		return nil, errors.Wrap(err, "fail on createCoinbaseTx")
	}
	txEntries[0] = b.Transactions[0].Tx

	if config.SupportBalanceInStateDB {
		bcBlock.Transactions = append(bcBlock.Transactions, b.Transactions[0].Tx)
		_, err = validation.ValidateTx(b.Transactions[0].Tx, bcBlock, c, stateDB)
		if err != nil {
			return nil, errors.Wrap(err, "fail on validate CoinbaseTx")
		}
	}

	b.BlockHeader.BlockCommitment.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.BlockHeader.BlockCommitment.TransactionStatusHash, err = types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		return nil, err
	}

	b.StateRoot = bc.NewHash(stateDB.IntermediateRoot(true))

	return b, err
}
