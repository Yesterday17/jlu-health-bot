package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

type HealthConfig struct {
	Token string `json:"token"`
	Proxy string `json:"proxy"`

	Owner        string `json:"owner"`
	AccountsPath string `json:"accounts_path"`
	MaxUsers     uint   `json:"max_users"`
}

var Config HealthConfig

func LoadConfig(token, proxy, owner, accountsPath string, maxUsers uint) {
	var config HealthConfig
	data, err := ioutil.ReadFile("config.json")
	if err == nil {
		err = json.Unmarshal(data, &config)
		if err != nil {
			log.Println("配置文件解析错误，将使用参数替代。")
		} else {
			if token == "" {
				token = config.Token
			}
			if proxy == "" {
				proxy = config.Proxy
			}
			if owner == "" {
				owner = config.Owner
			}
			if accountsPath == "" {
				accountsPath = config.AccountsPath
			}
			if maxUsers == 0 {
				maxUsers = config.MaxUsers
			}
		}
	}

	if token == "" {
		log.Fatal("未提供 Telegram Bot Token！")
	}

	if proxy != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			log.Panic(err)
		}
		http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	}

	_, err = os.Stat(accountsPath)
	if os.IsNotExist(err) {
		log.Panic("用户帐户目录不存在！")
	} else if err != nil {
		log.Panic(err.Error())
	}

	if owner == "" {
		log.Println("未设置拥有者 ID，管理功能将不可用。")
	}

	Config.Token = token
	Config.Proxy = proxy
	Config.AccountsPath = accountsPath
	Config.Owner = owner
	Config.MaxUsers = maxUsers

	err = SaveConfig()
	if err != nil {
		log.Fatal(err)
	}
}

func SaveConfig() error {
	data, err := json.Marshal(Config)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile("./config.json", data, 0755)
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
