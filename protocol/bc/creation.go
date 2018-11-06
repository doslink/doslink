package bc

import "io"

func (Creation) typ() string { return "creation1" }
func (c *Creation) writeForHash(w io.Writer) {
	mustWriteForHash(w, c.Nonce)
	mustWriteForHash(w, c.From)
	mustWriteForHash(w, c.Input)
}

// SetDestination will link the creation to the output
func (c *Creation) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	c.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewCreation creates a new Creation.
func NewCreation(nonce uint64, from, data *Program, arguments [][]byte, ordinal uint64) *Creation {
	return &Creation{
		Nonce:            nonce,
		From:             from,
		Input:            data,
		WitnessArguments: arguments,
		Ordinal:          ordinal,
	}
}
