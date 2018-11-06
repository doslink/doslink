// Package account stores and tracks accounts within a Chain Core.
package account

import (
	"sync"

	"github.com/golang/groupcache/lru"
	dbm "github.com/tendermint/tmlibs/db"

	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/protocol"
)

const (
	maxAccountCache = 1000
)

var (
	accountIndexKey     = []byte("AccountIndex")
	accountPrefix       = []byte("Account:")
	aliasPrefix         = []byte("AccountAlias:")
	contractIndexPrefix = []byte("ContractIndex:")
	contractPrefix      = []byte("Contract:")
	miningAddressKey    = []byte("MiningAddress")
	CoinbaseAbKey       = []byte("CoinbaseArbitrary")
)

// pre-define errors for supporting errorFormatter
var (
	ErrDuplicateAlias  = errors.New("duplicate account alias")
	ErrFindAccount     = errors.New("fail to find account")
	ErrMarshalAccount  = errors.New("failed marshal account")
	ErrInvalidAddress  = errors.New("invalid address")
	ErrFindCtrlProgram = errors.New("fail to find account control program")
)

// Manager stores accounts and their associated control programs.
type Manager struct {
	db         dbm.DB
	chain      *protocol.Chain
	utxoKeeper *utxoKeeper

	cacheMu    sync.Mutex
	cache      *lru.Cache
	aliasCache *lru.Cache

	delayedACPsMu sync.Mutex
	delayedACPs   map[*txbuilder.TemplateBuilder][]*CtrlProgram

	accIndexMu sync.Mutex
	accountMu  sync.Mutex
}

// NewManager creates a new account manager
func NewManager(walletDB dbm.DB, chain *protocol.Chain) *Manager {
	return &Manager{
		db:          walletDB,
		chain:       chain,
		utxoKeeper:  newUtxoKeeper(chain.BestBlockHeight, walletDB),
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*CtrlProgram),
	}
}
