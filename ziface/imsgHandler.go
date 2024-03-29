package ziface

// 此接口要放在在server 中
type IMessageHandler interface {
	// 调度，执行对应的router消息处理方法
	DoMsgHandler(IRequest)
	// 给server添加具体的router 处理逻辑
	AddRouter(msgID uint32, router IRouter)
}
