package api

import (
	"context"

	"github.com/doslink/doslink/core/account"
	"github.com/doslink/doslink/core/asset"
	"github.com/doslink/doslink/core/pseudohsm"
	"github.com/doslink/doslink/core/rpc"
	"github.com/doslink/doslink/core/signers"
	"github.com/doslink/doslink/core/txbuilder"
	"github.com/doslink/doslink/basis/errors"
	"github.com/doslink/doslink/net/http/httperror"
	"github.com/doslink/doslink/net/http/httpjson"
	"github.com/doslink/doslink/protocol/validation"
	"github.com/doslink/doslink/protocol/vm"
)

var (
	// ErrDefault is default API Error
	ErrDefault = errors.New("API Error")
)

func isTemporary(info httperror.Info, err error) bool {
	switch info.ChainCode {
	case "000": // internal server error
		return true
	case "001": // request timed out
		return true
	case "761": // outputs currently reserved
		return true
	case "706": // 1 or more action errors
		errs := errors.Data(err)["actions"].([]httperror.Response)
		temp := true
		for _, actionErr := range errs {
			temp = temp && isTemporary(actionErr.Info, nil)
		}
		return temp
	default:
		return false
	}
}

var respErrFormatter = map[error]httperror.Info{
	ErrDefault: {500, "000", "API Error"},

	// Signers error namespace (2xx)
	signers.ErrBadQuorum: {400, "200", "Quorum must be greater than 1 and less than or equal to the length of xpubs"},
	signers.ErrBadXPub:   {400, "201", "Invalid xpub format"},
	signers.ErrNoXPubs:   {400, "202", "At least one xpub is required"},
	signers.ErrBadType:   {400, "203", "Retrieved type does not match expected type"},
	signers.ErrDupeXPub:  {400, "204", "Root XPubs cannot contain the same key more than once"},

	// Transaction error namespace (7xx)
	// Build transaction error namespace (70x ~ 72x)
	account.ErrInsufficient:         {400, "700", "Funds of account are insufficient"},
	account.ErrImmature:             {400, "701", "Available funds of account are immature"},
	account.ErrReserved:             {400, "702", "Available UTXOs of account have been reserved"},
	account.ErrMatchUTXO:            {400, "703", "Not found UTXO with given hash"},
	ErrBadActionType:                {400, "704", "Invalid action type"},
	ErrBadAction:                    {400, "705", "Invalid action object"},
	ErrBadActionConstruction:        {400, "706", "Invalid action construction"},
	txbuilder.ErrMissingFields:      {400, "707", "One or more fields are missing"},
	txbuilder.ErrBadAmount:          {400, "708", "Invalid asset amount"},
	account.ErrFindAccount:          {400, "709", "Not found account"},
	asset.ErrFindAsset:              {400, "710", "Not found asset"},
	txbuilder.ErrBadContractArgType: {400, "711", "Invalid contract argument type"},
	txbuilder.ErrOrphanTx:           {400, "712", "Not found transaction input utxo"},
	txbuilder.ErrExtTxFee:           {400, "713", "Transaction fee exceed max limit"},

	// Submit transaction error namespace (73x ~ 79x)
	// Validation error (73x ~ 75x)
	validation.ErrTxVersion:                 {400, "730", "Invalid transaction version"},
	validation.ErrWrongTransactionSize:      {400, "731", "Invalid transaction size"},
	validation.ErrBadTimeRange:              {400, "732", "Invalid transaction time range"},
	validation.ErrNotStandardTx:             {400, "733", "Not standard transaction"},
	validation.ErrWrongCoinbaseTransaction:  {400, "734", "Invalid coinbase transaction"},
	validation.ErrWrongCoinbaseAsset:        {400, "735", "Invalid coinbase assetID"},
	validation.ErrCoinbaseArbitraryOversize: {400, "736", "Invalid coinbase arbitrary size"},
	validation.ErrEmptyResults:              {400, "737", "No results in the transaction"},
	validation.ErrMismatchedAssetID:         {400, "738", "Mismatched assetID"},
	validation.ErrMismatchedPosition:        {400, "739", "Mismatched value source/dest position"},
	validation.ErrMismatchedReference:       {400, "740", "Mismatched reference"},
	validation.ErrMismatchedValue:           {400, "741", "Mismatched value"},
	validation.ErrMissingField:              {400, "742", "Missing required field"},
	validation.ErrNoSource:                  {400, "743", "No source for value"},
	validation.ErrOverflow:                  {400, "744", "Arithmetic overflow/underflow"},
	validation.ErrPosition:                  {400, "745", "Invalid source or destination position"},
	validation.ErrUnbalanced:                {400, "746", "Unbalanced asset amount between input and output"},
	validation.ErrOverGasCredit:             {400, "747", "Gas credit has been spent"},
	validation.ErrGasCalculate:              {400, "748", "Gas usage calculate got a math error"},

	// VM error (76x ~ 78x)
	vm.ErrAltStackUnderflow:  {400, "760", "Alt stack underflow"},
	vm.ErrBadValue:           {400, "761", "Bad value"},
	vm.ErrContext:            {400, "762", "Wrong context"},
	vm.ErrDataStackUnderflow: {400, "763", "Data stack underflow"},
	vm.ErrDisallowedOpcode:   {400, "764", "Disallowed opcode"},
	vm.ErrDivZero:            {400, "765", "Division by zero"},
	vm.ErrFalseVMResult:      {400, "766", "False result for executing VM"},
	vm.ErrLongProgram:        {400, "767", "Program size exceeds max int32"},
	vm.ErrRange:              {400, "768", "Arithmetic range error"},
	vm.ErrReturn:             {400, "769", "RETURN executed"},
	vm.ErrRunLimitExceeded:   {400, "770", "Run limit exceeded because the Fee is insufficient"},
	vm.ErrShortProgram:       {400, "771", "Unexpected end of program"},
	vm.ErrToken:              {400, "772", "Unrecognized token"},
	vm.ErrUnexpected:         {400, "773", "Unexpected error"},
	vm.ErrUnsupportedVM:      {400, "774", "Unsupported VM because the version of VM is mismatched"},
	vm.ErrVerifyFailed:       {400, "775", "VERIFY failed"},

	// Mock HSM error namespace (8xx)
	pseudohsm.ErrDuplicateKeyAlias:    {400, "800", "Key Alias already exists"},
	pseudohsm.ErrInvalidAfter:         {400, "801", "Invalid `after` in query"},
	pseudohsm.ErrLoadKey:              {400, "802", "Key not found or wrong password"},
	pseudohsm.ErrTooManyAliasesToList: {400, "803", "Requested key aliases exceeds limit"},
	pseudohsm.ErrDecrypt:              {400, "804", "Could not decrypt key with given passphrase"},
}

// Map error values to standard error codes. Missing entries
// will map to internalErrInfo.
//
// TODO(jackson): Share one error table across Chain
// products/services so that errors are consistent.
var errorFormatter = httperror.Formatter{
	Default:     httperror.Info{500, "000", "Chain API Error"},
	IsTemporary: isTemporary,
	Errors: map[error]httperror.Info{
		// General error namespace (0xx)
		context.DeadlineExceeded:     {408, "001", "Request timed out"},
		httpjson.ErrBadRequest:       {400, "002", "Invalid request body"},
		rpc.ErrWrongNetwork:          {502, "103", "A peer core is operating on a different blockchain network"},

		//accesstoken authz err namespace (86x)
		errNotAuthenticated: {401, "860", "Request could not be authenticated"},
	},
}
