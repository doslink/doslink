package types

import (
	"fmt"
	"io"

	"github.com/doslink/doslink/basis/encoding/blockchain"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/protocol/bc"
)

// serflag variables for input types.
const (
	IssuanceInputType uint8 = iota
	SpendInputType
	CoinbaseInputType
	CreationInputType
	CallInputType
	ContractInputType
	WithdrawalInputType
)

type (
	// TxInput is the top level struct of tx input.
	TxInput struct {
		AssetVersion uint64
		//ReferenceData    []byte
		TypedInput
		CommitmentSuffix []byte
		WitnessSuffix    []byte
	}

	// TypedInput return the txinput type.
	TypedInput interface {
		InputType() uint8
	}
)

var errBadAssetID = errors.New("asset ID does not match other issuance parameters")

// AssetAmount return the asset id and amount of the txinput.
func (t *TxInput) AssetAmount() bc.AssetAmount {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		assetID := inp.AssetID()
		return bc.AssetAmount{
			AssetId: &assetID,
			Amount:  inp.Amount,
		}
	case *SpendInput:
		return inp.AssetAmount
	case *WithdrawalInput:
		return inp.AssetAmount
	}
	return bc.AssetAmount{}
}

// AssetID return the assetID of the txinput
func (t *TxInput) AssetID() bc.AssetID {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.AssetID()
	case *SpendInput:
		return *inp.AssetId
	case *WithdrawalInput:
		return *inp.AssetId

	}
	return bc.AssetID{}
}

// Amount return the asset amount of the txinput
func (t *TxInput) Amount() uint64 {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Amount
	case *SpendInput:
		return inp.Amount
	case *WithdrawalInput:
		return inp.Amount
	}
	return 0
}

// ControlProgram return the control program of the spend input
func (t *TxInput) ControlProgram() []byte {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.IssuanceProgram
	case *SpendInput:
		return inp.ControlProgram
	case *ContractInput:
		return inp.ControlProgram
	case *WithdrawalInput:
		return inp.ControlProgram
	}
	return nil
}

// Arguments get the args for the input
func (t *TxInput) Arguments() [][]byte {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		return inp.Arguments
	case *SpendInput:
		return inp.Arguments
	case *CreationInput:
		return inp.Arguments
	case *CallInput:
		return inp.Arguments
	case *ContractInput:
		return inp.Arguments
	case *WithdrawalInput:
		return inp.Arguments
	}
	return nil
}

// SetArguments set the args for the input
func (t *TxInput) SetArguments(args [][]byte) {
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		inp.Arguments = args
	case *SpendInput:
		inp.Arguments = args
	case *CreationInput:
		inp.Arguments = args
	case *CallInput:
		inp.Arguments = args
	case *ContractInput:
		inp.Arguments = args
	case *WithdrawalInput:
		inp.Arguments = args
	}
}

// SpentOutputID calculate the hash of spended output
func (t *TxInput) SpentOutputID() (o bc.Hash, err error) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		o, err = ComputeOutputID(&si.SpendCommitment)
	}
	return o, err
}

func (t *TxInput) readFrom(r *blockchain.Reader) (err error) {
	if t.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}

	var assetID bc.AssetID
	t.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if t.AssetVersion != 1 {
			return nil
		}
		var icType [1]byte
		if _, err = io.ReadFull(r, icType[:]); err != nil {
			return errors.Wrap(err, "reading input commitment type")
		}
		switch icType[0] {
		case IssuanceInputType:
			ii := new(IssuanceInput)
			t.TypedInput = ii

			if ii.Nonce, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if _, err = assetID.ReadFrom(r); err != nil {
				return err
			}
			if ii.Amount, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}

		case SpendInputType:
			si := new(SpendInput)
			t.TypedInput = si
			if si.SpendCommitmentSuffix, err = si.SpendCommitment.readFrom(r, 1); err != nil {
				return err
			}

		case CoinbaseInputType:
			ci := new(CoinbaseInput)
			t.TypedInput = ci
			if ci.Arbitrary, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}

		case CreationInputType:
			ci := new(CreationInput)
			t.TypedInput = ci

			if ci.Nonce, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if ci.Data, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if ci.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if ci.ControlProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}

		case CallInputType:
			ci := new(CallInput)
			t.TypedInput = ci

			if ci.Nonce, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if ci.To, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if ci.Data, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if ci.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if ci.ControlProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}

		case ContractInputType:
			ci := new(ContractInput)
			t.TypedInput = ci

			if ci.Nonce, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if ci.To, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if ci.Data, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if ci.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if ci.ControlProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}

		case WithdrawalInputType:
			wi := new(WithdrawalInput)
			t.TypedInput = wi

			if err = wi.AssetAmount.ReadFrom(r); err != nil {
				return errors.Wrap(err, "reading asset+amount")
			}
			if wi.WithdrawProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if wi.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if wi.ControlProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unsupported input type %d", icType[0])
		}
		return nil
	})
	if err != nil {
		return err
	}

	t.WitnessSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if t.AssetVersion != 1 {
			return nil
		}

		switch inp := t.TypedInput.(type) {
		case *IssuanceInput:
			if inp.AssetDefinition, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if inp.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
				return err
			}
			if inp.IssuanceProgram, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}
			if inp.AssetID() != assetID {
				return errBadAssetID
			}
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}

		case *SpendInput:
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}

		case *CreationInput:
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}

		case *CallInput:
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}

		case *ContractInput:
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}

		case *WithdrawalInput:
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}

		}
		return nil
	})

	return err
}

func (t *TxInput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, t.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	if _, err := blockchain.WriteExtensibleString(w, t.CommitmentSuffix, t.writeInputCommitment); err != nil {
		return errors.Wrap(err, "writing input commitment")
	}

	_, err := blockchain.WriteExtensibleString(w, t.WitnessSuffix, t.writeInputWitness)
	return errors.Wrap(err, "writing input witness")
}

func (t *TxInput) writeInputCommitment(w io.Writer) (err error) {
	if t.AssetVersion != 1 {
		return nil
	}

	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		if _, err = w.Write([]byte{IssuanceInputType}); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.Nonce); err != nil {
			return err
		}
		assetID := t.AssetID()
		if _, err = assetID.WriteTo(w); err != nil {
			return err
		}
		_, err = blockchain.WriteVarint63(w, inp.Amount)
		return err

	case *SpendInput:
		if _, err = w.Write([]byte{SpendInputType}); err != nil {
			return err
		}
		return inp.SpendCommitment.writeExtensibleString(w, inp.SpendCommitmentSuffix, t.AssetVersion)

	case *CoinbaseInput:
		if _, err = w.Write([]byte{CoinbaseInputType}); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.Arbitrary); err != nil {
			return errors.Wrap(err, "writing coinbase arbitrary")
		}

	case *CreationInput:
		if _, err = w.Write([]byte{CreationInputType}); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarint63(w, inp.Nonce); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.Data); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarint63(w, inp.VMVersion); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarstr31(w, inp.ControlProgram); err != nil {
			return err
		}
		return err

	case *CallInput:
		if _, err = w.Write([]byte{CallInputType}); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarint63(w, inp.Nonce); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.To); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.Data); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarint63(w, inp.VMVersion); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarstr31(w, inp.ControlProgram); err != nil {
			return err
		}
		return err

	case *ContractInput:
		if _, err = w.Write([]byte{ContractInputType}); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarint63(w, inp.Nonce); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.To); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.Data); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarint63(w, inp.VMVersion); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarstr31(w, inp.ControlProgram); err != nil {
			return err
		}
		return err

	case *WithdrawalInput:
		if _, err = w.Write([]byte{WithdrawalInputType}); err != nil {
			return err
		}
		if _, err = inp.AssetAmount.WriteTo(w); err != nil {
			return errors.Wrap(err, "writing asset amount")
		}
		if _, err = blockchain.WriteVarstr31(w, inp.WithdrawProgram); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarint63(w, inp.VMVersion); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarstr31(w, inp.ControlProgram); err != nil {
			return err
		}
		return err

	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) error {
	if t.AssetVersion != 1 {
		return nil
	}
	switch inp := t.TypedInput.(type) {
	case *IssuanceInput:
		if _, err := blockchain.WriteVarstr31(w, inp.AssetDefinition); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarint63(w, inp.VMVersion); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarstr31(w, inp.IssuanceProgram); err != nil {
			return err
		}
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	case *SpendInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	case *CreationInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	case *CallInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	case *ContractInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	case *WithdrawalInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err

	}
	return nil
}
