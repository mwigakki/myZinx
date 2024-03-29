package znet

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/myZinx/ziface"
	"github.com/sirupsen/logrus"
)

type Connection struct {
	//  socket TCP套接字 c
	Conn *net.TCPConn
	//  链接ID
	ConnID uint32
	//  链接的状态（是否关闭）
	isClosed bool
	//  等待连接被动退出的channel（管理连接状态，连接断开要通知此channel
	ExitChan chan bool
	// 无缓冲通道，用于读、写 goroutine 之间的消息通信
	msgChan chan []byte
	// 当前 连接对应的 处理业务的router
	MsgHandler ziface.IMessageHandler
	// 与连接管理器通信的通道
	ConnMgrChan chan ziface.IConnection // 每次客户端连接成功或断开连接会将会连接信息放进这个通道，connManage方法才去添加或删除这个连接
	// 该连接的心跳检测器
	hbc ziface.IHeartBeatChecker
	// 链接所在的server，必须使用SetServer 添加
	server ziface.IServer

	// 连接属性的集合，里面都是用户自定义的属性
	property map[string]any // 需要在new函数里初始化
	// 保护连接属性的锁
	propertyLock sync.RWMutex
}

func NewConnection(c *net.TCPConn, connID uint32, msgHandler ziface.IMessageHandler, connMgrChan chan ziface.IConnection) *Connection {
	conn := &Connection{ // 声明的msgChan 是不带缓冲区的，ExitChan 也是。
		Conn:        c,
		ConnID:      connID,
		isClosed:    false,
		ExitChan:    make(chan bool),
		msgChan:     make(chan []byte),
		MsgHandler:  msgHandler,
		ConnMgrChan: connMgrChan, // 此通道由连接管理器管理并维护
		hbc:         nil,         // 默认不开心跳检测器，把开启权限交给server
		property:    make(map[string]any),
	}
	// 将当前连接加入到 与连接管理器通信的通道 中，把本连接注册到连接管理器
	conn.ConnMgrChan <- conn
	return conn
}

// 启动连接，让当前连接准备开始工作
func (c *Connection) Start() {
	logrus.Debug("connection start. Connection id = ", c.ConnID)
	// 启动 当前连接的 读写数据的业务goroutine
	go c.StartReader()
	go c.StartWriter()
	if c.hbc != nil {
		c.hbc.Start()
	}
}

// 关闭连接。结束连接的工作
func (c *Connection) Stop() {
	logrus.Debug("connection stop. Connection id = ", c.ConnID)
	// 如果已经关闭就不管了
	if !c.isClosed {
		c.isClosed = true
		// 关闭连接的心跳检测器
		if c.hbc != nil {
			c.hbc.Stop()
		}
		// 关闭socket 连接
		c.Conn.Close()
		// 在连接管理器中删除自己
		// 将当前连接加入到 与连接管理器通信的通道 中，把本连接从连接管理器中删除
		c.ConnMgrChan <- c
		// 回收资源
		close(c.ExitChan)
		close(c.msgChan)
	}
}

// 获取当前连接绑定的 c
func (c *Connection) GetTCPConnection() *net.TCPConn {
	return c.Conn
}

// 获取连接ID
func (c *Connection) GetConnID() uint32 {
	return c.ConnID
}

// 获取客户端的TCP状态 IP和Port
func (c *Connection) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// 连接的read 业务方法
func (c *Connection) StartReader() {
	defer func() { // reader 线程任何一个return 都会给退出通道传入值，
		c.ExitChan <- true //  StartWriter() 方法接收此通道的值，用来退出 writer 线程
	}()
	for {
		// 按 TLV 的格式进行拆包读取
		dp := NewDataPack() // 创建一个解包的对象
		headData := make([]byte, dp.GetFixedHeadLen())
		_, err := io.ReadFull(c.Conn, headData) // 读出头部数据 []byte类型；客户端关闭的话，这里会收到EOF 的错误
		if err != nil {
			if err != io.EOF {
				logrus.Error("server GetFixedHeadLen err :", err)
			}
			return
		}
		if c.hbc != nil {
			c.hbc.UpdateActiveTime() // 更新心跳检测器时间
		}
		msg, err := dp.Unpack(headData, c.GetTCPConnection()) // 直接从 conn 中读取data
		if err != nil {
			logrus.Error("server read Fixed Head err :", err)
			return
		}
		// 每个connection 得到的数据都封装成request，然后将request 交给router 进行处理
		// 得到当前conn 数据的Request 请求数据
		req := &Request{conn: c, msg: msg}
		// server端收到的所有数据都在handle里面处理
		go c.MsgHandler.DoMsgHandler(req)
	}
}

// 连接的 write 业务方法，给客户端发送消息的模块
func (c *Connection) StartWriter() {
	// 不停阻塞，一直等待 reader给同步通道发送通知
	for {
		select {
		case data := <-c.msgChan: // data 就是reader 收到客户消息后，执行完业务逻辑，封装好的要发回客户的信息
			if _, err := c.GetTCPConnection().Write(data); err != nil {
				// 向对端写入数据时，由于对端关闭，这里便报错了。
				// 但是由于reader 一直阻塞在 io.ReadFull 方法，所以在这里也不好去控制reader 关闭
				// 只能关闭程序时先关闭server，再关闭client
				logrus.Error("[Writer] Send dada err: ", err)
				return
			}
		case <-c.ExitChan:
			// reader 收到客户端退出的消息，就发送信息告诉writer关闭自己
			// （从没值的通道中取值会阻塞）从退出通道中取值，取到了说明要退出程序了。（一般是reader goroutine向退出通道中传入值）
			c.Stop()
			return
		}
	}
}

// 此方法将我们要发送给客户端的数据先进行封包，得二进制数据，再发送给写的goroutine
func (c *Connection) SendMsg(msgID uint32, length uint32, data []byte) error {
	if c.isClosed {
		return errors.New("Connection is closed when send msg. ")
	}
	msg := &Message{
		MsgId:  msgID,
		Length: length,
		Data:   data,
	}
	dp := DataPack{}
	sendData, err := dp.Pack(msg)
	if err != nil {
		logrus.Error("when SendMsg Pack msg, err = ", err)
		return err
	}
	// 将要发送的数据发给writer 线程
	c.msgChan <- sendData
	return nil
}

// 绑定心跳检测器
func (c *Connection) BindHeartBeatChecker(hbc ziface.IHeartBeatChecker) {
	c.hbc = hbc
}

// 设置连接属性
func (c *Connection) SetProperty(key string, value any) {
	// 加写锁
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	c.property[key] = value
}

// 获取连接属性
func (c *Connection) GetProperty(key string) (any, error) {
	// 加读锁
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	if value, has := c.property[key]; has {
		return value, nil
	} else {
		return nil, fmt.Errorf("property %s NOT FOUND ! please set before ", key)
	}

}

// 删除连接属性
func (c *Connection) RemoveProperty(key string) {
	// 加写锁
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	delete(c.property, key)
}

// 该连接是否还存活
func (c *Connection) IsAlive() bool {
	return !c.isClosed
}

// 虽然这样耦合太严重了，但为了实现在服务器关闭正在传输的文件，必须把server 加到每个连接中
func (c *Connection) SetServer(server ziface.IServer) {
	c.server = server
}
func (c *Connection) GetServer() ziface.IServer {
	return c.server
}
