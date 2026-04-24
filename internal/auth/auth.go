// Package auth implements the Slack OAuth 2.0 PKCE authorization flow.
package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	callbackPort = 7777
	callbackPath = "/callback"
	authTimeout  = 5 * time.Minute

	slackAuthURL  = "https://slack.com/oauth/v2/authorize"
	slackTokenURL = "https://slack.com/api/oauth.v2.access"

	// userScopes lists all OAuth user token scopes required by scli.
	userScopes = "channels:read,groups:read,im:read,im:write,mpim:read," +
		"channels:history,groups:history,im:history,mpim:history," +
		"chat:write,files:read,files:write,search:read,users:read"
)

// Config holds the OAuth application credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// DefaultRedirectURI returns the local HTTPS callback URL used during OAuth.
// Register this URL in your Slack app's Redirect URLs list.
// Note: the browser will show a self-signed certificate warning; this is expected
// and unavoidable for local HTTPS callbacks.
func DefaultRedirectURI() string {
	return fmt.Sprintf("https://localhost:%d%s", callbackPort, callbackPath)
}

// TokenResponse is returned after a successful OAuth login.
type TokenResponse struct {
	AccessToken string
	TeamID      string
	UserID      string
	TeamName    string
}

// Authorizer performs the OAuth PKCE authorization flow against Slack.
type Authorizer struct {
	cfg         Config
	httpClient  *http.Client
	openBrowser func(string) error
	// tokenURL and authURL are overridable in tests.
	tokenURL string
	authURL  string
}

// New creates an Authorizer with production defaults.
func New(cfg Config) *Authorizer {
	return &Authorizer{
		cfg:         cfg,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		openBrowser: openBrowserCmd,
		tokenURL:    slackTokenURL,
		authURL:     slackAuthURL,
	}
}

// Login runs the full OAuth PKCE flow using a local HTTPS server to receive
// the callback automatically. A self-signed certificate is generated at runtime
// for localhost; the browser will display a certificate warning that the user
// must accept before the redirect is followed.
func (a *Authorizer) Login(ctx context.Context) (*TokenResponse, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate code verifier: %w", err)
	}
	challenge := codeChallenge(verifier)

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("generate TLS certificate: %w", err)
	}

	authURL := a.buildAuthURL(challenge, state)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", callbackPort),
		Handler:     mux,
		ReadTimeout: authTimeout,
		TLSConfig:   &tls.Config{Certificates: []tls.Certificate{cert}},
	}

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch: possible CSRF attack")
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("missing code in callback (error: %s)", r.URL.Query().Get("error"))
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "<html><body><h2>Authentication successful.</h2><p>You may close this tab.</p></body></html>")
		codeCh <- code
	})

	listener, err := tls.Listen("tcp", fmt.Sprintf(":%d", callbackPort),
		&tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		return nil, fmt.Errorf("start HTTPS listener on port %d: %w", callbackPort, err)
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server: %w", err)
		}
	}()
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	fmt.Printf("Opening browser for Slack authentication...\n")
	fmt.Printf("NOTE: your browser will show a certificate warning for localhost.\n")
	fmt.Printf("      This is expected — accept it to complete the login.\n\n")
	fmt.Printf("If the browser does not open, visit:\n  %s\n\n", authURL)
	if err := a.openBrowser(authURL); err != nil {
		fmt.Printf("Note: could not open browser automatically (%v)\n", err)
	}

	var code string
	select {
	case code = <-codeCh:
	case err = <-errCh:
		return nil, err
	case <-time.After(authTimeout):
		return nil, fmt.Errorf("authentication timed out after %s", authTimeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return a.exchangeCode(ctx, code, verifier)
}

// LoginManual runs the OAuth PKCE flow without a local server.
// Use this in headless environments or when the automatic browser flow fails.
// Slack redirects the browser to the redirect URI; the user copies the full
// redirect URL (or just the code value) from the address bar and pastes it here.
func (a *Authorizer) LoginManual(ctx context.Context) (*TokenResponse, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate code verifier: %w", err)
	}
	challenge := codeChallenge(verifier)

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	authURL := a.buildAuthURL(challenge, state)

	fmt.Printf("Open the following URL in your browser:\n\n  %s\n\n", authURL)
	fmt.Printf("After authorizing, Slack redirects to:\n  %s?code=...\n", a.cfg.RedirectURI)
	fmt.Printf("The page will fail to load — that is expected.\n")
	fmt.Printf("Copy the full URL from the browser's address bar and paste it here:\n> ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no input received")
	}
	input := strings.TrimSpace(scanner.Text())

	code := extractCode(input)
	if code == "" {
		return nil, fmt.Errorf("could not find authorization code in: %q", input)
	}

	return a.exchangeCode(ctx, code, verifier)
}

// extractCode parses the authorization code from a full redirect URL or a bare code string.
func extractCode(input string) string {
	if strings.Contains(input, "?") {
		parsed, err := url.Parse(input)
		if err == nil {
			if code := parsed.Query().Get("code"); code != "" {
				return code
			}
		}
	}
	return input
}

func (a *Authorizer) buildAuthURL(challenge, state string) string {
	params := url.Values{
		"client_id":             {a.cfg.ClientID},
		"user_scope":            {userScopes},
		"redirect_uri":          {a.cfg.RedirectURI},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return a.authURL + "?" + params.Encode()
}

type slackTokenResponse struct {
	OK         bool   `json:"ok"`
	Error      string `json:"error"`
	AuthedUser struct {
		ID          string `json:"id"`
		AccessToken string `json:"access_token"`
	} `json:"authed_user"`
	Team struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"team"`
}

func (a *Authorizer) exchangeCode(ctx context.Context, code, verifier string) (*TokenResponse, error) {
	data := url.Values{
		"client_id":     {a.cfg.ClientID},
		"client_secret": {a.cfg.ClientSecret},
		"code":          {code},
		"redirect_uri":  {a.cfg.RedirectURI},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.tokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp slackTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if !tokenResp.OK {
		return nil, fmt.Errorf("slack authentication error: %s", tokenResp.Error)
	}

	return &TokenResponse{
		AccessToken: tokenResp.AuthedUser.AccessToken,
		TeamID:      tokenResp.Team.ID,
		UserID:      tokenResp.AuthedUser.ID,
		TeamName:    tokenResp.Team.Name,
	}, nil
}

// generateSelfSignedCert creates an in-memory self-signed TLS certificate for localhost.
// The certificate is valid for 24 hours and covers both the "localhost" hostname
// and the loopback IP addresses (127.0.0.1 and ::1).
func generateSelfSignedCert() (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate RSA key: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"scli local OAuth callback"},
		},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return tls.X509KeyPair(certPEM, keyPEM)
}

// generateCodeVerifier creates a cryptographically random PKCE code verifier.
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// codeChallenge computes the S256 PKCE challenge from a verifier string.
func codeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// generateState creates a random CSRF state token.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// openBrowserCmd opens the given URL in the default system browser.
func openBrowserCmd(rawURL string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{rawURL}
	case "linux":
		cmd, args = "xdg-open", []string{rawURL}
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", rawURL}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return exec.Command(cmd, args...).Start() //nolint:gosec // URL is constructed internally
}
