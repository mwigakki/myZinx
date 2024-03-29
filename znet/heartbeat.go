package znet

import (
	"time"

	"github.com/myZinx/utils"
	"github.com/myZinx/ziface"
	"github.com/sirupsen/logrus"
)

type HeartbeatChecher struct {
	//  此心跳检测器所属的连接` conn`
	conn ziface.IConnection
	// 该连接的心跳包发送间隔 `SendInterval`（每个连接的发送间隔不一样，因为如果有太多连接 100W个，所有连接同时发会引起较大的流量，随机间隔可以给网络减负）
	SendInterval time.Duration
	// 连接的上一次活跃时间 `LastActiveTime`（不仅心跳包，连接发送其他任何消息进行通信都会更新此数据）
	LastActiveTime time.Time
	// 构造心跳包的方法  `HeartbeatMsgMakeFunc`。（框架提供一个默认的，就写入一些文字。提供此属性的set方法给开发者）
	HeartbeatMsgMakeFunc func(ziface.IConnection) []byte
	// 收到心跳包的回包时的处理路由  `HeartbeatRouter`，要集成自baseRouter的。（框架提供一个默认的，就打印到日志。但提供此属性的set方法给开发者）
	HeartbeatRouter ziface.IRouter
	// 连接已经断开的信号通道（无阻塞）`ExitChan`。（因为定时发心跳包的程序肯定是另开的goroutine执行的，所以当连接断开，需要通信此gorontine结束，不要空等）
	ExitChan chan bool
	// 远程连接不存话时的处理方法  `OnRemoteNotAlive`。（框架提供一个默认的，就打印一些日志。但提供此属性的set方法给开发者）
	OnRemoteNotAlive func(ziface.IConnection)
}

func NewHeartbeatChecher(conn ziface.IConnection, sendInterval time.Duration) *HeartbeatChecher {
	return &HeartbeatChecher{
		conn:                 conn,
		SendInterval:         sendInterval,
		LastActiveTime:       time.Now(),
		HeartbeatMsgMakeFunc: heartbeatMsgMakeFunc,
		HeartbeatRouter:      &HeartbeatDefaultRouter{},
		ExitChan:             make(chan bool),
		OnRemoteNotAlive:     onRemoteNotAlive,
	}
}

// 收到心跳包的回包时的处理路由的 setRouter 方法。
func (hbc *HeartbeatChecher) SetHeartbeatRouter(r ziface.IRouter) {
	hbc.HeartbeatRouter = r
}

// 给该心跳检测器绑定对应连接的方法
func (hbc *HeartbeatChecher) BindConn(conn ziface.IConnection) {
	hbc.conn = conn
}

// 构造心跳包的方法的 set 方法
func (hbc *HeartbeatChecher) SetHeartbeatMsgMakeFunc(f func(ziface.IConnection) []byte) {
	hbc.HeartbeatMsgMakeFunc = f
}

// 构造心跳包的默认方法
func heartbeatMsgMakeFunc(conn ziface.IConnection) []byte {
	msg := "来自[服务器]的心跳包"
	logrus.Debugf("发送心跳包给客户端 %s", conn.GetTCPConnection().RemoteAddr())
	return []byte(msg)
}

// 远程连接不存话时的处理方法的 set 方法。
func (hbc *HeartbeatChecher) SetOnRemoteNotAlive(f func(ziface.IConnection)) {
	hbc.OnRemoteNotAlive = f
}

// 远程连接不存话时的 默认处理方法，默认方法就只执行 Stop().
func onRemoteNotAlive(conn ziface.IConnection) {
	conn.Stop()
}

// 更新心跳检测器活跃时间的方法
func (hbc *HeartbeatChecher) UpdateActiveTime() {
	hbc.LastActiveTime = time.Now()
}

// 该心跳检测器的Start 方法
func (hbc *HeartbeatChecher) Start() {
	go func() { // 开启此心跳检测器
		ticker := time.NewTicker(hbc.SendInterval)
		for {
			select {
			case <-ticker.C:
				if hbc.conn == nil {
					logrus.Warn("该计时器没有绑定连接!")
				} else {
					if !hbc.conn.IsAlive() {
						// 连接已经不存在了，关闭本心跳检测器即可
						hbc.OnRemoteNotAlive(hbc.conn)
					} else {
						hbc.SendHeartbeat()
					}
				}
			case <-hbc.ExitChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// 该心跳检测器的Stop 方法
func (hbc *HeartbeatChecher) Stop() {
	logrus.Debugf("关闭 连接 id = %d 的心跳检测器 \n", hbc.conn.GetConnID())
	hbc.ExitChan <- true
}

// 发送心跳包的方法 （这个方法就没有必要交给用户去自定义了）
func (hbc *HeartbeatChecher) SendHeartbeat() error {
	msg := hbc.HeartbeatMsgMakeFunc(hbc.conn)
	err := hbc.conn.SendMsg(utils.MSGID_HEARTBEAT, uint32(len(msg)), msg)
	if err != nil {
		logrus.Error("心跳发送出错：err = ", err)
		return err
	}
	return nil
}
