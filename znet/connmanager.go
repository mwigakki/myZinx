package znet

import (
	"fmt"
	"sync"

	"github.com/myZinx/ziface"
	"github.com/sirupsen/logrus"
)

type ConnManager struct {
	server ziface.IServer // 绑定了此连接管理模块的服务器对象
	// 已经创建的connection 集合
	conns map[uint32]ziface.IConnection
	// 每个server 都有的MessageHandler 模块，其中用map 为每个消息ID 定义了handler方法（开发者自己加的），
	// 如果这些handler 方法有增删改查，那么就有并发问题，所以需要一个锁
	connLock    sync.RWMutex            // 保护连接集合的读写锁
	ConnMgrChan chan ziface.IConnection // 每次客户端连接成功或断开连接会将会连接信息放进这个通道，connManage方法才去添加或删除这个连接
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		conns:       make(map[uint32]ziface.IConnection),
		ConnMgrChan: make(chan ziface.IConnection, 32), // 暂时随便定义一个长度。高并发时长度太小会导致连接缓慢，因为通道满了还写入就会阻塞。但通道太长又浪费空间
		// 锁不用初始化了
	}
}

// 连接管理的方法，每次客户端连接成功或断开连接会将会连接信息放进通道，connManage方法从通道中读取后才去添加或删除这个连接
func (cm *ConnManager) ConnManage() {
	for {
		conn := <-cm.ConnMgrChan
		// 如果该conn 在管理器中有就删除，没有就加入
		if _, has := cm.conns[conn.GetConnID()]; has {
			cm.Remove(conn)
			cm.server.CallOnConnStop(conn) // 此连接conn 中就不应该再调用通道，tcp连接等等成员变量了，因为它们已经被关闭了。可以调用其他的数值变量
			// 这里也是一点点缺陷吧，按理来说此钩子函数应该在连接回收资源以前进行调用。现在这个做法是回收资源和钩子函数在两个goroutine了。解决办法当然就是把server对象耦合到每个connection中
		} else {
			cm.Add(conn)
			cm.server.CallOnConnStart(conn)
			if cm.Len()%1000 == 0 {
				logrus.Infof("[ConnManager] 当前连接人数=  %d, \n", cm.Len())
			}
		}
		logrus.Debugf("[ConnManager] 当前连接人数=  %d, \n", cm.Len())
	}
}

// 增加连接
func (cm *ConnManager) Add(conn ziface.IConnection) {
	// 保护共享资源 map，加 写锁
	cm.connLock.Lock()
	defer cm.connLock.Unlock()
	cm.conns[conn.GetConnID()] = conn
	logrus.Debug("connection (id= ", conn.GetConnID(), ") added to ConnManager successfuly")
}

// 删除连接
func (cm *ConnManager) Remove(conn ziface.IConnection) {
	// 保护共享资源 map，加 写锁
	cm.connLock.Lock()
	defer cm.connLock.Unlock()
	delete(cm.conns, conn.GetConnID())
}

// 得到一个连接
func (cm *ConnManager) Get(connId uint32) (ziface.IConnection, error) {
	// 保护共享资源 map，加 读锁
	cm.connLock.RLock()
	defer cm.connLock.RUnlock()
	if conn, has := cm.conns[connId]; has {
		return conn, nil
	} else {
		return nil, fmt.Errorf("connection id : %d NOT FOUND! ", connId)
	}
}

// 总连接数
func (cm *ConnManager) Len() int {
	return len(cm.conns)
}

// 终止并清楚所有连接，关闭服务器时
func (cm *ConnManager) Clear() {
	// 保护共享资源 map，加 写锁
	cm.connLock.Lock()
	defer cm.connLock.Unlock()
	for connId, conn := range cm.conns {
		conn.Stop()
		delete(cm.conns, connId)
	}
	logrus.Infoln("Clear All connections successfully! ")
}

// 得到 与连接管理器通信的通道
func (cm *ConnManager) GetConnMgrChan() chan ziface.IConnection {
	return cm.ConnMgrChan
}

// 设置连接管理模块对应的server
func (cm *ConnManager) SetServer(s ziface.IServer) {
	cm.server = s
}
