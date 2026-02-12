# https://developer.cisco.com/docs/nexus-dashboard-fabric-controller/latest/api-reference-lan/
import json

import requests
import urllib3

urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

IP = ""
USR = ""
PWD = ""

login_payload = {"userName": USR, "userPasswd": PWD, "domain": "DefaultAuth"}


class Client:
    def __init__(self, ip, usr, pwd):
        self.ip = ip
        self.usr = usr
        self.pwd = pwd
        self.session = requests.Session()

    def login(self):
        login_payload = {
            "userName": self.usr,
            "userPasswd": self.pwd,
            "domain": "DefaultAuth",
        }
        self.session.post(f"https://{self.ip}/login", json=login_payload, verify=False)

    def get(self, endpoint):
        res = self.session.get(f"https://{self.ip}{endpoint}", verify=False)
        return res.json()

    def get_log(self, endpoint):
        res = self.get(endpoint)
        print(json.dumps(res, indent=2))


client = Client(IP, USR, PWD)
client.login()
client.get_log("/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/fabrics")
client.get_log("/appcenter/cisco/ndfc/api/v1/fm/about/version")
client.get_log("/appcenter/cisco/ndfc/api/v1/lan-fabric/rest/control/switches/overview")
