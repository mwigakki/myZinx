package znet

import (
	"github.com/myZinx/ziface"
	"github.com/sirupsen/logrus"
)

// 每个server有一个MessageHandler属性，只是这个属性会同时传给所有connection
type MessageHandler struct {
	// 每一个消息ID所对应的处理方法
	Apis map[uint32]ziface.IRouter
}

func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		Apis: make(map[uint32]ziface.IRouter),
	}
}

// 调度，执行对应的router消息处理方法
func (m *MessageHandler) DoMsgHandler(req ziface.IRequest) {
	msgId := req.GetMsgId()
	handler, has := m.Apis[msgId]
	if !has {
		logrus.Warnf("[WARNING] api msg id [%d] is NOT FOUND! need register!", msgId)
		return
	}
	handler.PreHandle(req)
	handler.Handle(req)
	handler.PostHandle(req)
}

// 给server添加具体的router 处理逻辑
func (m *MessageHandler) AddRouter(msgID uint32, router ziface.IRouter) {
	m.Apis[msgID] = router
}
