/*
This files interacts with the API rest to make petitions like searching some user
*/

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type SippoClient struct {
	Host string
	Auth Auth

	client *http.Client
}

type Auth struct {
	GrantType string `json:"grant_type"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type Meeting struct {
	User SippoUser `json:"user"`
}

type SippoUser struct {
	Username string `json:"username"`
	Domain   string `json:"domain"`
}

func (sc *SippoClient) httpClient() *http.Client {
	if sc.client == nil {
		return http.DefaultClient
	}
	return sc.client
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func (sc *SippoClient) GetMeetingsByPhone(token, phone string) ([]Meeting, error) {
	form := make(url.Values)
	form.Set("participants.phone", phone)
	req, err := sc.newRequest("GET", "/sapi/meetings", nil, form)
	req.Header.Set("Authorization", "Bearer "+token)
	var meetings []Meeting
	sc.do(req, &meetings)
	if len(meetings) == 0 {
		return nil, errors.New("No meetings for phone " + phone)
	}
	return meetings, err
}

/* func (sc *SippoClient) getAuthToken() (Token, error) {
	req, err := sc.newRequest("POST", "/sapi/o/token", sc.Auth, nil)
	token := Token{}
	_, err = sc.do(req, &token)

	return token, err
} */

func (sc *SippoClient) newRequest(method, path string, body interface{}, form url.Values) (*http.Request, error) {
	u := sc.Host + path
	log.Println("Send method ", method, " to endpoint: "+u)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		req, err := http.NewRequest(method, u, buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		return req, nil
	} else {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		req, err := http.NewRequest(method, u, nil)
		if err != nil {
			return nil, err
		}
		req.URL.RawQuery = form.Encode()
		return req, nil
	}
}

func (sc *SippoClient) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := sc.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(v)
	return resp, err
}
