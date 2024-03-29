package znet

import (
	"fmt"
	"io"
	"os"

	"github.com/myZinx/utils"
	"github.com/myZinx/ziface"
	"github.com/sirupsen/logrus"
)

// 定义实现 IRouter 的基类，但这个基类不具体实现任何方法
// 用户实现自己的router时，先嵌入这个基类，然后对这个基类的方法进行重写就好了
// 有点像适配层，对接口进行隔离
// 当然用户也可以不继承，直接实现 IRouter 也行。
// 只是实现接口的话就必须三个方法都实现，继承基类的话就可以按需求实现想实现的方法即可
type BaseRouter struct{}

// 在处理 conn 业务之前的钩子方法
func (br *BaseRouter) PreHandle(req ziface.IRequest) {}

// 处理 conn 业务的主方法
func (br *BaseRouter) Handle(req ziface.IRequest) {}

// 在处理 conn 业务之后的钩子方法
func (br *BaseRouter) PostHandle(req ziface.IRequest) {}

// 默认的 收到心跳包 回包时的路由处理
type HeartbeatDefaultRouter struct {
	BaseRouter
}

func (br *HeartbeatDefaultRouter) Handle(req ziface.IRequest) {
	conn := req.GetConnection()
	data := req.GetData() // 得到的只是数据，不包含message 的头
	logrus.Debugf("[connId: %d | remote: %v | msgId: %s]: %s", conn.GetConnID(),
		conn.GetTCPConnection().RemoteAddr(), utils.GlobalObj.MsgIdDesc[req.GetMsgId()], string(data))
}

// 默认的 客户端发给server的普通消息 的路由处理
type GeneralMsgRouter struct {
	BaseRouter
}

func (br *GeneralMsgRouter) Handle(req ziface.IRequest) {
	conn := req.GetConnection()
	data := req.GetData() // 得到的只是数据，不包含message 的头
	logrus.Infof("[connId: %d | remote: %v | msgId: %s]: %s", conn.GetConnID(),
		conn.GetTCPConnection().RemoteAddr(), utils.GlobalObj.MsgIdDesc[req.GetMsgId()], string(data))
}

// 默认的 客户端希望得到server消息响应 的路由处理
type PingRouter struct {
	BaseRouter
}

func (br *PingRouter) Handle(req ziface.IRequest) {
	conn := req.GetConnection()
	data := req.GetData() // 得到的只是数据，不包含message 的头
	logrus.Infof("[connId: %d | remote: %v | msgId: %s]: %s", conn.GetConnID(),
		conn.GetTCPConnection().RemoteAddr(), utils.GlobalObj.MsgIdDesc[req.GetMsgId()], string(data))
	// 数据回复
	respondMsg := []byte("server respond!")
	err := conn.SendMsg(utils.MSGID_PING, uint32(len(respondMsg)), respondMsg)
	if err != nil {
		logrus.Errorln("router handle err :", err)
		return
	}
}

// 默认的 处理文件下载请求 的路由处理
type FileRequestRouter struct {
	BaseRouter
}

// FileRequest数据包中，data就是文件名
func (br *FileRequestRouter) Handle(req ziface.IRequest) {
	// 先用Pack把一个 FILE_RESPOND 数据包头准备好的包发过去，包头中length就是文件大小
	filePath := fmt.Sprintf("files/%s", string(req.GetData()))
	file, err := os.Open(filePath)
	if err != nil {
		logrus.Warn("文件 " + filePath + " 不存在")
		return
	}
	defer file.Close()
	// 先把文件按 小块 读到内存，然后这一小块发出去 , 可以用conn.SetWriteBuffer() 设置tcp发送缓冲区大小
	conn := req.GetConnection()
	buffer := make([]byte, utils.GlobalObj.MaxFilePackageSize) // 定义每个文件块的最大大小，但实际进入tcp传输还是会切分，但我们不管
	for {
		// 先读取是否允许传输文件，如果接收到不允许文件传输的命令了，就在这里停止传输并跳出循环
		if !req.GetConnection().GetServer().IsAllowFileReq() {
			logrus.Info("未开启或已关闭文件传输")
			return
		}
		n, err := file.Read(buffer) // 读到文件末尾（即最后一次读）会返回 0, io.EOF
		if err == io.EOF {          // 文件读取完毕
			// logrus.Infof("向 %d号连接发送文件 %s 成功。", req.GetConnection().GetConnID(), string(req.GetData()))
			break
		}
		if err != nil {
			logrus.Errorf("read file %s occurs err: %v", filePath, err)
			return
		}
		err = conn.SendMsg(utils.MSGID_FILE_RESPOND, uint32(n), buffer[:n])
		if err != nil {
			logrus.Error("发送文件信息出错， err= ", err)
			return
		}
	}
}
