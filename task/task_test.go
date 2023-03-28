package task

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func compareDuration(expect time.Duration, real time.Duration) error {
	if real < expect || real > expect+100*time.Millisecond {
		return fmt.Errorf("expect %dms, got %dms", expect.Milliseconds(), real.Milliseconds())
	}

	return nil
}

func TestTask(t *testing.T) {
	task := NewTask(2)
	now := time.Now()
	task.SetJobDoneHandler(func(j *Job) {
		if j.Error != nil {
			t.Logf("panic error: %v", j.Error)
		}
	})

	task.Submit(func(ctx context.Context) {
		time.Sleep(100 * time.Millisecond)

		real := time.Since(now)
		if err := compareDuration(100*time.Millisecond, real); err != nil {
			t.Errorf("A: %v", err)
		}
		t.Logf("%s: A, delta: %dms", time.Now(), real.Milliseconds())
	})
	// B运行时（500ms），A-B-C依次运行（总300ms）
	task.Submit(func(ctx context.Context) {
		time.Sleep(500 * time.Millisecond)

		real := time.Since(now)
		if err := compareDuration(500*time.Millisecond, real); err != nil {
			t.Errorf("B: %v", err)
		}

		panic(fmt.Errorf("B panic, delta: %dms", real.Milliseconds()))
	})
	// 因为并发限制，A任务执行完毕之后才执行C，即C的时间应该是200ms
	task.Submit(func(ctx context.Context) {
		time.Sleep(100 * time.Millisecond)

		real := time.Since(now)
		if err := compareDuration(200*time.Millisecond, real); err != nil {
			t.Errorf("C: %v", err)
		}
		t.Logf("%s: C, delta: %dms", time.Now(), real.Milliseconds())
	})
	// 因为并发限制，C任务执行完毕之后才执行D，即D的时间应该是300ms
	task.Submit(func(ctx context.Context) {
		time.Sleep(100 * time.Millisecond)
		real := time.Since(now)
		if err := compareDuration(300*time.Millisecond, real); err != nil {
			t.Errorf("D: %v", err)
		}
		t.Logf("%s: D, delta: %dms", time.Now(), time.Since(now).Milliseconds())
	})
	// E任务在D任务之后执行，但是E任务不会被执行，因为E是在300+500秒后，而shutdown是在700ms之后执行
	task.Submit(func(ctx context.Context) {
		time.Sleep(500 * time.Millisecond)
		t.Errorf("E: should not be execution")
		t.Logf("%s: E", time.Now()) // Stop之后，还来不及运行
	})

	go func() {
		time.Sleep(500 * time.Millisecond)
		task.Shutdown(200 * time.Millisecond)
	}()

	task.RunServe()
	t.Logf("%s:shutdown after 700ms, delta: %dms", time.Now(), time.Since(now).Milliseconds())

}
