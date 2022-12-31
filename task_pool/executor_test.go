package task_pool

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestExecutor(t *testing.T) {

	t.Logf("初始程数: %d", runtime.NumGoroutine())

	executor, _ := NewExecutor(NewExecutorParams(2, 1*time.Second, "测试"), logger.NewDefaultLogger())

	now := time.Now()
	executor.Submit(func(ctx context.Context) {
		time.Sleep(1 * time.Second)
		t.Logf("task 1 , 应该1s 实际: %.4f", time.Since(now).Seconds())
		t.Logf("task 1, 协程数: %d", runtime.NumGoroutine())
	}, func(ctx context.Context) {
		time.Sleep(1 * time.Second)
		t.Logf("task 2, 应该1s, 实际: %.4f", time.Since(now).Seconds())
		t.Logf("task 2, 协程数: %d", runtime.NumGoroutine())
	}, func(ctx context.Context) {
		time.Sleep(2 * time.Second)
		t.Logf("task 3 应该3s, 实际: %.4fs", time.Since(now).Seconds())
		t.Logf("task 3, 协程数: %d", runtime.NumGoroutine())
	})
	executor.SubmitWithTimeout(func(ctx context.Context) {
		time.Sleep(4_500 * time.Millisecond)
		t.Logf("task 4, 会被强制终止")
	}, 3*time.Second)
	t.Logf("wait前协程数: %d", runtime.NumGoroutine())

	executor.Wait()
	t.Logf("wait后协程数: %d", runtime.NumGoroutine())

	executor.Stop()

	t.Logf("stop后协程数: %d, 因为task4还在运行, 应该比初始协程多1", runtime.NumGoroutine())

	time.Sleep(1_500 * time.Millisecond)

	t.Logf("程序结束时协程数: %d, 此时应该有task 4的消息打印", runtime.NumGoroutine())

}
