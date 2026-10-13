package taskgroup

import "testing"

func TestWithoutSession(t *testing.T) {
	sess, ctx := New(t.Context(), DefaultLimits())
	detached := WithoutSession(ctx)
	if FromContext(detached) != nil {
		t.Fatal("session leaked")
	}
	if FromContext(ctx) != sess {
		t.Fatal("parent session")
	}
	if err := sess.Wait(); err != nil {
		t.Fatal(err)
	}
}
