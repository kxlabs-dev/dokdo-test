package main

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

// TestGORMMSSQLQuestionMark verifies that using dokdo.DialectMySQL (? placeholders)
// with a GORM SQL Server connection avoids the NamedExpr @p1 failure.
func TestGORMMSSQLQuestionMark(t *testing.T) {
	mssqlDSN := os.Getenv("DOKDO_TEST_MSSQL_DSN")
	if mssqlDSN == "" {
		t.Skip("DOKDO_TEST_MSSQL_DSN not set")
	}

	dqMySQL, err := dokdo.Load("query", dokdo.DialectMySQL)
	if err != nil {
		t.Fatalf("Load DialectMySQL: %v", err)
	}

	db, err := gorm.Open(sqlserver.Open(mssqlDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open MSSQL: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	pass := func(name string) { fmt.Printf("OK  %s\n", name) }
	fail := func(name string, e error) {
		fmt.Printf("NG  %s — %v\n", name, e)
		t.Errorf("%s FAIL: %v", name, e)
	}

	// ── CASE-1: users#insertUser with ? placeholders via db.Exec ─────────
	s, a, err := dqMySQL.Build("users#insertUser", query.InsertUserParams{
		Name: "qmark-test", Status: "active", Score: 50, Age: 20,
	})
	if err != nil {
		fail("CASE-1 Build", err)
		return
	}
	fmt.Printf("CASE-1 SQL : %s\n", s)
	fmt.Printf("CASE-1 Args: %v\n", a)

	r1 := db.Exec(s, a...)
	if r1.Error != nil {
		fail("CASE-1 db.Exec insertUser (? dialect on MSSQL)", r1.Error)
	} else {
		pass(fmt.Sprintf("CASE-1 db.Exec insertUser (RowsAffected=%d)", r1.RowsAffected))
	}

	// inserted row 조회 후 ID 확보 (cleanup용)
	var insertedID int64
	if r1.Error == nil {
		db.Raw("SELECT TOP 1 id FROM users WHERE name = ? ORDER BY id DESC", "qmark-test").Scan(&insertedID)
	}

	// ── CASE-2: orders#updateOrder with ? placeholders via db.Exec ───────
	// updateOrder는 동적 SET절 — 실제 row가 있어야 의미있지만, SQL 실행 가능 여부만 확인
	// 존재하지 않는 id=0 업데이트는 RowsAffected=0이지만 Error는 없어야 함
	s2, a2, err := dqMySQL.Build("orders#updateOrder", query.OrderUpdateParams{
		Id: 0,
		Updates: []struct{ Key, Value string }{
			{Key: "item", Value: "pencil"},
			{Key: "amount", Value: "1"},
		},
	})
	if err != nil {
		fail("CASE-2 Build", err)
	} else {
		fmt.Printf("CASE-2 SQL : %s\n", s2)
		fmt.Printf("CASE-2 Args: %v\n", a2)
		r2 := db.Exec(s2, a2...)
		if r2.Error != nil {
			fail("CASE-2 db.Exec updateOrder (? dialect on MSSQL)", r2.Error)
		} else {
			pass(fmt.Sprintf("CASE-2 db.Exec updateOrder (RowsAffected=%d)", r2.RowsAffected))
		}
	}

	// cleanup
	if insertedID != 0 {
		sc, ac, _ := dqMySQL.Build("users#deleteUser", query.UserIdParams{Id: insertedID})
		db.Exec(sc, ac...)
	}
}
