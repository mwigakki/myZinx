package ziface

type IHeartBeatChecker interface {
	// 给该心跳检测器绑定对应连接的方法
	BindConn(IConnection)
	// 构造心跳包的方法的 set 方法
	SetHeartbeatMsgMakeFunc(func(IConnection) []byte)
	// 远程连接不存话时的处理方法的 set 方法。
	SetOnRemoteNotAlive(func(IConnection))
	// 收到心跳包的回包时的处理路由的 setRouter 方法。
	SetHeartbeatRouter(IRouter)
	// 它们三个的默认的执行方法，不需要交给接口拓展
	// 该心跳检测器的Start 方法
	Start()
	// 该心跳检测器的Stop 方法
	Stop()
	// 发送心跳包的方法  。（这个方法就没有必要交给用户去自定义了）
	SendHeartbeat() error
	// 更新心跳检测器活跃时间的方法
	UpdateActiveTime()
}
