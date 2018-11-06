package protocol

import (
	"testing"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/consensus/difficulty"
	"github.com/doslink/doslink/config"
	"github.com/doslink/doslink/protocol/bc"
	"math/big"
	"github.com/doslink/doslink/basis/crypto"
	evm_common "github.com/ethereum/go-ethereum/common"
	evm_state "github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"bytes"
)

func TestChain_initChainStatus(t *testing.T) {
	config.SupportBalanceInStateDB = false
	for _, netParams := range consensus.NetParams {
		t.Log("ActiveNetParams:", netParams)
		consensus.ActiveNetParams = netParams
		genesisBlock := config.GenesisBlock()

		if config.SupportBalanceInStateDB {
			// TODO genesisBlock stateRoot
			database := evm_state.NewDatabase(ethdb.NewMemDatabase())
			stateDB, _ := evm_state.New(genesisBlock.StateRoot.Byte32(), database)
			for _, tx := range genesisBlock.Transactions {
				for _, output := range tx.Outputs {
					if bytes.Compare(output.AssetId.Bytes(), consensus.NativeAssetID.Bytes()) == 0 {
						address := evm_common.BytesToAddress(crypto.Ripemd160(output.ControlProgram))
						amount := new(big.Int).SetUint64(output.Amount)
						stateDB.AddBalance(address, amount)
					}
				}
			}
			root := stateDB.IntermediateRoot(true)
			t.Log("stateRoot:", root.Hex())
			genesisBlock.StateRoot = bc.NewHash(root)

			for _, tx := range genesisBlock.Transactions {
				for _, output := range tx.Outputs {
					if output.AssetId.String() == consensus.NativeAssetID.String() {
						address := evm_common.BytesToAddress(crypto.Ripemd160(output.ControlProgram))
						amount := new(big.Int).SetUint64(output.Amount)
						t.Log("address:", address.Hex(), "balance:", stateDB.GetBalance(address), "amount", amount)
					}
				}
			}
		}

		nonce := genesisBlock.Nonce
		hash := genesisBlock.Hash()
		for {
			if difficulty.CheckProofOfWork(&hash, consensus.InitialSeed, genesisBlock.Bits) {
				t.Log("genesisHash:", &hash)
				t.Log("genesisHash:", hash)
				break
			}
			genesisBlock.Nonce++
			hash = genesisBlock.Hash()
		}
		if genesisBlock.Nonce != nonce {
			t.Errorf("correct nonce is %d, but get %d", genesisBlock.Nonce, nonce)
		}
		if hash != *config.GenesisBlockHash() {
			t.Errorf("expect hash is %v, but get %v", config.GenesisBlockHash(), &hash)
		}
	}
}
