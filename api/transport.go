package api

import (
	"encoding/json"
	"fmt"
	"github.com/Forau/nordnet/util/models"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

type RequestGeneratorFn func(method, baseUrl, path string, params *Params) (*http.Request, error)
type RequestExecutorFn func(*http.Request) (*http.Response, error)

type HttpTransport struct {
	BaseURL, Service       string
	LoginState             *models.Login
	ExpiresAt, LastUsageAt time.Time

	credProvFn CredentialsProviderFn // If we need to re-login, new credentials must be generated

	ReqGenFn  RequestGeneratorFn
	ReqExecFn RequestExecutorFn
}

// If we already are logged in, a touch is preformed and then the login struct is returned
func (hs *HttpTransport) LoginHook() (res *models.Login, err error) {
	// If we have a cached state, see if we can use it
	if hs.LoginState != nil {
		// Do a touch to see if we status still is valid
		status := &models.LoggedInStatus{}
		err := hs.Perform("PUT", "login", nil, res)

		if err == nil && status.LoggedIn {
			// We eigher have an error, or is already logged in
			log.Print("Already logged in")
			return hs.LoginState, err
		}
	}
	params := &Params{"auth": hs.credProvFn(), "service": hs.Service}
	err = hs.Perform("POST", "login", params, &hs.LoginState)
	return hs.LoginState, err
}

func NewHttpTransport(url, version, service string, credentialFn CredentialsProviderFn) *HttpTransport {
	execFn := NewDefaultRequestExecutorFn()
	execFn = WrapConditionalLogging(execFn, log.New(os.Stderr, "", log.Lshortfile), func(status int) bool {
		return status < 200 || status >= 300
	})

	return &HttpTransport{
		BaseURL:    fmt.Sprintf("%s/%s", url, version),
		credProvFn: credentialFn,
		Service:    service,
		ReqGenFn:   DefaultRequestGeneratorFn,
		ReqExecFn:  execFn,
	}
}

func (hs *HttpTransport) Perform(method, path string, params *Params, res interface{}) (err error) {
	// First we check for login request with no parameters, and hijack the login call
	if path == "login" && method == "POST" && params == nil {
		// And we only run the hook if res is of type Login. Also, we need it as pointer
		if resPtr, ok := res.(*models.Login); ok {
			hookRes, err := hs.LoginHook()
			if hookRes != nil {
				*resPtr = *hookRes
			}
			return err
		} else {
      log.Printf("Called login, but could not hook, due to %T is not Login", res)
    }
	}

	req, err := hs.ReqGenFn(method, hs.BaseURL, path, params)

	if hs.LoginState != nil && hs.LoginState.SessionKey != "" {
		req.SetBasicAuth(hs.LoginState.SessionKey, hs.LoginState.SessionKey)
	}

	if err != nil {
		return
	}

	resp, err := hs.ReqExecFn(req)
	if err != nil {
		return
	}

	hs.LastUsageAt = time.Now()
	defer resp.Body.Close()

	if resp != nil {
		switch resp.StatusCode {
		case 204:
			return
		case 400, 401, 404:
			errRes := APIError{}
			if err = json.NewDecoder(resp.Body).Decode(&errRes); err != nil {
				return
			}
			return errRes
		case 429:
			return TooManyRequestsError
		}
	}

	err = json.NewDecoder(resp.Body).Decode(&res)
	return
}

func (hs *HttpTransport) formatURL(path string, params *Params) (*url.URL, error) {
	var absURL string
	if path == "" {
		absURL = hs.BaseURL
	} else {
		absURL = hs.BaseURL + "/" + path
	}

	if reqURL, err := url.Parse(absURL); err != nil {
		return nil, err
	} else {
		if params != nil {
			reqQuery := reqURL.Query()
			for key, value := range *params {
				reqQuery.Set(key, value)
			}
			reqURL.RawQuery = reqQuery.Encode()
		}

		return reqURL, nil
	}
}

var DefaultRequestGeneratorFn = func(method, baseUrl, path string, params *Params) (req *http.Request, err error) {
	var absURL string
	if path == "" {
		absURL = baseUrl
	} else {
		absURL = baseUrl + "/" + path
	}

	// Values will become postData or query, depending on method
	values := &url.Values{}
	if params != nil {
		for key, value := range *params {
			values.Set(key, value)
		}
	}

	if reqURL, err := url.Parse(absURL); err != nil {
		return nil, err
	} else {
		if method == "POST" || method == "PUT" {
			body := values.Encode()
			req, err = http.NewRequest(method, reqURL.String(), strings.NewReader(body))
			if req != nil {
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			}
		} else {

			reqURL.RawQuery = values.Encode()
			req, err = http.NewRequest(method, reqURL.String(), nil)
		}
	}
	if req != nil {
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Language", "en")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	}
	return
}

func NewDefaultRequestExecutorFn() RequestExecutorFn {
	client := &http.Client{}

	return func(req *http.Request) (resp *http.Response, err error) {
		resp, err = client.Do(req)
		return
	}
}

func WrapConditionalLogging(rexec RequestExecutorFn, logger *log.Logger, cond func(status int) bool) RequestExecutorFn {
	logDataOrError := func(data []byte, err error) {
		if err != nil {
			logger.Println(err)
		} else {
			logger.Println(string(data))
		}
	}

	return func(req *http.Request) (resp *http.Response, err error) {
		statusCode := 0 // Will be used in case of error
		reqData, reqErr := httputil.DumpRequest(req, true)

		resp, err = rexec(req)
		if resp != nil {
			statusCode = resp.StatusCode
		}
		if cond(statusCode) {
			logDataOrError(reqData, reqErr)

			if resp != nil {
				logDataOrError(httputil.DumpResponse(resp, true))
			}
		}
		return
	}
}
