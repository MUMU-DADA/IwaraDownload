package middleware

import "context"

type Context struct {
	Next    HandlerFunc     // 执行下一个中间件
	Context context.Context // 上下文
	End     func()          // 提前结束
}

// HandlerFunc 中间件函数
type HandlerFunc func(ctx Context) error

// Control 控制器
type Control struct {
	middlewareFuncs []HandlerFunc   // 中间件函数
	mainFunc        HandlerFunc     // 主函数
	endFuncs        []HandlerFunc   // 结束函数
	ctx             context.Context // 上下文
}

// Use 添加中间件
func (c *Control) Use(handleFuncs ...HandlerFunc) {
	for _, v := range handleFuncs {
		c.middlewareFuncs = append(c.middlewareFuncs, v)
	}
}

// UseEnd 添加结束函数
func (c *Control) UseEnd(handleFuncs ...HandlerFunc) {
	for _, v := range handleFuncs {
		c.endFuncs = append(c.endFuncs, v)
	}
}

// Run 执行
func (c *Control) Run() error {
	// 执行中间件
	for i, v := range c.middlewareFuncs {
		var end bool
		var next HandlerFunc
		if i == len(c.middlewareFuncs)-1 {
			next = func(ctx Context) error {
				return nil
			}
		} else {
			next = func(ctx Context) error {
				return v(Context{
					Next:    c.middlewareFuncs[i+1],
					Context: ctx.Context,
				})
			}
		}
		err := v(Context{
			Next: next,
			End: func() {
				end = true
			},
			Context: c.ctx,
		})
		if err != nil {
			return err
		}
		if end {
			return nil
		}
	}

	// 执行主函数
	var end bool
	err := c.mainFunc(Context{
		Next: func(ctx Context) error {
			return nil
		},
		End: func() {
			end = true
		},
		Context: c.ctx,
	})
	if err != nil {
		return err
	}
	if end {
		return nil
	}

	// 执行结束函数
	for i, v := range c.endFuncs {
		var end bool
		var next HandlerFunc
		if i == len(c.endFuncs)-1 {
			next = func(ctx Context) error {
				return nil
			}
		} else {
			next = func(ctx Context) error {
				return v(Context{
					Next:    c.endFuncs[i+1],
					Context: ctx.Context,
				})
			}
		}
		err := v(Context{
			Next: next,
			End: func() {
				end = true
			},
			Context: c.ctx,
		})
		if err != nil {
			return err
		}
		if end {
			return nil
		}
	}
	return nil
}

// NewControl 创建控制器
func NewControl(ctx context.Context, mainFunc HandlerFunc) *Control {
	return &Control{
		middlewareFuncs: make([]HandlerFunc, 0),
		mainFunc:        mainFunc,
		ctx:             ctx,
	}
}
