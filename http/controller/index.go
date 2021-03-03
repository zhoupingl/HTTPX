package controller

import (
	"gitee.com/zhucheer/orange/app"
	"github.com/tidwall/gjson"
	"httpx/proxy"
	"io/ioutil"
)

func Println(ctx *app.Context) error {

	data, _ := ioutil.ReadAll(ctx.Request().Body)
	println("\033[37m" + string(data) + "\033[0m")

	return nil
}

func Put(ctx *app.Context) error {

	body, err := ioutil.ReadAll(ctx.Request().Body)
	if err != nil {
		return err
	}

	rid := gjson.GetBytes(body, "rid").String()
	url := gjson.GetBytes(body, "url").String()
	data := gjson.GetBytes(body, "data").String()
	timeout := gjson.GetBytes(body, "timeout").Int()

	resp := proxy.PushTask(rid, url, data, int(timeout), 0)

	ctx.ToString(resp)

	return nil
}
