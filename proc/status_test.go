package proc

import (
	"context"
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
