package ziface

// 定义服务器接口
type IServer interface {
	// 开始
	Start()
	// 结束
	Stop()
	// 运行
	Serve()
	// 路由功能：给当前的服务注册一个路由功能，供客户端的连接使用
	AddRouter(msgID uint32, router IRouter)
	// 得到连接管理器
	GetConnMgr() IConnManager

	// 设置该server 创建连接之后自动调用 hook 函数
	SetOnConnStart(func(IConnection))
	// 调用该server 创建连接之后自动调用 hook 函数
	CallOnConnStart(IConnection)
	// 设置该server 断开连接之前自动调用 hook 函数
	SetOnConnStop(func(IConnection))
	// 调用该server 断开连接之前自动调用 hook 函数
	CallOnConnStop(IConnection)

	IsAllowFileReq() bool
}
