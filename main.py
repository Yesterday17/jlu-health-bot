import json
import os
import glob
import schedule
import threading
import time
import telebot

import health
from telebot.types import Message
from telebot import apihelper

if os.environ.__contains__("TG_PROXY"):
    apihelper.proxy = {"https": os.environ["TG_PROXY"]}

os.chdir(os.path.dirname(os.path.realpath(__file__)))
bot = telebot.TeleBot(os.environ["BOT_TOKEN"], parse_mode=None, threaded=True)
user_dict = {}


def load_config():
    for filename in glob.glob("./accounts/*.json"):
        data = json.load(open(filename))
        user_dict[data["chat_id"]] = data


def save_user_config(chat_id):
    info = user_dict[chat_id]
    json.dump(info, open("./accounts/{}.json".format(chat_id), "w"))


@bot.message_handler(commands=["info"])
def info(message: Message):
    chat_id = message.chat.id
    if chat_id not in user_dict:
        bot.reply_to(message, "无用户信息！")
    else:
        user = user_dict[chat_id]
        bot.reply_to(message, "用户名：{}\n"
                              "密码：[隐藏]\n"
                              "校区：{}\n"
                              "寝室楼号：{}\n"
                              "寝室号：{}".format(user["username"], user["fields"]["fieldSQxq"],
                                              user["fields"]["fieldSQgyl"], user["fields"]["fieldSQqsh"]))


@bot.message_handler(commands=["start"])
def start(message: Message):
    msg = bot.reply_to(message, "欢迎使用本科生每日打卡 Bot。\n"
                                "为正常使用该 Bot,请按照提示的步骤进行信息填写。\n"
                                "\n"
                                "请输入用户名：")
    bot.register_next_step_handler(msg, step_username)


def step_username(message: Message):
    try:
        chat_id = message.chat.id
        user_dict[chat_id] = {
            "chat_id": chat_id,
            "username": message.text,
            "password": "",
            "fields": {
                "fieldSQxq": "",
                "fieldSQgyl": "",
                "fieldSQqsh": "",
            }
        }
        msg = bot.reply_to(message, "请输入密码：")
        bot.register_next_step_handler(msg, step_password)
    except Exception as e:
        bot.reply_to(message, e.__str__())


def step_password(message: Message):
    try:
        chat_id = message.chat.id
        user_dict[chat_id]["password"] = message.text

        msg = bot.reply_to(message, "请输入校区号（1 为中心校区）：")

        # delete password message immediately
        bot.delete_message(chat_id, message.message_id)
        bot.register_next_step_handler(msg, step_district)
    except Exception as e:
        bot.reply_to(message, e.__str__())
        bot.delete_message(chat_id, message.message_id)


def step_district(message: Message):
    try:
        chat_id = message.chat.id
        user_dict[chat_id]["fields"]["fieldSQxq"] = message.text

        msg = bot.reply_to(message, "请输入寝室楼号（1 为北苑一公寓）：")
        bot.register_next_step_handler(msg, step_dormitory)
    except Exception as e:
        bot.reply_to(message, e.__str__())


def step_dormitory(message: Message):
    try:
        chat_id = message.chat.id
        user_dict[chat_id]["fields"]["fieldSQgyl"] = message.text

        msg = bot.reply_to(message, "请输入寝室号：")
        bot.register_next_step_handler(msg, step_room)
    except Exception as e:
        bot.reply_to(message, e.__str__())


def step_room(message: Message):
    try:
        chat_id = message.chat.id
        user_dict[chat_id]["fields"]["fieldSQqsh"] = message.text

        save_user_config(chat_id)
        bot.send_message(chat_id, "信息填写完成！请使用 /info 查看信息。\n"
                                  "如有信息错误，请使用 /start 重新开始本步骤。")
    except Exception as e:
        bot.reply_to(message, e.__str__())


def rpt(name):
    def report():
        for chat_id in user_dict:
            health.report(bot, chat_id, user_dict[chat_id], name)

    return report


schedule.every().day.at("07:01").do(rpt("早打卡"))
schedule.every().day.at("11:01").do(rpt("午打卡"))
schedule.every().day.at("18:01").do(rpt("晚打卡"))
schedule.every().day.at("21:01").do(rpt("晚点名"))


def schedule_continuous_run(interval=1):
    cease_continuous_run = threading.Event()

    class ScheduleThread(threading.Thread):
        @classmethod
        def run(cls):
            # debug
            # time.sleep(2)
            # rpt("测试打卡")()
            while not cease_continuous_run.is_set():
                schedule.run_pending()
                time.sleep(interval)

    continuous_thread = ScheduleThread()
    continuous_thread.start()


if __name__ == '__main__':
    load_config()
    schedule_continuous_run(10)
    bot.polling()
