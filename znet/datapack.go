package znet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/myZinx/utils"
	"github.com/myZinx/ziface"
	"github.com/sirupsen/logrus"
)

/**
封包，拆包 模块，用于解决tcp 粘包问题
TLV 格式， [TYPE | HEADER LENGTH | BODY LENGTH | HEADER | BODY ]
*/

// 我感觉这里DataPack 有点多余，它的方法完全可以交给 Message 去完成
type DataPack struct {
}

func NewDataPack() *DataPack {
	return &DataPack{}
}
func (dp *DataPack) GetFixedHeadLen() uint32 { // 得到应用层包头总长，固定头部的长度
	return MsgHeaderLength
}

// 相当于结构体的序列化
func (dp *DataPack) Pack(msg ziface.IMessage) ([]byte, error) {
	buf := bytes.NewBuffer([]byte{}) // 创建一个空的缓冲
	// 把 msg 对象的所有成员 按顺序写入缓冲
	if err := binary.Write(buf, binary.LittleEndian, msg.GetMsgId()); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, msg.GetLength()); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, msg.GetData()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// 相当于将字节切片反序列化为结构体
func (dp *DataPack) Unpack(headData []byte, conn net.Conn) (ziface.IMessage, error) {
	// 先读 head（len和id）的信息
	buf := bytes.NewReader(headData)
	msg := &Message{}
	if err := binary.Read(buf, binary.LittleEndian, &msg.MsgId); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &msg.Length); err != nil {
		return nil, err
	}
	if msg.GetLength() > utils.GlobalObj.MaxFilePackageSize {
		return nil, fmt.Errorf("收到的数据包长度太长，请检查msgid = %d", msg.GetMsgId())
	}
	msg.Data = make([]byte, msg.GetLength())
	_, err := io.ReadFull(conn, msg.Data) // 继续读取消息内容
	if err != nil {
		if err != io.EOF {
			logrus.Error("unpack message body err :", err)
		}
		return nil, err
	}
	return msg, nil
}
