package proc

import (
	"context"
	"errors"
	"testing"
)

func TestInterpreterStatusDoesNotInventAnOSProcess(t *testing.T) {
	for _, status := range []int{0, 7} {
		m := NewManager()
		info, err := m.Start(t.Context(), Spec{Name: "interpreter"}, AttachFunc(func(context.Context) (*Attached, error) {
			return &Attached{Shape: ShapeFunc, Wait: func() Result { return Result{Status: &status} }}, nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		final := waitFinished(t, m, info.ID)
		want := StateCompleted
		if status != 0 {
			want = StateFailed
		}
		if final.State != want || final.ExitStatus() != status || final.Status == nil || *final.Status != status || final.Proc != nil {
			t.Fatalf("status %d: final session = %+v", status, final)
		}
	}
}

func TestInterpreterErrorPreservedAlongsideStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		err    error
		state  State
		reason string
	}{
		{"zero with error", 0, errors.New("interpreter I/O failure"), StateFailed, "interpreter I/O failure"},
		{"nonzero with error", 7, errors.New("execution failed"), StateFailed, "execution failed"},
		{"canceled", 0, context.Canceled, StateKilled, "canceled"},
		{"deadline", 0, context.DeadlineExceeded, StateKilled, "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewManager()
			info, err := m.Start(t.Context(), Spec{Name: "interpreter"}, AttachFunc(func(context.Context) (*Attached, error) {
				return &Attached{Shape: ShapeFunc, Wait: func() Result {
					return Result{Status: &tc.status, Err: tc.err}
				}}, nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			final := waitFinished(t, m, info.ID)
			if final.State != tc.state || final.Reason != tc.reason || final.ExitStatus() != tc.status {
				t.Fatalf("unexpected terminal result: %+v", final)
			}
		})
	}
}
