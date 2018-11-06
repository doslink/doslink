package types

// CallInput satisfies the TypedInput interface and represents a call.
type CallInput struct {
	Nonce          uint64 // sender's nonce
	To             []byte
	Data           []byte
	VMVersion      uint64
	ControlProgram []byte   // sender
	Arguments      [][]byte // Witness
}

// NewCallInput create a new CallInput struct.
func NewCallInput(controlProgram []byte, nonce uint64, to []byte, data []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &CallInput{
			VMVersion:      1,
			ControlProgram: controlProgram,
			Nonce:          nonce,
			To:             to,
			Data:           data,
			Arguments:      arguments,
		},
	}
}

// InputType is the interface function for return the input type.
func (ci *CallInput) InputType() uint8 { return CallInputType }
