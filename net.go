package main

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	browser "github.com/EDDYCJY/fake-useragent"
	"golang.org/x/net/publicsuffix"
)

type HealthJar interface {
	http.CookieJar
	IsEmpty(u *url.URL) bool
}

type healthJar struct {
	jar http.CookieJar
}

func (h *healthJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	h.jar.SetCookies(u, cookies)
}

func (h *healthJar) Cookies(u *url.URL) []*http.Cookie {
	return h.jar.Cookies(u)
}

func (h *healthJar) IsEmpty(u *url.URL) bool {
	return len(h.Cookies(u)) == 0
}

func NewHealthJar() HealthJar {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	return &healthJar{jar: jar}
}

func (u *User) NeedRedirect(url *url.URL) bool {
	c := &http.Client{
		Jar: u.Jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req := u.prepareRequest("GET", url)
	resp, err := c.Do(req)
	if err != nil {
		return true
	}

	// 30x
	return resp.StatusCode > 300 && resp.StatusCode < 400
}

func (u *User) Get(url *url.URL) (string, error) {
	c := &http.Client{Jar: u.Jar, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	req := u.prepareRequest("GET", url)
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (u *User) Post(url *url.URL, body url.Values) (*http.Response, error) {
	c := &http.Client{Jar: u.Jar, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	req := u.prepareRequest("POST", url)

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	b := strings.NewReader(body.Encode())
	req.ContentLength = int64(b.Len())
	req.Body = ioutil.NopCloser(b)
	return c.Do(req)
}

func (u *User) PostB(url *url.URL, body url.Values) ([]byte, error) {
	resp, err := u.Post(url, body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (u *User) prepareRequest(method string, url *url.URL) *http.Request {
	return &http.Request{
		URL:    url,
		Method: method,
		Header: map[string][]string{
			"Origin":     {"https://ehall.jlu.edu.cn"},
			"Referer":    {"https://ehall.jlu.edu.cn/"},
			"User-Agent": {browser.Chrome()},
		},
	}
}
