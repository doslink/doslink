package account

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/vmutil"
)

// VM as a bank
// 1.deposit: UTXO -> VM(stateDB)
// 2.withdraw: VM(stateDB) -> UTXO

// DecodeDepositAction convert input data to action struct
func (m *Manager) DecodeDepositAction(data []byte) (txbuilder.Action, error) {
	a := new(depositAction)
	err := json.Unmarshal(data, a)
	return a, err
}

type depositAction struct {
	bc.AssetAmount
	Address string `json:"address"`
	VM      int64  `json:"vm"`
}

func (a *depositAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	address, err := hex.DecodeString(a.Address)
	if err != nil {
		return err
	}
	program, err := vmutil.DepositProgram(a.VM, address)
	if err != nil {
		return err
	}

	out := types.NewTxOutput(*a.AssetId, a.Amount, program)
	return b.AddOutput(out)
}

// DecodeWithdrawAction convert input data to action struct
func (m *Manager) DecodeWithdrawAction(data []byte) (txbuilder.Action, error) {
	a := &withdrawAction{accounts: m}
	err := json.Unmarshal(data, a)
	return a, err
}

type withdrawAction struct {
	accounts *Manager
	bc.AssetAmount
	AccountID string `json:"account_id"`
	Address   string `json:"address"`
	VM        int64  `json:"vm"`
}

func (a *withdrawAction) Build(ctx context.Context, b *txbuilder.TemplateBuilder) error {
	var missing []string
	if a.AccountID == "" {
		missing = append(missing, "account_id")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return txbuilder.MissingFieldsError(missing...)
	}

	acct, err := a.accounts.FindByID(a.AccountID)
	if err != nil {
		return errors.Wrap(err, "get account info")
	}

	sender, err := getSender(a.accounts, a.AccountID, a.Address)
	if err != nil {
		return err
	}
	address, _ := hex.DecodeString(sender.Address)

	can, err := b.Chain().CanTransfer(address, new(big.Int).SetUint64(a.Amount))
	if err != nil {
		return err
	}
	if !can {
		return ErrInsufficientBalance
	}

	withdrawProgram, err := vmutil.WithdrawProgram(a.VM, address)
	if err != nil {
		return err
	}
	txInput := types.NewWithdrawalInput(sender.ControlProgram, a.AssetId, a.Amount, withdrawProgram, nil)
	sigInst, err := SigningInstruction(acct.Signer, sender.KeyIndex, sender.Address)
	if err != nil {
		return err
	}

	if err = b.AddInput(txInput, sigInst); err != nil {
		return errors.Wrap(err, "adding inputs")
	}

	return nil
}
