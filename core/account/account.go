package account

import (
	"encoding/json"
	"github.com/doslink/doslink/core/signers"
	"github.com/doslink/doslink/common"
	"github.com/doslink/doslink/basis/crypto/ed25519/chainkd"
	"github.com/doslink/doslink/basis/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

// Key account store prefix
func Key(name string) []byte {
	return append(accountPrefix, []byte(name)...)
}

func aliasKey(name string) []byte {
	return append(aliasPrefix, []byte(name)...)
}

// Account is structure of Chain account
type Account struct {
	*signers.Signer
	ID    string `json:"id"`
	Alias string `json:"alias"`
}

// Create creates a new Account.
func (m *Manager) Create(xpubs []chainkd.XPub, quorum int, alias string) (*Account, error) {
	m.accountMu.Lock()
	defer m.accountMu.Unlock()

	normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
	if existed := m.db.Get(aliasKey(normalizedAlias)); existed != nil {
		return nil, ErrDuplicateAlias
	}

	signer, err := signers.Create("account", xpubs, quorum, m.getNextAccountIndex())
	id := signers.IDGenerate()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	account := &Account{Signer: signer, ID: id, Alias: normalizedAlias}
	rawAccount, err := json.Marshal(account)
	if err != nil {
		return nil, ErrMarshalAccount
	}

	accountID := Key(id)
	storeBatch := m.db.NewBatch()
	storeBatch.Set(accountID, rawAccount)
	storeBatch.Set(aliasKey(normalizedAlias), []byte(id))
	storeBatch.Write()
	return account, nil
}

// DeleteAccount deletes the account's ID or alias matching accountInfo.
func (m *Manager) DeleteAccount(aliasOrID string) (err error) {
	account := &Account{}
	if account, err = m.FindByAlias(aliasOrID); err != nil {
		if account, err = m.FindByID(aliasOrID); err != nil {
			return err
		}
	}

	m.cacheMu.Lock()
	m.aliasCache.Remove(account.Alias)
	m.cacheMu.Unlock()

	storeBatch := m.db.NewBatch()
	storeBatch.Delete(aliasKey(account.Alias))
	storeBatch.Delete(Key(account.ID))
	storeBatch.Write()
	return nil
}

// FindByAlias retrieves an account's Signer record by its alias
func (m *Manager) FindByAlias(alias string) (*Account, error) {
	m.cacheMu.Lock()
	cachedID, ok := m.aliasCache.Get(alias)
	m.cacheMu.Unlock()
	if ok {
		return m.FindByID(cachedID.(string))
	}

	rawID := m.db.Get(aliasKey(alias))
	if rawID == nil {
		return nil, ErrFindAccount
	}

	accountID := string(rawID)
	m.cacheMu.Lock()
	m.aliasCache.Add(alias, accountID)
	m.cacheMu.Unlock()
	return m.FindByID(accountID)
}

// FindByID returns an account's Signer record by its ID.
func (m *Manager) FindByID(id string) (*Account, error) {
	m.cacheMu.Lock()
	cachedAccount, ok := m.cache.Get(id)
	m.cacheMu.Unlock()
	if ok {
		return cachedAccount.(*Account), nil
	}

	rawAccount := m.db.Get(Key(id))
	if rawAccount == nil {
		return nil, ErrFindAccount
	}

	account := &Account{}
	if err := json.Unmarshal(rawAccount, account); err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	m.cache.Add(id, account)
	m.cacheMu.Unlock()
	return account, nil
}

// GetAccountByProgram return Account by given CtrlProgram
func (m *Manager) GetAccountByProgram(program *CtrlProgram) (*Account, error) {
	rawAccount := m.db.Get(Key(program.AccountID))
	if rawAccount == nil {
		return nil, ErrFindAccount
	}

	account := &Account{}
	return account, json.Unmarshal(rawAccount, account)
}

// GetAliasByID return the account alias by given ID
func (m *Manager) GetAliasByID(id string) string {
	rawAccount := m.db.Get(Key(id))
	if rawAccount == nil {
		log.Warn("GetAliasByID fail to find account")
		return ""
	}

	account := &Account{}
	if err := json.Unmarshal(rawAccount, account); err != nil {
		log.Warn(err)
	}
	return account.Alias
}

// ListAccounts will return the accounts in the db
func (m *Manager) ListAccounts(id string) ([]*Account, error) {
	accounts := []*Account{}
	accountIter := m.db.IteratorPrefix(Key(strings.TrimSpace(id)))
	defer accountIter.Release()

	for accountIter.Next() {
		account := &Account{}
		if err := json.Unmarshal(accountIter.Value(), &account); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (m *Manager) getNextAccountIndex() uint64 {
	m.accIndexMu.Lock()
	defer m.accIndexMu.Unlock()

	var nextIndex uint64 = 1
	if rawIndexBytes := m.db.Get(accountIndexKey); rawIndexBytes != nil {
		nextIndex = common.BytesToUnit64(rawIndexBytes) + 1
	}
	m.db.Set(accountIndexKey, common.Unit64ToBytes(nextIndex))
	return nextIndex
}
