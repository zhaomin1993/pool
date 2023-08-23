package pool

import "github.com/zhaomin1993/pool/internal"

type Pool interface {
	OnPanic(onPanic func(msg interface{}))
	Accept(job internal.Job) (err error)
	Len() uint16
	AdjustSize(workNum uint16)
	Pause()
	Continue(num uint16)
	Close()
}
