package ziface

// 路由抽象接口，其中的数据都是IRequest
type IRouter interface {
	// 在处理 conn 业务之前的钩子方法
	PreHandle(IRequest)
	// 处理 conn 业务的主方法
	Handle(IRequest)
	// 在处理 conn 业务之后的钩子方法
	PostHandle(IRequest)
}
