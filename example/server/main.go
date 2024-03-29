package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/myZinx/utils"
	"github.com/myZinx/znet"
	"github.com/sirupsen/logrus"
)

// 基于 zinx开发的服务器端应用程序
func main() {
	// logrus.SetLevel(logrus.InfoLevel)
	logrus.SetLevel(logrus.DebugLevel)
	log.SetPrefix("[服务端]：")
	// 1 创建一个server 句柄，使用 zinx 的api
	s := znet.NewServer("[MILLION TCP CONN SERVER]")
	go startGin(s)
	s.Serve()
}

// 开一个 gin 服务器去等待命令去开启或关闭 文件请求
func startGin(s *znet.Server) {
	addr := fmt.Sprintf("%s:%d", utils.GlobalObj.Host, utils.GlobalObj.ServerGinPort)
	logrus.Infof("开启 控制文件传输的接口，监听地址：%s", addr)
	// 创建一个默认路由
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	r.GET("/StartFileReq", func(ctx *gin.Context) {
		s.AllowFileReq = true
		ctx.JSON(http.StatusOK, gin.H{
			"err": "",
		})
	})
	r.GET("/StopFileReq", func(ctx *gin.Context) {
		s.AllowFileReq = false
		ctx.JSON(http.StatusOK, gin.H{
			"err": "",
		})
	})
	r.Run(addr)
}
