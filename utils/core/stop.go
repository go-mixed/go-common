package core

import "context"

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
