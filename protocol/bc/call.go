package bc

import "io"

func (Call) typ() string { return "call1" }
func (c *Call) writeForHash(w io.Writer) {
	mustWriteForHash(w, c.Nonce)
	mustWriteForHash(w, c.From)
	mustWriteForHash(w, c.To)
	mustWriteForHash(w, c.Input)
}

// SetDestination will link the call to the output
func (c *Call) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	c.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewCall creates a new Call.
func NewCall(nonce uint64, from, to, data *Program, arguments [][]byte, ordinal uint64) *Call {
	return &Call{
		Nonce:            nonce,
		From:             from,
		To:               to,
		Input:            data,
		WitnessArguments: arguments,
		Ordinal:          ordinal,
	}
}
