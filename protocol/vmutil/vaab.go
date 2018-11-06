package vmutil

import "github.com/doslink/doslink/protocol/vm"

// DepositProgram generates the script for deposit output
func DepositProgram(vmType int64, address []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_0)
	builder.AddInt64(vmType)
	builder.AddData(address)
	builder.AddOp(vm.OP_DEPOSIT)
	return builder.Build()
}

// WithdrawProgram generates the script for withdraw output
func WithdrawProgram(vmType int64, address []byte) ([]byte, error) {
	builder := NewBuilder()
	builder.AddOp(vm.OP_0)
	builder.AddInt64(vmType)
	builder.AddData(address)
	builder.AddOp(vm.OP_WITHDRAW)
	return builder.Build()
}
