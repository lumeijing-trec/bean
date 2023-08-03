package helpers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

var (
	singleFlightGroup = new(singleflight.Group)
	waitTime          = time.Duration(100) * time.Millisecond
	maxWaitTime       = time.Duration(2000) * time.Millisecond
)

func SingleDo[T any](ctx context.Context, key string, call func() (T, error), retry uint, ttl ...time.Duration) (data T, err error) {
	result, e, _ := singleFlightGroup.Do(key, func() (result interface{}, err error) {
		if len(ttl) > 0 {
			forgetTimer := time.AfterFunc(ttl[0], func() {
				singleFlightGroup.Forget(key)
			})
			defer forgetTimer.Stop()
		}

		for i := 0; i <= int(retry); i++ {
			result, err = call()
			if err == nil {
				return result, nil
			}
			if i == int(retry) {
				return nil, err
			}

			waitTime := JitterBackoff(waitTime, maxWaitTime, i)
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				switch {
				case errors.Is(err, context.DeadlineExceeded) && errors.Is(ctx.Err(), context.DeadlineExceeded):
					fallthrough
				case errors.Is(err, context.Canceled) && errors.Is(ctx.Err(), context.Canceled):
					return nil, err
				default:
					return nil, errors.WithMessage(err, ctx.Err().Error())
				}
			}
		}
		return nil, err
	})

	if e != nil {
		err = e
		return
	}

	val, ok := result.(T)
	if !ok {
		err = errors.WithStack(fmt.Errorf("expected type %T but got type %T", data, result))
		return
	}
	return val, nil
}
