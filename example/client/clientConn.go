package main

import (
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/myZinx/utils"
	"github.com/myZinx/ziface"
	"github.com/myZinx/znet"
	"github.com/sirupsen/logrus"
)

type ClientConn struct {
	id               uint32
	conn             net.Conn                                      // 与server 的tcp连接
	dp               *znet.DataPack                                // 封包解包的结构体
	handler          map[uint32]func(ziface.IMessage, *ClientConn) // 给客户端就写简单的handler 完成消息的处理把
	msgChan          chan []byte                                   // handler把消息处理后可能会需要给server回复一些信息，于是把消息放在此通道中
	exitChan         chan bool                                     // 退出的通道，无缓冲
	fileReqSignal    chan bool                                     // 文件传输的信息，传入true表示开启文件请求，传入false表示停止文件请求
	isFileRequesting bool                                          // 当前连接的文件请求已开启，默认false
	fileTrans        *FileTransfer                                 // 正在传输的 文件对象，每个客户端每次只能接收一个文件
	saveFile         bool                                          // 确认保存文件，为false表示只将文件传过来而不保存
	ticker           *time.Ticker                                  // 文件传输时，等待时间的定时器
}

func newClientConn(conn net.Conn, id uint32) *ClientConn {
	return &ClientConn{
		id:   id,
		conn: conn,
		dp:   znet.NewDataPack(),
		handler: map[uint32]func(ziface.IMessage, *ClientConn){
			utils.MSGID_HEARTBEAT:    heartBeatHandler,
			utils.MSGID_GENERAL_MSG:  generalMsgHandler,
			utils.MSGID_PING:         pingHandler,
			utils.MSGID_FILE_REQUEST: nil, // CLIENT 不会收到文件请求，只会发出
			utils.MSGID_FILE_RESPOND: fileRespondHandler,
		},
		msgChan:          make(chan []byte), // 无阻塞通道即可，每次只处理一个消息
		exitChan:         make(chan bool),
		fileReqSignal:    make(chan bool, 1), // 防止 写入fileReqSignal的地方阻塞
		isFileRequesting: false,
		fileTrans:        &FileTransfer{fileWriter: nil},
		saveFile:         false,                                    // 默认只传文件而不保存。
		ticker:           time.NewTicker(time.Duration(1<<63 - 1)), // 因为默认不开启文件传输，故此定时器触发时间是无限大
	}
}

func (c *ClientConn) clientReader() {
	logrus.Debugf("client %d started READER !", c.id)
	defer func() {
		logrus.Debugf("client %d exit READER !", c.id)
		wg.Done()
		c.exitChan <- true
	}()
	for {
		// 先接收首部，再根据首部判断之后的内容如何接收
		headData := make([]byte, c.dp.GetFixedHeadLen())
		_, err := io.ReadFull(c.conn, headData)
		if err != nil {
			if err == io.EOF {
				logrus.Info("远端server 已关闭!")
				return
			}
			logrus.Errorf("client %d read err : %v", c.id, err)
			return
		}
		// 在Unpack 中再次调用 io.ReadFull 按照头部中定义的数据长度去读取
		msgReceived, err := c.dp.Unpack(headData, c.conn)
		if err != nil {
			logrus.Error("server read Unpack err :", err)
			return
		}
		c.handler[msgReceived.GetMsgId()](msgReceived, c) // 根据消息ID 调用对应的handler, 不开额外线程去处理
	}
}
func (c *ClientConn) clientWriter() {
	logrus.Debugf("client %d started WRITER !", c.id)
	defer func() {
		logrus.Debugf("client %d exit WRITER !", c.id)
		c.stopClientConn()
		wg.Done()
	}()
	// writer 中只将通道中传来的 字节数组 发出去
	for {
		select {
		case msg := <-c.msgChan:
			_, err := c.conn.Write(msg)
			if err != nil {
				logrus.Error("client write err,err = ", err)
				return
			}
		case <-c.exitChan:
			return
		}
	}
}

func (c *ClientConn) stopClientConn() {
	// 连接退出，释放资源
	// 但是没有在主函数的 conns 切片中删除自己，但懒得管了
	logrus.Debug("连接退出，释放资源")
	c.conn.Close()
	close(c.msgChan)
	close(c.exitChan)
	// close(c.fileReqSignal)  这里关闭后会给StartFileRequest 函数的<-c.fileReqSignal 发送false，会一直发，所以这里不要管了
	c.ticker.Stop()
}

func (c *ClientConn) StartFileRequest() {
	var err error
	var maxDuration time.Duration = 1<<63 - 1 // largest representable duration to approximately 290 years.
	// 使用可控的定时器，不要使用不可控的 time.Sleep()
	for {
		select {
		case fileReq := <-c.fileReqSignal: // 结束文件请求的信号
			if fileReq {
				// 收到的fileReq 为true，表示让此连接开始文件请求
				logrus.Infof("[client %d] 连接开启文件请求", c.id)
				WaitTime := generateRandomWaitTime()           // 根据指数分布随机生成一个等待时间
				dur := time.Duration(WaitTime+2) * time.Second // 随机休眠一段时间，至少等 2 秒
				c.ticker.Reset(dur)                            // 重设定时器的计时周期
				c.isFileRequesting = true
			} else {
				// 收到的fileReq 为false，表示让此连接停止文件请求。（正在接收的不管）
				logrus.Infof("[client %d] 连接结束文件请求", c.id)
				c.isFileRequesting = false
				c.fileTrans.Close()         // 当前正在使用的这个 文件传输对象，关闭之
				c.ticker.Reset(maxDuration) // 结束文件请求，设置无限等待
				continue
			}
		case <-c.ticker.C: //（不开fileRequest 不可能到这儿）
			fId := 0 // 随机得到
			c.fileTrans, err = NewFileTransfer(c.saveFile, utils.GlobalObj.FileNames[fId], utils.GlobalObj.FileSizes[fId])
			if err != nil {
				logrus.Error("打开/创建文件出错，err = ", err)
				return
			}
			data := []byte(c.fileTrans.fileName) // 把请求的文件名传过去
			err = c.SendMsg(utils.MSGID_FILE_REQUEST, uint32(len(data)), data)
			logrus.Debug("开始请求文件")
			if err != nil {
				logrus.Errorln("router handle err :", err)
				return
			}
			c.ticker.Reset(maxDuration) // 发送一个请求后，就设置无限等待，直到上一个包接收完毕
		}
	}
}

func (c *ClientConn) SendMsg(msgID uint32, length uint32, data []byte) error {
	msg := &znet.Message{
		MsgId:  msgID,
		Length: length,
		Data:   data,
	}
	dp := znet.DataPack{}
	sendData, err := dp.Pack(msg)
	if err != nil {
		logrus.Error("when SendMsg Pack msg, err = ", err)
		return err
	}
	// 将要发送的数据发给writer 线程
	c.msgChan <- sendData
	return nil
}

func generateRandomWaitTime() int {
	// 生成一个[0.0, 1.0)的随机数r，看r落在cdf的哪个区间，区间下标就是结果，按泊松分布概率生成的随机值
	rnd := rand.Float64()
	// 用二分查找返回其区间下标
	l, r := 0, len(cdf)
	for l != r-1 {
		mid := (l + r) / 2
		if cdf[mid] <= rnd {
			l = mid
		} else {
			r = mid
		}
	}
	// logrus.Infof("生成的随机数为 %f, -->随机数落在区间 %f - %f，生成下标 %d \n", rnd, cdf[l], cdf[r], l)
	return l
}
