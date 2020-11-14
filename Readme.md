# JLU Health Bot

![image](https://user-images.githubusercontent.com/8667822/90133379-efbf8280-dda1-11ea-9182-809572e7e258.png)

为吉林大学本科生每日打卡所做的 Telegram Bot。

以 WTFPL 协议开源。

**反馈请到 [@JLU@LUG Ch.](https://t.me/jlulugch) 附属群组。**

## 声明

本 Bot 实现的功能仅为方便打卡操作，简化健康状态下的打卡过程。如有发热情况请手动进入系统准确填写个人情况！

## 命令
- [ ] /start：录入用户信息
- [x] /info：查看当前录入的用户信息（不包括密码）
- [x] /field：设置打卡过程中使用到的字段，以空格分隔 `key` 和 `value`
- [x] /del：清除已录入的用户信息
- [x] /mode: 查看当前的打卡模式
- [x] /report：手动触发打卡，会根据当前时间自动判断打卡类型
- [x] /pause：暂停自动打卡
- [x] /resume：恢复自动打卡
- [x] /reportall \[31|11\]：（管理）触发全体重打卡
- [x] /broadcast：（管理）通过 Bot 向所有用户发布消息