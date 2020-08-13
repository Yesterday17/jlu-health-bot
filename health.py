import re
import json
from time import time, sleep
from logging import debug, error
import urllib3
import requests

from telebot import TeleBot

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
config = {
    "transaction": "BKSMRDK",
    "fields": {
        "fieldZtw": "1",
        "fieldZhongtw": "1",
        "fieldWantw": "1",
    }
}


def report(bot: TeleBot, chat_id, user, name, max_retry=15, retry_interval=10):
    for tries in range(max_retry):
        try:
            if tries == 0:
                bot.send_message(chat_id, "开始进行{}…".format(name))
            else:
                bot.send_message(chat_id, "开始第 {}/{} 次{}重试".format(tries + 1, max_retry, name))
            msg_login = bot.send_message(chat_id, "登录中……")

            s = requests.Session()
            s.headers.update({'Referer': 'https://ehall.jlu.edu.cn/',
                              'User-Agent': 'ozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36'})
            s.verify = False

            r = s.get('https://ehall.jlu.edu.cn/jlu_portal/login')
            pid = re.search('(?<=name="pid" value=")[a-z0-9]{8}', r.text)[0]
            debug('PID: ' + pid)

            postPayload = {'username': user['username'], 'password': user['password'], 'pid': pid}
            r = s.post('https://ehall.jlu.edu.cn/sso/login', data=postPayload)

            bot.delete_message(chat_id, msg_login.message_id)
            msg_form = bot.send_message(chat_id, "登录成功！正在获取表单……")

            r = s.get('https://ehall.jlu.edu.cn/infoplus/form/' + config['transaction'] + '/start')
            csrfToken = re.search('(?<=csrfToken" content=").{32}', r.text)[0]
            debug('CSRF: ' + csrfToken)

            postPayload = {'idc': config['transaction'], 'csrfToken': csrfToken}
            r = s.post('https://ehall.jlu.edu.cn/infoplus/interface/start', data=postPayload)
            if r.text[0] == '{':
                data = json.loads(r.text)
                bot.send_message(chat_id, data["error"])
                return
            sid = re.search('(?<=form/)\\d*(?=/render)', r.text)[0]
            debug('Step ID: ' + sid)

            postPayload = {'stepId': sid, 'csrfToken': csrfToken}
            r = s.post('https://ehall.jlu.edu.cn/infoplus/interface/render', data=postPayload)
            data = json.loads(r.content)['entities'][0]
            payload_1 = data['data']
            for u, v in config['fields'].items(): payload_1[u] = v
            for u, v in user['fields'].items(): payload_1[u] = v
            payload_1 = json.dumps(payload_1)
            debug('DATA: ' + payload_1)
            payload_2 = ','.join(data['fields'].keys())
            debug('FIELDS: ' + payload_2)

            bot.delete_message(chat_id, msg_form.message_id)
            msg_submit = bot.send_message(chat_id, "表单获取成功，正在打卡……")
            postPayload = {
                'actionId': 1,
                'formData': payload_1,
                'nextUsers': '{}',
                'stepId': sid,
                'timestamp': int(time()),
                'boundFields': payload_2,
                'csrfToken': csrfToken
            }
            r = s.post('https://ehall.jlu.edu.cn/infoplus/interface/doAction', data=postPayload)
            debug(r.text)

            if json.loads(r.content)['ecode'] != 'SUCCEED':
                raise Exception('The server returned a non-successful status.')

            bot.delete_message(chat_id, msg_submit.message_id)
            bot.send_message(chat_id, "打卡成功！")
            break

        except Exception as e:
            bot.send_message(chat_id, "打卡失败！" + e.__str__())
            error(e)
            sleep(retry_interval)
