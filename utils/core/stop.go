package core

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// IsStopped 通道是否已经停止或close, 非阻塞,
// 如果有多个通道, 任意一个通道停止都会返回true
func IsStopped(stopChannels ...<-chan struct{}) bool {
	for _, stopChan := range stopChannels {
		select {
		case <-stopChan:
			return true
		default:
		}
	}
	return false
}

// IsAllStopped 所有通道是否被关闭
func IsAllStopped(stopChannels ...<-chan struct{}) bool {
	for _, stopChan := range stopChannels {
		select {
		case <-stopChan:
		default:
			return false
		}
	}
	return true
}

// IsContextDone ctx是否被cancel
// 如果有多个通道, 任意一个通道被cancel都会返回true
func IsContextDone(contexts ...context.Context) bool {
	for _, ctx := range contexts {
		select {
		case <-ctx.Done():
			return true
		default:
		}
	}
	return false
}

// IsAllContextDone 所有Context是否被cancelled
func IsAllContextDone(contexts ...context.Context) bool {
	for _, ctx := range contexts {
		select {
		case <-ctx.Done():
		default:
			return false
		}
	}
	return true
}

// WaitForStopped 阻塞等待通道停止或close
func WaitForStopped(stopChan <-chan struct{}) {
	select {
	case <-stopChan:
		return
	}
}

// WaitForStopped2 任意一个chan停止或close则退出
func WaitForStopped2(stopChan1 <-chan struct{}, stopChan2 <-chan struct{}) {
	select {
	case <-stopChan1:
		return
	case <-stopChan2:
		return
	}
}

// WaitForStopped3 任意一个chan停止或close则退出
func WaitForStopped3(stopChan1 <-chan struct{}, stopChan2 <-chan struct{}, stopChan3 <-chan struct{}) {
	select {
	case <-stopChan1:
		return
	case <-stopChan2:
		return
	case <-stopChan3:
		return
	}
}

// WaitForStopped4 任意一个chan停止或close则退出
func WaitForStopped4(stopChan1 <-chan struct{}, stopChan2 <-chan struct{}, stopChan3 <-chan struct{}, stopChan4 <-chan struct{}) {
	select {
	case <-stopChan1:
		return
	case <-stopChan2:
		return
	case <-stopChan3:
		return
	case <-stopChan4:
		return
	}
}

// StopChanToContext 将StopChan转成Cancel Context
// 注意 cancel需要被调取, 不然内存泄露
func StopChanToContext(stopChan <-chan struct{}) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		WaitForStopped2(stopChan, ctx.Done())
		cancel()
	}()

	return ctx, cancel
}

// ListenStopSignal 监听进程退出信号, 结束时回调exitCallback
//  例子:
//  如下为正确做法，除了监听信号会退出ListenStopSignal的协程，在runXXX因为其它原因退出时，也会反向触发停止监听信号，并退出ListenStopSignal协程
//  func runXXX() {
//  	ctx, cancel := context.WithCancel(context.Background())
//  	defer cancel()
//  	ListenStopSignal(ctx, cancel)
//      redis.Keys(ctx, "*") // ctrl+c，会停止redis的任务
//  }
//
//  如果不像上面那么做，ListenStopSignal的协程会阻塞到监听到信号或进程退出：
//  ListenStopSignal(context.Background(), func(){})
func ListenStopSignal(ctx context.Context, exitCallback context.CancelFunc) {
	go func() {
		exitSign := make(chan os.Signal)
		//监听指定信号: 终端断开, ctrl+c, kill, ctrl+/
		signal.Notify(exitSign, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		defer close(exitSign)

		select {
		case <-ctx.Done():
			// 正常退出协程
		case <-exitSign:
			exitCallback()
		}
	}()
}

// ListenContext 监听ctx.Done, 结束时回调exitCallback
func ListenContext(ctx context.Context, exitCallback context.CancelFunc) {
	go func() {
		WaitForStopped(ctx.Done())
		exitCallback()
	}()
}
