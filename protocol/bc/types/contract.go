package types

// ContractInput satisfies the TypedInput interface and represents a contract call.
type ContractInput struct {
	Nonce          uint64 // sender's nonce
	To             []byte
	Data           []byte
	VMVersion      uint64
	ControlProgram []byte   // sender
	Arguments      [][]byte // Witness
}

// NewContractInput create a new ContractInput struct.
func NewContractInput(controlProgram []byte, nonce uint64, to []byte, data []byte, arguments [][]byte) *TxInput {
	return &TxInput{
		AssetVersion: 1,
		TypedInput: &ContractInput{
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
func (ci *ContractInput) InputType() uint8 { return ContractInputType }
