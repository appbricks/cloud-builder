package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/appbricks/cloud-builder/config"
	"github.com/mevansam/goutils/logger"
	"github.com/mevansam/goutils/utils"
)

type Authenticator struct {
	context config.AuthContext
	config *oauth2.Config

	authCallbackHandler func(w http.ResponseWriter, r *http.Request)

	// opaque value used to validate 
	// against CSRF attacks
	state string

	localServerExit *sync.WaitGroup
	localHttpServer *http.Server
	serverError error
}

func NewAuthenticator(
	context config.AuthContext,
	config *oauth2.Config,
	authCallbackHandler func(w http.ResponseWriter, r *http.Request),
) *Authenticator {

	return &Authenticator{
		context: context,
		config: config,
		authCallbackHandler: authCallbackHandler,

		localServerExit: &sync.WaitGroup{},
	}
}

// Starts an http listener locally to listen for
// the oauth redirect with authcode once the 
// user has been authenticated by the auth service.
func (authn *Authenticator) StartOAuthFlow(port int) (string, error) {

	// construct callback URL for auth code exchange
	authn.config.RedirectURL = fmt.Sprintf(
		"http://localhost:%d/callback",
		port,
	)

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/callback", authn.OAuthHandler)

	authn.localHttpServer = &http.Server{ 
		Addr: fmt.Sprintf(":%d", port),
		Handler: serveMux,
	}

	// mutex to wait on until server shuts down
	authn.localServerExit.Add(1)

	go func() {
		// signal server has shutdown
		defer func() {
			authn.config.RedirectURL = ""
			authn.localHttpServer = nil
			authn.localServerExit.Done()
		}()

		// always returns error. ErrServerClosed on graceful close
		if err := authn.localHttpServer.ListenAndServe(); err != http.ErrServerClosed {
			authn.serverError = err
			
			logger.DebugMessage(
				"Error serving local HTTP OAuth callback server: %# v",
				err)
		}
	}()

	// generate authorize URL where user will sign 
	// in and redirect back to the local server
	authn.state = utils.RandomString(10)
	authURL := authn.config.AuthCodeURL(authn.state)

	return authURL, nil
}

// Wait until OAuth flow has completed. Returns
// false is oath flow completes with callback
// to local server
func (authn *Authenticator) WaitForOAuthFlowCompletion(timeout time.Duration) (bool, error) {
	c := make(chan struct{})
	go func() {
			defer close(c)
			authn.localServerExit.Wait()
	}()
	select {
		case <-c:
			// server exited with callback 
			// from authentication service
			return false, authn.serverError
		case <-time.After(timeout):
			// timed out
			return true, nil
	}
}

// Handles the OAuth callback which exchanges the
// auth code in the request for a token and saves
// the token.
func (authn *Authenticator) OAuthHandler(w http.ResponseWriter, r *http.Request) {

	var (
		err error
	)

	logger.TraceMessage(
		"Received authorization callback: %s",
		r.RequestURI)

	defer func() {
		authn.state = ""
	}()

	if err = r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse request parameters", http.StatusBadRequest)
		return
	}
	state := r.Form.Get("state")
	if state != authn.state {
		http.Error(w, "State invalid", http.StatusBadRequest)
		return
	}
	code := r.Form.Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}
	if err = authn.RetrieveToken(code); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if authn.authCallbackHandler != nil {
		authn.authCallbackHandler(w, r)
	}
	
	if authn.localHttpServer != nil {	
		go func() {	
			if err = authn.localHttpServer.Shutdown(context.Background()); err != nil {
				authn.serverError = err
					
				logger.DebugMessage(
					"Error shutting down local HTTP OAuth callback server: %# v",
					err)
			}
		}()
	}
}

// Exchange given auth code for a token
func (authn *Authenticator) RetrieveToken(authCode string) error {

	var (
		err error

		token *oauth2.Token
	)

	if token, err = authn.config.Exchange(context.Background(), authCode); err != nil {
		return err
	}
	authn.context.SetToken(token)
	return nil
}

// Checks if the current auth context has been 
// authenticated. This will refresh the oauth 
// token if the access token has expired and 
// the refresh token has not expired
func (authn *Authenticator) IsAuthenticated() (bool, error) {

	var (
		err error
		token *oauth2.Token
	)

	token = authn.context.GetToken()
	if token == nil {
		return false, fmt.Errorf("not authenticated")
	}
	token.Expiry = time.Now()
	if token, err = authn.config.TokenSource(context.Background(), token).Token(); err != nil {
		return false, err
	}
	authn.context.SetToken(token)
	return true, nil
}
