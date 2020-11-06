package main

import (
	"flag"
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func main() {
	var token, proxy, accountsPath, owner string
	var maxUsers uint
	flag.StringVar(&token, "token", "", "Telegram Bot Token")
	flag.StringVar(&proxy, "proxy", "", "http 代理地址")
	flag.StringVar(&owner, "owner", "", "Bot 拥有者 ID")
	flag.StringVar(&accountsPath, "accounts-path", "./accounts/", "存储用户帐户的路径")
	flag.UintVar(&maxUsers, "max-users", 8, "最大用户数量")
	flag.Parse()

	LoadConfig(token, proxy, owner, accountsPath, maxUsers)
	err := LoadUsers()
	if err != nil {
		log.Fatal(err)
	}

	b, err := tb.NewBot(tb.Settings{
		Token:  Config.Token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	InitCronJobs(b)

	b.Handle("/report", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		t := u.Mode.GetReportTime()
		if t == ReportTimeUnknown {
			_, _ = b.Reply(m, "不在打卡时间段内。")
			return
		}

		Report(b, t, u)
	})

	b.Handle("/mode", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		_, _ = b.Reply(m, "打卡模式："+u.Mode.Name())
	})

	b.Handle("/pause", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		u.Pause = true
		u.Save()
		_, _ = b.Reply(m, "自动打卡已暂停。")
	})

	b.Handle("/resume", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		u.Pause = false
		u.Save()
		_, _ = b.Reply(m, "自动打卡已恢复。")
	})

	b.Start()
}
