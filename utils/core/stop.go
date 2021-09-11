package core

// IsStopped 通道是否已经停止或close, 非阻塞
func IsStopped(stopChan <-chan bool) bool {
	select {
	case <-stopChan:
		return true
	default:
		return false
	}
}

// WaitForStopped 阻塞等待通道停止或close
func WaitForStopped(stopChan <-chan bool) {
	select {
	case <-stopChan:
		return
	}
}
