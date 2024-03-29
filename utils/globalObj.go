package utils

import (
	"encoding/json"
	"os"

	"github.com/myZinx/ziface"
)

/*
	存储一切zinx 框架使用的全局参数，供其他模块使用
	一些参数应该通过 zinx.json由用户去配置
*/

// 消息ID 定义。不同消息的默认处理路由在router.go 中定义，同时在server.go中newServer的时候给默认路由加入
const (
	MSGID_HEARTBEAT    = 0
	MSGID_GENERAL_MSG  = 1
	MSGID_PING         = 2
	MSGID_FILE_REQUEST = 3
	MSGID_FILE_RESPOND = 4
)

type GlobalObject struct {
	// server 的配置
	TcpServer     ziface.IServer // 当前zinx 全局的server对象
	Host          string         // 当前服务器监听的IP
	Port          int            // 当前服务器监听的tcp 端口
	ServerGinPort int            // Server 使用 gin 部署额外的服务
	ClientGinPort int            // client 使用 gin 部署额外的服务
	Name          string         // 当前服务器名称
	// zinx 的配置
	Version            string // 当前 zinx 版本号
	MaxConn            int    // 当前服务器主机允许的最大连接数
	MaxPackageSize     uint32 // 当前框架数据包的最大值
	MaxFilePackageSize uint32 // 当前框架中发送文件数据包的最大值
	// 心跳检测器配置,定义全局的心跳包发送间隔
	// （设定最大值和最小值，具体连接的发送间隔去其中的随机数。因为设定唯一值会使所有连接同时发心跳包，当连接过多时会导致突发流量）
	MinSendInterval int
	MaxSendInterval int     // 心跳包发送最大间隔  以秒为单位
	MinWaitTimt     int     // 文件传输中的最小等待时间，与下面两个不会冲突，它是加在随机出来的时间上的
	MeanWaitTimt    float64 // 文件传输中的平均等待时间
	MaxWaitTimt     int     // 文件传输中的最长等待时间

	FileNames []string // 认为客户端是知道所有文件名和文件大小的
	FileSizes []int64

	MsgIdDesc map[uint32]string // 不同消息id的描述
}

// 定义全局对外的globalObj 对象
var GlobalObj *GlobalObject

// 提供init方法 初始化对象
func init() {
	GlobalObj = &GlobalObject{ // 现在配置一些默认值
		Name:               "Zinx Server App",
		Host:               "127.0.0.1",
		Port:               8990, // TCP 服务器断开
		ServerGinPort:      8991, // 服务器程序接收 文件传输命令 的服务器端口
		ClientGinPort:      8992, // 客户端程序接收 文件传输命令 的服务器端口
		Version:            "V1.0",
		MaxConn:            60000,
		MaxPackageSize:     1024,
		MaxFilePackageSize: 1 << 15, // 暂定32KB，本机器的tcp发送缓存大小为200KB；修改此处可以明显改变文件传输速度
		MinSendInterval:    100,     // 心跳包发送时间间隔设置
		MaxSendInterval:    200,
		MinWaitTimt:        2, // 最小等待时间是直接加在下面两个值算出来的随机等待时间上的
		MeanWaitTimt:       30,
		MaxWaitTimt:        60,
		FileNames:          []string{"bigfile.mp4", "v1_hpzg.mp4", "v2_hpzg.mp4", "v3_4k.mp4", "v4_4k.mp4"},
		FileSizes:          []int64{2147479552, 5949948, 75313964, 124565867, 324563298},
	}
	GlobalObj.MsgIdDesc = map[uint32]string{
		MSGID_HEARTBEAT:    "HEARTBEAT",
		MSGID_GENERAL_MSG:  "GENERAL_MSG",
		MSGID_PING:         "PING",
		MSGID_FILE_REQUEST: "FILE_REQUEST",
		MSGID_FILE_RESPOND: "FILE_RESPOND",
	}
	// GlobalObj.Reload("")
}

// 暂时不要用
func (g *GlobalObject) Reload(filePath string) {
	if filePath == "" {
		filePath = "conf/zinx.json"
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &GlobalObj) // 会自动将data 接触成的数据装入 对象中
	if err != nil {
		panic(err)
	}
}
