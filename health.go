package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

type ReportTime uint8

func ReportAll(b *tb.Bot, m ReportMode) {
	Users.Range(func(key, value interface{}) bool {
		user := value.(*User)
		if m == user.Mode && !user.Pause {
			Report(b, m.GetReportTime(), user)
			time.Sleep(30 * time.Second)
		}
		return true
	})
}

func Report(bot *tb.Bot, t ReportTime, u *User) {
	var fields = GetReportFields(t)
	var msg *tb.Message

	for i := 0; i < u.MaxRetry; i++ {
		if i == 0 {
			_, _ = bot.Send(tb.ChatID(u.ChatId), fmt.Sprintf("开始进行%s……", t.Name()))
		} else {
			_, _ = bot.Send(tb.ChatID(u.ChatId), fmt.Sprintf("开始进行第 %d/%d 次%s重试……", i, u.MaxRetry-1, t.Name()))
		}

		msg, _ = bot.Send(tb.ChatID(u.ChatId), "登录中……")
		err := u.Login()
		if msg != nil {
			_ = bot.Delete(msg)
		}
		if err != nil {
			_, _ = bot.Send(tb.ChatID(u.ChatId), fmt.Sprintf("登录失败：%s", err.Error()))
			time.Sleep(10 * time.Second)
			continue
		}

		msg, _ = bot.Send(tb.ChatID(u.ChatId), fmt.Sprintf("登录成功！正在获取表单……"))
		body, form, err := u.GetForm()
		if msg != nil {
			_ = bot.Delete(msg)
		}
		if err != nil {
			_, _ = bot.Send(tb.ChatID(u.ChatId), fmt.Sprintf("表单获取失败：%s", err.Error()))
			if errors.Is(err, EhallSystemError) {
				break
			}

			time.Sleep(10 * time.Second)
			continue
		}

		u.MergeTo(&form)
		fields.MergeTo(&form)
		formJson, _ := json.Marshal(form)
		body.Set("formData", string(formJson))

		msg, _ = bot.Send(tb.ChatID(u.ChatId), "表单获取成功，正在打卡……")
		err = u.DoReport(body)
		if msg != nil {
			_ = bot.Delete(msg)
		}
		if err != nil {
			_, _ = bot.Send(tb.ChatID(u.ChatId), fmt.Sprintf("打卡失败：%s", err.Error()))
			if errors.Is(err, EhallSystemError) {
				break
			}

			time.Sleep(10 * time.Second)
			continue
		}

		// success
		_, _ = bot.Send(tb.ChatID(u.ChatId), "打卡成功！")
		break
	}
	if msg != nil {
		_ = bot.Delete(msg)
	}
}
