package bc

import "io"

func (Contract) typ() string { return "contract1" }
func (c *Contract) writeForHash(w io.Writer) {
	mustWriteForHash(w, c.Nonce)
	mustWriteForHash(w, c.From)
	mustWriteForHash(w, c.To)
	mustWriteForHash(w, c.Input)
}

// SetDestination will link the contract to the output
func (c *Contract) SetDestination(id *Hash, val *AssetAmount, pos uint64) {
	c.WitnessDestination = &ValueDestination{
		Ref:      id,
		Value:    val,
		Position: pos,
	}
}

// NewContract creates a new Contract.
func NewContract(nonce uint64, from *Program, to []byte, data *Program, arguments [][]byte, ordinal uint64) *Contract {
	return &Contract{
		Nonce:            nonce,
		From:             from,
		To:               to,
		Input:            data,
		WitnessArguments: arguments,
		Ordinal:          ordinal,
	}
}
