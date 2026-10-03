package cmdlog

import "context"

type resultKey struct{}

// ResultBox holds the value a command wants written on its log out line.
type ResultBox struct {
	v any
}

// Set records v. A nil box is a no-op.
func (b *ResultBox) Set(v any) {
	if b == nil {
		return
	}
	b.v = v
}

// Get returns the noted value, or nil when nothing was noted.
func (b *ResultBox) Get() any {
	if b == nil {
		return nil
	}
	return b.v
}

// WithResult returns a context the command handler can note a result on.
func WithResult(ctx context.Context, box *ResultBox) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, resultKey{}, box)
}

// Note stores v on the ResultBox in ctx. Missing context or box is ignored.
func Note(ctx context.Context, v any) {
	if ctx == nil {
		return
	}
	box, _ := ctx.Value(resultKey{}).(*ResultBox)
	box.Set(v)
}
