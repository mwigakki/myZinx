package ziface

// 消息管理模块

type IConnManager interface {
	// 增加连接
	Add(IConnection)
	// 删除连接
	Remove(IConnection)
	// 得到一个连接
	Get(uint32) (IConnection, error)
	// 总连接数
	Len() int
	// 终止并清楚所有连接，关闭服务器时
	Clear()
	// 连接管理的方法，每次客户端连接成功或断开连接会将会连接信息放进通道，connManage方法从通道中读取后才去添加或删除这个连接
	ConnManage()
	// 得到 与连接管理器通信的通道
	GetConnMgrChan() chan IConnection
	// 设置连接管理模块对应的server
	SetServer(IServer)
}
