package protocol

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/config"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/consensus/segwit"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/state"
	vm_state "github.com/doslink/doslink/protocol/vm/state"

	evm_common "github.com/ethereum/go-ethereum/common"
	evm_state "github.com/ethereum/go-ethereum/core/state"
)

const maxProcessBlockChSize = 1024

// Chain provides functions for working with the block chain.
type Chain struct {
	index          *state.BlockIndex
	orphanManage   *OrphanManage
	txPool         *TxPool
	store          Store
	processBlockCh chan *processBlockMsg

	cond     sync.Cond
	bestNode *state.BlockNode
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(store Store, txPool *TxPool) (*Chain, error) {
	c := &Chain{
		orphanManage:   NewOrphanManage(),
		txPool:         txPool,
		store:          store,
		processBlockCh: make(chan *processBlockMsg, maxProcessBlockChSize),
	}
	c.cond.L = new(sync.Mutex)

	storeStatus := store.GetStoreStatus()
	if storeStatus == nil {
		if err := c.initChainStatus(); err != nil {
			return nil, err
		}
		storeStatus = store.GetStoreStatus()
	}

	var err error
	if c.index, err = store.LoadBlockIndex(); err != nil {
		return nil, err
	}

	c.bestNode = c.index.GetNode(storeStatus.Hash)
	c.index.SetMainChain(c.bestNode)
	go c.blockProcesser()
	return c, nil
}

func (c *Chain) initChainStatus() error {
	genesisBlock := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()
	for i := range genesisBlock.Transactions {
		txStatus.SetStatus(i, false)
	}

	if config.SupportBalanceInStateDB {
		// TODO genesisBlock stateRoot
		stateDB, err := NewState(&genesisBlock.StateRoot, c)
		if err != nil {
			return err
		}
		for _, tx := range genesisBlock.Transactions {
			for _, output := range tx.Outputs {
				if bytes.Compare(output.AssetId.Bytes(), consensus.NativeAssetID.Bytes()) == 0 {
					hash, err := segwit.GetHashFromStandardProg(output.ControlProgram)
					if err != nil {
						return err
					}
					address := evm_common.BytesToAddress(hash)
					amount := new(big.Int).SetUint64(output.Amount)
					stateDB.AddBalance(address, amount)
				}
			}
		}
		root := stateDB.IntermediateRoot(true)

		genesisBlock.StateRoot = bc.NewHash(root)

		hash := genesisBlock.Hash()
		if hash != *config.GenesisBlockHash() {
			return fmt.Errorf("expect hash is %v, but get %v", config.GenesisBlockHash(), &hash)
		}

		stateDB.Commit(true)
		stateDB.Database().TrieDB().Commit(root, true)
	} else {
		hash := genesisBlock.Hash()
		if hash != *config.GenesisBlockHash() {
			return fmt.Errorf("expect hash is %v, but get %v", config.GenesisBlockHash(), &hash)
		}
	}

	if err := c.store.SaveBlock(genesisBlock, txStatus); err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	bcBlock := types.MapBlock(genesisBlock)
	if err := utxoView.ApplyBlock(bcBlock, txStatus); err != nil {
		return err
	}

	node, err := state.NewBlockNode(&genesisBlock.BlockHeader, nil)
	if err != nil {
		return err
	}
	return c.store.SaveChainStatus(node, utxoView)
}

// BestBlockHeight returns the current height of the blockchain.
func (c *Chain) BestBlockHeight() uint64 {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestNode.Height
}

// BestBlockHash return the hash of the chain tail block
func (c *Chain) BestBlockHash() *bc.Hash {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return &c.bestNode.Hash
}

// BestBlockHeader returns the chain tail block
func (c *Chain) BestBlockHeader() *types.BlockHeader {
	node := c.index.BestNode()
	return node.BlockHeader()
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(hash bc.Hash) bool {
	return c.index.InMainchain(hash)
}

// CalcNextSeed return the seed for the given block
func (c *Chain) CalcNextSeed(preBlock *bc.Hash) (*bc.Hash, error) {
	node := c.index.GetNode(preBlock)
	if node == nil {
		return nil, errors.New("can't find preblock in the blockindex")
	}
	return node.CalcNextSeed(), nil
}

// CalcNextBits return the seed for the given block
func (c *Chain) CalcNextBits(preBlock *bc.Hash) (uint64, error) {
	node := c.index.GetNode(preBlock)
	if node == nil {
		return 0, errors.New("can't find preblock in the blockindex")
	}
	return node.CalcNextBits(), nil
}

// This function must be called with mu lock in above level
func (c *Chain) setState(node *state.BlockNode, view *state.UtxoViewpoint) error {
	if err := c.store.SaveChainStatus(node, view); err != nil {
		return err
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	c.index.SetMainChain(node)
	c.bestNode = node

	log.WithFields(log.Fields{"height": c.bestNode.Height, "hash": c.bestNode.Hash}).Debug("chain best status has been update")
	c.cond.Broadcast()
	return nil
}

// BlockWaiter returns a channel that waits for the block at the given height.
func (c *Chain) BlockWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.cond.L.Lock()
		defer c.cond.L.Unlock()
		for c.bestNode.Height < height {
			c.cond.Wait()
		}
		ch <- struct{}{}
	}()

	return ch
}

// GetTxPool return chain txpool.
func (c *Chain) GetTxPool() *TxPool {
	return c.txPool
}

func (c *Chain) ProcessTransaction(tx *types.Tx, statusFail bool, height, fee uint64) (bool, error) {
	return c.txPool.ProcessTransaction(tx, statusFail, height, fee)
}

func (c *Chain) Store() *Store {
	return &c.store
}

func NewState(stateRoot *bc.Hash, c *Chain) (*evm_state.StateDB, error) {
	db := vm_state.NewEvmDbWrapper((*c.Store()).DB())
	stateDB := evm_state.NewDatabase(db)
	return evm_state.New(stateRoot.Byte32(), stateDB)
}

func (c *Chain) CurrentState() (*evm_state.StateDB, error) {
	return NewState(&c.BestBlockHeader().StateRoot, c)
}

func (c *Chain) GetAccountNonce(address []byte) (uint64, error) {
	stateDB, err := c.CurrentState()
	if err != nil {
		return 0, err
	}
	return stateDB.GetNonce(evm_common.BytesToAddress(address)), nil
}

func (c *Chain) GetAccountBalance(address []byte) (*big.Int, error) {
	stateDB, err := c.CurrentState()
	if err != nil {
		return nil, err
	}
	return stateDB.GetBalance(evm_common.BytesToAddress(address)), nil
}

func (c *Chain) CanTransfer(address []byte, amount *big.Int) (bool, error) {
	stateDB, err := c.CurrentState()
	if err != nil {
		return false, err
	}
	return stateDB.GetBalance(evm_common.BytesToAddress(address)).Cmp(amount) >= 0, nil
}

func (c *Chain) BestBlockInfo() (height, timestamp, difficulty uint64) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestNode.Height, c.bestNode.Timestamp, c.bestNode.Bits
}

func (c *Chain) GetBlockHashByHeight(height uint64) [32]byte {
	if header, _ := c.GetHeaderByHeight(height); header != nil {
		return header.Hash().Byte32()
	}
	return bc.Hash{}.Byte32()
}
