package main

import (
	"context"
	"encoding/json"
	"fmt"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"mux/internal/accounts"
	"mux/internal/codexoauth"
	"mux/internal/securestore"
)

type App struct {
	ctx     context.Context
	store   *accounts.Store
	initErr error
}

type AccountView struct {
	accounts.Account
	Active bool `json:"active"`
}

func NewApp() *App {
	store, err := accounts.NewStore()
	return &App{store: store, initErr: err}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) ListAccounts() ([]AccountView, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	activeID := a.store.Active()
	accountsList := a.store.List()
	result := make([]AccountView, 0, len(accountsList))
	for _, account := range accountsList {
		result = append(result, AccountView{Account: account, Active: account.ID == activeID})
	}
	return result, nil
}

func (a *App) AddAccount(name string) (AccountView, error) {
	if a.initErr != nil {
		return AccountView{}, a.initErr
	}
	account, err := a.store.Create(name)
	if err != nil {
		return AccountView{}, err
	}
	return AccountView{Account: account, Active: account.ID == a.store.Active()}, nil
}

func (a *App) SetActiveAccount(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	return a.store.SetActive(id)
}

func (a *App) LoginAccount(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	for _, account := range a.store.List() {
		if account.ID != id {
			continue
		}
		if err := a.store.SetStatus(id, accounts.StatusLoggingIn, ""); err != nil {
			return err
		}
		go a.runLogin(account)
		return nil
	}
	return fmt.Errorf("account not found")
}

func (a *App) runLogin(account accounts.Account) {
	credentials, err := codexoauth.Login(a.ctx, func() {
		wailsruntime.WindowShow(a.ctx)
		wailsruntime.WindowUnminimise(a.ctx)
	})
	if err != nil {
		_ = a.store.SetStatus(account.ID, accounts.StatusError, err.Error())
		return
	}
	//nolint:gosec // Credentials are immediately stored in the macOS Keychain and never written to disk.
	secret, err := json.Marshal(credentials)
	if err != nil {
		_ = a.store.SetStatus(account.ID, accounts.StatusError, "encode credentials failed")
		return
	}
	if err := securestore.Set(account.ID, string(secret)); err != nil {
		_ = a.store.SetStatus(account.ID, accounts.StatusError, "save credentials to Keychain failed")
		return
	}
	if err := a.store.SetStatus(account.ID, accounts.StatusReady, ""); err != nil {
		return
	}
}

func (a *App) RemoveAccount(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	if err := securestore.Delete(id); err != nil {
		return fmt.Errorf("delete account credentials from Keychain: %w", err)
	}
	if err := a.store.Remove(id); err != nil {
		return fmt.Errorf("remove account data: %w", err)
	}
	return nil
}
