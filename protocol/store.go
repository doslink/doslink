package protocol

import (
	"github.com/doslink/doslink/database/storage"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/state"
	dbm "github.com/tendermint/tmlibs/db"
)

// Store provides storage interface for blockchain data
type Store interface {
	BlockExist(*bc.Hash) bool

	GetBlock(*bc.Hash) (*types.Block, error)
	GetStoreStatus() *BlockStoreState
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)

	LoadBlockIndex() (*state.BlockIndex, error)
	SaveBlock(*types.Block, *bc.TransactionStatus) error
	SaveChainStatus(*state.BlockNode, *state.UtxoViewpoint) error

	DB() dbm.DB
}

// BlockStoreState represents the core's db status
type BlockStoreState struct {
	Height uint64
	Hash   *bc.Hash
}
