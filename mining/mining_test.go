package mining

import (
	"testing"
	"fmt"
	"encoding/json"
	"github.com/doslink/doslink/protocol/validation"
	"github.com/doslink/doslink/protocol/bc"
)

func TestCreateCoinbaseTx(t *testing.T) {
	reductionInterval := uint64(840000)
	baseSubsidy := uint64(41250000000)
	cases := []struct {
		height  uint64
		txFee   uint64
		subsidy uint64
	}{
		{
			height:  reductionInterval - 1,
			txFee:   100000000,
			subsidy: baseSubsidy + 100000000,
		},
		{
			height:  reductionInterval,
			txFee:   2000000000,
			subsidy: baseSubsidy/2 + 2000000000,
		},
		{
			height:  reductionInterval + 1,
			txFee:   0,
			subsidy: baseSubsidy / 2,
		},
		{
			height:  reductionInterval * 2,
			txFee:   100000000,
			subsidy: baseSubsidy/4 + 100000000,
		},
	}

	for _, c := range cases {
		coinbaseTx, err := createCoinbaseTx(nil, c.txFee, c.height)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(coinbaseTx.String())
		txb, err := json.Marshal(coinbaseTx.Tx)
		fmt.Println(string(txb))
		block := new(bc.Block)
		block.BlockHeader = new(bc.BlockHeader)
		block.Version = 1
		block.Transactions = []*bc.Tx{coinbaseTx.Tx}
		vs, err := validation.ValidateTx(coinbaseTx.Tx, block, nil, nil)
		fmt.Println(vs)
		fmt.Println(vs.GasState())
		fmt.Println(vs.GasState().GasValid)
		fmt.Println(vs.GasState().AssetValue)

		outputAmount := coinbaseTx.Outputs[0].OutputCommitment.Amount
		if outputAmount != c.subsidy {
			t.Fatalf("coinbase tx reward dismatch, expected: %d, have: %d", c.subsidy, outputAmount)
		}
	}
}
