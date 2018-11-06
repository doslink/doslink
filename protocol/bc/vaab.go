package bc

import "io"

func (Deposit) typ() string { return "deposit1" }
func (d *Deposit) writeForHash(w io.Writer) {
	mustWriteForHash(w, d.Source)
	mustWriteForHash(w, d.ControlProgram)
}

// NewDeposit creates a new Deposit.
func NewDeposit(source *ValueSource, controlProgram *Program, ordinal uint64) *Deposit {
	return &Deposit{
		Source:         source,
		ControlProgram: controlProgram,
		Ordinal:        ordinal,
	}
}

func (Withdrawal) typ() string { return "withdrawal1" }
func (m *Withdrawal) writeForHash(w io.Writer) {
	mustWriteForHash(w, m.ControlProgram)
	mustWriteForHash(w, m.Value)
	mustWriteForHash(w, m.WithdrawProgram)
}

// SetDestination will link the withdrawal to the output
func (m *Withdrawal) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	m.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewWithdrawal creates a new Withdrawal.
func NewWithdrawal(controlProgram *Program, value *AssetAmount, withdrawProgram *Program, arguments [][]byte, ordinal uint64) *Withdrawal {
	return &Withdrawal{
		ControlProgram:   controlProgram,
		Value:            value,
		WithdrawProgram:  withdrawProgram,
		WitnessArguments: arguments,
		Ordinal:          ordinal,
	}
}
