## 基于 Zinx 的轻量级 TCP 文件下载服务器

> 
> 此仓库是学习刘丹冰老师的 B 站 zinx 教程一步步自己实现的，做了一些修改，记录于此，恐后用之。
> Zinx 基于 golang 的轻量级 TCP 服务器框架：[github 地址](https://github.com/aceld/zinx)
> 视频教程：[B 站刘丹冰 Aceld 的 zinx 教程](https://www.bilibili.com/video/BV1wE411d7th)

本项目在最简单的 zinx 框架的基础中加入了心跳包和文件传输的功能，客户端在包头 type 为 `MSGID_FILE_REQUEST` 的数据包中把想要下载的文件名放在包的载荷中传给 server，server 端就会将该文件传回去。

为了测试大量连接，在 client 端的代码中开启了多个 goroutine 去模拟了多个客户端。

为了适配项目需求给 client 和 server 加入了 gin 服务器，去接收传输文件或停止传输文件的命令。
