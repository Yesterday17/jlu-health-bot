package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func main() {
	var token, proxy, accountsPath, owner string
	var maxUsers uint
	flag.StringVar(&token, "token", "", "Telegram Bot Token")
	flag.StringVar(&proxy, "proxy", "", "http 代理地址")
	flag.StringVar(&owner, "owner", "", "Bot 拥有者用户名")
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

		go Report(b, t, u)
	})

	b.Handle("/field", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		err := u.SetBotField(m.Payload)
		if err != nil {
			_, _ = b.Reply(m, err.Error())
		}

		_, _ = b.Reply(m, "设置完成。")
	})

	b.Handle("/botfield", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		err := u.SetBotField("bot/" + m.Payload)
		if err != nil {
			_, _ = b.Reply(m, err.Error())
		}

		_, _ = b.Reply(m, "设置完成。")
	})

	b.Handle("/info", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		_, _ = b.Reply(m, fmt.Sprintf("用户名：%s\n"+
			"密码：[隐藏]\n"+
			"校区：%s\n"+
			"寝室楼号：%s\n"+
			"寝室号：%s\n",
			u.Username, u.Fields["fieldSQxq"], u.Fields["fieldSQgyl"], u.Fields["fieldSQqsh"],
		))
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

	b.Handle("/del", func(m *tb.Message) {
		user, ok := Users.Load(m.Chat.ID)
		if !ok {
			return
		}

		u := user.(*User)
		u.Remove()
		_, _ = b.Reply(m, "用户信息已删除。")
	})

	b.Handle("/reportall", func(m *tb.Message) {
		if m.Sender.Username != Config.Owner {
			return
		}

		switch m.Payload {
		case "31":
			go ReportAll(b, ReportMode31, "管理员触发了一次全体打卡。")
		case "11":
			go ReportAll(b, ReportMode11, "管理员触发了一次全体打卡。")
		default:
			return
		}

		_, _ = b.Reply(m, "已触发一次全体打卡。")
	})

	b.Handle("/broadcast", func(m *tb.Message) {
		if m.Sender.Username != Config.Owner {
			return
		}

		go Users.Range(func(key, value interface{}) bool {
			u := value.(*User)
			_, _ = b.Send(tb.ChatID(u.ChatId), m.Payload)
			return true
		})
	})

	b.Start()
}
