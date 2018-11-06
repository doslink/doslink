package config

import (
	"encoding/hex"

	log "github.com/sirupsen/logrus"

	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
)

func genesisTx() *types.Tx {
	contract, err := hex.DecodeString("0014678f9a43d1de0809ff2bbf9b00312a166dfacce8")
	if err != nil {
		log.Panicf("fail on decode genesis tx output control program")
	}

	txData := types.TxData{
		Version: 1,
		Inputs: []*types.TxInput{
			types.NewCoinbaseInput([]byte("Knowledge is power. Learning to get rid of poverty. -- Sep/01/2018.")),
		},
		Outputs: []*types.TxOutput{
			types.NewTxOutput(*consensus.NativeAssetID, consensus.InitialBlockSubsidy, contract),
		},
	}
	return types.NewTx(txData)
}

func mainNetGenesisBlock() *types.Block {
	tx := genesisTx()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     1530935879,
			Timestamp: 1535735358,
			Bits:      2305843009214892324,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
				StateRoot:              bc.Hash{},
			},
		},
		Transactions: []*types.Tx{tx},
	}
	if SupportBalanceInStateDB {
		block.Nonce = 1530935912
	}
	return block
}

func testNetGenesisBlock() *types.Block {
	tx := genesisTx()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     1530936083,
			Timestamp: 1535703376,
			Bits:      2305843009214892324,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
				StateRoot:              bc.Hash{},
			},
		},
		Transactions: []*types.Tx{tx},
	}
	if SupportBalanceInStateDB {
		block.Nonce = 1530936107
	}
	return block
}

func soloNetGenesisBlock() *types.Block {
	tx := genesisTx()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	txStatusHash, err := types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	if err != nil {
		log.Panicf("fail on calc genesis tx status merkle root")
	}

	merkleRoot, err := types.TxMerkleRoot([]*bc.Tx{tx.Tx})
	if err != nil {
		log.Panicf("fail on calc genesis tx merkel root")
	}

	block := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:   1,
			Height:    0,
			Nonce:     42,
			Timestamp: 1535703376,
			Bits:      2305843009214892324,
			BlockCommitment: types.BlockCommitment{
				TransactionsMerkleRoot: merkleRoot,
				TransactionStatusHash:  txStatusHash,
				StateRoot:              bc.Hash{},
			},
		},
		Transactions: []*types.Tx{tx},
	}
	if SupportBalanceInStateDB {
		block.Nonce = 85
	}
	return block
}

// GenesisBlock will return genesis block
func GenesisBlock() *types.Block {
	return map[string]func() *types.Block{
		"main": mainNetGenesisBlock,
		"test": testNetGenesisBlock,
		"solo": soloNetGenesisBlock,
	}[consensus.ActiveNetParams.Name]()
}

var SupportBalanceInStateDB = false

func GenesisBlockHash() *bc.Hash {
	if !SupportBalanceInStateDB {
		return map[string]*bc.Hash{
			"main": {
				V0: uint64(1771503047052980175),
				V1: uint64(17015547388716129558),
				V2: uint64(17123631615990298558),
				V3: uint64(14588836744182067041),
			},
			"test": {
				V0: uint64(6839815717976224938),
				V1: uint64(6440849165116631448),
				V2: uint64(15993109950913757143),
				V3: uint64(11782620654590800031),
			},
			"solo": {
				V0: uint64(13579783433843684229),
				V1: uint64(6083100809378826044),
				V2: uint64(1833562050281205522),
				V3: uint64(8672917913308055504),
			},
		}[consensus.ActiveNetParams.Name]
	} else {
		return map[string]*bc.Hash{
			"main": {
				V0: uint64(8325532997334157898),
				V1: uint64(4189984660549501270),
				V2: uint64(5027510122468721539),
				V3: uint64(18379455307324088015),
			},
			"test": {
				V0: uint64(16672815734135948520),
				V1: uint64(14559210567994881926),
				V2: uint64(426186010121443363),
				V3: uint64(2053255217068846892),
			},
			"solo": {
				V0: uint64(195960844143915746),
				V1: uint64(17468542655660531027),
				V2: uint64(17784038276451449838),
				V3: uint64(661814422175617024),
			},
		}[consensus.ActiveNetParams.Name]
	}
}
