package web

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	c "github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/contracts"
	"github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/internal/testutil"
	"github.com/go-jose/go-jose/v4"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestOIDCVerifiedSubjectNonceAndSingleUseState(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	owner := c.Principal{Workspace: "team", Person: "owner", Role: "owner"}
	db.SetPolicy(ctx, owner, c.Policy{Version: 1, Content: "full", Redaction: "none", Visibility: "team"})
	db.SetMember(ctx, owner, "owner", "owner", "verified-subject")
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	signer, e := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithHeader("kid", "test"))
	if e != nil {
		t.Fatal(e)
	}
	var issuer, nonce string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(map[string]any{"issuer": issuer, "authorization_endpoint": issuer + "/authorize", "token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys", "id_token_signing_alg_values_supported": []string{"RS256"}})
		case "/keys":
			json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"}}})
		case "/token":
			r.ParseForm()
			if r.Form.Get("code_verifier") == "" {
				t.Error("missing PKCE")
			}
			claims, _ := json.Marshal(map[string]any{"iss": issuer, "aud": "test-client", "sub": "verified-subject", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": nonce})
			signed, _ := signer.Sign(claims)
			token, _ := signed.CompactSerialize()
			json.NewEncoder(w).Encode(map[string]any{"access_token": "synthetic", "token_type": "Bearer", "id_token": token})
		default:
			w.WriteHeader(404)
		}
	}))
	defer provider.Close()
	issuer = provider.URL
	app, e := New(ctx, db, Config{Origin: "https://telemetry.example", Workspace: "team", Issuer: issuer, ClientID: "test-client", ClientSecret: "test-secret"})
	if e != nil {
		t.Fatal(e)
	}
	login := httptest.NewRecorder()
	app.ServeHTTP(login, httptest.NewRequest("GET", "/auth/login", nil))
	location, _ := url.Parse(login.Header().Get("Location"))
	state := location.Query().Get("state")
	nonce = location.Query().Get("nonce")
	if state == "" || nonce == "" || location.Query().Get("code_challenge") == "" {
		t.Fatal("missing state/nonce/PKCE")
	}
	callback := func(cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/auth/callback?state="+state+"&code=fixture", nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, r)
		return w
	}
	result := callback(login.Result().Cookies()[0])
	if result.Code != 303 {
		t.Fatal("valid OIDC rejected", result.Code, result.Body.String())
	}
	found := false
	for _, cookie := range result.Result().Cookies() {
		if cookie.Name == "team_session" {
			found = true
			if !cookie.HttpOnly || !cookie.Secure {
				t.Fatal("insecure cookie")
			}
			p, e := db.Authenticate(ctx, cookie.Value)
			if e != nil || p.Person != "owner" {
				t.Fatal("wrong identity", e)
			}
		}
	}
	if !found {
		t.Fatal("no session")
	}
	if replay := callback(login.Result().Cookies()[0]); replay.Code != 403 {
		t.Fatal("state replay", replay.Code)
	}
	login = httptest.NewRecorder()
	app.ServeHTTP(login, httptest.NewRequest("GET", "/auth/login", nil))
	location, _ = url.Parse(login.Header().Get("Location"))
	state = location.Query().Get("state")
	nonce = "wrong-nonce"
	if result = callback(login.Result().Cookies()[0]); result.Code != 403 {
		t.Fatal("nonce not verified", result.Code)
	}
}
