package main

import (
	"io/ioutil"
	"path"
	"strings"
	"sync"
)

// sync map[int64]*User
var Users sync.Map

func LoadUsers() error {
	info, err := ioutil.ReadDir(Config.AccountsPath)
	if err != nil {
		return err
	}

	for _, i := range info {
		n := i.Name()
		if !i.IsDir() && strings.HasSuffix(n, ".json") {
			p := path.Join(Config.AccountsPath, n)
			u, err := NewUser(p)
			if err != nil {
				return err
			}
			Users.Store(u.ChatId, u)
		}
	}
	return nil
}
