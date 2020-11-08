package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type User struct {
	ChatId   int64                  `json:"chat_id"`
	Username string                 `json:"username"`
	Password string                 `json:"password"`
	Pause    bool                   `json:"pause"`
	Fields   map[string]interface{} `json:"fields"`

	Mode     ReportMode `json:"mode"`
	MaxRetry int        `json:"max_retry"`

	Jar HealthJar `json:"-"`
}

func NewUser(path string) (*User, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var u User
	err = json.Unmarshal(data, &u)
	if err != nil {
		return nil, err
	}

	u.Jar = NewHealthJar()

	if u.MaxRetry == 0 {
		u.MaxRetry = 8
	}
	// FIXME: Remove mode modification
	if u.Mode == ReportModeNone {
		u.Mode = ReportMode11
	}

	u.Save()
	return &u, nil
}

func (u *User) Save() {
	data, _ := json.Marshal(*u)

	p := path.Join(Config.AccountsPath, strconv.FormatInt(u.ChatId, 10)+".json")
	_ = ioutil.WriteFile(p, data, 0755)
}

func (u *User) Remove() {
	// Remove in map
	Users.Delete(u.ChatId)

	// Remove json
	p := path.Join(Config.AccountsPath, strconv.FormatInt(u.ChatId, 10)+".json")
	_ = os.Remove(p)
}

func (u *User) Login() error {
	r, err := u.Get(EhallLoginPendingPage)
	if err != nil {
		return err
	} else if !strings.Contains(r, "统一身份认证") {
		// cookie in jar
		return nil
	}

	regexPid, _ := regexp.Compile("(?:name=\"pid\" value=\")([a-z0-9]{8})")
	match := regexPid.FindStringSubmatch(r)
	if match == nil {
		return errors.New("无法获取登录 PID")
	}
	pid := match[1]

	resp, err := u.Post(EhallSSOLoginPage, map[string][]string{
		"username": {u.Username},
		"password": {u.Password},
		"pid":      {pid},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("登录状态 %d", resp.StatusCode)
	}

	return nil
}

func (u *User) GetForm() (url.Values, map[string]interface{}, error) {
	csrf, err := u.getFormCSRF()
	if err != nil {
		return nil, nil, err
	}

	sid, err := u.getFormStepId(csrf)
	if err != nil {
		return nil, nil, err
	}

	return u.getFormPage(csrf, sid)
}

func (u *User) getFormCSRF() (string, error) {
	r, err := u.Get(EhallFormCSRFPage)
	if err != nil {
		return "", err
	}

	regexCsrf, _ := regexp.Compile("(?:csrfToken\" content=\")(.{32})")
	match := regexCsrf.FindStringSubmatch(r)
	if match == nil {
		return "", errors.New("无法获取 CSRF Token")
	}
	return match[1], nil
}

func (u *User) getFormStepId(csrf string) (string, error) {
	r, err := u.PostB(EhallFormStartPage, map[string][]string{
		"idc":       {"BKSMRDK"},
		"csrfToken": {csrf},
	})
	if err != nil {
		return "", err
	}

	var j struct {
		Errno int    `json:"errno"`
		Error string `json:"error"`
	}
	err = json.Unmarshal(r, &j)
	if err != nil {
		return "", err
	} else if j.Errno != 0 {
		return "", NewEhallSystemError(j.Error, j.Errno)
	}

	regexSid, _ := regexp.Compile("(?:form/)(\\d*)(?:/render)")
	match := regexSid.FindStringSubmatch(string(r))
	if match == nil {
		return "", errors.New("无法获取 Step Id")
	}

	return match[1], nil
}

func (u *User) getFormPage(csrf, sid string) (url.Values, map[string]interface{}, error) {
	r, err := u.PostB(EhallFormRenderPage, map[string][]string{
		"stepId":    {sid},
		"csrfToken": {csrf},
	})
	if err != nil {
		return nil, nil, err
	}
	var resp struct {
		Errno    int    `json:"errno"`
		Error    string `json:"error"`
		Entities []struct {
			Data   map[string]interface{} `json:"data"`
			Fields map[string]interface{} `json:"fields"`
		} `json:"entities"`
	}
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return nil, nil, err
	} else if resp.Errno != 0 {
		return nil, nil, NewEhallSystemError(resp.Error, resp.Errno)
	}
	if len(resp.Entities) == 0 {
		return nil, nil, errors.New("entities 数量为 0")
	}

	var boundFields string
	for k := range resp.Entities[0].Fields {
		boundFields += "," + k
	}
	form := url.Values{
		"actionId":    {"1"},
		"nextUsers":   {"{}"},
		"stepId":      {sid},
		"timestamp":   {strconv.FormatInt(time.Now().Unix(), 10)},
		"csrfToken":   {csrf},
		"boundFields": {boundFields[1:]},
		"formData":    {},
	}
	return form, resp.Entities[0].Data, nil
}

func (u *User) DoReport(body url.Values) error {
	r, err := u.PostB(EhallDoActionPage, body)
	if err != nil {
		return err
	}

	var resp struct {
		Errno int    `json:"errno"`
		Error string `json:"error"`
	}
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return err
	} else if resp.Errno != 0 {
		return NewEhallSystemError(resp.Error, resp.Errno)
	}

	return nil
}

func (u *User) MergeTo(m *map[string]interface{}) {
	for k, v := range u.Fields {
		(*m)[k] = v
	}
}
