package rsapi

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/rightscale/rsc/cmd"
	"github.com/rightscale/rsc/metadata"
)

// RightScale client
// Instances of this struct should be created through `New`, `NewRL10` or `FromCommandLine`.
type Api struct {
	Auth                  Authenticator // Authenticator, signs requests for auth
	Logger                *log.Logger   // Optional logger, if specified requests and responses get logged
	Host                  string        // API host, e.g. "us-3.rightscale.com"
	Client                HttpClient    // Underlying http client
	Unsecure              bool          // Whether HTTP should be used instead of HTTPS (used by RL10 proxied requests)
	DumpRequestResponse   Format        // Whether to dump HTTP requests and responses to STDOUT, and if so in which format
	FetchLocationResource bool          // Whether to fetch resource pointed by Location header
	Metadata              ApiMetadata   // Generated API metadata
}

// Request/response dump format
type Format int

const (
	NoDump Format = iota
	Debug
	Json
)

// Api metadata consists of resource metadata indexed by resource name
type ApiMetadata map[string]*metadata.Resource

// Generic API parameter type, used to specify optional parameters for example
type ApiParams map[string]interface{}

// Use interface instead of raw http.Client to ease testing
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// New returns a API client that uses the given authenticator.
// logger and client are optional.
// host may be blank in which case client attempts to resolve it using auth.
// If no HTTP client is specified then the default client is used.
func New(host string, auth Authenticator, logger *log.Logger, client HttpClient) *Api {
	if auth != nil {
		auth.SetHost(host)
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &Api{
		Auth:   auth,
		Logger: logger,
		Host:   host,
		Client: client,
	}
}

// NewRL10 returns a API client that uses the information stored in /var/run/rightlink/secret to do
// auth and configure the host. The client behaves identically to the client returned by New in
// all other regards.
func NewRL10(logger *log.Logger, client HttpClient) (*Api, error) {
	rllConfig, err := os.Open(RllSecret)
	if err != nil {
		return nil, fmt.Errorf("Failed to load RLL config: %s", err)
	}
	defer rllConfig.Close()
	var port string
	var secret string
	scanner := bufio.NewScanner(rllConfig)
	for scanner.Scan() {
		line := scanner.Text()
		elems := strings.Split(line, "=")
		if len(elems) != 2 {
			return nil, fmt.Errorf("Invalid RLL configuration line '%s'", line)
		}
		switch elems[0] {
		case "RS_RLL_PORT":
			port = elems[1]
			if _, err := strconv.Atoi(elems[1]); err != nil {
				return nil, fmt.Errorf("Invalid port value '%s'", port)
			}
		case "RS_RLL_SECRET":
			secret = elems[1]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Failed to load RLL config: %s", err)
	}
	host := "localhost:" + port
	auth := NewRL10Authenticator(secret)
	auth.SetHost(host)
	return &Api{
		Auth:     auth,
		Logger:   logger,
		Host:     host,
		Client:   client,
		Unsecure: true,
	}, nil
}

// Build client from command line
func FromCommandLine(cmdLine *cmd.CommandLine) (*Api, error) {
	var client *Api
	var httpClient *http.Client
	if cmdLine.NoRedirect {
		httpClient = &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return fmt.Errorf("Client configured to prevent redirection")
			},
		}
	} else {
		httpClient = http.DefaultClient
	}
	if cmdLine.RL10 {
		var err error
		if client, err = NewRL10(nil, httpClient); err != nil {
			return nil, err
		}
	} else if cmdLine.OAuthToken != "" {
		auth := NewOAuthAuthenticator(cmdLine.OAuthToken)
		client = New(cmdLine.Host, auth, nil, httpClient)
	} else if cmdLine.OAuthAccessToken != "" {
		auth := NewTokenAuthenticator(cmdLine.OAuthAccessToken)
		client = New(cmdLine.Host, auth, nil, httpClient)
	} else if cmdLine.APIToken != "" {
		auth := NewInstanceAuthenticator(cmdLine.APIToken, cmdLine.Account)
		client = New(cmdLine.Host, auth, nil, httpClient)
	} else if cmdLine.Username != "" && cmdLine.Password != "" {
		auth := NewBasicAuthenticator(cmdLine.Username, cmdLine.Password, cmdLine.Account)
		client = New(cmdLine.Host, auth, nil, httpClient)
	} else {
		// No auth, used by tests
		client = New(cmdLine.Host, nil, nil, httpClient)
		client.Unsecure = true
	}
	if !cmdLine.ShowHelp && !cmdLine.NoAuth {
		if cmdLine.OAuthToken == "" && cmdLine.OAuthAccessToken == "" && cmdLine.APIToken == "" && cmdLine.Username == "" && !cmdLine.RL10 {
			return nil, fmt.Errorf("Missing authentication information, use '--email EMAIL --password PWD', '--token TOKEN' or 'setup'")
		}
		client.DumpRequestResponse = NoDump
		if cmdLine.Dump == "json" {
			client.DumpRequestResponse = Json
		} else if cmdLine.Dump == "debug" {
			client.DumpRequestResponse = Debug
		}
		client.FetchLocationResource = cmdLine.FetchResource
	}
	return client, nil
}
