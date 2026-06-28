package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

func writeKX(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

func writeGo(t *testing.T, dir, relPath, content string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

// TestSetRef_SameDir_DifferentFiles: 같은 디렉토리의 다른 파일 타입을 올바르게 resolve하는지 검증
func TestSetRef_SameDir_DifferentFiles(t *testing.T) {
	dq, err := dokdo.Load("query")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// 검증 2: searchUserByName → users.go#UserSearchParams
	name := "kim"
	sql, args, err := dq.Build("users#searchUserByName", query.UserSearchParams{Name: &name})
	if err != nil {
		t.Fatalf("Build searchUserByName: %v", err)
	}
	if !strings.Contains(sql, "WHERE") {
		t.Errorf("searchUserByName: WHERE absent, sql=%q", sql)
	}
	if len(args) != 1 {
		t.Fatalf("searchUserByName args count: got %d, want 1", len(args))
	}
	if ptr, ok := args[0].(*string); !ok || *ptr != "kim" {
		t.Errorf("searchUserByName args[0]: got %v (%T), want *string(\"kim\")", args[0], args[0])
	}

	// 검증 3: getUserById → common.go#CommonParams
	sql, args, err = dq.Build("users#getUserById", query.CommonParams{Id: 42})
	if err != nil {
		t.Fatalf("Build getUserById: %v", err)
	}
	if !strings.Contains(sql, "id = ?") {
		t.Errorf("getUserById: 'id = ?' absent, sql=%q", sql)
	}
	if len(args) != 1 {
		t.Fatalf("getUserById args count: got %d, want 1", len(args))
	}
	if args[0] != int64(42) {
		t.Errorf("getUserById args[0]: got %v (%T), want int64(42)", args[0], args[0])
	}

	// 검증 4: listUsers → common.go#PageParams
	sql, args, err = dq.Build("users#listUsers", query.PageParams{Limit: 10, Offset: 20})
	if err != nil {
		t.Fatalf("Build listUsers: %v", err)
	}
	if !strings.Contains(sql, "LIMIT ?") {
		t.Errorf("listUsers: 'LIMIT ?' absent, sql=%q", sql)
	}
	if !strings.Contains(sql, "OFFSET ?") {
		t.Errorf("listUsers: 'OFFSET ?' absent, sql=%q", sql)
	}
	if len(args) != 2 {
		t.Fatalf("listUsers args count: got %d, want 2", len(args))
	}
	if args[0] != int(10) {
		t.Errorf("listUsers args[0] (limit): got %v (%T), want int(10)", args[0], args[0])
	}
	if args[1] != int(20) {
		t.Errorf("listUsers args[1] (offset): got %v (%T), want int(20)", args[1], args[1])
	}

	// 검증 5: getUserById(common.go 타입)에 users.go 타입을 넣으면 타입 불일치 에러
	_, _, err = dq.Build("users#getUserById", query.UserSearchParams{Name: &name})
	if err == nil {
		t.Error("getUserById with UserSearchParams: expected error, got nil")
	}
}

// TestSetRef_FileBoundary: set:{"파일명#타입"}에서 파일명이 실제로 검사되는지 증명
// UserSearchParams는 users.go에만 존재 — common#UserSearchParams로 참조하면 Load() 실패해야 함
func TestSetRef_FileBoundary(t *testing.T) {
	dir := t.TempDir()

	writeGo(t, dir, "users.go", `
package query

type UserSearchParams struct {
	Name *string
}
`)
	// common.go는 없고, kx는 common#UserSearchParams를 참조 → 파일 없음
	writeKX(t, dir, "users.kx", `
<users>
  <searchUser set:{"common#UserSearchParams"}>
    SELECT * FROM users WHERE name = #{name}
  </>
</>
`)

	_, err := dokdo.Load(dir)
	if err == nil {
		t.Fatal("Load should fail: UserSearchParams is in users.go, not common.go")
	}
}
