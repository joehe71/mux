package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"mux/internal/accounts"
	"mux/internal/codexoauth"
	"mux/internal/securestore"
)

type App struct {
	ctx          context.Context
	store        *accounts.Store
	initErr      error
	loginMu      sync.Mutex
	loginCancels map[string]context.CancelFunc
}

type AccountView struct {
	accounts.Account
	Active bool `json:"active"`
}

func NewApp() *App {
	store, err := accounts.NewStore()
	return &App{
		store:        store,
		initErr:      err,
		loginCancels: make(map[string]context.CancelFunc),
	}
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

func (a *App) UpdateAccount(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	secret, err := securestore.Get(id)
	if err != nil {
		return fmt.Errorf("get account credentials from Keychain: %w", err)
	}
	var credentials codexoauth.Credentials
	if err := json.Unmarshal([]byte(secret), &credentials); err != nil {
		return fmt.Errorf("decode account credentials: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	profile, err := codexoauth.UserInfo(ctx, credentials.AccessToken)
	if err != nil {
		return err
	}
	name := profile.Name
	if name == "" {
		name = profile.Email
	}
	if name == "" {
		name = credentials.AccountID
	}
	planType := credentials.PlanType
	if tokenPlan, err := codexoauth.PlanTypeFromAccessToken(credentials.AccessToken); err == nil {
		planType = tokenPlan
	}
	return a.store.UpdateProfile(id, name, profile.Email, profile.Avatar, planType)
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
		loginCtx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
		a.loginMu.Lock()
		if _, exists := a.loginCancels[id]; exists {
			a.loginMu.Unlock()
			cancel()
			return errors.New("account login is already in progress")
		}
		a.loginCancels[id] = cancel
		a.loginMu.Unlock()
		go func() {
			defer func() {
				a.loginMu.Lock()
				delete(a.loginCancels, id)
				a.loginMu.Unlock()
			}()
			_ = a.runLogin(loginCtx, account)
		}()
		return nil
	}
	return fmt.Errorf("account not found")
}

func (a *App) runLogin(ctx context.Context, account accounts.Account) error {
	credentials, err := codexoauth.Login(ctx, func() {
		wailsruntime.WindowShow(a.ctx)
		wailsruntime.WindowUnminimise(a.ctx)
	})
	if err != nil {
		return a.loginFailure(account.ID, err)
	}
	//nolint:gosec // Credentials are immediately stored in the macOS Keychain and never written to disk.
	secret, err := json.Marshal(credentials)
	if err != nil {
		return a.loginFailure(account.ID, fmt.Errorf("encode credentials: %w", err))
	}
	if err := securestore.Set(account.ID, string(secret)); err != nil {
		return a.loginFailure(account.ID, fmt.Errorf("save credentials to Keychain: %w", err))
	}
	profile, err := codexoauth.UserInfo(ctx, credentials.AccessToken)
	if err != nil {
		profile.Name = credentials.DisplayName
	}
	name := profile.Name
	if name == "" {
		name = profile.Email
	}
	if name == "" {
		name = credentials.AccountID
	}
	if err := a.store.UpdateProfile(account.ID, name, profile.Email, profile.Avatar, credentials.PlanType); err != nil {
		return a.loginFailure(account.ID, fmt.Errorf("save account profile: %w", err))
	}
	if err := a.store.SetStatus(account.ID, accounts.StatusReady, ""); err != nil {
		return fmt.Errorf("set account ready status: %w", err)
	}
	return nil
}

func (a *App) CancelLogin(id string) error {
	a.loginMu.Lock()
	cancel, exists := a.loginCancels[id]
	a.loginMu.Unlock()
	if !exists {
		return errors.New("account login is not in progress")
	}
	cancel()
	return nil
}

func (a *App) loginFailure(id string, loginErr error) error {
	statusErr := a.store.SetStatus(id, accounts.StatusError, loginErr.Error())
	if statusErr != nil {
		return errors.Join(loginErr, fmt.Errorf("set account error status: %w", statusErr))
	}
	return loginErr
}

func (a *App) RemoveAccount(id string) error {
	if a.initErr != nil {
		return a.initErr
	}
	a.loginMu.Lock()
	cancel, exists := a.loginCancels[id]
	a.loginMu.Unlock()
	if exists {
		cancel()
	}
	if err := securestore.Delete(id); err != nil {
		return fmt.Errorf("delete account credentials from Keychain: %w", err)
	}
	if err := a.store.Remove(id); err != nil {
		return fmt.Errorf("remove account data: %w", err)
	}
	return nil
}
