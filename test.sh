#/bin/bash

curl 127.0.0.1:8991/StartFileReq                 # 服务端的
curl 127.0.0.1:8992/StartFileReq?amount=2        # 客户端的
echo ""
sleep 12s

curl 127.0.0.1:8991/StopFileReq          # 服务端的
curl 127.0.0.1:8992/StopFileReq          # 客户端的
echo ""