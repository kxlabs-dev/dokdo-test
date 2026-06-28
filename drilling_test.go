package main

import (
	"strings"
	"testing"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

// TestDrilling_StructAccess: struct 드릴링 정상 동작 확인
// item.Score (기본 타입) + item.Address.City (nested struct 드릴링)
func TestDrilling_StructAccess(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	params := query.DrillParams{
		Items: []query.UserDetailItem{
			{Score: int64(90), Address: query.AddressInfo{City: "Seoul", Country: "KR"}},
			{Score: int64(80), Address: query.AddressInfo{City: "Busan", Country: "KR"}},
		},
	}

	sql, args, err := dq.Build("users#selectWithDrilling", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "score IN") {
		t.Errorf("expected score IN — sql: %q", sql)
	}
	if !strings.Contains(sql, "city IN") {
		t.Errorf("expected city IN — sql: %q", sql)
	}
	if got := strings.Count(sql, "?"); got != 4 {
		t.Errorf("? count: got %d, want 4 — sql: %q", got, sql)
	}
	if len(args) != 4 {
		t.Fatalf("args length: got %d, want 4", len(args))
	}
	if args[0] != int64(90) {
		t.Errorf("args[0]: got %v, want int64(90)", args[0])
	}
	if args[1] != int64(80) {
		t.Errorf("args[1]: got %v, want int64(80)", args[1])
	}
	if args[2] != "Seoul" {
		t.Errorf("args[2]: got %v, want Seoul", args[2])
	}
	if args[3] != "Busan" {
		t.Errorf("args[3]: got %v, want Busan", args[3])
	}
}

// TestDrilling_CircularReference: 순환참조 감지 — Load() 시점 BuildError
func TestDrilling_CircularReference(t *testing.T) {
	dir := t.TempDir()

	writeGo(t, dir, "users.go", `
package query

type NodeB struct {
	Next *NodeA
}

type NodeA struct {
	Items []NodeB
}

type CircularParams struct {
	Root NodeA
}
`)
	writeKX(t, dir, "users.kx", `
<users>
  <selectCircular set:{"users#CircularParams"}>
    SELECT 1
  </>
</>
`)

	_, err := dokdo.Load(dir)
	if err == nil {
		t.Fatal("Load should fail: circular reference NodeA → NodeB → NodeA")
	}
	if !strings.Contains(err.Error(), "circular reference detected") {
		t.Errorf("expected circular reference error, got: %v", err)
	}
}

// TestDrilling_SwitchFor: switch → for
func TestDrilling_SwitchFor(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	params := query.SwitchForParams{
		Type: "bulk",
		Items: []query.BulkInsertRow{
			{Name: "Alice", Email: "alice@example.com", Age: 30},
			{Name: "Bob", Email: "bob@example.com", Age: 25},
		},
	}

	sql, args, err := dq.Build("users#selectSwitchFor", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "INSERT INTO") {
		t.Errorf("expected INSERT INTO — sql: %q", sql)
	}
	if got := strings.Count(sql, "?"); got != 6 {
		t.Errorf("? count: got %d, want 6 — sql: %q", got, sql)
	}
	if len(args) != 6 {
		t.Fatalf("args length: got %d, want 6", len(args))
	}
}

// TestDrilling_ForSwitch: for → switch
func TestDrilling_ForSwitch(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	params := query.ForSwitchParams{
		Items: []query.StatusItem{
			{Id: int64(1), Status: "active"},
			{Id: int64(2), Status: "inactive"},
		},
	}

	sql, args, err := dq.Build("users#selectForSwitch", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "WHERE id IN") {
		t.Errorf("expected WHERE id IN — sql: %q", sql)
	}
	if got := strings.Count(sql, "?"); got != 1 {
		t.Errorf("? count: got %d, want 1 — sql: %q", got, sql)
	}
	if len(args) != 1 {
		t.Fatalf("args length: got %d, want 1", len(args))
	}
	if args[0] != int64(1) {
		t.Errorf("args[0]: got %v, want int64(1)", args[0])
	}
}

// TestDrilling_ForSwitchFor: for → switch → for
func TestDrilling_ForSwitchFor(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	params := query.ForSwitchForParams{
		Groups: []query.IdGroup{
			{Type: "include", Ids: []int64{1, 2, 3}},
		},
	}

	sql, args, err := dq.Build("users#selectForSwitchFor", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "id IN") {
		t.Errorf("expected id IN — sql: %q", sql)
	}
	if got := strings.Count(sql, "?"); got != 3 {
		t.Errorf("? count: got %d, want 3 — sql: %q", got, sql)
	}
	if len(args) != 3 {
		t.Fatalf("args length: got %d, want 3", len(args))
	}
	if args[0] != int64(1) {
		t.Errorf("args[0]: got %v, want int64(1)", args[0])
	}
}

// TestDrilling_NamedStructSlice: named struct 슬라이스 정상 동작 확인
// bulkInsert — #{} 만 사용, named struct 슬라이스 순회
func TestDrilling_NamedStructSlice(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	params := query.BulkInsertParams{
		Rows: []query.BulkInsertRow{
			{Name: "Alice", Email: "alice@example.com", Age: 30},
			{Name: "Bob", Email: "bob@example.com", Age: 25},
		},
	}

	sql, args, err := dq.Build("users#bulkInsert", params)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(sql, "INSERT INTO") {
		t.Errorf("expected INSERT INTO — sql: %q", sql)
	}
	if !strings.Contains(sql, "VALUES") {
		t.Errorf("expected VALUES — sql: %q", sql)
	}
	if got := strings.Count(sql, "?"); got != 6 {
		t.Errorf("? count: got %d, want 6 — sql: %q", got, sql)
	}
	if len(args) != 6 {
		t.Fatalf("args length: got %d, want 6", len(args))
	}
	if args[0] != "Alice" {
		t.Errorf("args[0]: got %v, want Alice", args[0])
	}
	if args[3] != "Bob" {
		t.Errorf("args[3]: got %v, want Bob", args[3])
	}
}
