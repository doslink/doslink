// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/doslink/doslink/protocol/vm/evm"
	"github.com/ethereum/go-ethereum/core"
	"github.com/doslink/doslink/consensus"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	BestBlockInfo() (height, timestamp, difficulty uint64)
	GetBlockHashByHeight(uint64) ([32]byte)
}

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(msg core.Message, height, timestamp, difficulty uint64, chain ChainContext, author *common.Address) evm.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary common.Address = *author
	return evm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn(chain),
		Origin:      msg.From(),
		Coinbase:    beneficiary,
		BlockNumber: new(big.Int).SetUint64(height),
		Time:        new(big.Int).SetUint64(timestamp),
		Difficulty:  new(big.Int).SetUint64(difficulty),
		GasLimit:    consensus.MaxBlockGas,
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(chain ChainContext) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		return common.Hash(chain.GetBlockHashByHeight(n))
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db evm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db evm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
