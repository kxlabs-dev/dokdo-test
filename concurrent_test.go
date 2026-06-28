package main

import (
	"sync"
	"testing"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

// TestConcurrentBuild verifies that concurrent Build() calls on a shared *dokdo.Dokdo
// instance do not trigger data races. Run with: go test -race ./...
func TestConcurrentBuild(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	const iterations = 100

	type job struct {
		name   string
		target string
		params func() any
	}

	name := "kim"
	status := "active"
	gradeA := "A"
	nestedStatus := "active"

	jobs := []job{
		{
			name:   "searchUser",
			target: "users#searchUser",
			params: func() any {
				return query.IfParams{Name: &name, Status: &status}
			},
		},
		{
			name:   "selectByIds",
			target: "users#selectByIds",
			params: func() any {
				return query.IdListParams{IdList: []int64{1, 2, 3}}
			},
		},
		{
			name:   "updateUser",
			target: "users#updateUser",
			params: func() any {
				return query.UpdateParams{
					Id: 1,
					Fields: []query.UpdateField{{"name", "kim"}, {"age", "30"}},
				}
			},
		},
		{
			name:   "gradeUser",
			target: "users#gradeUser",
			params: func() any {
				return query.ElseIfParams{Grade: "A"}
			},
		},
		{
			name:   "switchGrade",
			target: "users#switchGrade",
			params: func() any {
				return query.SwitchParams{Grade: &gradeA}
			},
		},
		{
			name:   "nestedIfFor",
			target: "users#nestedIfFor",
			params: func() any {
				return query.NestedParams{Ids: []int64{10, 20, 30}, Status: &nestedStatus}
			},
		},
	}

	var wg sync.WaitGroup

	for _, j := range jobs {
		// 3 goroutines per query type to increase concurrency pressure
		for range 3 {
			wg.Go(func() {
				for range iterations {
					if _, _, err := dq.Build(j.target, j.params()); err != nil {
						t.Errorf("[%s] Build error: %v", j.name, err)
						return
					}
				}
			})
		}
	}

	wg.Wait()
}
