from os import times
import requests
from pathlib import Path
import json
import io
import time

from requests.models import Response

import urllib3
from urllib3.exceptions import InsecureRequestWarning

# 証明書エラーは抑止
urllib3.disable_warnings(InsecureRequestWarning)

class SimpleClient:

    ENDPOINT_AUTHORIZE = "/v1/login/api"
    ENDPOINT_REFRESH = "/v1/refresh/api"

    def __init__(self, url: str, jsonPath: str, timeout=(2.0, 5.0)):

        if url.endswith("/"):
            self._url = url[:-1]
        else:
            self._url = url
        self._jsonPath = jsonPath
        self._json = ""
        self._timeout = timeout

    def Authorize(self, apikey:str):

        verified = False

        # 保存済み JWT Token をチェック
        if Path(self._jsonPath).exists():
            with open(self._jsonPath, 'r') as f:
                self._json = json.load(f)

            # token の有効期限をチェックする
            current = int(time.time())
            if current < int(self._json["expire_in"]):
                # access_token を再取得
                try:
                    self.Refresh()
                    verified = True
                except:
                    verified = False

        # API KEY を使ってサーバーに 認証をかける
        if verified == False and len(apikey) != 0:
            url_auth = self._url + SimpleClient.ENDPOINT_AUTHORIZE
            post_data = json.dumps({"token": apikey}).encode("utf-8")
            response = requests.post(url_auth, data=post_data, verify=False, timeout=self._timeout)

            if response.status_code != 200:
                raise RuntimeError("authentication failed")
            else:
                try:
                    self._json = json.loads(response.content)
                    with open(self._jsonPath, 'w') as f:
                        json.dump(self._json, f)
                    verified = True
                except Exception as e:
                    raise RuntimeError(e)

        if not verified:
            raise RuntimeError("authentication failed")

    def Request(self, url:str):

        requestUrl = url

        if not(requestUrl.startswith(self._url)):
            requestUrl = self._url + requestUrl
            
        # no auth header
        response = requests.post(url, headers=None, verify=False, timeout=self._timeout)

        if response.status_code == 401 or response.status_code == 403:
            if not "access_token" in self._json:
                self.Refresh()
            headers = {"authorization": "Bearer " + self._json["access_token"]}
            response = requests.post(url, headers=headers, verify=False, timeout=self._timeout)

            if response.status_code == 401 or response.status_code == 403:
                self.Refresh()
                headers = {"authorization": "Bearer " + self._json["access_token"]}
                response = requests.post(url, headers=headers, verify=False, timeout=self._timeout)

        if response.status_code == 200:
            try:
                return json.loads(response.content)
            except Exception as e:
                if isinstance(response.content, bytes):
                    return io.BytesIO(response.content)
                else:
                    raise RuntimeError(e)
        elif response.status_code == 401 or response.status_code == 403:
            raise RuntimeError("[Request] authentication faild")
        elif response.status_code == 404:
            raise RuntimeError("[Request] file not found")

    def Refresh(self):

        url_refresh = self._url + SimpleClient.ENDPOINT_REFRESH

        if not "refresh_token" in self._json:
            raise RuntimeError("refresh token is not found, cannnot refresh")

        post_data = json.dumps({"refresh_token": self._json["refresh_token"]}).encode("utf-8")
        response = requests.post(url_refresh, data=post_data, verify=False, timeout=self._timeout)

        if response.status_code != 200:
            raise RuntimeError("[Refresh] authentication failed")

        else:
            try:
                self._json = json.loads(response.content)
                with open(self._jsonPath, 'w') as f:
                    json.dump(self._json, f)
            except Exception as e:
                raise RuntimeError(e)
