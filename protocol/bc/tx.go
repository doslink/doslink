package bc

import (
	"github.com/doslink/doslink/basis/crypto/sha3pool"
	"github.com/doslink/doslink/basis/errors"
)

// Tx is a wrapper for the entries-based representation of a transaction.
type Tx struct {
	*TxHeader
	ID       Hash
	Entries  map[Hash]Entry
	InputIDs []Hash // 1:1 correspondence with TxData.Inputs

	SpentOutputIDs []Hash
	GasInputIDs    []Hash
}

// SigHash ...
func (tx *Tx) SigHash(n uint32) (hash Hash) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	tx.InputIDs[n].WriteTo(hasher)
	tx.ID.WriteTo(hasher)
	hash.ReadFrom(hasher)
	return hash
}

// Convenience routines for accessing entries of specific types by ID.
var (
	ErrEntryType    = errors.New("invalid entry type")
	ErrMissingEntry = errors.New("missing entry")
)

// Output try to get the output entry by given hash
func (tx *Tx) Output(id Hash) (*Output, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*Output)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}

// Spend try to get the spend entry by given hash
func (tx *Tx) Spend(id Hash) (*Spend, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	sp, ok := e.(*Spend)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return sp, nil
}

// Issuance try to get the issuance entry by given hash
func (tx *Tx) Issuance(id Hash) (*Issuance, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	iss, ok := e.(*Issuance)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return iss, nil
}

// Creation try to get the creation entry by given hash
func (tx *Tx) Creation(id Hash) (*Creation, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	ct, ok := e.(*Creation)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return ct, nil
}

// Call try to get the creation entry by given hash
func (tx *Tx) Call(id Hash) (*Call, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	ct, ok := e.(*Call)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return ct, nil
}

// Contract try to get the contract entry by given hash
func (tx *Tx) Contract(id Hash) (*Contract, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	ct, ok := e.(*Contract)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return ct, nil
}

// Deposit try to get the deposit entry by given hash
func (tx *Tx) Deposit(id Hash) (*Deposit, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*Deposit)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}

// Withdrawal try to get the withdrawal entry by given hash
func (tx *Tx) Withdrawal(id Hash) (*Withdrawal, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	ct, ok := e.(*Withdrawal)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return ct, nil
}
