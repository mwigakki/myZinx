package ziface

// 将请求的消息封装在message中
type IMessage interface {
	GetMsgId() uint32
	GetLength() uint32
	GetData() []byte
	// 这俩需要额外的set方法
	SetBodyContent(buf []byte)
}
