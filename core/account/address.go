package account

import (
	"encoding/json"
	"github.com/doslink/doslink/core/signers"
	"github.com/doslink/doslink/common"
	"github.com/doslink/doslink/basis/crypto"
	"github.com/doslink/doslink/basis/crypto/ed25519/chainkd"
	"github.com/doslink/doslink/basis/crypto/sha3pool"
	"github.com/doslink/doslink/protocol/vmutil"
	"encoding/hex"
)

func contractIndexKey(accountID string) []byte {
	return append(contractIndexPrefix, []byte(accountID)...)
}

// ContractKey account control promgram store prefix
func ContractKey(hash common.Hash) []byte {
	return append(contractPrefix, hash[:]...)
}

//CtrlProgram is structure of account control program
type CtrlProgram struct {
	AccountID      string
	Address        string
	KeyIndex       uint64
	ControlProgram []byte
	Change         bool // Mark whether this control program is for UTXO change
}

// CreateAddress generate an address for the select account
func (m *Manager) CreateAddress(accountID string, change bool) (cp *CtrlProgram, err error) {
	account, err := m.FindByID(accountID)
	if err != nil {
		return nil, err
	}
	return m.createAddress(account, change)
}

// GetAddressIndex return the current index
func (m *Manager) GetContractIndex(accountID string) uint64 {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	index := uint64(1)
	if rawIndexBytes := m.db.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		index = common.BytesToUnit64(rawIndexBytes)
	}
	return index
}

// GetLocalCtrlProgramByAddress return CtrlProgram by given address
func (m *Manager) GetLocalCtrlProgramByAddress(address string) (*CtrlProgram, error) {
	program, err := m.GetProgramByAddress(address)
	if err != nil {
		return nil, err
	}

	var hash [32]byte
	sha3pool.Sum256(hash[:], program)
	rawProgram := m.db.Get(ContractKey(hash))
	if rawProgram == nil {
		return nil, ErrFindCtrlProgram
	}

	cp := &CtrlProgram{}
	return cp, json.Unmarshal(rawProgram, cp)
}

// IsLocalControlProgram check is the input control program belong to local
func (m *Manager) IsLocalControlProgram(prog []byte) bool {
	var hash common.Hash
	sha3pool.Sum256(hash[:], prog)
	bytes := m.db.Get(ContractKey(hash))
	return bytes != nil
}

// ListControlProgram return all the local control program
func (m *Manager) ListControlProgram() ([]*CtrlProgram, error) {
	cps := []*CtrlProgram{}
	cpIter := m.db.IteratorPrefix(contractPrefix)
	defer cpIter.Release()

	for cpIter.Next() {
		cp := &CtrlProgram{}
		if err := json.Unmarshal(cpIter.Value(), cp); err != nil {
			return nil, err
		}
		cps = append(cps, cp)
	}
	return cps, nil
}

// CreateAddress generate an address for the select account
func (m *Manager) createAddress(account *Account, change bool) (cp *CtrlProgram, err error) {
	cp, err = m.createP2SH(account, change)
	if err != nil {
		return nil, err
	}
	return cp, m.insertControlPrograms(cp)
}

func (m *Manager) createP2SH(account *Account, change bool) (*CtrlProgram, error) {
	idx := m.getNextContractIndex(account.ID)
	path := signers.Path(account.Signer, signers.AccountKeySpace, idx)
	derivedXPubs := chainkd.DeriveXPubs(account.XPubs, path)
	derivedPKs := chainkd.XPubKeys(derivedXPubs)
	signScript, err := vmutil.P2SPMultiSigProgram(derivedPKs, account.Quorum)
	if err != nil {
		return nil, err
	}
	scriptHash := crypto.Ripemd160(signScript)

	address := common.BytesToAddress(scriptHash)

	control, err := vmutil.P2WSHProgram(scriptHash)
	if err != nil {
		return nil, err
	}

	return &CtrlProgram{
		AccountID:      account.ID,
		Address:        address.Hex(),
		KeyIndex:       idx,
		ControlProgram: control,
		Change:         change,
	}, nil
}

func (m *Manager) getNextContractIndex(accountID string) uint64 {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	nextIndex := uint64(1)
	if rawIndexBytes := m.db.Get(contractIndexKey(accountID)); rawIndexBytes != nil {
		nextIndex = common.BytesToUnit64(rawIndexBytes) + 1
	}
	m.db.Set(contractIndexKey(accountID), common.Unit64ToBytes(nextIndex))
	return nextIndex
}

func (m *Manager) GetProgramByAddress(address string) ([]byte, error) {
	addr, err := hex.DecodeString(address)
	if err != nil {
		return nil, err
	}

	program := []byte{}
	program, err = vmutil.P2WSHProgram(addr)
	if err != nil {
		return nil, err
	}
	return program, nil
}

func (m *Manager) insertControlPrograms(progs ...*CtrlProgram) error {
	var hash common.Hash
	for _, prog := range progs {
		accountCP, err := json.Marshal(prog)
		if err != nil {
			return err
		}

		sha3pool.Sum256(hash[:], prog.ControlProgram)
		m.db.Set(ContractKey(hash), accountCP)
	}
	return nil
}
