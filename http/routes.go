package http

import (
	"gitee.com/zhucheer/orange/app"
	"httpx/http/controller"
	"httpx/proxy"
)

type Route struct {
}

func (s *Route) ServeMux() {
	commonGp := app.NewRouter("")
	{
		// 任务加入代理http api请求队列中
		commonGp.POST("/x", controller.Put)
		commonGp.POST("/printx", controller.Println)
	}
}

// Register 服务注册器, 可以对业务服务进行初始化调用
func (s *Route) Register() {
	proxy.RecoveryData()
}
