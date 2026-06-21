package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	gormoracle "github.com/godoes/gorm-oracle"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

// gormUserRow: SELECT * FROM users 결과를 Scan할 구조체.
// gorm:"column:..." 태그로 Oracle의 대문자 컬럼명(ID, NAME, ...)을 처리.
type gormUserRow struct {
	Id     int64  `gorm:"column:id"`
	Name   string `gorm:"column:name"`
	Status string `gorm:"column:status"`
	Score  int    `gorm:"column:score"`
	Age    int    `gorm:"column:age"`
}

// gormOrderRow: SELECT * FROM orders 결과를 Scan할 구조체.
type gormOrderRow struct {
	Id     int64  `gorm:"column:id"`
	UserId int64  `gorm:"column:user_id"`
	Item   string `gorm:"column:item"`
	Amount int    `gorm:"column:amount"`
}

func TestGORMVerify(t *testing.T) {
	mysqlDSN  := os.Getenv("DOKDO_TEST_MYSQL_DSN")
	pgDSN     := os.Getenv("DOKDO_TEST_PG_DSN")
	oracleDSN := os.Getenv("DOKDO_TEST_ORACLE_DSN")
	mssqlDSN  := os.Getenv("DOKDO_TEST_MSSQL_DSN")

	if mysqlDSN == "" && pgDSN == "" && oracleDSN == "" && mssqlDSN == "" {
		t.Skip("no DB DSN set")
	}

	dqMySQL, err := dokdo.Load("query", dokdo.DialectMySQL)
	if err != nil { t.Fatalf("Load MySQL: %v", err) }
	dqPG, err := dokdo.Load("query", dokdo.DialectPostgres)
	if err != nil { t.Fatalf("Load PG: %v", err) }
	dqOracle, err := dokdo.Load("query", dokdo.DialectOracle)
	if err != nil { t.Fatalf("Load Oracle: %v", err) }
	if _, err = dokdo.Load("query", dokdo.DialectSQLServer); err != nil { t.Fatalf("Load MSSQL: %v", err) }

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}

	type gormCase struct {
		label  string
		opener func() (*gorm.DB, error)
		dq     *dokdo.Dokdo
		kind   dbKind
	}

	var cases []gormCase
	if mysqlDSN != "" {
		cases = append(cases, gormCase{
			label:  "GORM/MySQL",
			opener: func() (*gorm.DB, error) { return gorm.Open(mysql.Open(mysqlDSN), gormCfg) },
			dq:     dqMySQL,
			kind:   kindMySQL,
		})
	}
	if pgDSN != "" {
		cases = append(cases, gormCase{
			label:  "GORM/PG",
			opener: func() (*gorm.DB, error) { return gorm.Open(postgres.Open(pgDSN), gormCfg) },
			dq:     dqPG,
			kind:   kindPG,
		})
	}
	if oracleDSN != "" {
		cases = append(cases, gormCase{
			label:  "GORM/Oracle",
			opener: func() (*gorm.DB, error) { return gorm.Open(gormoracle.Open(oracleDSN), gormCfg) },
			dq:     dqOracle,
			kind:   kindOracle,
		})
	}
	if mssqlDSN != "" {
		cases = append(cases, gormCase{
			label:  "GORM/MSSQL",
			opener: func() (*gorm.DB, error) { return gorm.Open(sqlserver.Open(mssqlDSN), gormCfg) },
			dq:     dqMySQL, // ? placeholders — @p1 style triggers GORM NamedExpr bug
			kind:   kindMSSQL,
		})
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			db, err := c.opener()
			if err != nil {
				t.Errorf("%s: gorm.Open FAIL — %v", c.label, err)
				return
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Errorf("%s: db.DB() FAIL — %v", c.label, err)
				return
			}
			defer sqlDB.Close()
			runGORMCases(t, c.label, db, c.dq, c.kind)
		})
	}
}

func runGORMCases(t *testing.T, label string, db *gorm.DB, dq *dokdo.Dokdo, kind dbKind) {
	t.Helper()

	pass := func(name string) { fmt.Printf("%s: %s OK\n", label, name) }
	fail := func(name string, err error) {
		fmt.Printf("%s: %s FAIL — %v\n", label, name, err)
		t.Errorf("%s: %s FAIL: %v", label, name, err)
	}

	mustBuild := func(name, target string, params interface{}) (string, []interface{}) {
		s, a, err := dq.Build(target, params)
		if err != nil {
			fail(name+" (Build)", err)
		}
		return s, a
	}

	// gormInsertUser: DB별 INSERT 후 ID 반환 (GORM 인터페이스 사용)
	gormInsertUser := func(name string, params query.InsertUserParams) int64 {
		switch kind {
		case kindMySQL:
			s, a := mustBuild(name, "users#insertUser", params)
			if r := db.Exec(s, a...); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			var id int64
			if r := db.Raw("SELECT LAST_INSERT_ID()").Scan(&id); r.Error != nil {
				fail(name+" (LAST_INSERT_ID)", r.Error)
				return 0
			}
			return id

		case kindPG:
			s, a := mustBuild(name, "users#insertUserPg", params)
			var id int64
			if r := db.Raw(s, a...).Scan(&id); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			return id

		case kindOracle:
			s, a := mustBuild(name, "users#insertUser", params)
			if r := db.Exec(s, a...); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			ms, ma := mustBuild(name+" (maxId)", "users#selectUserMaxId", params)
			var nid sql.NullInt64
			if r := db.Raw(ms, ma...).Scan(&nid); r.Error != nil {
				fail(name+" (maxId)", r.Error)
				return 0
			}
			if !nid.Valid {
				fail(name, fmt.Errorf("id=NULL"))
				return 0
			}
			return nid.Int64

		case kindMSSQL:
			// ? placeholders (DialectMySQL) → no @ → GORM uses clause.Expr → args bound correctly.
			// OUTPUT INSERTED.id returns ID atomically without SCOPE_IDENTITY() cross-connection issue.
			s, a := mustBuild(name, "users#insertUserMssql", params)
			var id int64
			if r := db.Raw(s, a...).Scan(&id); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			return id
		}
		return 0
	}

	// gormInsertOrder: DB별 INSERT orders 후 ID 반환
	gormInsertOrder := func(name string, params query.OrderInsertParams) int64 {
		switch kind {
		case kindMySQL:
			s, a := mustBuild(name, "orders#insertOrder", params)
			if r := db.Exec(s, a...); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			var id int64
			if r := db.Raw("SELECT LAST_INSERT_ID()").Scan(&id); r.Error != nil {
				fail(name+" (LAST_INSERT_ID)", r.Error)
				return 0
			}
			return id

		case kindPG:
			s, a := mustBuild(name, "orders#insertOrderPg", params)
			var id int64
			if r := db.Raw(s, a...).Scan(&id); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			return id

		case kindOracle:
			s, a := mustBuild(name, "orders#insertOrder", params)
			if r := db.Exec(s, a...); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			ms, ma := mustBuild(name+" (maxId)", "orders#selectOrderMaxId", params)
			var nid sql.NullInt64
			if r := db.Raw(ms, ma...).Scan(&nid); r.Error != nil {
				fail(name+" (maxId)", r.Error)
				return 0
			}
			if !nid.Valid {
				fail(name, fmt.Errorf("id=NULL"))
				return 0
			}
			return nid.Int64

		case kindMSSQL:
			s, a := mustBuild(name, "orders#insertOrderMssql", params)
			var id int64
			if r := db.Raw(s, a...).Scan(&id); r.Error != nil {
				fail(name, r.Error)
				return 0
			}
			return id
		}
		return 0
	}

	// ── SETUP: SELECT 테스트용 데이터 삽입 ────────────────────────────────
	userID1 := gormInsertUser("setup user1", query.InsertUserParams{Name: "gorm-alice", Status: "active", Score: 85, Age: 30})
	userID2 := gormInsertUser("setup user2", query.InsertUserParams{Name: "gorm-bob", Status: "inactive", Score: 70, Age: 25})
	userID3 := gormInsertUser("setup user3", query.InsertUserParams{Name: "gorm-carol", Status: "active", Score: 90, Age: 35})
	orderID := gormInsertOrder("setup order", query.OrderInsertParams{UserId: userID1, Item: "keyboard", Amount: 100})

	defer func() {
		for _, uid := range []int64{userID1, userID2, userID3} {
			if uid == 0 {
				continue
			}
			s, a := mustBuild("cleanup user", "users#deleteUser", query.UserIdParams{Id: uid})
			db.Exec(s, a...)
		}
		if orderID != 0 {
			s, a := mustBuild("cleanup order", "orders#deleteOrder", query.OrderIdParams{Id: orderID})
			db.Exec(s, a...)
		}
	}()

	if userID1 == 0 || userID2 == 0 || userID3 == 0 || orderID == 0 {
		fail("SETUP", fmt.Errorf("insert failed (uid1=%d uid2=%d uid3=%d oid=%d)", userID1, userID2, userID3, orderID))
		return
	}

	// ── CASE-1: SELECT searchUser — 동적 WHERE + GORM Raw().Scan() ─────────
	name := "gorm-alice"
	s, a := mustBuild("CASE-1", "users#searchUser", query.IfParams{Name: &name})
	var found []gormUserRow
	if r := db.Raw(s, a...).Scan(&found); r.Error != nil {
		fail("CASE-1 SELECT searchUser", r.Error)
	} else if len(found) >= 1 && found[0].Name == "gorm-alice" {
		pass("CASE-1 SELECT searchUser (dynamic WHERE)")
	} else {
		firstName := ""
		if len(found) > 0 {
			firstName = found[0].Name
		}
		fail("CASE-1 SELECT searchUser", fmt.Errorf("got %d rows, first.Name=%q", len(found), firstName))
	}

	// ── CASE-2: SELECT selectByIds — for IN절 + 슬라이스 Scan ─────────────
	s, a = mustBuild("CASE-2", "users#selectByIds", query.IdListParams{IdList: []int64{userID1, userID2, userID3}})
	var byIds []gormUserRow
	if r := db.Raw(s, a...).Scan(&byIds); r.Error != nil {
		fail("CASE-2 SELECT selectByIds", r.Error)
	} else if len(byIds) == 3 {
		pass("CASE-2 SELECT selectByIds (IN clause slice scan)")
	} else {
		fail("CASE-2 SELECT selectByIds", fmt.Errorf("want 3, got %d", len(byIds)))
	}

	// ── CASE-3: INSERT via db.Exec — RowsAffected 검증 ────────────────────
	// base insertUser 쿼리 사용 (dialect별 플레이스홀더 자동 적용, RETURNING 없음)
	s, a = mustBuild("CASE-3", "users#insertUser", query.InsertUserParams{
		Name: "gorm-extra", Status: "active", Score: 50, Age: 20,
	})
	r3 := db.Exec(s, a...)
	if r3.Error != nil {
		fail("CASE-3 INSERT via db.Exec", r3.Error)
	} else if r3.RowsAffected == 1 {
		pass("CASE-3 INSERT via db.Exec (RowsAffected=1)")
	} else {
		fail("CASE-3 INSERT via db.Exec", fmt.Errorf("RowsAffected=%d", r3.RowsAffected))
	}
	// CASE-3 extra 행 정리: searchUser로 ID 찾아서 deleteUser
	extraName := "gorm-extra"
	se, ae := mustBuild("CASE-3 cleanup find", "users#searchUser", query.IfParams{Name: &extraName})
	var extras []gormUserRow
	if r := db.Raw(se, ae...).Scan(&extras); r.Error == nil {
		for _, u := range extras {
			sd, ad := mustBuild("CASE-3 cleanup del", "users#deleteUser", query.UserIdParams{Id: u.Id})
			db.Exec(sd, ad...)
		}
	}

	// ── CASE-4: UPDATE updateOrder (동적 SET) + Raw().Scan() 으로 검증 ─────
	s, a = mustBuild("CASE-4 update", "orders#updateOrder", query.OrderUpdateParams{
		Id: orderID,
		Updates: []struct{ Key, Value string }{
			{Key: "item", Value: "monitor"},
			{Key: "amount", Value: "200"},
		},
	})
	if r4 := db.Exec(s, a...); r4.Error != nil {
		fail("CASE-4 UPDATE updateOrder", r4.Error)
	} else {
		vs, va := mustBuild("CASE-4 verify", "orders#selectOrderById", query.OrderIdParams{Id: orderID})
		var orders []gormOrderRow
		if rv := db.Raw(vs, va...).Scan(&orders); rv.Error != nil {
			fail("CASE-4 UPDATE verify", rv.Error)
		} else if len(orders) == 1 && orders[0].Item == "monitor" && orders[0].Amount == 200 {
			pass("CASE-4 UPDATE updateOrder (dynamic SET, verified via Raw+Scan)")
		} else {
			item, amount := "", 0
			if len(orders) > 0 {
				item, amount = orders[0].Item, orders[0].Amount
			}
			fail("CASE-4 UPDATE updateOrder", fmt.Errorf("want item=monitor amount=200, got item=%q amount=%d (rows=%d)", item, amount, len(orders)))
		}
	}
}
