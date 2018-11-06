package types

import "github.com/doslink/doslink/protocol/bc"

// WithdrawalInput satisfies the TypedInput interface and represents a withdrawal.
type WithdrawalInput struct {
	bc.AssetAmount
	WithdrawProgram []byte
	VMVersion       uint64
	ControlProgram  []byte
	Arguments       [][]byte // Witness
}

// NewWithdrawalInput create a new WithdrawInput struct.
func NewWithdrawalInput(controlProgram []byte, assetID *bc.AssetID, amount uint64, withdrawProgram []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &WithdrawalInput{
			AssetAmount: bc.AssetAmount{
				AssetId: assetID,
				Amount:  amount,
			},
			WithdrawProgram: withdrawProgram,
			VMVersion:       1,
			ControlProgram:  controlProgram,
			Arguments:       arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (ci *WithdrawalInput) InputType() uint8 { return WithdrawalInputType }
