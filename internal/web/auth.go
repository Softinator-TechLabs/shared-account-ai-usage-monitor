package web

import (
	"context"
	"crypto/subtle"
	"errors"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"net/http"
	"sync"
	"time"
)

type loginState struct {
	nonce, verifier, browser string
	expires                  time.Time
}
type oidcFlow struct {
	config   oauth2.Config
	verifier *oidc.IDTokenVerifier
	mu       sync.Mutex
	pending  map[string]loginState
}

func newOIDC(ctx context.Context, cfg Config) (*oidcFlow, error) {
	if cfg.Issuer == "" || cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("OIDC issuer/client ID/client secret required")
	}
	provider, e := oidc.NewProvider(ctx, cfg.Issuer)
	if e != nil {
		return nil, errors.New("OIDC discovery failed")
	}
	return &oidcFlow{config: oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, Endpoint: provider.Endpoint(), RedirectURL: cfg.Origin + "/auth/callback", Scopes: []string{oidc.ScopeOpenID}}, verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}), pending: map[string]loginState{}}, nil
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		http.Redirect(w, r, "/", 303)
		return
	}
	state := store.ID()
	flow := loginState{nonce: store.ID(), verifier: oauth2.GenerateVerifier(), browser: store.ID(), expires: time.Now().Add(5 * time.Minute)}
	s.auth.mu.Lock()
	for k, v := range s.auth.pending {
		if time.Now().After(v.expires) {
			delete(s.auth.pending, k)
		}
	}
	if len(s.auth.pending) > 10000 {
		s.auth.mu.Unlock()
		w.WriteHeader(429)
		return
	}
	s.auth.pending[state] = flow
	s.auth.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: "team_oidc", Value: flow.browser, Path: "/auth/callback", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 300})
	http.Redirect(w, r, s.auth.config.AuthCodeURL(state, oidc.Nonce(flow.nonce), oauth2.S256ChallengeOption(flow.verifier)), 302)
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	if s.auth == nil {
		fail(w, c.ErrForbidden)
		return
	}
	state := r.URL.Query().Get("state")
	s.auth.mu.Lock()
	flow, ok := s.auth.pending[state]
	delete(s.auth.pending, state)
	s.auth.mu.Unlock()
	cookie, e := r.Cookie("team_oidc")
	if !ok || e != nil || time.Now().After(flow.expires) || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(flow.browser)) != 1 {
		fail(w, c.ErrForbidden)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	tok, e := s.auth.config.Exchange(ctx, r.URL.Query().Get("code"), oauth2.VerifierOption(flow.verifier))
	if e != nil {
		fail(w, c.ErrForbidden)
		return
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok {
		fail(w, c.ErrForbidden)
		return
	}
	id, e := s.auth.verifier.Verify(ctx, raw)
	if e != nil || id.Nonce != flow.nonce {
		fail(w, c.ErrForbidden)
		return
	}
	session, e := s.store.HumanToken(ctx, s.config.Workspace, id.Subject)
	if e != nil {
		fail(w, e)
		return
	}
	s.cookie(w, session, 28800)
	http.SetCookie(w, &http.Cookie{Name: "team_oidc", Value: "", Path: "/auth/callback", Secure: true, HttpOnly: true, MaxAge: -1})
	http.Redirect(w, r, "/", 303)
}
