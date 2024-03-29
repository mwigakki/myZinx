package main

import (
	"time"

	"github.com/myZinx/utils"
	"github.com/myZinx/ziface"
	"github.com/myZinx/znet"
	"github.com/sirupsen/logrus"
)

// 不同消息的handler
func heartBeatHandler(msg ziface.IMessage, c *ClientConn) {
	logrus.Debugf("[remote: %v | msgId: %s]: %s ", c.conn.RemoteAddr(), utils.GlobalObj.MsgIdDesc[msg.GetMsgId()], string(msg.GetData()))
	// 心跳包需要回复
	data := []byte("来自 [客户端] 的心跳包")
	msgSend := &znet.Message{ // 测试心跳包
		MsgId:  utils.MSGID_HEARTBEAT,
		Length: uint32(len(data)),
		Data:   data}
	buf, err := c.dp.Pack(msgSend)
	if err != nil {
		logrus.Error("client Pack err,err = ", err)
		return
	}
	c.msgChan <- buf
}

func generalMsgHandler(msg ziface.IMessage, c *ClientConn) {
	logrus.Infof("[remote: %v | msgId: %s]: %s ", c.conn.RemoteAddr(), utils.GlobalObj.MsgIdDesc[msg.GetMsgId()], string(msg.GetData()))
}
func pingHandler(msg ziface.IMessage, c *ClientConn) {
	logrus.Infof("[remote: %v | msgId: %s]: %s ", c.conn.RemoteAddr(), utils.GlobalObj.MsgIdDesc[msg.GetMsgId()], string(msg.GetData()))
}
func fileRespondHandler(msg ziface.IMessage, c *ClientConn) {
	if !c.isFileRequesting { // 如果文件传输请求已经被关闭，那么现在传输的这些就直接不要了
		logrus.Debug(" 如果文件传输请求已经被关闭，那么现在传输的这些就直接不要了")
		return
	}
	// 接收server 发来的文件，msg的length就是文件大小
	logrus.Tracef("[remote: %v | msgId: %s]: 收到文件块大小 : %d", c.conn.RemoteAddr(), utils.GlobalObj.MsgIdDesc[msg.GetMsgId()], msg.GetLength())
	// 不能用io.CopyN去接收，因为我们需要保证  io.ReadFull(c.conn, headData) 是数据包的唯一入口
	// 所以把所有文件数据包都用 FILE_RESPOND 头进行封装
	if c.saveFile && c.fileTrans.fileWriter == nil {
		logrus.Warn("未开启文件 writer，无法保存，接收数据包全部丢弃")
		return
	} else {
		if c.saveFile { // 需要保存文件才把数据写入file writer
			_, err := c.fileTrans.fileWriter.Write(msg.GetData())
			if err != nil {
				logrus.Error("写入文件出错，err = ", err)
				return
			}
		}
		c.fileTrans.byteReceived += int64(msg.GetLength())
		if c.fileTrans.byteReceived == c.fileTrans.fileSize { // 刚好接收完毕
			duration := time.Since(c.fileTrans.startTime)
			downloadSpeed := 8 * 1000 * float64(c.fileTrans.fileSize) / float64(duration)
			logrus.Infof("文件 %s 下载完毕，总共用时 %.3f ms, 平均下载速度 %.3f Mbps", c.fileTrans.fileName, float64(duration)/1e6, downloadSpeed)
			// 此数据包接收完毕，重新开始等待
			WaitTime := generateRandomWaitTime()                                     // 根据指数分布随机生成一个等待时间
			dur := time.Duration(WaitTime+utils.GlobalObj.MinWaitTimt) * time.Second // 随机休眠一段时间
			c.ticker.Reset(dur)                                                      // 重设定时器的计时周期

			c.fileTrans.Close()
		} else if c.fileTrans.byteReceived > c.fileTrans.fileSize {
			// 一般错误不会从这里报出来
			logrus.Fatal("传输数据量大于文件大小，无法处理")
			c.fileTrans.Close()
			return
		}
	}
}
