##### easy_go_pool 特性
```
1.动态扩容至指定size
2.size可异步改变：AdjustSize(workNum uint16)
3.可异步关闭也可同步等待任务完成后关闭
适合对任务处理时间短任务量有限的工作进行处理
```
##### go_pool 特性
```
1.动态扩容至指定size
2.size可异步改变：AdjustSize(workNum uint16)
3.当任务量小于需要时会按设置的间隔时间动态缩容
4.可异步关闭也可同步等待任务完成后关闭
适合对任务处理时间较长任务量不确定的工作进行处理
```
##### 示例代码
```go
package main

import (
	epool "github.com/z-yuanhao/pool/easy_go_pool"
	"log"
	"math/rand"
	"time"
)

type Score struct {
	Num int
}

func (s *Score) Do() {
	if s.Num%10000 == 0 {
		log.Println("num:", s.Num)
	}
	if s.Num%2 == 0 {
		panic(s.Num)
	}
	time.Sleep(time.Millisecond * 100)
}

func main() {
	//创建协程池
	pool := epool.NewWorkerPool(1000, 1100)
	defer pool.Close() //关闭协程池
	pool.OnPanic(func(msg interface{}) {
		//log.Println("error:", msg)
	})
	datanum := 100 * 100 * 10
	for i := 1; i <= datanum; i++ {
		//接收任务
		if err := pool.Accept(&Score{Num: i}); err != nil {
			log.Println("err:\t", err)
			break
		}
		if i%10000 == 0 {
			log.Println("send num:", i)
			randNum := rand.Intn(10) + 1000
			//调整协程池大小
			pool.AdjustSize(uint16(randNum))
		}
	}
}
```

##### benchmark 测试
![](https://raw.githubusercontent.com/z-yuanhao/pool/master/images/1.png)
![](https://raw.githubusercontent.com/z-yuanhao/pool/master/images/2.png)
![](https://raw.githubusercontent.com/z-yuanhao/pool/master/images/3.png)
![](https://raw.githubusercontent.com/z-yuanhao/pool/master/images/4.png)
![](https://raw.githubusercontent.com/z-yuanhao/pool/master/images/5.png)