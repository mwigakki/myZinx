package ziface

import "net"

// 定义连接 模块的抽象层

type IConnection interface {
	// 启动连接，让当前连接准备开始工作
	Start()
	// 关闭连接。结束连接的工作
	Stop()
	// 获取当前连接绑定的 conn
	GetTCPConnection() *net.TCPConn
	// 获取连接ID
	GetConnID() uint32
	// 获取客户端的TCP状态 IP和Port
	RemoteAddr() net.Addr
	// 发送数据
	SendMsg(uint32, uint32, []byte) error
	// 绑定心跳检测器
	BindHeartBeatChecker(IHeartBeatChecker)

	// 设置连接属性
	SetProperty(string, any)
	// 获取连接属性
	GetProperty(string) (any, error)
	// 删除连接属性
	RemoveProperty(string)
	// 是否还存活
	IsAlive() bool
	// 虽然这样耦合太严重了，但为了实现在服务器关闭正在传输的文件，必须把server 加到每个连接中
	SetServer(IServer)
	GetServer() IServer
}

// 定义一个处理连接所绑定的业务的方法
// 通过第一个参数得到连接的信息，通过第二三个参数得到处理的数据
// 每一个IConnection 的实现都应该有这个成员属性
type HandleFunc func(*net.TCPConn, []byte, int) error // 它的意义在于将读数据的方法与处理数据的方法进行解耦以及抽象
// 此函数已停用
