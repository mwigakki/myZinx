package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/myZinx/utils"
	"github.com/sirupsen/logrus"
)

var (
	// server_ip = flag.String("server_ip", "192.168.199.164", "server IP")
	// client_ip = flag.String("client_ip", "192.168.199.162", "client IP")
	server_ip   = flag.String("server_ip", utils.GlobalObj.Host, "server IP")
	client_ip   = flag.String("client_ip", "127.0.0.1", "client IP")
	connections = flag.Int("conn", 3, "number of tcp connections")
	lambda      = flag.Float64("lambda", 1/utils.GlobalObj.MeanWaitTimt, "lambda in neg exp") // 平均等待时间的倒数是 lambda
	maxWaitTime = flag.Int("mwt", utils.GlobalObj.MaxWaitTimt, "max Wait Time")
	cdf         []float64      // 根据上述两个值算得的负指数分布的cdf，放在全局变量这儿以供其他地方算随机等待时间
	wg          sync.WaitGroup // 等待组
)

type ClientConnMgr struct {
	beginPort         int           // 客户端端口起点
	conns             []*ClientConn // 所有的连接
	fileReqConnAmount int           // 开启了文件请求功能的连接数，默认是conns[:amount] 它们开启了
	cId               uint32        // 每来一个连接给分配一个cId使用原子方法进行自增
}

func main() {
	// logrus.SetLevel(logrus.InfoLevel)
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("CLIENT start ...")
	flag.Parse()
	// 生成负指数分布的cdf，用于请求文件后到下一次再次请求之间的随机等待时间
	cdf = generateNegExpDistributionCDF(*lambda, *maxWaitTime)
	cmgr := &ClientConnMgr{
		beginPort:         10000, // 客户端端口起点
		conns:             make([]*ClientConn, 0, *connections),
		fileReqConnAmount: 0,
	}
	for i := 0; i < *connections; i++ {
		client_port := cmgr.beginPort + i
		localAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", *client_ip, client_port)) // 客户端端口从 clientPortBegin 累加
		if err != nil {
			logrus.Error("ResolveTCPAddr 解析本地tcp地址 出错")
			return
		}
		conn, err := net.DialTCP("tcp", localAddr, &net.TCPAddr{IP: net.ParseIP(*server_ip), Port: utils.GlobalObj.Port}) // 第二个参数写客户端地址，第三个参数写服务器地址
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && strings.Contains(opErr.Error(), "address already in use") {
				logrus.Warnf("端口 %d 被占用，跳过此端口。", client_port)
				*connections++ // 增加一个值以保证跳过此端口后，最终的连接总数还是程序开头设定的值
				continue
			}
			logrus.Error("连接服务器 Error：", i, err) // 其他需要处理的异常
			return
		}
		newcId := atomic.AddUint32(&cmgr.cId, 1)
		c := newClientConn(conn, newcId)
		cmgr.conns = append(cmgr.conns, c)
		go c.clientReader()
		go c.clientWriter()
		wg.Add(2)
		go c.StartFileRequest() // 每个连接的文件请求的goroutine一直是开着的，但默认是阻塞的，通过 给fileReqSignal 通道传值来开启
	}
	go startGin(cmgr) // 开启 控制文件传输的接口
	logrus.Infof("完成初始化 %d 条连接，最大端口号是：%d", len(cmgr.conns), *connections+cmgr.beginPort)
	wg.Wait() // 等待其他线程结束
	logrus.Infof("所有连接均已关闭，结束程序")
}

func startGin(cmgr *ClientConnMgr) {
	addr := fmt.Sprintf("%s:%d", *server_ip, utils.GlobalObj.ClientGinPort)
	logrus.Infof("开启 控制文件传输的接口，监听地址：%s", addr)
	// 创建一个默认路由
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	r.GET("/StartFileReq", func(ctx *gin.Context) {
		amount, err := strconv.Atoi(ctx.Query("amount")) // 让amount 个连接去开始请求文件
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"err": "传入的数量无法解析，请检查: ip:port/StartFileReq?amount=xxx",
			})
		} else {
			for i := 0; i < amount; i++ {
				if !cmgr.conns[i].isFileRequesting { // 如果已经开启了文件请求就不要再给通道传值了
					cmgr.conns[i].fileReqSignal <- true
				}
			}
			cmgr.fileReqConnAmount = amount // 更新开启了文件请求连接的数量
			ctx.JSON(http.StatusOK, gin.H{
				"err": "",
			})
		}
	})
	r.GET("/StopFileReq", func(ctx *gin.Context) {
		if cmgr.fileReqConnAmount == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"err": "此时没有任何连接开启文件请求",
			})
			return
		}
		for i := 0; i < cmgr.fileReqConnAmount; i++ {
			cmgr.conns[i].fileReqSignal <- false
		}
		cmgr.fileReqConnAmount = 0 // 更新开启了文件请求连接的数量
		ctx.JSON(http.StatusOK, gin.H{
			"err": "",
		})
	})
	r.Run(addr)
}

func generateNegExpDistributionCDF(lambda float64, maxWaitTime int) []float64 {
	/*
		我们认为每个连接请求文件的等待时间间隔服从负指数分布
		泊松过程的等待时间间隔服从负指数分布。负指数分布是连续概率分布，无法计算单个点的概率。
		随机变量X 服从参数为λ的负指数分布，则记为 X ~ Exp(λ)，其均值为 1/λ，
		但我们先计算CDF，即F(X)=P(X<=x)。然后将CDF划分为离散的区间，
		再生成随机数看落在哪个区间里，取区间下标即可按负指数分布生成的随机数
		在此问题中，CDF = {P(X<=1), P(X<=2), P(X<=3),... , P(X<=maxWaitTime) }={F(1), F(2), F(3), F(4),...， F(maxWaitTime)}
		其中，P(X<=x) = F(x) = f(x)在0到x上的积分。
		f(x) = λe^(-λx) ; F(x) = 1-e^(-λx)  ,x只取非负值,
	*/
	cdf := make([]float64, maxWaitTime)
	for i := 1; i < maxWaitTime; i++ { // 让第一个数为0
		cdf[i] = 1 - math.Pow(math.E, -float64(i)*lambda)
	}
	// 计算出的cdf可能由于lambda和mwt的设置导致成这个样子 [0,0.03,..., 0.71, 0.72]
	// 需要将cdf归一化到 0~1 之间
	for i := 1; i < maxWaitTime; i++ {
		cdf[i] = cdf[i] / cdf[maxWaitTime-1]
	}
	return cdf
}
