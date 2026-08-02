package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"mux/internal/accounts"
	"mux/internal/codexoauth"
	"mux/internal/gateway"
	"mux/internal/logger"
	"mux/internal/securestore"
)

type App struct {
	ctx          context.Context
	store        *accounts.Store
	initErr      error
	loginMu      sync.Mutex
	loginCancels map[string]context.CancelFunc
	syncMu       sync.Mutex
	syncMinutes  int
	syncChanges  chan time.Duration
	settingsPath string
	logger       *logger.Logger
	gateway      *gateway.Gateway
	gatewayPort  int
	affinityMu   sync.Mutex
	affinity     map[string]string
	gatewayMu    sync.Mutex
	gatewayRun   bool
}

type AccountView struct {
	accounts.Account
	Active bool `json:"active"`
}

type settings struct {
	SyncIntervalMinutes int `json:"syncIntervalMinutes"`
	GatewayPort         int `json:"gatewayPort"`
}

func NewApp() *App {
	store, err := accounts.NewStore()
	app := &App{
		store:        store,
		initErr:      err,
		loginCancels: make(map[string]context.CancelFunc),
		syncMinutes:  10,
		syncChanges:  make(chan time.Duration),
		affinity:     make(map[string]string),
	}
	if store != nil {
		app.logger, err = logger.New(store.Root())
		if err != nil {
			app.initErr = err
		}
		app.settingsPath = filepath.Join(store.Root(), "settings.json")
		if contents, readErr := os.ReadFile(app.settingsPath); readErr == nil {
			var saved settings
			if json.Unmarshal(contents, &saved) == nil {
				if saved.SyncIntervalMinutes >= 5 {
					app.syncMinutes = saved.SyncIntervalMinutes
				}
				if saved.GatewayPort >= 1024 && saved.GatewayPort <= 65535 {
					app.gatewayPort = saved.GatewayPort
				}
			}
		}
	}
	return app
}

func (a *App) shutdown(ctx context.Context) {
	if a.gateway != nil {
		_ = a.stopGateway(ctx)
	}
	if a.logger != nil {
		_ = a.logger.Close()
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.logger != nil {
		a.logger.Info("application started")
	}
	if a.gatewayPort == 0 {
		a.gatewayPort = 8787
	}
	go a.startAccountSync(ctx)
}

func (a *App) selectGatewayCredential(ctx context.Context, model string) (gateway.Credential, error) {
	accountsList := a.store.List()
	a.affinityMu.Lock()
	preferredID := a.affinity[model]
	a.affinityMu.Unlock()
	ordered := make([]accounts.Account, 0, len(accountsList))
	for _, account := range accountsList {
		if account.ID == preferredID {
			ordered = append(ordered, account)
		}
	}
	for _, account := range accountsList {
		if account.ID != preferredID {
			ordered = append(ordered, account)
		}
	}
	var lastErr error
	for _, account := range ordered {
		if account.Status != accounts.StatusReady || account.Usage == nil || account.Usage.PrimaryWindow == nil || account.Usage.PrimaryWindow.UsedPercent >= 100 {
			continue
		}
		secret, err := securestore.Get(account.ID)
		if err != nil {
			lastErr = err
			continue
		}
		var credentials codexoauth.Credentials
		if err := json.Unmarshal([]byte(secret), &credentials); err != nil {
			lastErr = err
			continue
		}
		if credentials.ExpiresAt <= time.Now().Add(5*time.Minute).UnixMilli() {
			credentials, err = codexoauth.Refresh(ctx, credentials)
			if err != nil {
				lastErr = err
				continue
			}
			//nolint:gosec // credentials are immediately written to the Keychain.
			encoded, _ := json.Marshal(credentials)
			if err := securestore.Set(account.ID, string(encoded)); err != nil {
				lastErr = err
				continue
			}
		}
		a.affinityMu.Lock()
		a.affinity[model] = account.ID
		a.affinityMu.Unlock()
		return gateway.Credential{AccountID: credentials.AccountID, AccessToken: credentials.AccessToken}, nil
	}
	if lastErr != nil {
		return gateway.Credential{}, fmt.Errorf("no available account: %w", lastErr)
	}
	return gateway.Credential{}, errors.New("no account with available quota")
}

func (a *App) startAccountSync(ctx context.Context) {
	sync := func() {
		for _, account := range a.store.List() {
			if a.logger != nil {
				a.logger.Info("background account sync started", slog.String("account", account.ID))
			}
			if err := a.UpdateAccount(account.ID); err != nil {
				if a.logger != nil {
					a.logger.Error("account sync failed", slog.String("account", account.ID), slog.Any("error", err))
				}
				continue
			}
			if a.logger != nil {
				a.logger.Info("account synced", slog.String("account", account.ID))
			}
		}
	}

	sync()
	a.syncMu.Lock()
	ticker := time.NewTicker(time.Duration(a.syncMinutes) * time.Minute)
	a.syncMu.Unlock()
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sync()
		case interval := <-a.syncChanges:
			ticker.Stop()
			ticker = time.NewTicker(interval)
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) SetSyncInterval(minutes int) error {
	if minutes < 5 {
		return errors.New("sync interval must be at least 5 minutes")
	}
	a.syncMu.Lock()
	a.syncMinutes = minutes
	saveErr := os.WriteFile(a.settingsPath, mustJSON(settings{SyncIntervalMinutes: minutes, GatewayPort: a.gatewayPort}), 0o600)
	a.syncMu.Unlock()
	if saveErr != nil {
		return fmt.Errorf("save settings: %w", saveErr)
	}
	if a.logger != nil {
		a.logger.Info("sync interval changed", slog.Int("minutes", minutes))
	}
	select {
	case a.syncChanges <- time.Duration(minutes) * time.Minute:
	default:
	}
	return nil
}

func mustJSON(value any) []byte {
	contents, _ := json.MarshalIndent(value, "", "  ")
	return append(contents, '\n')
}

func (a *App) GetSyncInterval() int {
	a.syncMu.Lock()
	defer a.syncMu.Unlock()
	return a.syncMinutes
}

func (a *App) StartGateway() error {
	a.gatewayMu.Lock()
	defer a.gatewayMu.Unlock()
	if a.gatewayRun {
		return nil
	}
	if a.gatewayPort == 0 {
		a.gatewayPort = 8787
	}
	a.gateway = gateway.New(a.gatewayPort, a.selectGatewayCredential, a.logger)
	a.gatewayRun = true
	go func(server *gateway.Gateway, port int) {
		if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) && a.logger != nil {
			a.logger.Error("gateway stopped", slog.Any("error", err), slog.Int("port", port))
		}
		a.gatewayMu.Lock()
		a.gatewayRun = false
		a.gatewayMu.Unlock()
	}(a.gateway, a.gatewayPort)
	return nil
}

func (a *App) StopGateway() error {
	return a.stopGateway(context.Background())
}

func (a *App) stopGateway(ctx context.Context) error {
	a.gatewayMu.Lock()
	server := a.gateway
	a.gatewayRun = false
	a.gatewayMu.Unlock()
	if server == nil {
		return nil
	}
	return server.Close(ctx)
}

func (a *App) IsGatewayRunning() bool {
	a.gatewayMu.Lock()
	defer a.gatewayMu.Unlock()
	return a.gatewayRun
}

func (a *App) GetGatewayPort() int {
	if a.gatewayPort == 0 {
		return 8787
	}
	return a.gatewayPort
}

func (a *App) SetGatewayPort(port int) error {
	if port < 1024 || port > 65535 {
		return errors.New("gateway port must be between 1024 and 65535")
	}
	if err := a.stopGateway(context.Background()); err != nil {
		return err
	}
	a.gatewayPort = port
	if err := os.WriteFile(a.settingsPath, mustJSON(settings{SyncIntervalMinutes: a.syncMinutes, GatewayPort: port}), 0o600); err != nil {
		return fmt.Errorf("save gateway settings: %w", err)
	}
	if a.ctx != nil {
		a.gateway = gateway.New(port, a.selectGatewayCredential, a.logger)
		go func() { _ = a.gateway.Start() }()
	}
	if a.logger != nil {
		a.logger.Info("gateway port changed", slog.Int("port", port))
	}
	return nil
}

func randomGatewayAPIKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate gateway API key: %w", err)
	}
	return "mux-local-" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (a *App) ConfigureFinchGateway() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home directory: %w", err)
	}
	finchHome := filepath.Join(home, ".finch")
	devModels := filepath.Join(home, ".finch-dev", "models.json")
	modelsPath := filepath.Join(finchHome, "models.json")
	if _, err := os.Stat(devModels); err == nil {
		modelsPath = devModels
	}
	stored := map[string]any{}
	//nolint:gosec // modelsPath is a Finch configuration path selected by the application.
	if contents, err := os.ReadFile(modelsPath); err == nil {
		if err := json.Unmarshal(contents, &stored); err != nil {
			return fmt.Errorf("decode Finch models: %w", err)
		}
	}
	apiKey, err := randomGatewayAPIKey()
	if err != nil {
		return err
	}
	provider := map[string]any{
		"id":       "mux-gateway",
		"name":     "Mux Gateway",
		"api":      "openai-responses",
		"baseUrl":  fmt.Sprintf("http://127.0.0.1:%d", a.GetGatewayPort()),
		"apiKey":   apiKey,
		"enabled":  true,
		"isCustom": true,
		"models": []any{
			map[string]any{"id": "gpt-5.6-sol", "apiId": "gpt-5.6-sol", "name": "GPT-5.6 Sol", "enabled": true, "contextLength": 372000, "isDefault": true, "defaultReasoningEffort": "medium"},
			map[string]any{"id": "gpt-5.6-terra", "apiId": "gpt-5.6-terra", "name": "GPT-5.6 Terra", "enabled": true, "contextLength": 372000, "defaultReasoningEffort": "medium"},
			map[string]any{"id": "gpt-5.6-luna", "apiId": "gpt-5.6-luna", "name": "GPT-5.6 Luna", "enabled": true, "contextLength": 372000, "instant": true, "defaultReasoningEffort": "low"},
			map[string]any{"id": "gpt-5.5", "apiId": "gpt-5.5", "name": "GPT-5.5", "enabled": true, "contextLength": 272000, "defaultReasoningEffort": "medium"},
			map[string]any{"id": "gpt-5.4", "apiId": "gpt-5.4", "name": "GPT-5.4", "enabled": true, "contextLength": 272000, "defaultReasoningEffort": "medium"},
			map[string]any{"id": "gpt-5.4-mini", "apiId": "gpt-5.4-mini", "name": "GPT-5.4 mini", "enabled": true, "contextLength": 400000, "instant": true, "defaultReasoningEffort": "low"},
		},
	}
	providers, _ := stored["customProviders"].([]any)
	updated := false
	for i, item := range providers {
		if existing, ok := item.(map[string]any); ok && existing["id"] == "mux-gateway" {
			providers[i] = provider
			updated = true
			break
		}
	}
	if !updated {
		providers = append(providers, provider)
	}
	stored["customProviders"] = providers
	contents, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Finch models: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(modelsPath), 0o700); err != nil {
		return fmt.Errorf("create Finch config directory: %w", err)
	}
	if err := os.WriteFile(modelsPath, append(contents, '\n'), 0o600); err != nil {
		return fmt.Errorf("write Finch models: %w", err)
	}
	if a.logger != nil {
		a.logger.Info("Finch gateway provider configured", slog.String("path", modelsPath), slog.Int("port", a.GetGatewayPort()))
	}
	return nil
}

func (a *App) RemoveFinchGateway() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get user home directory: %w", err)
	}
	modelsPath := filepath.Join(home, ".finch", "models.json")
	devModels := filepath.Join(home, ".finch-dev", "models.json")
	if _, err := os.Stat(devModels); err == nil {
		modelsPath = devModels
	}
	//nolint:gosec // modelsPath is the Finch configuration path selected by the application.
	contents, err := os.ReadFile(modelsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read Finch models: %w", err)
	}
	stored := map[string]any{}
	if err := json.Unmarshal(contents, &stored); err != nil {
		return fmt.Errorf("decode Finch models: %w", err)
	}
	providers, _ := stored["customProviders"].([]any)
	filtered := make([]any, 0, len(providers))
	for _, item := range providers {
		provider, ok := item.(map[string]any)
		if !ok || provider["id"] != "mux-gateway" {
			filtered = append(filtered, item)
		}
	}
	stored["customProviders"] = filtered
	updated, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode Finch models: %w", err)
	}
	//nolint:gosec // modelsPath is the Finch configuration path selected by the application.
	if err := os.WriteFile(modelsPath, append(updated, '\n'), 0o600); err != nil {
		return fmt.Errorf("write Finch models: %w", err)
	}
	return nil
}

func (a *App) OpenConfigFileFolder() error {
	if a.initErr != nil {
		return a.initErr
	}
	//nolint:gosec // the path is the application directory resolved by Store.
	return exec.Command("open", a.store.Root()).Run()
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
	if err := a.store.UpdateProfile(id, name, profile.Email, profile.Avatar, planType); err != nil {
		return err
	}
	usage, err := codexoauth.UsageInfo(ctx, credentials.AccessToken, credentials.AccountID)
	if err != nil {
		return err
	}
	if err := a.store.SetUsage(id, usageView(usage)); err != nil {
		return err
	}
	if a.logger != nil {
		a.logger.Info("account information and usage updated", slog.String("account", id))
	}
	return nil
}

func usageView(usage codexoauth.Usage) *accounts.Usage {
	result := &accounts.Usage{PlanType: usage.PlanType}
	if usage.RateLimit != nil {
		result.LimitReached = usage.RateLimit.LimitReached
		result.PrimaryWindow = usageWindowView(usage.RateLimit.PrimaryWindow)
		result.SecondaryWindow = usageWindowView(usage.RateLimit.SecondaryWindow)
	}
	if usage.Credits != nil {
		result.HasCredits = usage.Credits.HasCredits
		result.Unlimited = usage.Credits.Unlimited
		result.Balance = usage.Credits.Balance
	}
	return result
}

func usageWindowView(window *codexoauth.UsageWindow) *accounts.UsageWindow {
	if window == nil {
		return nil
	}
	return &accounts.UsageWindow{UsedPercent: window.UsedPercent, LimitWindowSeconds: window.LimitWindowSeconds, ResetAt: window.ResetAt}
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
	profile := codexoauth.UserProfile{}
	fetchedProfile, err := codexoauth.UserInfo(ctx, credentials.AccessToken)
	if err == nil {
		profile = fetchedProfile
	} else {
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
