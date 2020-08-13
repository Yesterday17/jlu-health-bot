# JLU Health Bot

![image](https://user-images.githubusercontent.com/8667822/90133379-efbf8280-dda1-11ea-9182-809572e7e258.png)

为吉林大学本科生每日打卡所做的 Telegram Bot。

核心代码引用自 [TechCiel/jlu-health-reporter](https://github.com/TechCiel/jlu-health-reporter) ，有部分修改。

以 WTFPL 协议开源。

**已部署到服务器，正在测试中。[@ehall_jlu_bot](http://t.me/ehall_jlu_bot)**

## 部署向导

```bash
export BOT_TOKEN=<YOUR_BOT_TOKEN>
export TG_PROXY=socks5://127.0.0.1:1080

# work with jlu-http-proxy
export REPORT_PROXY=https://127.0.0.1:1081

pip install -r requirements.txt
python main.py
```

## 命令
- /start：录入/修改用户信息
- /info：查看当前录入的用户信息（不包括密码）