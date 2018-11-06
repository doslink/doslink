package account

import (
	"github.com/doslink/doslink/consensus/segwit"
	"github.com/doslink/doslink/protocol/bc"
)

// AddUnconfirmedUtxo add utxo list to utxoKeeper
func (m *Manager) AddUnconfirmedUtxo(utxos []*UTXO) {
	m.utxoKeeper.AddUnconfirmedUtxo(utxos)
}

func (m *Manager) ListUnconfirmedUtxo(isSmartContract bool) []*UTXO {
	utxos := m.utxoKeeper.ListUnconfirmed()
	result := []*UTXO{}
	for _, utxo := range utxos {
		if segwit.IsP2WScript(utxo.ControlProgram) != isSmartContract {
			result = append(result, utxo)
		}
	}
	return result
}

// RemoveUnconfirmedUtxo remove utxos from the utxoKeeper
func (m *Manager) RemoveUnconfirmedUtxo(hashes []*bc.Hash) {
	m.utxoKeeper.RemoveUnconfirmedUtxo(hashes)
}
