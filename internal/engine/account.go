package engine

import (
	"fmt"

	"github.com/termite-mail/termite/internal/config"
	"github.com/termite-mail/termite/internal/providers"
)

// Account binds an account configuration to its resolved provider.
type Account struct {
	Config   config.AccountConfig
	Provider providers.Provider
}

// NewAccount creates an Account from the given config, resolving the
// appropriate provider by name.
func NewAccount(cfg config.AccountConfig) (Account, error) {
	p, err := providers.NewProvider(cfg.Provider)
	if err != nil {
		return Account{}, fmt.Errorf("failed to create provider for account %q: %w", cfg.ID, err)
	}
	return Account{
		Config:   cfg,
		Provider: p,
	}, nil
}

// AccountManager manages a set of email accounts for the engine.
type AccountManager struct {
	accounts map[string]Account
}

// NewAccountManager creates a new AccountManager from the given account configs.
func NewAccountManager(configs []config.AccountConfig) (*AccountManager, error) {
	mgr := &AccountManager{
		accounts: make(map[string]Account, len(configs)),
	}
	for _, cfg := range configs {
		acct, err := NewAccount(cfg)
		if err != nil {
			return nil, err
		}
		mgr.accounts[cfg.ID] = acct
	}
	return mgr, nil
}

// Get returns the account with the given ID, or an error if not found.
func (m *AccountManager) Get(id string) (Account, error) {
	acct, ok := m.accounts[id]
	if !ok {
		return Account{}, fmt.Errorf("account not found: %s", id)
	}
	return acct, nil
}

// All returns all registered accounts.
func (m *AccountManager) All() []Account {
	accts := make([]Account, 0, len(m.accounts))
	for _, a := range m.accounts {
		accts = append(accts, a)
	}
	return accts
}

// IDs returns all registered account IDs.
func (m *AccountManager) IDs() []string {
	ids := make([]string, 0, len(m.accounts))
	for id := range m.accounts {
		ids = append(ids, id)
	}
	return ids
}
