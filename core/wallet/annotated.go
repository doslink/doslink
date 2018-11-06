package wallet

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/tendermint/tmlibs/db"

	"github.com/doslink/doslink/basis/crypto/sha3pool"
	chainjson "github.com/doslink/doslink/basis/encoding/json"
	"github.com/doslink/doslink/common"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/consensus/segwit"
	"github.com/doslink/doslink/core/account"
	"github.com/doslink/doslink/core/asset"
	"github.com/doslink/doslink/core/query"
	"github.com/doslink/doslink/core/signers"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/vm"
	"github.com/doslink/doslink/protocol/vmutil"
)

// annotateTxs adds asset data to transactions
func annotateTxsAsset(w *Wallet, txs []*query.AnnotatedTx) {
	for i, tx := range txs {
		for j, input := range tx.Inputs {
			alias, definition := w.getAliasDefinition(input.AssetID)
			txs[i].Inputs[j].AssetAlias, txs[i].Inputs[j].AssetDefinition = alias, &definition
		}
		for k, output := range tx.Outputs {
			alias, definition := w.getAliasDefinition(output.AssetID)
			txs[i].Outputs[k].AssetAlias, txs[i].Outputs[k].AssetDefinition = alias, &definition
		}
	}
}

func (w *Wallet) getExternalDefinition(assetID *bc.AssetID) json.RawMessage {
	definitionByte := w.DB.Get(asset.ExtAssetKey(assetID))
	if definitionByte == nil {
		return nil
	}

	definitionMap := make(map[string]interface{})
	if err := json.Unmarshal(definitionByte, &definitionMap); err != nil {
		return nil
	}

	alias := assetID.String()
	externalAsset := &asset.Asset{
		AssetID:           *assetID,
		Alias:             &alias,
		DefinitionMap:     definitionMap,
		RawDefinitionByte: definitionByte,
		Signer:            &signers.Signer{Type: "external"},
	}

	if err := w.AssetReg.SaveAsset(externalAsset, alias); err != nil {
		log.WithFields(log.Fields{"err": err, "assetID": alias}).Warning("fail on save external asset to internal asset DB")
	}
	return definitionByte
}

func (w *Wallet) getAliasDefinition(assetID bc.AssetID) (string, json.RawMessage) {
	//nativeAsset
	if assetID.String() == consensus.NativeAssetID.String() {
		alias := consensus.NativeAssetAlias
		definition := []byte(asset.DefaultNativeAsset.RawDefinitionByte)

		return alias, definition
	}

	//local asset and saved external asset
	if localAsset, err := w.AssetReg.FindByID(nil, &assetID); err == nil {
		alias := *localAsset.Alias
		definition := []byte(localAsset.RawDefinitionByte)
		return alias, definition
	}

	//external asset
	if definition := w.getExternalDefinition(&assetID); definition != nil {
		return assetID.String(), definition
	}

	return "", nil
}

// annotateTxs adds account data to transactions
func annotateTxsAccount(txs []*query.AnnotatedTx, walletDB db.DB) {
	for i, tx := range txs {
		for j, input := range tx.Inputs {
			//issue asset tx input SpentOutputID is nil
			if input.SpentOutputID == nil {
				continue
			}
			localAccount, err := getAccountFromUTXO(*input.SpentOutputID, walletDB)
			if localAccount == nil || err != nil {
				continue
			}
			txs[i].Inputs[j].AccountAlias = localAccount.Alias
			txs[i].Inputs[j].AccountID = localAccount.ID
		}
		for j, output := range tx.Outputs {
			localAccount, err := getAccountFromACP(output.ControlProgram, walletDB)
			if localAccount == nil || err != nil {
				continue
			}
			txs[i].Outputs[j].AccountAlias = localAccount.Alias
			txs[i].Outputs[j].AccountID = localAccount.ID
		}
	}
}

func getAccountFromUTXO(outputID bc.Hash, walletDB db.DB) (*account.Account, error) {
	accountUTXO := account.UTXO{}
	localAccount := account.Account{}

	accountUTXOValue := walletDB.Get(account.StandardUTXOKey(outputID))
	if accountUTXOValue == nil {
		return nil, fmt.Errorf("failed get account utxo:%x ", outputID)
	}

	if err := json.Unmarshal(accountUTXOValue, &accountUTXO); err != nil {
		return nil, err
	}

	accountValue := walletDB.Get(account.Key(accountUTXO.AccountID))
	if accountValue == nil {
		return nil, fmt.Errorf("failed get account:%s ", accountUTXO.AccountID)
	}
	if err := json.Unmarshal(accountValue, &localAccount); err != nil {
		return nil, err
	}

	return &localAccount, nil
}

func getAccountFromACP(program []byte, walletDB db.DB) (*account.Account, error) {
	var hash common.Hash
	accountCP := account.CtrlProgram{}
	localAccount := account.Account{}

	sha3pool.Sum256(hash[:], program)

	rawProgram := walletDB.Get(account.ContractKey(hash))
	if rawProgram == nil {
		return nil, fmt.Errorf("failed get account control program:%x ", hash)
	}

	if err := json.Unmarshal(rawProgram, &accountCP); err != nil {
		return nil, err
	}

	accountValue := walletDB.Get(account.Key(accountCP.AccountID))
	if accountValue == nil {
		return nil, fmt.Errorf("failed get account:%s ", accountCP.AccountID)
	}

	if err := json.Unmarshal(accountValue, &localAccount); err != nil {
		return nil, err
	}

	return &localAccount, nil
}

var emptyJSONObject = json.RawMessage(`{}`)

func isValidJSON(b []byte) bool {
	var v interface{}
	err := json.Unmarshal(b, &v)
	return err == nil
}

func (w *Wallet) buildAnnotatedTransaction(orig *types.Tx, b *types.Block, txStatus *bc.TransactionStatus, indexInBlock int) *query.AnnotatedTx {
	tx := &query.AnnotatedTx{
		ID:                     orig.ID,
		Timestamp:              b.Timestamp,
		BlockID:                b.Hash(),
		BlockHeight:            b.Height,
		Position:               uint32(indexInBlock),
		BlockTransactionsCount: uint32(len(b.Transactions)),
		Inputs:                 make([]*query.AnnotatedInput, 0, len(orig.Inputs)),
		Outputs:                make([]*query.AnnotatedOutput, 0, len(orig.Outputs)),
		Size:                   orig.SerializedSize,
		Logs:                   []*query.AnnotatedLog{},
		ReferenceData:          &emptyJSONObject,
	}
	for i := range orig.Inputs {
		tx.Inputs = append(tx.Inputs, w.BuildAnnotatedInput(orig, uint32(i)))
	}
	for i := range orig.Outputs {
		tx.Outputs = append(tx.Outputs, w.BuildAnnotatedOutput(orig, i))
	}
	tx.StatusFail, _ = txStatus.GetStatus(indexInBlock)
	logs, _ := txStatus.GetLogs(indexInBlock)
	for _, log := range logs {
		var topics []chainjson.HexBytes
		for _, topic := range log.Topics {
			topics = append(topics, topic)
		}
		tx.Logs = append(tx.Logs,
			&query.AnnotatedLog{
				Address: log.Address,
				Topics:  topics,
				Data:    log.Data,
			},
		)
	}
	if len(orig.ReferenceData) > 0 {
		referenceData := json.RawMessage(orig.ReferenceData)
		tx.ReferenceData = &referenceData
	}

	return tx
}

// BuildAnnotatedInput build the annotated input.
func (w *Wallet) BuildAnnotatedInput(tx *types.Tx, i uint32) *query.AnnotatedInput {
	orig := tx.Inputs[i]
	in := &query.AnnotatedInput{
		AssetDefinition: &emptyJSONObject,
	}
	if orig.InputType() != types.CoinbaseInputType {
		in.AssetID = orig.AssetID()
		in.Amount = orig.Amount()
	}

	id := tx.Tx.InputIDs[i]
	in.InputID = id
	e := tx.Entries[id]
	switch e := e.(type) {
	case *bc.Spend:
		in.Type = "spend"
		in.ControlProgram = orig.ControlProgram()
		in.Address = w.getAddressFromControlProgram(in.ControlProgram)
		in.SpentOutputID = e.SpentOutputId
	case *bc.Issuance:
		in.Type = "issue"
		in.IssuanceProgram = orig.ControlProgram()
	case *bc.Coinbase:
		in.Type = "coinbase"
		in.Arbitrary = e.Arbitrary
	case *bc.Creation:
		in.Type = "creation"
		in.ControlProgram = e.From.Code
		in.Address = w.getAddressFromControlProgram(in.ControlProgram)
		in.Nonce = e.Nonce
		in.Contract = vm.ContractAddress(e.From.Code, e.Nonce)
		in.Data = e.Input.Code
	case *bc.Call:
		in.Type = "call"
		in.ControlProgram = e.From.Code
		in.Address = w.getAddressFromControlProgram(in.ControlProgram)
		in.Nonce = e.Nonce
		in.Contract = e.To.Code
		in.Data = e.Input.Code
	case *bc.Contract:
		in.Type = "contract"
		in.ControlProgram = e.From.Code
		in.Address = w.getAddressFromControlProgram(in.ControlProgram)
		in.Nonce = e.Nonce
		if e.To != nil {
			in.Contract = e.To
		} else {
			in.Contract = vm.ContractAddress(e.From.Code, e.Nonce)
		}
		in.Data = e.Input.Code
	case *bc.Withdrawal:
		in.Type = "withdrawal"
		in.ControlProgram = orig.ControlProgram()
		in.Address = w.getAddressFromControlProgram(in.ControlProgram)
	}
	return in
}

func (w *Wallet) getAddressFromControlProgram(prog []byte) string {
	if segwit.IsP2WSHScript(prog) {
		if scriptHash, err := segwit.GetHashFromStandardProg(prog); err == nil {
			return buildP2SHAddress(scriptHash)
		}
	} else if segwit.IsP2ContractProgram(prog) {
		if addr, err := segwit.GetHashFromStandardProg(prog); err == nil {
			return buildP2ContractAddress(addr)
		}
	} else if vm.IsOpDeposit(prog) {
		if addr, err := vm.GetAddressFromOpDeposit(prog); err == nil {
			return buildAddress(addr)
		}
	} else if vm.IsOpWithdraw(prog) {
		if addr, err := vm.GetAddressFromOpWithdraw(prog); err == nil {
			return buildAddress(addr)
		}
	}

	return ""
}

func buildP2SHAddress(scriptHash []byte) string {
	address := common.BytesToAddress(scriptHash)
	return address.Hex()
}

func buildP2ContractAddress(addr []byte) string {
	address := common.BytesToAddress(addr)
	return address.Hex()
}

func buildAddress(addr []byte) string {
	address := common.BytesToAddress(addr)
	return address.Hex()
}

// BuildAnnotatedOutput build the annotated output.
func (w *Wallet) BuildAnnotatedOutput(tx *types.Tx, idx int) *query.AnnotatedOutput {
	orig := tx.Outputs[idx]
	outid := tx.OutputID(idx)
	out := &query.AnnotatedOutput{
		OutputID:        *outid,
		Position:        idx,
		AssetID:         *orig.AssetId,
		AssetDefinition: &emptyJSONObject,
		Amount:          orig.Amount,
		ControlProgram:  orig.ControlProgram,
		Address:         w.getAddressFromControlProgram(orig.ControlProgram),
	}

	if vmutil.IsUnspendable(out.ControlProgram) {
		out.Type = "retire"
	} else if vm.IsOpDeposit(out.ControlProgram) {
		out.Type = "deposit"
	} else {
		out.Type = "control"
	}
	return out
}
