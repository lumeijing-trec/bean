// MIT License

// Copyright (c) The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Safe way to execute `go routine` without crashing the parent process while having a `panic`.
package sync_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/retail-ai-inc/bean/v2/config"
	"github.com/retail-ai-inc/bean/v2/sync"
	"github.com/retail-ai-inc/bean/v2/trace"
)

func Test_ResultPool(t *testing.T) {
	config.Bean = &config.Config{
		Sentry: config.Sentry{
			On:               true,
			TracesSampleRate: 1.0,
		},
	}

	var (
		req                = httptest.NewRequest(http.MethodGet, "/hoge", nil)
		taskDur            = time.Duration(50) * time.Millisecond
		shorterDur         = taskDur / 10
		ctxTimeout, cancel = context.WithTimeoutCause(
			context.Background(),
			shorterDur,
			ErrTimeout,
		)
	)
	defer cancel()

	type args struct {
		ctx   context.Context
		opts  []sync.ResultPoolOption
		tasks []func(context.Context) (data, error)
	}
	tests := []struct {
		name        string
		args        args
		wantCertain bool
		want        []data
		errCount    map[resultType]int
	}{
		{
			name: "go tasks with request",
			args: args{
				ctx: context.Background(),
				opts: []sync.ResultPoolOption{
					sync.WithRltRequest(req),
				},
				tasks: genRltTasks(t, taskDur, n, n, n, n, n),
			},
			wantCertain: true,
			want:        []data{{v: 0}, {v: 1}, {v: 2}, {v: 3}, {v: 4}},
			errCount:    nil,
		},
		{
			name: "go tasks with max less than tasks",
			args: args{
				ctx: context.Background(),
				opts: []sync.ResultPoolOption{
					sync.WithRltMaxGoroutines(2),
				},
				tasks: genRltTasks(t, taskDur, n, n, n, n, n),
			},
			wantCertain: true,
			want:        []data{{v: 0}, {v: 1}, {v: 2}, {v: 3}, {v: 4}},
			errCount:    nil,
		},
		{
			name: "go tasks with errors",
			args: args{
				ctx:   context.Background(),
				opts:  nil,
				tasks: genRltTasks(t, taskDur, n, e, n, n, e),
			},
			wantCertain: true,
			want:        []data{{v: 0}, {v: 2}, {v: 3}},
			errCount: map[resultType]int{
				e: 2,
			},
		},
		{
			name: "go tasks with panic",
			args: args{
				ctx:   context.Background(),
				opts:  nil,
				tasks: genRltTasks(t, taskDur, n, n, p, n, n),
			},
			wantCertain: true,
			want:        []data{{v: 0}, {v: 1}, {v: 3}, {v: 4}},
			errCount: map[resultType]int{
				p: 1,
			},
		},
		{
			name: "go tasks with timeout",
			args: args{
				ctx:   ctxTimeout,
				opts:  nil,
				tasks: genRltTasks(t, taskDur, n, n, n, n, n),
			},
			wantCertain: true,
			want:        nil,
			errCount: map[resultType]int{
				pto: 5,
			},
		},
		{
			name: "go tasks with cancel on first error",
			args: args{
				ctx: context.Background(),
				opts: []sync.ResultPoolOption{
					sync.WithRltCancelOnFirstErr(),
				},
				tasks: genRltTasks(t, taskDur, n, n, e, n, n),
			},
			// Due to the cancel on first error option, the results are uncertain.
			// We have no idea which tasks will be done before the first error is returned by a task with error.
			wantCertain: false,
			// Skip assertion for uncertain results
			want: []data{},
			errCount: map[resultType]int{
				e: 1,
			},
		},
		{
			name: "go tasks with errored results",
			args: args{
				ctx: context.Background(),
				opts: []sync.ResultPoolOption{
					sync.WithCollectErroredRlts(),
				},
				tasks: genRltTasks(t, taskDur, n, e, n, n, e),
			},
			wantCertain: true,
			want:        []data{{v: 0}, {v: 1}, {v: 2}, {v: 3}, {v: 4}},
			errCount: map[resultType]int{
				e: 2,
			},
		},
	}
	for _, tt := range tests {
		// tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// t.Parallel()
			// TODO: Investigate why timeout case returns context.Canceled instead of ErrTimeout when running in parallel.

			pool := sync.NewResultPool[data](tt.args.ctx, tt.args.opts...)

			for _, task := range tt.args.tasks {
				pool.Go(task)
			}

			results, err := pool.Wait()
			checkErr(t, tt.errCount, err)

			if tt.wantCertain {
				// Sort results by the `v` field
				sort.Slice(results, func(i, j int) bool {
					return results[i].v < results[j].v
				})

				if !reflect.DeepEqual(results, tt.want) {
					t.Errorf("GoPools() results = %v, want %v", results, tt.want)
				}
			}
		})
	}
}

type data struct {
	v int
}

func genRltTasks(t *testing.T, dur time.Duration, types ...resultType) []func(context.Context) (data, error) {
	t.Helper()

	if dur < 0 {
		t.Fatalf("task dur is less than 0")
	}

	tasks := make([]func(context.Context) (data, error), 0, len(types))

	for i, typ := range types {
		i := i
		switch typ {
		case n:
			tasks = append(tasks, func(c context.Context) (data, error) {
				ctx, finish := trace.StartSpan(c, fmt.Sprintf("task %d", i))
				defer finish()

				select {
				case <-ctx.Done():
					return data{v: i}, context.Cause(ctx)
				default:
				}
				fmt.Printf("task %d started for %v\n", i, dur)
				time.Sleep(dur)
				fmt.Printf("task %d executed\n", i)

				return data{v: i}, nil
			})

		case e:
			tasks = append(tasks, func(c context.Context) (data, error) {
				ctx, finish := trace.StartSpan(c, fmt.Sprintf("task error %d", i))
				defer finish()

				select {
				case <-ctx.Done():
					return data{v: i}, context.Cause(ctx)
				default:
				}
				fmt.Printf("task error %d started\n", i)

				return data{v: i}, ErrTask
			})

		case p:
			tasks = append(tasks, func(c context.Context) (data, error) {
				ctx, finish := trace.StartSpan(c, fmt.Sprintf("task panic %d", i))
				defer finish()

				select {
				case <-ctx.Done():
					return data{v: i}, context.Cause(ctx)
				default:
				}
				fmt.Printf("task panic %d started\n", i)

				panic(PanicMsg)
			})

		default:
			t.Fatalf("unknown task type %d: %s", i, typ)
		}
	}

	return tasks
}
