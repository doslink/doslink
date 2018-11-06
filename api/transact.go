package api

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/basis/math/checked"
	"github.com/doslink/doslink/consensus"
	"github.com/doslink/doslink/consensus/segwit"
	"github.com/doslink/doslink/core/account"
	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/net/http/reqid"
	"github.com/doslink/doslink/protocol"
	"github.com/doslink/doslink/protocol/bc"
	"github.com/doslink/doslink/protocol/bc/types"
	"github.com/doslink/doslink/protocol/validation"
)

var (
	defaultTxTTL    = 5 * time.Minute
	defaultBaseRate = float64(100000)
)

func (a *API) actionDecoder(action string) (func([]byte) (txbuilder.Action, error), bool) {
	decoders := map[string]func([]byte) (txbuilder.Action, error){
		"control_address":                txbuilder.DecodeControlAddressAction,
		"control_program":                txbuilder.DecodeControlProgramAction,
		"issue":                          a.wallet.AssetReg.DecodeIssueAction,
		"retire":                         txbuilder.DecodeRetireAction,
		"spend_account":                  a.wallet.AccountMgr.DecodeSpendAction,
		"spend_account_unspent_output":   a.wallet.AccountMgr.DecodeSpendUTXOAction,
		"create_contract":                a.wallet.AccountMgr.DecodeCreateContractAction,
		"sendto_contract":                a.wallet.AccountMgr.DecodeSendToContractAction,
		"contract":                       a.wallet.AccountMgr.DecodeContractAction,
		"set_transaction_reference_data": txbuilder.DecodeSetTxRefDataAction,
		"deposit":                        a.wallet.AccountMgr.DecodeDepositAction,
		"withdraw":                       a.wallet.AccountMgr.DecodeWithdrawAction,
	}
	decoder, ok := decoders[action]
	return decoder, ok
}

func onlyHaveInputActions(req *BuildRequest) (bool, error) {
	count := 0
	for i, act := range req.Actions {
		actionType, ok := act["type"].(string)
		if !ok {
			return false, errors.WithDetailf(ErrBadActionType, "no action type provided on action %d", i)
		}

		if strings.HasPrefix(actionType, "spend") || actionType == "issue" {
			count++
		}
	}

	return count == len(req.Actions), nil
}

func (a *API) buildSingle(ctx context.Context, req *BuildRequest) (*txbuilder.Template, error) {
	if err := a.completeMissingIDs(ctx, req); err != nil {
		return nil, err
	}

	//if ok, err := onlyHaveInputActions(req); err != nil {
	//	return nil, err
	//} else if ok {
	//	return nil, errors.WithDetail(ErrBadActionConstruction, "transaction contains only input actions and no output actions")
	//}

	actions := make([]txbuilder.Action, 0, len(req.Actions))
	for i, act := range req.Actions {
		typ, ok := act["type"].(string)
		if !ok {
			return nil, errors.WithDetailf(ErrBadActionType, "no action type provided on action %d", i)
		}
		decoder, ok := a.actionDecoder(typ)
		if !ok {
			return nil, errors.WithDetailf(ErrBadActionType, "unknown action type %q on action %d", typ, i)
		}

		// Remarshal to JSON, the action may have been modified when we
		// filtered aliases.
		b, err := json.Marshal(act)
		if err != nil {
			return nil, err
		}
		action, err := decoder(b)
		if err != nil {
			return nil, errors.WithDetailf(ErrBadAction, "%s on action %d", err.Error(), i)
		}
		actions = append(actions, action)
	}
	actions = account.MergeSpendAction(actions)

	ttl := req.TTL.Duration
	if ttl == 0 {
		ttl = defaultTxTTL
	}
	maxTime := time.Now().Add(ttl)

	tpl, err := txbuilder.Build(ctx, req.Tx, actions, maxTime, req.TimeRange, a.chain)
	if errors.Root(err) == txbuilder.ErrAction {
		// append each of the inner errors contained in the data.
		var Errs string
		var rootErr error
		for i, innerErr := range errors.Data(err)["actions"].([]error) {
			if i == 0 {
				rootErr = errors.Root(innerErr)
			}
			Errs = Errs + innerErr.Error()
		}
		err = errors.WithDetail(rootErr, Errs)
	}
	if err != nil {
		return nil, err
	}

	// ensure null is never returned for signing instructions
	if tpl.SigningInstructions == nil {
		tpl.SigningInstructions = []*txbuilder.SigningInstruction{}
	}
	return tpl, nil
}

// POST /build-transaction
func (a *API) build(ctx context.Context, buildReqs *BuildRequest) Response {
	subctx := reqid.NewSubContext(ctx, reqid.New())

	tmpl, err := a.buildSingle(subctx, buildReqs)
	if err != nil {
		return NewErrorResponse(err)
	}

	return NewSuccessResponse(tmpl)
}

type submitTxResp struct {
	TxID       *bc.Hash `json:"tx_id"`
	AssetValue uint64   `json:"asset_value"`
	GasLeft    int64    `json:"gas_left"`
	GasUsed    int64    `json:"gas_used"`
	GasValid   bool     `json:"gas_valid"`
	StorageGas int64    `json:"storage_gas"`
	VMGas      int64    `json:"vm_gas"`
}

// POST /submit-transaction
func (a *API) submit(ctx context.Context, ins struct {
	Tx           types.Tx `json:"raw_transaction"`
	OnlyValidate bool     `json:"only_validate" default:"false"`
}) (resp Response) {
	if ins.OnlyValidate {
		log.WithField("tx_id", ins.Tx.ID.String()).Info("validate single tx")
	} else {
		log.WithField("tx_id", ins.Tx.ID.String()).Info("submit single tx")
	}

	gasStatus, err := txbuilder.FinalizeTx(ctx, a.chain, &ins.Tx, ins.OnlyValidate)
	if err != nil {
		resp = NewErrorResponse(err)
	} else {
		resp = NewSuccessResponse(&submitTxResp{TxID: &ins.Tx.ID})
	}

	if gasStatus != nil {
		resp.Data = &submitTxResp{
			TxID:       &ins.Tx.ID,
			AssetValue: gasStatus.AssetValue,
			GasLeft:    gasStatus.GasLeft,
			GasUsed:    gasStatus.GasUsed,
			GasValid:   gasStatus.GasValid,
			StorageGas: gasStatus.StorageGas,
			VMGas:      gasStatus.GasUsed - gasStatus.StorageGas,
		}
	}
	return resp
}

// EstimateTxGasResp estimate transaction consumed gas
type EstimateTxGasResp struct {
	TotalUny   int64 `json:"total_uny"`
	StorageUny int64 `json:"storage_uny"`
	VMUny      int64 `json:"vm_uny"`
}

// EstimateTxGas estimate consumed uny for transaction
func EstimateTxGas(template txbuilder.Template, chain *protocol.Chain) (*EstimateTxGasResp, error) {
	// base tx size and not include sign
	data, err := template.Transaction.TxData.MarshalText()
	if err != nil {
		return nil, err
	}
	baseTxSize := int64(len(data))

	// extra tx size for sign witness parts
	signSize := estimateSignSize(template.SigningInstructions)

	// total gas for tx storage
	totalTxSizeGas, ok := checked.MulInt64(baseTxSize+signSize, consensus.StorageGasRate)
	if !ok {
		return nil, errors.New("calculate txsize gas got a math error")
	}

	// consume gas for run VM
	totalP2WSHGas := int64(0)
	totalVMGas := int64(0)

	tx := template.Transaction.Tx

	bh := chain.BestBlockHeader()
	block := types.MapBlock(&types.Block{BlockHeader: *bh})

	stateDB, err := protocol.NewState(&bh.StateRoot, chain)
	if err != nil {
		err = errors.Wrap(err, "open stateDB")
		log.Error(err)
		return nil, err
	}
	stateDB.Prepare(tx.ID.Byte32(), [32]byte{}, 0)

	for pos, inpID := range template.Transaction.Tx.InputIDs {
		e := template.Transaction.Entries[inpID]
		switch e := template.Transaction.Entries[inpID].(type) {
		case *bc.Spend:
			resOut, err := template.Transaction.Output(*e.SpentOutputId)
			if err != nil {
				continue
			}

			if segwit.IsP2WSHScript(resOut.ControlProgram.Code) {
				sigInst := template.SigningInstructions[pos]
				totalP2WSHGas += estimateP2WSHGas(sigInst)
			}
			continue
		case *bc.Issuance:
			if segwit.IsP2WSHScript(e.WitnessAssetDefinition.IssuanceProgram.Code) {
				sigInst := template.SigningInstructions[pos]
				totalP2WSHGas += estimateP2WSHGas(sigInst)
			}
			continue
		case *bc.Creation:
			if segwit.IsP2WSHScript(e.From.Code) {
				sigInst := template.SigningInstructions[pos]
				totalP2WSHGas += estimateP2WSHGas(sigInst)
			}
		case *bc.Call:
			if segwit.IsP2WSHScript(e.From.Code) {
				sigInst := template.SigningInstructions[pos]
				totalP2WSHGas += estimateP2WSHGas(sigInst)
			}
		case *bc.Contract:
			if segwit.IsP2WSHScript(e.From.Code) {
				sigInst := template.SigningInstructions[pos]
				totalP2WSHGas += estimateP2WSHGas(sigInst)
			}
		case *bc.Withdrawal:
			if segwit.IsP2WSHScript(e.ControlProgram.Code) {
				sigInst := template.SigningInstructions[pos]
				totalP2WSHGas += estimateP2WSHGas(sigInst)
			}
		default:
			continue
		}

		gasStatus, err := validation.EstimateContractGas(e, template.Transaction.Tx, block, chain, stateDB)
		if err != nil {
			log.Infoln(err)
			continue
		}
		totalVMGas += gasStatus.GasUsed

	}

	for _, e := range template.Transaction.Entries {
		switch e.(type) {
		case *bc.Deposit:
		default:
			continue
		}

		gasStatus, err := validation.EstimateContractGas(e, template.Transaction.Tx, block, chain, stateDB)
		if err != nil {
			log.Infoln(err)
			continue
		}
		totalVMGas += gasStatus.GasUsed

	}

	// total estimate gas
	totalGas := totalTxSizeGas + totalP2WSHGas + totalVMGas

	// rounding totalUny with base rate 100000
	totalUny := float64(totalGas*consensus.VMGasRate) / defaultBaseRate
	roundingUny := math.Ceil(totalUny)
	estimateUny := int64(roundingUny) * int64(defaultBaseRate)

	// TODO add priority

	log.WithField("baseTxSize", baseTxSize).
		WithField("signSize", signSize).
		WithField("totalTxSizeGas", totalTxSizeGas).
		WithField("totalP2WSHGas", totalP2WSHGas).
		WithField("VMGas", totalVMGas).
		WithField("totalVMGas", totalP2WSHGas+totalVMGas).
		Println("EstimateTxGas")
	return &EstimateTxGasResp{
		TotalUny:   estimateUny,
		StorageUny: totalTxSizeGas * consensus.VMGasRate,
		VMUny:      (totalP2WSHGas + totalVMGas) * consensus.VMGasRate,
	}, nil
}

// estimate p2wsh gas.
// OP_CHECKMULTISIG consume (984 * a - 72 * b - 63) gas,
// where a represent the num of public keys, and b represent the num of quorum.
func estimateP2WSHGas(sigInst *txbuilder.SigningInstruction) int64 {
	P2WSHGas := int64(0)
	baseP2WSHGas := int64(1314)

	for _, witness := range sigInst.WitnessComponents {
		switch t := witness.(type) {
		case *txbuilder.SignatureWitness:
			P2WSHGas += baseP2WSHGas + (984*int64(len(t.Keys)) - 72*int64(t.Quorum) - 63)
		case *txbuilder.RawTxSigWitness:
			P2WSHGas += baseP2WSHGas + (984*int64(len(t.Keys)) - 72*int64(t.Quorum) - 63)
		}
	}
	return P2WSHGas
}

// estimate signature part size.
// if need multi-sign, calculate the size according to the length of keys.
func estimateSignSize(signingInstructions []*txbuilder.SigningInstruction) int64 {
	signSize := int64(0)
	baseWitnessSize := int64(206)

	for _, sigInst := range signingInstructions {
		for _, witness := range sigInst.WitnessComponents {
			switch t := witness.(type) {
			case *txbuilder.SignatureWitness:
				signSize += int64(t.Quorum) * baseWitnessSize
			case *txbuilder.RawTxSigWitness:
				signSize += int64(t.Quorum) * baseWitnessSize
			}
		}
	}
	return signSize
}

// POST /estimate-transaction-gas
func (a *API) estimateTxGas(ctx context.Context, in struct {
	TxTemplate txbuilder.Template `json:"transaction_template"`
}) Response {
	txGasResp, err := EstimateTxGas(in.TxTemplate, a.chain)
	if err != nil {
		return NewErrorResponse(err)
	}
	return NewSuccessResponse(txGasResp)
}
