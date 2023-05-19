package internal

// job 工作接口
type Job interface {
	Do() //不允许永远阻塞,代码要求高
}
