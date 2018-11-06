package account

import (
	"encoding/json"
	"github.com/doslink/doslink/protocol/vmutil"
	log "github.com/sirupsen/logrus"
)

func (m *Manager) GetCoinbaseArbitrary() []byte {
	if arbitrary := m.db.Get(CoinbaseAbKey); arbitrary != nil {
		return arbitrary
	}
	return []byte{}
}

func (m *Manager) SetCoinbaseArbitrary(arbitrary []byte) {
	m.db.Set(CoinbaseAbKey, arbitrary)
}

// GetCoinbaseControlProgram will return a coinbase script
func (m *Manager) GetCoinbaseControlProgram() ([]byte, error) {
	cp, err := m.GetCoinbaseCtrlProgram()
	if err == ErrFindAccount {
		log.Warningf("GetCoinbaseControlProgram: can't find any account in db")
		return vmutil.DefaultCoinbaseProgram()
	}
	if err != nil {
		return nil, err
	}
	return cp.ControlProgram, nil
}

// GetCoinbaseCtrlProgram will return the coinbase CtrlProgram
func (m *Manager) GetCoinbaseCtrlProgram() (*CtrlProgram, error) {
	if data := m.db.Get(miningAddressKey); data != nil {
		cp := &CtrlProgram{}
		return cp, json.Unmarshal(data, cp)
	}

	accountIter := m.db.IteratorPrefix([]byte(accountPrefix))
	defer accountIter.Release()
	if !accountIter.Next() {
		return nil, ErrFindAccount
	}

	account := &Account{}
	if err := json.Unmarshal(accountIter.Value(), account); err != nil {
		return nil, err
	}

	program, err := m.createAddress(account, false)
	if err != nil {
		return nil, err
	}

	rawCP, err := json.Marshal(program)
	if err != nil {
		return nil, err
	}

	m.db.Set(miningAddressKey, rawCP)
	return program, nil
}

// GetMiningAddress will return the mining address
func (m *Manager) GetMiningAddress() (string, error) {
	cp, err := m.GetCoinbaseCtrlProgram()
	if err != nil {
		return "", err
	}
	return cp.Address, nil
}

// SetMiningAddress will set the mining address
func (m *Manager) SetMiningAddress(miningAddress string) (string, error) {
	program, err := m.GetProgramByAddress(miningAddress)
	if err != nil {
		return "", err
	}

	cp := &CtrlProgram{
		Address:        miningAddress,
		ControlProgram: program,
	}
	rawCP, err := json.Marshal(cp)
	if err != nil {
		return "", err
	}

	m.db.Set(miningAddressKey, rawCP)
	return m.GetMiningAddress()
}
