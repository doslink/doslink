package wallet

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/doslink/doslink/core/account"
	"github.com/doslink/doslink/core/asset"
	"github.com/doslink/doslink/core/pseudohsm"
	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/basis/crypto/ed25519/chainkd"
	"github.com/doslink/doslink/database/leveldb"
	"github.com/doslink/doslink/protocol"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
)

func TestWalletUpdate(t *testing.T) {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	store := leveldb.NewStore(testDB)
	txPool := protocol.NewTxPool(store)

	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		t.Fatal(err)
	}

	accountManager := account.NewManager(testDB, chain)
	hsm, err := pseudohsm.New(dirPath)
	if err != nil {
		t.Fatal(err)
	}

	xpub1, err := hsm.XCreate("test_pub1", "password")
	if err != nil {
		t.Fatal(err)
	}

	testAccount, err := accountManager.Create([]chainkd.XPub{xpub1.XPub}, 1, "testAccount")
	if err != nil {
		t.Fatal(err)
	}

	controlProg, err := accountManager.CreateAddress(testAccount.ID, false)
	if err != nil {
		t.Fatal(err)
	}

	controlProg.KeyIndex = 1

	reg := asset.NewRegistry(testDB, chain)
	asset, err := reg.Define([]chainkd.XPub{xpub1.XPub}, 1, nil, "TESTASSET", nil)
	if err != nil {
		t.Fatal(err)
	}

	utxos := []*account.UTXO{}
	nativeAssetUtxo := mockUTXO(controlProg, consensus.NativeAssetID)
	utxos = append(utxos, nativeAssetUtxo)
	OtherUtxo := mockUTXO(controlProg, &asset.AssetID)
	utxos = append(utxos, OtherUtxo)

	_, txData, err := mockTxData(utxos, testAccount)
	if err != nil {
		t.Fatal(err)
	}

	tx := types.NewTx(*txData)
	block := mockSingleBlock(tx)
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)
	store.SaveBlock(block, txStatus)

	w := mockWallet(testDB, accountManager, reg, chain)
	err = w.AttachBlock(block)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.GetTransactionByTxID(tx.ID.String()); err != nil {
		t.Fatal(err)
	}

	wants, err := w.GetTransactions("")
	if len(wants) != 1 {
		t.Fatal(err)
	}
}

func mockUTXO(controlProg *account.CtrlProgram, assetID *bc.AssetID) *account.UTXO {
	utxo := &account.UTXO{}
	utxo.OutputID = bc.Hash{V0: 1}
	utxo.SourceID = bc.Hash{V0: 2}
	utxo.AssetID = *assetID
	utxo.Amount = 1000000000
	utxo.SourcePos = 0
	utxo.ControlProgram = controlProg.ControlProgram
	utxo.AccountID = controlProg.AccountID
	utxo.Address = controlProg.Address
	utxo.ControlProgramIndex = controlProg.KeyIndex
	return utxo
}

func mockTxData(utxos []*account.UTXO, testAccount *account.Account) (*txbuilder.Template, *types.TxData, error) {
	tplBuilder := txbuilder.NewBuilder(time.Now())

	for _, utxo := range utxos {
		txInput, sigInst, err := account.UtxoToInputs(testAccount.Signer, utxo)
		if err != nil {
			return nil, nil, err
		}
		tplBuilder.AddInput(txInput, sigInst)

		out := &types.TxOutput{}
		if utxo.AssetID == *consensus.NativeAssetID {
			out = types.NewTxOutput(utxo.AssetID, 100, utxo.ControlProgram)
		} else {
			out = types.NewTxOutput(utxo.AssetID, utxo.Amount, utxo.ControlProgram)
		}
		tplBuilder.AddOutput(out)
	}

	return tplBuilder.Build()
}

func mockWallet(walletDB dbm.DB, account *account.Manager, asset *asset.Registry, chain *protocol.Chain) *Wallet {
	wallet := &Wallet{
		DB:         walletDB,
		AccountMgr: account,
		AssetReg:   asset,
		chain:      chain,
	}
	return wallet
}

func mockSingleBlock(tx *types.Tx) *types.Block {
	return &types.Block{
		BlockHeader: types.BlockHeader{
			Version: 1,
			Height:  1,
			Bits:    2305843009230471167,
		},
		Transactions: []*types.Tx{tx},
	}
}
