package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	var ExitChan chan bool = make(chan bool, 1)
	var i int = 0
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			dur := time.Duration((rand.Intn(5))+1) * time.Second
			ticker.Reset(dur) // 重设定时器的计时周期
			fmt.Printf("定时器触发了，当前时间：%s\n", time.Now().Format("15:04:05"))
			i++
			if i > 1 {
				ExitChan <- true
			}
		case <-ExitChan:
			fmt.Println("退出该goroutine")
			ticker.Stop() // 回收计时器里的资源
			return
		}
	}
}
