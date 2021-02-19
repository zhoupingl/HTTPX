package main

import (
	"fmt"
	"gitee.com/zhucheer/orange/app"
	"httpx/http"
	"httpx/proxy"
	"time"
)

func main() {

	router := &http.Route{}

	// 等待任务全部执行完成  优雅退出
	app.AppDefer(func() {
		// 关闭服务
		fmt.Println(fmt.Sprintf("[SYS] \033[31m 系统准备退出\033[0m "))
		proxy.Exit()
		fmt.Println(fmt.Sprintf("[SYS] \033[31m 系统退出...\033[0m "))
		time.Sleep(time.Second * 2)
		proxy.Wait()
		fmt.Println(fmt.Sprintf("[SYS] \033[31m 系统退出!!!\033[0m "))

	})
	app.AppStart(router)

}
