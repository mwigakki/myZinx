package znet

/*
定义应用层的消息结构体来解决粘包的问题
将请求的消息封装在message中
TLV 格式的应用层数据
+--------+-------------+-------------+--------------+--------------+
|  TYPE  |           LENGTH          |            VALUE            |
+--------+-------------+-------------+--------------+--------------+
| Msg ID |           length          |            content          |
+--------+-------------+-------------+--------------+--------------+
| 4 byte |           8 byte          |          xxxxxxxxxxxxx      |
+--------+-------------+-------------+--------------+--------------+
对于普通文本消息，head content 是没有的。
对于传输文件消息，head content 可以放文件大小，文件名等信息。
但其实发文件的过程不是只调用Pack就行，而是把MsgId，Length 准备好，content 不写。
然后用 io.Copy把文件发过去
*/
type Message struct {
	MsgId  uint32
	Length uint32
	Data   []byte
}

var MsgHeaderLength uint32 = 8 // 数据包的总的包头长度，包括type和length（固定头部长度）

func (m *Message) GetMsgId() uint32 {
	return m.MsgId
}
func (m *Message) GetLength() uint32 {
	return m.Length
}

func (m *Message) GetData() []byte {
	return m.Data
}

// 这俩需要额外的set方法
func (m *Message) SetBodyContent(buf []byte) {
	m.Data = buf
}
