package chanUtils

func ClearChan[T any](c <-chan T) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}
