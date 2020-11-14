package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type User struct {
	ChatId   int64             `json:"chat_id"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Pause    bool              `json:"pause"`
	Fields   map[string]string `json:"fields"`

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

func (u *User) GetForm() (string, url.Values, *FormInfo, error) {
	csrf, err := u.getFormCSRF()
	if err != nil {
		return "", nil, nil, err
	}

	sid, err := u.getFormStepId(csrf)
	if err != nil {
		return csrf, nil, nil, err
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

type FormInfo struct {
	Data   map[string]interface{} `json:"data"`
	Fields map[string]FieldInfo   `json:"fields"`
	Step   StepInfo               `json:"step"`
}

type FieldInfo struct {
	Label  string `json:"label"` // display field
	Name   string `json:"name"`  // form[field.Name] = xx
	Code   string `json:"code"`  // code in suggest
	Type   string `json:"type"`  // Code
	Parent string `json:"parent"`
}

type StepInfo struct {
	InstanceId string `json:"instanceId"`
	EntryId    int64  `json:"entryId"`
	StepId     int64  `json:"flowStepId"`
}

func (u *User) getFormPage(csrf, sid string) (string, url.Values, *FormInfo, error) {
	r, err := u.PostB(EhallFormRenderPage, map[string][]string{
		"stepId":    {sid},
		"csrfToken": {csrf},
	})
	if err != nil {
		return csrf, nil, nil, err
	}
	var resp struct {
		Errno    int        `json:"errno"`
		Error    string     `json:"error"`
		Entities []FormInfo `json:"entities"`
	}
	err = json.Unmarshal(r, &resp)
	if err != nil {
		return csrf, nil, nil, err
	} else if resp.Errno != 0 {
		return csrf, nil, nil, NewEhallSystemError(resp.Error, resp.Errno)
	}
	if len(resp.Entities) == 0 {
		return csrf, nil, nil, errors.New("entities 数量为 0")
	}

	var boundFields string
	for k := range resp.Entities[0].Fields {
		boundFields += "," + k
	}
	body := url.Values{
		"actionId":    {"1"},
		"nextUsers":   {"{}"},
		"stepId":      {sid},
		"timestamp":   {strconv.FormatInt(time.Now().Unix(), 10)},
		"csrfToken":   {csrf},
		"boundFields": {boundFields[1:]},
		"formData":    {},
	}
	return csrf, body, &resp.Entities[0], nil
}

type SuggestResult struct {
	CodeId   string `json:"codeId"`
	CodeName string `json:"codeName"`
	ParentId string `json:"parentId"`
	Enabled  bool   `json:"enabled"`
}

func (u *User) SuggestField(form *FormInfo, field FieldInfo, choice, csrf string) error {
	var parent string
	if field.Parent != "" {
		p := form.Data[field.Parent]
		switch p {
		case p.(string):
			parent = p.(string)
		default:
			return fmt.Errorf("字段 %s 的 parent 字段 %s 不存在！", field.Name, field.Parent)
		}
		if parent == "" {
			return fmt.Errorf("字段 %s(%s) 的 parent 字段 %s(%s) 为空！",
				field.Label, field.Name, form.Fields[field.Parent].Label, field.Parent)
		}
	}

	var pageId = 0
	for {
		r, err := u.PostB(EhallFieldSuggestPage, map[string][]string{
			"prefix":     {""},
			"type":       {field.Type},
			"code":       {field.Code},
			"parent":     {parent},
			"isTopLevel": {"false"},
			"pageNo":     {strconv.Itoa(pageId)},
			"rand":       {strconv.FormatFloat(rand.Float64()*999, 'G', 30, 32)},
			"settings":   {"{}"},
			"csrfToken":  {csrf},
			"lang":       {"zh"},
			"instanceId": {form.Step.InstanceId},
			"stepId":     {strconv.FormatInt(form.Step.StepId, 10)},
			"entryId":    {strconv.FormatInt(form.Step.EntryId, 10)},
			"workflowId": {"null"},
		})
		if err != nil {
			return err
		}

		var result struct {
			Items []SuggestResult `json:"items"`
		}
		err = json.Unmarshal(r, &result)
		if err != nil {
			return err
		}

		if len(result.Items) == 0 {
			return fmt.Errorf("未找到 %s(%s) 为 %s 的选项！", field.Label, field.Name, choice)
		}

		for _, s := range result.Items {
			if s.CodeName == choice {
				form.Data[field.Name] = s.CodeId
				form.Data[field.Name+"_Attr"] = fmt.Sprintf(`{"_parent": "%s"}`, parent)
				return nil
			}
		}

		// Last page
		if len(result.Items) < SuggestPageItemCount {
			return fmt.Errorf("未找到 %s(%s) 为 %s 的选项！", field.Label, field.Name, choice)
		}

		// Next page
		pageId++
	}
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
		// Ignore "bot/${field}"
		if !strings.HasPrefix(k, "bot/") {
			(*m)[k] = v
		}
	}
}

func (u *User) GetBotField(key string) (string, error) {
	v, ok := u.Fields["bot/"+key]
	if !ok {
		return "", errors.New(key + " 字段不存在！")
	}

	return v, nil
}
