package mycontext

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestBackgroundIsNotTODO(t *testing.T) {
	bg := fmt.Sprint(Background())
	todo := fmt.Sprint(TODO())

	if todo == bg {
		t.Fatalf("TODO and Background should not be equal: %q vs %q", todo, bg)
	}
}

func TestWithCancel(t *testing.T) {
	ctx, cancel := WithCancel(Background())

	if err := ctx.Err(); err != nil {
		t.Fatalf("Error should be nil first, got %v", err)
	}
	cancel()

	<-ctx.Done()
	if err := ctx.Err(); err != Canceled {
		t.Fatalf("Error should be canceled by now, got %v", err)
	}
}

func TestWithCancelConcurrency(t *testing.T) {
	ctx, cancel := WithCancel(Background())

	time.AfterFunc(1*time.Second, cancel)

	if err := ctx.Err(); err != nil {
		t.Fatalf("Error should be nil first, got %v", err)
	}
	<-ctx.Done()
	if err := ctx.Err(); err != Canceled {
		t.Fatalf("Error should be canceled by now, got %v", err)
	}
}

func TestWithCancelPropagation(t *testing.T) {
	ctxA, cancelA := WithCancel(Background())
	ctxB, cancelB := WithCancel(ctxA)
	defer cancelB()

	cancelA()

	select {
	case <-ctxB.Done():
	case <-time.After(1 * time.Second):
		t.Fatalf("Time out!")
	}

	if err := ctxB.Err(); err != Canceled {
		t.Fatalf("Error should be canceled by now, got %v", err)
	}

}

func TestWithDeadline(t *testing.T) {
	deadline := time.Now().Add(2 * time.Second)
	ctx, cancel := WithDeadline(Background(), deadline)

	if d, ok := ctx.Deadline(); !ok || d != deadline {
		t.Fatalf("Expected deadline %v; got %v", deadline, d)
	}

	then := time.Now()
	<-ctx.Done()
	if d := time.Since(then); math.Abs(d.Seconds()-2.0) > 0.1 {
		t.Fatalf("Should have been done after 2.0 seconds, took %v", d)
	}
	if err := ctx.Err(); err != DeadlineExceeded {
		t.Fatalf("Error should be DeadlineExceeded, got %v", err)
	}

	cancel()
	if err := ctx.Err(); err != DeadlineExceeded {
		t.Fatalf("Error should still be DeadlineExceeded, got %v", err)
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 2 * time.Second
	deadline := time.Now().Add(timeout)
	ctx, cancel := WithTimeout(Background(), timeout)

	if d, ok := ctx.Deadline(); !ok || d.Sub(deadline) > time.Millisecond {
		t.Fatalf("Expected deadline %v; got %v", deadline, d)
	}

	then := time.Now()
	<-ctx.Done()
	if d := time.Since(then); math.Abs(d.Seconds()-2.0) > 0.1 {
		t.Fatalf("Should have been done after 2.0 seconds, took %v", d)
	}
	if err := ctx.Err(); err != DeadlineExceeded {
		t.Fatalf("Error should be DeadlineExceeded, got %v", err)
	}

	cancel()
	if err := ctx.Err(); err != DeadlineExceeded {
		t.Fatalf("Error should still be DeadlineExceeded, got %v", err)
	}
}

func TestWithValue(t *testing.T) {
	tc := []struct {
		key, val, keyRet, valRet interface{}
		shouldPanic              bool
	}{
		{"a", "b", "a", "b", false},
		{"a", "b", "c", nil, false},
		{42, true, 42, true, false},
		{42, true, int64(42), nil, false},
		{nil, true, nil, nil, true},
		{[]int{1, 2, 3}, true, []int{1, 2, 3}, nil, true},
	}

	for _, tt := range tc {
		var panicked interface{}
		func() {
			defer func() { panicked = recover() }()

			ctx := WithValue(Background(), tt.key, tt.val)
			if val := ctx.Value(tt.keyRet); val != tt.valRet {
				t.Errorf("Expected value %v, got %v", tt.valRet, val)
			}
		}()

		if panicked != nil && !tt.shouldPanic {
			t.Errorf("Unexpected panic: %v", panicked)
		}
		if panicked == nil && tt.shouldPanic {
			t.Errorf("Expected panic, but didn't get it")
		}
	}
}

func TestDeadlineExceededIsTimeouter(t *testing.T) {
	f := func(ctx Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	type timeouter interface {
		Timeout() bool
	}

	ctx, cancel := WithCancel(Background())
	cancel()
	err := f(ctx)
	if _, ok := err.(timeouter); ok {
		t.Fatalf("Canceled context should not have Timeout method")
	}

	ctx, cancel = WithTimeout(Background(), 1*time.Millisecond)
	defer cancel()
	err = f(ctx)
	if _, ok := err.(timeouter); !ok {
		t.Fatalf("Deadline exceeded context should have Timeout method")
	}
}


