package main

import (
	"errors"
	"net/url"
	"time"
)

var (
	EhallLoginPendingPage, _ = url.Parse("https://ehall.jlu.edu.cn/taskcenter/workflow/index")
	EhallSSOLoginPage, _     = url.Parse("https://ehall.jlu.edu.cn/sso/login")
	EhallFormCSRFPage, _     = url.Parse("https://ehall.jlu.edu.cn/infoplus/form/BKSMRDK/start")
	EhallFormStartPage, _    = url.Parse("https://ehall.jlu.edu.cn/infoplus/interface/start")
	EhallFormRenderPage, _   = url.Parse("https://ehall.jlu.edu.cn/infoplus/interface/render")
	EhallDoActionPage, _     = url.Parse("https://ehall.jlu.edu.cn/infoplus/interface/doAction")

	EhallSystemError = errors.New("系统返回错误，将不再重试。\n错误信息：")
)

const (
	// 三测温一点名
	ReportTimeAsa ReportTime = iota
	ReportTimeHiru
	ReportTimeYoru
	ReportTimeFin

	// 一测温一点名
	ReportTimeDay
	ReportTimeNight

	// 未知
	ReportTimeUnknown
)

func (t ReportTime) Name() string {
	switch t {
	case ReportTimeAsa:
		return "早打卡"
	case ReportTimeHiru:
		return "午打卡"
	case ReportTimeYoru:
		return "晚打卡"
	case ReportTimeFin:
		return "晚点名"
	case ReportTimeDay:
		return "午前测温"
	case ReportTimeNight:
		return "晚点名"
	default:
		return "未知"
	}
}

type ReportFields struct {
	Ztw     string `json:"fieldZtw"`
	Zhongtw string `json:"fieldZhongtw"`
	Wantw   string `json:"fieldWantw"`
}

func GetReportFields(t ReportTime) ReportFields {
	switch t {
	case ReportTimeAsa:
		return ReportFields{"1", "", ""}
	case ReportTimeHiru:
		return ReportFields{"1", "1", ""}
	case ReportTimeYoru:
		return ReportFields{"1", "1", "1"}
	case ReportTimeFin:
		return ReportFields{"1", "1", "1"}
	case ReportTimeDay, ReportTimeNight:
		return ReportFields{"1", "", ""}
	default:
		return ReportFields{}
	}
}

func (f ReportFields) MergeTo(m *map[string]interface{}) {
	(*m)["fieldZtw"] = f.Ztw
	(*m)["fieldZhongtw"] = f.Zhongtw
	(*m)["fieldWantw"] = f.Wantw
}

type ReportMode uint8

const (
	ReportModeNone ReportMode = iota
	ReportMode31
	ReportMode11
	ReportModeLeaveSchool
)

func (m ReportMode) Name() string {
	switch m {
	case ReportMode31:
		return "三测温一打卡"
	case ReportMode11:
		return "一测温一打卡"
	case ReportModeLeaveSchool:
		return "本科生健康状况申报(尚未实现)"
	default:
		return "未知"
	}
}

func (m ReportMode) GetReportTime() ReportTime {
	hour := time.Now().Hour()
	switch m {
	case ReportMode31:
		switch hour {
		case 7:
			return ReportTimeAsa
		case 11:
			return ReportTimeHiru
		case 17:
			return ReportTimeYoru
		case 21:
			return ReportTimeFin
		}
	case ReportMode11:
		switch hour {
		case 6, 7, 8, 9, 10, 11:
			return ReportTimeDay
		case 21, 22:
			return ReportTimeNight
		}
	case ReportModeLeaveSchool:
		// TODO
	}
	return ReportTimeUnknown
}
