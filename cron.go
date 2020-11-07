package main

import (
	"github.com/robfig/cron/v3"
	tb "gopkg.in/tucnak/telebot.v2"
)

func InitCronJobs(b *tb.Bot) {
	c := cron.New()

	// 3-1: 07:10, 11:10, 17:10, 21:10
	_, _ = c.AddFunc("CRON_TZ=Asia/Shanghai 10 07,11,17,21 * * *", func() {
		ReportAll(b, ReportMode31, "")
	})

	// 1-1: 08:30, 21:30
	_, _ = c.AddFunc("CRON_TZ=Asia/Shanghai 30 08,21 * * *", func() {
		ReportAll(b, ReportMode11, "")
	})

	// TODO: ReportModeLeaveSchool
	c.Start()
}
