from random import randint


class UserAgent:
    @staticmethod
    def chrome():
        return "Mozilla/5.0 (Windows NT 10.0; Win64; x64)AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{}.{}.{}.{} Safari/537.36".format(
            randint(66, 86),
            0,
            randint(3000, 5000),
            0,
        )
