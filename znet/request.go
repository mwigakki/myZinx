package znet

import "github.com/myZinx/ziface"

type Request struct {
	//  当前连接
	conn ziface.IConnection
	//  当前请求的消息数据
	msg ziface.IMessage
}

// 得到当前连接
func (r *Request) GetConnection() ziface.IConnection {
	return r.conn
}

// 得到当前请求的消息数据
func (r *Request) GetData() []byte {
	return r.msg.GetData()
}
func (r *Request) GetMsgId() uint32 {
	return r.msg.GetMsgId()
}
func (r *Request) GetMsgLen() uint32 {
	// 返回消息的内容长度
	return uint32(len(r.msg.GetData()))
}
