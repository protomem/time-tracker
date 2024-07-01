package ctxstore

import "context"

type Key string

func (k Key) String() string {
	return string(k)
}

func With[T any](ctx context.Context, key Key, value T) context.Context {
	return context.WithValue(ctx, key, value)
}

func From[T any](ctx context.Context, key Key) (T, bool) {
	value, ok := ctx.Value(key).(T)
	return value, ok
}

func MustFrom[T any](ctx context.Context, key Key) T {
	value, ok := ctx.Value(key).(T)
	if !ok {
		panic("ctxstore: " + key + "  not found")
	}
	return value
}
