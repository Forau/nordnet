package api

import (
	"encoding/json"
	"fmt"
	"github.com/denro/nordnet/util/models"
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

type HttpSession struct {
	BaseURL, credentials, Service string
	LoginState                    *models.Login
	ExpiresAt, LastUsageAt        time.Time

	ReqGenFn  RequestGeneratorFn
	ReqExecFn RequestExecutorFn
}

// Login: Before any other of the services (except for the system info request) can be called the user must login. The username, password and phrase must be sent encrypted.
// If we already are logged in, a touch is preformed and then the login struct is returned
func (hs *HttpSession) Login() (res *models.Login, err error) {
	if hs.LoginState != nil {
		status, err := hs.Touch()
		if err != nil || status.LoggedIn {
			// We eigher have an error, or is already logged in
			return hs.LoginState, err
		}
	}
	params := &Params{"auth": hs.credentials, "service": hs.Service}

	err = hs.Perform("POST", "login", params, &hs.LoginState)
	return hs.LoginState, err
}

// Invalidates the session.
func (hs *HttpSession) Logout() (res *models.LoggedInStatus, err error) {
	res = &models.LoggedInStatus{}
	err = hs.Perform("DELETE", "login", nil, res)
	return
}

// If the application needs to keep the session alive the session can be touched. Note the basic auth header field must be set as for all other calls. All calls to any REST service is touching the session. So touching the session manually is only needed if no other calls are done during the session timeout interval.
func (hs *HttpSession) Touch() (res *models.LoggedInStatus, err error) {
	res = &models.LoggedInStatus{}
	err = hs.Perform("PUT", "login", nil, res)
	return
}

func NewHttpSession(url, version, service, credentials string) *HttpSession {
	execFn := NewDefaultRequestExecutorFn()
	execFn = WrapConditionalLogging(execFn, log.New(os.Stderr, "", log.Lshortfile), func(status int) bool {
		return status < 200 || status >= 300
	})
	execFn = WrapParseDefaultStatusCodes(execFn)

	return &HttpSession{
		BaseURL:     fmt.Sprintf("%s/%s", url, version),
		credentials: credentials,
		Service:     service,
		ReqGenFn:    NewDefaultRequestGeneratorFn(),
		ReqExecFn:   execFn,
	}
}

func (hs *HttpSession) Perform(method, path string, params *Params, res interface{}) (err error) {

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

	err = json.NewDecoder(resp.Body).Decode(&res)
	return
}

func (hs *HttpSession) formatURL(path string, params *Params) (*url.URL, error) {
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

func NewDefaultRequestGeneratorFn() RequestGeneratorFn {
	return func(method, baseUrl, path string, params *Params) (req *http.Request, err error) {
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
}

func NewDefaultRequestExecutorFn() RequestExecutorFn {
	client := &http.Client{}

	return func(req *http.Request) (resp *http.Response, err error) {
		resp, err = client.Do(req)
		return
	}
}

func WrapParseDefaultStatusCodes(rexec RequestExecutorFn) RequestExecutorFn {
	return func(req *http.Request) (resp *http.Response, err error) {
		resp, err = rexec(req)
		if resp != nil {
			switch resp.StatusCode {
			case 204:
				return
			case 400, 401, 404:
				errRes := APIError{}
				if err = json.NewDecoder(resp.Body).Decode(&errRes); err != nil {
					return
				}
				return resp, errRes
			case 429:
				return resp, TooManyRequestsError
			}
		}
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
