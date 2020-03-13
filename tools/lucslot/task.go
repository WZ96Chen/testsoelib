package lucslot

/**
  插槽消息： 主要用于精准执行 延迟任务的 环形插槽
  作者：     卢闯  2019-08-29
*/
import (
	"errors"
	"sync"
	"testsoelib/tools/ants"
	"time"
)

//执行的任务函数
type TaskFunc func(args ...interface{})

//任务
type Task struct {
	//转动圈数： 如果需要延迟到1小时候执行的任务 就会加圈
	turnNum int
	unique  bool //任务是不是全局唯一
	exec    TaskFunc
	params  []interface{}
}
type Slots [3600]map[string]*Task

//任务添加控制
var lock sync.Mutex

func (as *ArchetypeSlots) taskLoop() {
	lock.Lock()
	defer lock.Unlock()
	curIndex := as.curIndex
	//开启异步: 防止跳帧抖动 保重精确执行任务的关键
	ants.SubmitTask(func() {
		//异步下：单个map 任务25000 异常率0%
		tasks := as.slots[curIndex]
		if len(tasks) > 0 {
			//fmt.Println(fmt.Sprintf("卡槽：%d 执行前任务数量：%d", curIndex, len(tasks)))
			//遍历任务，判断任务循环次数等于0，则运行任务
			//否则任务循环次数减1
			for k, v := range tasks {
				if v.turnNum == 0 {
					//fmt.Println(fmt.Sprintf("定时任务：%s 已执行", k))
					//异步执行任务
					go v.exec(v.params...)
					delete(tasks, k)
					if v.unique {
						delete(as.globAllKeys, k)
					}
				} else {
					v.turnNum--
				}
			}
			//fmt.Println(fmt.Sprintf("卡槽：%d 执行后任务数量：%d", curIndex, len(tasks)))
		}
		//else {
		//	if curIndex == 50 {
		//		as.curIndex = 0
		//	}
		//	fmt.Println(fmt.Sprintf("卡槽：%d 已经没有任务", curIndex))
		//}
	})
}

func (as *ArchetypeSlots) AddTask(t time.Time, key string, exec TaskFunc, params []interface{}, unique bool) error {
	lock.Lock()
	defer lock.Unlock()
	if as.firstRunTime.After(t) { //任务执行不能小于任务运行时间
		return errors.New("时间错误")
	}
	//当前时间与指定时间相差秒数
	subSecond := t.Unix() - as.firstRunTime.Unix()
	//计算圈数：圈数不能加1 取整即可
	turnNum := int(subSecond / 3600)
	//任务在插槽中的索引
	ix := subSecond % 3600
	//加入相应的任务
	tasks := as.slots[ix]
	if _, ok := tasks[key]; ok {
		return errors.New("该slots中已存在key为" + key + "的任务")
	}
	if unique {
		if _, ok := as.globAllKeys[key]; ok {
			return errors.New("此任务" + key + "不能重复加入")
		}
		as.globAllKeys[key] = key
	}
	tasks[key] = &Task{
		turnNum: turnNum,
		exec:    exec,
		params:  params,
		unique:  unique,
	}
	return nil
}
