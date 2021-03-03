package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gitee.com/zhucheer/orange/database"
	"gitee.com/zhucheer/orange/httpclient"
	"github.com/tidwall/gjson"
	"os"
	"strings"
	"sync"
	"time"
)

var w = NewWorkers()

type PayResponse struct {
	Errno  int    `json:"error"`
	ErrMsg string `json:"error_info"`
}

type PayBody struct {
	StartAt      int64       `json:"start_at"`
	ElapsedTime  int64       `json:"elapsed_time"`
	HttpResponse PayResponse `json:"http_response"`
	Rid          string      `json:"rid"`
}

type Roby struct {
	T_rid     string `json:"t_rid"`
	T_url     string `json:"t_url"`
	T_data    string `json:"t_data"`
	T_timeout int    `json:"t_timeout"`
}

type command = string

const (
	SET command = "SET"
	DEL command = "DEL"
)

const REST_COUNT_MAX = 3

type Cmd struct {
	*Roby
	cmd string
}

var RobyPipeline = make(chan *Cmd, 100000)
var _RobyPipelineOnce = sync.Once{}

func init() {
	_RobyPipelineOnce.Do(func() {

		go func() {
			// add
			//启动一个协程， 处理
			for roby := range RobyPipeline {
				switch roby.cmd {
				case SET:
					HASH_XSET(roby.Roby)
				case DEL:
					HASH_XDEL(roby.T_rid)
				}
			}
		}()
	})
}

type BufStdOut struct {
	sync.Mutex
	b *bufio.Writer
}

func (o *BufStdOut) WriteString(s string) (int, error) {

	o.Lock()
	defer o.Unlock()

	n, err := o.b.WriteString(s)
	if err != nil {
		return n, err
	}
	return 0, o.b.WriteByte('\n')
}

var buf = BufStdOut{
	b: bufio.NewWriter(os.Stdout),
}

func PushTask(rid, url, data string, timeout, resetCount int) string {
	url = strings.TrimSpace(url)
	// # check
	if url == "" {
		return "ErrMsg:url不能为空"
	}

	if timeout < 1 {
		return "ErrMsg:timeout必须>1"
	}

	// 验证服务是否正在准备退出
	if !_exit.IsRun() {
		return "ErrMsg:服务正在退出..."
	}

	// 记录任务数
	_exit.Add(1)

	// 记录信息到数据库
	roby := &Roby{
		T_rid:     rid,
		T_url:     url,
		T_data:    data,
		T_timeout: timeout,
	}
	RobyPipeline <- &Cmd{roby, SET}

	buf.WriteString("[DEBUG]  \033[31m Add job! \033[0m")
	// push queue
	w.GetWorker(url).Schedule(func() {

		start := time.Now()
		defer func() {
			buf.WriteString(fmt.Sprintf("-------------D%d", time.Now().Sub(start).Milliseconds()))
		}()

		defer func() {
			if e := recover(); e != nil {
				buf.WriteString(fmt.Sprintf("[DEBUG] \033[0;33m 异常:%v\033[0m ", e))
			}
		}()
		// 记录数量
		defer func() {
			// 统计信息
			_stat.log(url)
			// 完成任务，标记
			_exit.Done()
			// 已经完成，从数据库删除数据
			RobyPipeline <- &Cmd{roby, DEL}
		}()

		buf.WriteString(fmt.Sprintf("[DEBUG] \033[0;33m 开始任务:%v, id:%s\033[0m ", time.Now().String(), url))
		defer func() {
			buf.WriteString(fmt.Sprintf("[DEBUG] \033[0;33m 完成任务:%v, id:%s\033[0m ", time.Now().String(), url))
		}()

		// 构建返回结构体信息
		resp := &PayBody{Rid: rid}
		defer func() {
			// 将请求结果-写入redis list中
			if resp.HttpResponse.ErrMsg == "SUCCESS" || resetCount >= REST_COUNT_MAX {
				pushToRedisList(resp)
			} else {
				resetCount++
				PushTask(rid, url, data, timeout, resetCount)
			}
		}()
		resp.StartAt = getMs()

		// 记录信息
		client := httpclient.NewClient()
		client.SetTimeout(timeout)
		client.WithBody(data)
		client.ContentType("multipart/form-data")
		httpResp, err := client.RunPost(url)
		resp.ElapsedTime = getMs() - resp.StartAt
		if err != nil {
			resp.HttpResponse.Errno = 500
			resp.HttpResponse.ErrMsg = fmt.Sprintf("HTTP POST %s fail", err.Error())
			return
		}

		if httpResp.BodyRaw.StatusCode < 200 || httpResp.BodyRaw.StatusCode >= 400 {
			resp.HttpResponse.Errno = httpResp.BodyRaw.StatusCode
			resp.HttpResponse.ErrMsg = httpResp.BodyRaw.Status
			return
		}

		// 判断返回结果是否为json
		if httpResp.String() == "SUCCESS" {
			resp.HttpResponse.Errno = 0
			resp.HttpResponse.ErrMsg = "SUCCESS"
			return
		}

		// 调用接口,接口系统错误 检查
		if gjson.GetBytes(httpResp.Body, "error").Exists() {
			resp.HttpResponse.Errno = int(gjson.GetBytes(httpResp.Body, "error").Int())
			resp.HttpResponse.ErrMsg = gjson.GetBytes(httpResp.Body, "error_info").String()
			return
		}

		// 调用接口,系统业务错误
		if gjson.GetBytes(httpResp.Body, "code").Exists() {
			resp.HttpResponse.Errno = int(gjson.GetBytes(httpResp.Body, "code").Int())
			resp.HttpResponse.ErrMsg = gjson.GetBytes(httpResp.Body, "msg").String()
			if resp.HttpResponse.Errno == 0 || resp.HttpResponse.Errno == 200 {
				resp.HttpResponse.ErrMsg = "SUCCESS"
			}
			return
		}

		resp.HttpResponse.Errno = -1
		resp.HttpResponse.ErrMsg = "未知错误-无法处理API数据"

	})

	return "SUCCESS"
}

// 记录信息
func HASH_XSET(roby *Roby) {
	data, _ := json.Marshal(roby)
	db, put, err := database.GetRedis("default")
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis pool error:%v\033[0m ", err))
		return
	}
	defer database.PutConn(put)

	_, err = db.Do("HSET", "HTTPX-DATA", roby.T_rid, string(data))
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis cmd HSET error:%v\033[0m ", err))
		return
	}
}

// 删除信息
func HASH_XDEL(id string) {
	db, put, err := database.GetRedis("default")
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis pool error:%v\033[0m ", err))
		return
	}
	defer database.PutConn(put)

	_, err = db.Do("HDEL", "HTTPX-DATA", id)
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis cmd HDEL error:%v\033[0m ", err))
		return
	}
}

func RecoveryData() {
	data := HASH_GETALL()
	for _, roby := range data {
		PushTask(roby.T_rid, roby.T_url, roby.T_data, roby.T_timeout, 0)
	}
}

// 恢复信息
func HASH_GETALL() []*Roby {
	db, put, err := database.GetRedis("default")
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis pool error:%v\033[0m ", err))
		return nil
	}
	defer database.PutConn(put)

	data, err := db.Do("HGETALL", "HTTPX-DATA")
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis cmd HGETALL error:%v\033[0m ", err))
		return nil
	}

	if v, ok := data.([]interface{}); ok {
		var resp = make([]*Roby, 0, len(v)/2)
		for i := 0; i < len(v); i += 2 {
			value := v[i+1].([]byte)
			var roby Roby
			json.Unmarshal(value, &roby)
			resp = append(resp, &roby)
		}
		return resp

	}

	return nil
}

func pushToRedisList(resp *PayBody) {
	data, _ := json.Marshal(resp)
	db, put, err := database.GetRedis("default")
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis pool error:%v\033[0m ", err))
		return
	}
	defer database.PutConn(put)

	// 记录错误信息
	_, err = db.Do("lpush", "HTTPX-LIST", string(data))
	if err != nil {
		buf.WriteString(fmt.Sprintf("[ERROR] \033[1;33m redis lpush error:%v\033[0m ", err))
	}
}

func getMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
