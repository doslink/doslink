package account

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/doslink/doslink/basis/crypto/ed25519/chainkd"
	chainjson "github.com/doslink/doslink/basis/encoding/json"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/config"
	"github.com/doslink/doslink/core/signers"
	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/vm"
	"github.com/doslink/doslink/protocol/vmutil"
)

//DecodeSpendAction unmarshal JSON-encoded data of spend action
func (m *Manager) DecodeSpendAction(data []byte) (txbuilder.Action, error) {
	a := &spendAction{accounts: m}
	return a, json.Unmarshal(data, a)
}

type spendAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID      string `json:"account_id"`
	UseUnconfirmed bool   `json:"use_unconfirmed"`
}

// MergeSpendAction merge common assetID and accountID spend action
func MergeSpendAction(actions []txbuilder.Action) []txbuilder.Action {
	var resultActions []txbuilder.Action
	spendActionMap := make(map[string]*spendAction)

	for _, act := range actions {
		switch act := act.(type) {
		case *spendAction:
			actionKey := act.AssetId.String() + act.AccountID
			if tmpAct, ok := spendActionMap[actionKey]; ok {
				tmpAct.Amount += act.Amount
				tmpAct.UseUnconfirmed = tmpAct.UseUnconfirmed || act.UseUnconfirmed
			} else {
				spendActionMap[actionKey] = act
				resultActions = append(resultActions, act)
			}
		default:
			resultActions = append(resultActions, act)
		}
	}
	return resultActions
}

func (a *spendAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	res, err := a.accounts.utxoKeeper.Reserve(a.AccountID, a.AssetId, a.Amount, a.UseUnconfirmed, b.MaxTime())
	if err != nil {
		return errors.Wrap(err, "reserving utxos")
	}

	// Cancel the reservation if the build gets rolled back.
	b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })
	for _, r := range res.utxos {
		txInput, sigInst, err := UtxoToInputs(acct.Signer, r)
		if err != nil {
			return errors.Wrap(err, "creating inputs")
		}

		if err = b.AddInput(txInput, sigInst); err != nil {
			return errors.Wrap(err, "adding inputs")
		}
	}

	if res.change > 0 {
		acp, err := a.accounts.CreateAddress(a.AccountID, true)
		if err != nil {
			return errors.Wrap(err, "creating control program")
		}

		// Don't insert the control program until callbacks are executed.
		a.accounts.insertControlProgramDelayed(b, acp)
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, res.change, acp.ControlProgram)); err != nil {
			return errors.Wrap(err, "adding change output")
		}
	}
	return nil
}

//DecodeSpendUTXOAction unmarshal JSON-encoded data of spend utxo action
func (m *Manager) DecodeSpendUTXOAction(data []byte) (txbuilder.Action, error) {
	a := &spendUTXOAction{accounts: m}
	return a, json.Unmarshal(data, a)
}

type spendUTXOAction struct {
	accounts       *Manager
	OutputID       *bc.Hash                     `json:"output_id"`
	UseUnconfirmed bool                         `json:"use_unconfirmed"`
	Arguments      []txbuilder.ContractArgument `json:"arguments"`
}

func (a *spendUTXOAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	if a.OutputID == nil {
		return txbuilder.MissingFieldsError("output_id")
	}

	res, err := a.accounts.utxoKeeper.ReserveParticular(*a.OutputID, a.UseUnconfirmed, b.MaxTime())
	if err != nil {
		return err
	}

	b.OnRollback(func() { a.accounts.utxoKeeper.Cancel(res.id) })
	var accountSigner *signers.Signer
	if len(res.utxos[0].AccountID) != 0 {
		account, err := a.accounts.FindByID(res.utxos[0].AccountID)
		if err != nil {
			return err
		}

		accountSigner = account.Signer
	}

	txInput, sigInst, err := UtxoToInputs(accountSigner, res.utxos[0])
	if err != nil {
		return err
	}

	if a.Arguments == nil {
		return b.AddInput(txInput, sigInst)
	}

	sigInst = &txbuilder.SigningInstruction{}
	if err := txbuilder.AddContractArgs(sigInst, a.Arguments); err != nil {
		return err
	}

	return b.AddInput(txInput, sigInst)
}

// UtxoToInputs convert an utxo to the txinput
func UtxoToInputs(signer *signers.Signer, u *UTXO) (*types.TxInput, *txbuilder.SigningInstruction, error) {
	txInput := types.NewSpendInput(nil, u.SourceID, u.AssetID, u.Amount, u.SourcePos, u.ControlProgram)
	sigInst, err := SigningInstruction(signer, u.ControlProgramIndex, u.Address)
	if err != nil {
		return nil, nil, err
	}
	return txInput, sigInst, nil
}

func SigningInstruction(signer *signers.Signer, keyIndex uint64, addr string) (*txbuilder.SigningInstruction, error) {
	sigInst := &txbuilder.SigningInstruction{}
	if signer == nil {
		return sigInst, nil
	}

	path := signers.Path(signer, signers.AccountKeySpace, keyIndex)
	if addr == "" {
		sigInst.AddWitnessKeys(signer.XPubs, path, signer.Quorum)
		return sigInst, nil
	}

	_, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	{
		sigInst.AddRawWitnessKeys(signer.XPubs, path, signer.Quorum)
		path := signers.Path(signer, signers.AccountKeySpace, keyIndex)
		derivedXPubs := chainkd.DeriveXPubs(signer.XPubs, path)
		derivedPKs := chainkd.XPubKeys(derivedXPubs)
		script, err := vmutil.P2SPMultiSigProgram(derivedPKs, signer.Quorum)
		if err != nil {
			return nil, err
		}
		sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(script))
	}
	return sigInst, nil
}

// insertControlProgramDelayed takes a template builder and an account
// control program that hasn't been inserted to the database yet. It
// registers callbacks on the TemplateBuilder so that all of the template's
// account control programs are batch inserted if building the rest of
// the template is successful.
func (m *Manager) insertControlProgramDelayed(b *txbuilder.TemplateBuilder, acp *CtrlProgram) {
	m.delayedACPsMu.Lock()
	m.delayedACPs[b] = append(m.delayedACPs[b], acp)
	m.delayedACPsMu.Unlock()

	b.OnRollback(func() {
		m.delayedACPsMu.Lock()
		delete(m.delayedACPs, b)
		m.delayedACPsMu.Unlock()
	})
	b.OnBuild(func() error {
		m.delayedACPsMu.Lock()
		acps := m.delayedACPs[b]
		delete(m.delayedACPs, b)
		m.delayedACPsMu.Unlock()

		// Insert all of the account control programs at once.
		if len(acps) == 0 {
			return nil
		}
		return m.insertControlPrograms(acps...)
	})
}

// DecodeCreateContractAction convert input data to action struct
func (m *Manager) DecodeCreateContractAction(data []byte) (txbuilder.Action, error) {
	a := &createContractAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type createContractAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID string             `json:"account_id"`
	Contract  chainjson.HexBytes `json:"input"`
	Creator   string             `json:"from"`
	Nonce     chainjson.HexBytes `json:"nonce"`
	VM        int64              `json:"vm"`
}

func (a *createContractAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if len(a.Contract) == 0 {
		missing = append(missing, "contract")
	}
	if a.AssetId.IsZero() && !(a.Amount == 0) {
		missing = append(missing, "asset_id")
	}
	//if !a.AssetId.IsZero() && (a.Amount == 0) {
	//	missing = append(missing, "amount")
	//}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	creator, err := getSender(a.accounts, a.AccountID, a.Creator)
	if err != nil {
		return err
	}
	address, _ := hex.DecodeString(creator.Address)

	nonce := uint64(0)
	if a.Nonce == nil {
		nonce, err = b.Chain().GetAccountNonce(address)
		if err != nil {
			return err
		}
	} else {
		nonce = new(big.Int).SetBytes(a.Nonce.Bytes()).Uint64()
	}

	createContractProgram, err := vmutil.CreateContractProgram(a.Contract)
	if err != nil {
		return err
	}

	txInput := types.NewCreationInput(creator.ControlProgram, nonce, createContractProgram, nil)
	sigInst, err := SigningInstruction(acct.Signer, creator.KeyIndex, creator.Address)
	if err != nil {
		return err
	}

	if err = b.AddInput(txInput, sigInst); err != nil {
		return errors.Wrap(err, "adding inputs")
	}

	if config.SupportBalanceInStateDB && a.Amount > 0 {
		toAddress := vm.ContractAddress(address, nonce)
		toProgram, err := vmutil.P2ContractProgram(a.VM, toAddress)
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, a.Amount, toProgram)); err != nil {
			return errors.Wrap(err, "adding create output")
		}
	}

	if a.Amount > 0 {
		depositProgram, err := vmutil.DepositProgram(a.VM, address)
		if err != nil {
			return err
		}
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, a.Amount, depositProgram)); err != nil {
			return errors.Wrap(err, "adding deposit output")
		}
	}

	return nil
}

func getSender(accounts *Manager, accountID, senderAddress string) (sender *CtrlProgram, err error) {
	var cp *CtrlProgram
	if senderAddress == "" {
		cp, err = accounts.CreateAddress(accountID, false)
	} else {
		cp, err = accounts.GetLocalCtrlProgramByAddress(senderAddress)
	}
	if err != nil {
		return nil, err
	}
	if accountID != cp.AccountID {
		return nil, errors.New("accountID mismatch")
	}
	return cp, nil
}

// DecodeSendToContractAction convert input data to action struct
func (m *Manager) DecodeSendToContractAction(data []byte) (txbuilder.Action, error) {
	a := &sendToContractAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type sendToContractAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID string             `json:"account_id"`
	Contract  chainjson.HexBytes `json:"to"`
	Input     chainjson.HexBytes `json:"input"`
	Sender    string             `json:"from"`
	Nonce     chainjson.HexBytes `json:"nonce"`
	VM        int64              `json:"vm"`
}

func (a *sendToContractAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if len(a.Contract) == 0 {
		missing = append(missing, "contract")
	}
	if a.AssetId.IsZero() && !(a.Amount == 0) {
		missing = append(missing, "asset_id")
	}
	//if !a.AssetId.IsZero() && (a.Amount == 0) {
	//	missing = append(missing, "amount")
	//}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	sender, err := getSender(a.accounts, a.AccountID, a.Sender)
	if err != nil {
		return err
	}
	address, _ := hex.DecodeString(sender.Address)

	nonce := uint64(0)
	if a.Nonce == nil {
		nonce, err = b.Chain().GetAccountNonce(address)
		if err != nil {
			return err
		}
	} else {
		nonce = new(big.Int).SetBytes(a.Nonce.Bytes()).Uint64()
	}

	callContractProgram, err := vmutil.CallContractProgram(a.Input)
	if err != nil {
		return err
	}

	txInput := types.NewCallInput(sender.ControlProgram, nonce, a.Contract, callContractProgram, nil)
	sigInst, err := SigningInstruction(acct.Signer, sender.KeyIndex, sender.Address)
	if err != nil {
		return err
	}

	if err = b.AddInput(txInput, sigInst); err != nil {
		return errors.Wrap(err, "adding inputs")
	}

	if config.SupportBalanceInStateDB && a.Amount > 0 {
		toAddress := a.Contract
		if toAddress == nil {
			toAddress = vm.ContractAddress(address, nonce)
		}
		toProgram, err := vmutil.P2ContractProgram(a.VM, toAddress)
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, a.Amount, toProgram)); err != nil {
			return errors.Wrap(err, "adding call output")
		}
	}

	if a.Amount > 0 {
		depositProgram, err := vmutil.DepositProgram(a.VM, address)
		if err != nil {
			return err
		}
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, a.Amount, depositProgram)); err != nil {
			return errors.Wrap(err, "adding deposit output")
		}
	}

	return nil
}

// DecodeContractAction convert input data to action struct
func (m *Manager) DecodeContractAction(data []byte) (txbuilder.Action, error) {
	a := &contractAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type contractAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID string             `json:"account_id"`
	From      string             `json:"from"`
	Nonce     chainjson.HexBytes `json:"nonce"`
	To        chainjson.HexBytes `json:"to"`
	Input     chainjson.HexBytes `json:"input"`
	VM        int64              `json:"vm"`
}

func (a *contractAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) (err error) {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if len(a.Input) == 0 {
		missing = append(missing, "input")
	}
	if a.AssetId.IsZero() && !(a.Amount == 0) {
		missing = append(missing, "asset_id")
	}
	//if !a.AssetId.IsZero() && (a.Amount == 0) {
	//	missing = append(missing, "amount")
	//}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	sender, err := getSender(a.accounts, a.AccountID, a.From)
	if err != nil {
		return err
	}
	address, _ := hex.DecodeString(sender.Address)

	nonce := uint64(0)
	if a.Nonce == nil {
		nonce, err = b.Chain().GetAccountNonce(address)
		if err != nil {
			return err
		}
	} else {
		nonce = new(big.Int).SetBytes(a.Nonce.Bytes()).Uint64()
	}

	contractProgram, err := vmutil.ContractProgram(a.Input)
	if err != nil {
		return err
	}

	txInput := types.NewContractInput(sender.ControlProgram, nonce, a.To, contractProgram, nil)
	sigInst, err := SigningInstruction(acct.Signer, sender.KeyIndex, sender.Address)
	if err != nil {
		return err
	}

	if err = b.AddInput(txInput, sigInst); err != nil {
		return errors.Wrap(err, "adding inputs")
	}

	if config.SupportBalanceInStateDB && a.Amount > 0 {
		toAddress := a.To
		if toAddress == nil {
			toAddress = vm.ContractAddress(address, nonce)
		}
		toProgram, err := vmutil.P2ContractProgram(a.VM, toAddress)
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, a.Amount, toProgram)); err != nil {
			return errors.Wrap(err, "adding contract output")
		}
	}

	if a.Amount > 0 {
		depositProgram, err := vmutil.DepositProgram(a.VM, address)
		if err != nil {
			return err
		}
		if err = b.AddOutput(types.NewTxOutput(*a.AssetId, a.Amount, depositProgram)); err != nil {
			return errors.Wrap(err, "adding deposit output")
		}
	}

	return nil
}
