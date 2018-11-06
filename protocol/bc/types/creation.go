package types

// CreationInput satisfies the TypedInput interface and represents a creation.
type CreationInput struct {
	Nonce          uint64 // creator's nonce
	Data           []byte
	VMVersion      uint64
	ControlProgram []byte   // creator
	Arguments      [][]byte // Witness
}

// NewCreationInput create a new CreationInput struct.
func NewCreationInput(controlProgram []byte, nonce uint64, data []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &CreationInput{
			VMVersion:      1,
			ControlProgram: controlProgram,
			Nonce:          nonce,
			Data:           data,
			Arguments:      arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (ci *CreationInput) InputType() uint8 { return CreationInputType }
