package network

import "sync"

type inMemoryPayeeDatabase struct {
	payeeElectronicAccounts map[string]PayeeElectronicAccount
	mu                      sync.RWMutex
}

// GetPayeeElectronicAccount implements PayeeDatabase.
func (d *inMemoryPayeeDatabase) GetPayeeElectronicAccount(payeeID string) (PayeeElectronicAccount, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	account, ok := d.payeeElectronicAccounts[payeeID]
	if !ok {
		return PayeeElectronicAccount{}, ErrPayeeNotFound{}
	}
	return account, nil
}

// CreatePayeeElectronicAccount implements PayeeDatabase.
func (d *inMemoryPayeeDatabase) CreatePayeeElectronicAccount(payeeID string, account PayeeElectronicAccount) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.payeeElectronicAccounts[payeeID] = account
	return nil
}

func newInMemoryPayeeDatabase() PayeeDatabase {
	return &inMemoryPayeeDatabase{
		payeeElectronicAccounts: map[string]PayeeElectronicAccount{},
		mu:                      sync.RWMutex{},
	}
}
