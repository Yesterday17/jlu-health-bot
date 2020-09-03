import json
import os
import re
from logging import debug, error
from time import time, sleep

import httpx

from main import BotAgent
from ua import UserAgent

transaction = "BKSMRDK"
fields = [
    {
        "fieldZtw": "1",
        "fieldZhongtw": "",
        "fieldWantw": "",
    },
    {
        "fieldZtw": "1",
        "fieldZhongtw": "1",
        "fieldWantw": "",
    },
    {
        "fieldZtw": "1",
        "fieldZhongtw": "1",
        "fieldWantw": "1",
    }
]

proxies = None
if "REPORT_PROXY" in os.environ:
    proxies = os.environ["REPORT_PROXY"]


def report(bot: BotAgent, chat_id, user, name, type: int, max_retry=15, retry_interval=10):
    msg_login = None
    msg_form = None
    msg_submit = None
    for tries in range(max_retry):
        c = httpx.Client(trust_env=False, verify=False, http2=True, proxies=proxies, headers={
            "origin": "https://ehall.jlu.edu.cn",
            "referer": "https://ehall.jlu.edu.cn/",
            "User-Agent": UserAgent.chrome(),
        })
        try:
            if tries == 0:
                bot.send_message(chat_id, "开始进行{}……".format(name))
            else:
                bot.send_message(chat_id, "开始进行第 {}/{} 次{}重试……".format(tries, max_retry, name))
            msg_login = bot.send_message(chat_id, "登录中……")

            r = c.get('https://ehall.jlu.edu.cn/jlu_portal/login')
            pid = re.search('(?<=name="pid" value=")[a-z0-9]{8}', r.text)[0]

            postPayload = {'username': user['username'], 'password': user['password'], 'pid': pid}
            r = c.post('https://ehall.jlu.edu.cn/sso/login', data=postPayload)

            bot.delete_message(chat_id, msg_login.message_id)
            msg_login = None
            msg_form = bot.send_message(chat_id, "登录成功！正在获取表单……")

            r = c.get('https://ehall.jlu.edu.cn/infoplus/form/' + transaction + '/start')
            csrfToken = re.search('(?<=csrfToken" content=").{32}', r.text)[0]
            debug('CSRF: ' + csrfToken)

            postPayload = {'idc': transaction, 'csrfToken': csrfToken}
            r = c.post('https://ehall.jlu.edu.cn/infoplus/interface/start', data=postPayload)
            data = json.loads(r.text)
            if data["errno"] != 0:
                bot.send_message(chat_id, data["error"])
                return
            sid = re.search('(?<=form/)\\d*(?=/render)', r.text)[0]

            postPayload = {'stepId': sid, 'csrfToken': csrfToken}
            r = c.post('https://ehall.jlu.edu.cn/infoplus/interface/render', data=postPayload)
            data = json.loads(r.content)['entities'][0]
            formData = data['data']
            for u, v in fields[type].items(): formData[u] = v
            for u, v in user['fields'].items(): formData[u] = v
            formData = json.dumps(formData)
            debug('DATA: ' + formData)
            boundFields = ','.join(data['fields'].keys())
            debug('FIELDS: ' + boundFields)

            bot.delete_message(chat_id, msg_form.message_id)
            msg_form = None
            msg_submit = bot.send_message(chat_id, "表单获取成功，正在打卡……")
            postPayload = {
                'actionId': 1,
                'formData': formData,
                'nextUsers': '{}',
                'stepId': sid,
                'timestamp': int(time()),
                'boundFields': boundFields,
                'csrfToken': csrfToken
            }
            r = c.post('https://ehall.jlu.edu.cn/infoplus/interface/doAction', data=postPayload)
            debug(r.text)

            if json.loads(r.content)['ecode'] != 'SUCCEED':
                raise Exception('The server returned a non-successful status.')

            bot.delete_message(chat_id, msg_submit.message_id)
            msg_submit = None
            bot.send_message(chat_id, "打卡成功！")
            break

        except Exception as e:
            bot.send_message(chat_id, "打卡失败！" + e.__str__())
            error(e)

            if msg_login is not None:
                bot.delete_message(chat_id, msg_login.message_id)
                msg_login = None
            if msg_form is not None:
                bot.delete_message(chat_id, msg_form.message_id)
                msg_form = None
            if msg_submit is not None:
                bot.delete_message(chat_id, msg_submit.message_id)
                msg_submit = None
            sleep(retry_interval)
        finally:
            c.close()
