package main

import (
	"os"
	"time"
)

// 当前客户端连接正在传输的文件信息
type FileTransfer struct {
	fileWriter   *os.File // 写入文件的接口
	fileName     string   // 规定文件名和文件大小在客户端都是已知的
	fileSize     int64
	byteReceived int64     // 已接收到的字节数，到达fileSize 时即接收完毕
	startTime    time.Time // 文件开始下载时间，用来计算下载用时的
}

func NewFileTransfer(saveFile bool, fileName string, fileSize int64) (*FileTransfer, error) {
	ft := &FileTransfer{}
	if saveFile { // 连接明确需要保存文件才会打开文件的 writer
		fw, err := os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		ft.fileWriter = fw
	}
	ft.fileName = fileName
	ft.fileSize = fileSize
	ft.byteReceived = 0
	ft.startTime = time.Now()
	return ft, nil
}

// 每次接收完毕后都关闭文件
func (ft *FileTransfer) Close() {
	ft.fileWriter.Close()
	ft.fileName = ""
	ft.fileSize = 0
	ft.byteReceived = 0
}
