package proxy

import (
	"fmt"
	"gitee.com/zhucheer/orange/database"
	"strconv"
	"sync"
	"time"
)

const TFormat = "2006.01.02.15.04"

var _stat = NewStat()

func NewStat() *stat {
	s := &stat{data: map[string]map[string]int64{}}
	s.tickAndPushImage()

	return s
}

type stat struct {
	sync.Mutex
	data map[string]map[string]int64
}

func (s *stat) log(url string) {
	s.Lock()
	defer s.Unlock()

	key := time.Now().Format(TFormat)
	group, ok := s.data[key]
	if !ok {
		group = map[string]int64{}
		s.data[key] = group
	}
	group[url]++
}

func (s *stat) image() (string, map[string]int64) {
	key := time.Now().Add(-time.Minute).Format(TFormat)
	if data, ok := s.getInfo(key); ok {
		return key, data
	}

	return "", nil
}

func (s *stat) tickAndPushImage() {
	go func() {
		tick := time.NewTicker(time.Second)
		for range tick.C {

			// 第一秒记录信息
			if !(time.Now().Unix() % 60 == 1) {
				continue
			}

			func() {
				defer func() {
					recover()
				}()

				fmt.Println(fmt.Sprintf("\033[1;32m INFO [%s] 创建image\033[0m ", time.Now().Format(time.RFC3339)))
				// 获取快照信息
				key, data := s.image()
				if key == "" {
					return
				}
				// 记录到redis
				s.writeToRedis(key, data)
			}()
		}
	}()
}

func (s *stat) writeToRedis(key string, data map[string]int64) {
	db, put, err := database.GetRedis("default")
	if err != nil {
		fmt.Println(fmt.Sprintf("[ERROR] \033[1;33m redis pool error:%v\033[0m ", err))
		return
	}
	defer database.PutConn(put)

	// redis key
	redisKey := "HTTPX-STAT-HSET." + key
	// 记录数据到统计表
	db.Do("HSET", "HTTPX-STAT." + key[0:len("2021.02.03")], redisKey, redisKey)
	// 构建redis指令
	var cmds = make([]interface{}, 0, len(data)*2+1)
	cmds = append(cmds, redisKey)
	for url, count := range data {
		cmds = append(cmds, url)
		cmds = append(cmds, strconv.Itoa(int(count)))
	}

	// 写入redis
	_, err = db.Do("HSET", cmds...)
	if err != nil {
		fmt.Println(fmt.Sprintf("[ERROR] \033[1;33m redis error:%v\033[0m ", err))
	}
}

func (s *stat) getInfo(key string) (map[string]int64, bool) {
	s.Lock()
	defer s.Unlock()
	defer func() {
		delete(s.data, key)
	}()

	data, ok := s.data[key]
	return data, ok
}
