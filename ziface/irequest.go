package ziface

type IRequest interface {
	// 得到当前连接
	GetConnection() IConnection
	// 得到当前请求的消息数据
	GetData() []byte

	GetMsgId() uint32
	GetMsgLen() uint32
}
