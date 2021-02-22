package controller

import (
	"fmt"
	"gitee.com/zhucheer/orange/app"
	"github.com/tidwall/gjson"
	"httpx/proxy"
	"io/ioutil"
)

func Put(ctx *app.Context) error {

	body, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("[INFO] \033[1;33mHttp requet body:%s\033[0m ", string(body)))

	rid := gjson.GetBytes(body, "rid").String()
	url := gjson.GetBytes(body, "url").String()
	data := gjson.GetBytes(body, "data").String()
	timeout := gjson.GetBytes(body, "timeout").Int()

	resp := proxy.PushTask(rid, url, data, int(timeout), 0)

	ctx.ToString(resp)

	return nil
}
