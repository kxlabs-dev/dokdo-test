package main

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/sijms/go-ora/v2"
	"github.com/joho/godotenv"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

func TestMain(m *testing.M) {
	_ = godotenv.Load()
	os.Exit(m.Run())
}

type dbKind int

const (
	kindMySQL  dbKind = iota
	kindPG
	kindOracle
	kindMSSQL
)

type dbCase struct {
	label  string
	driver string
	dsn    string
	dq     *dokdo.Dokdo
	kind   dbKind
}

func TestDBVerify(t *testing.T) {
	mysqlDSN  := os.Getenv("DOKDO_TEST_MYSQL_DSN")
	pgDSN     := os.Getenv("DOKDO_TEST_PG_DSN")
	oracleDSN := os.Getenv("DOKDO_TEST_ORACLE_DSN")
	mssqlDSN  := os.Getenv("DOKDO_TEST_MSSQL_DSN")

	if mysqlDSN == "" && pgDSN == "" && oracleDSN == "" && mssqlDSN == "" {
		t.Skip("no DB DSN set")
	}

	dqMySQL, err := dokdo.Load("query", dokdo.DialectMySQL)
	if err != nil {
		t.Fatalf("Load MySQL: %v", err)
	}
	dqPG, err := dokdo.Load("query", dokdo.DialectPostgres)
	if err != nil {
		t.Fatalf("Load PG: %v", err)
	}
	dqOracle, err := dokdo.Load("query", dokdo.DialectOracle)
	if err != nil {
		t.Fatalf("Load Oracle: %v", err)
	}
	dqMSSQL, err := dokdo.Load("query", dokdo.DialectSQLServer)
	if err != nil {
		t.Fatalf("Load MSSQL: %v", err)
	}

	var cases []dbCase
	if mysqlDSN != "" {
		cases = append(cases, dbCase{"MySQL", "mysql", mysqlDSN, dqMySQL, kindMySQL})
	}
	if pgDSN != "" {
		cases = append(cases, dbCase{"PG", "pgx", pgDSN, dqPG, kindPG})
	}
	if oracleDSN != "" {
		cases = append(cases, dbCase{"Oracle", "oracle", oracleDSN, dqOracle, kindOracle})
	}
	if mssqlDSN != "" {
		cases = append(cases, dbCase{"MSSQL", "sqlserver", mssqlDSN, dqMSSQL, kindMSSQL})
	}

	for _, c := range cases {
		c := c
		t.Run(c.label, func(t *testing.T) {
			db, err := sql.Open(c.driver, c.dsn)
			if err != nil {
				t.Errorf("%s: open: %v", c.label, err)
				return
			}
			defer db.Close()
			if err := db.Ping(); err != nil {
				t.Errorf("%s: ping: %v", c.label, err)
				return
			}
			runCRUD(t, c.label, db, c.dq, c.kind)
		})
	}
}

func runCRUD(t *testing.T, label string, db *sql.DB, dq *dokdo.Dokdo, kind dbKind) {
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

	// insertUser: DB별로 INSERT 후 ID 획득 방식이 다름
	insertUser := func(name string, params query.InsertUserParams) int64 {
		switch kind {
		case kindMySQL:
			s, a := mustBuild(name, "users#insertUser", params)
			res, err := db.Exec(s, a...)
			if err != nil {
				fail(name, err)
				return 0
			}
			id, err := res.LastInsertId()
			if err != nil {
				fail(name+" (LastInsertId)", err)
				return 0
			}
			if id == 0 {
				fail(name, fmt.Errorf("id=0"))
			}
			return id

		case kindPG:
			s, a := mustBuild(name, "users#insertUserPg", params)
			var id int64
			if err := db.QueryRow(s, a...).Scan(&id); err != nil {
				fail(name, err)
				return 0
			}
			if id == 0 {
				fail(name, fmt.Errorf("id=0"))
			}
			return id

		case kindOracle:
			s, a := mustBuild(name, "users#insertUser", params)
			if _, err := db.Exec(s, a...); err != nil {
				fail(name, err)
				return 0
			}
			ms, ma := mustBuild(name+" (selectMaxId)", "users#selectUserMaxId", params)
			var nid sql.NullInt64
			if err := db.QueryRow(ms, ma...).Scan(&nid); err != nil {
				fail(name+" (selectMaxId)", err)
				return 0
			}
			if !nid.Valid || nid.Int64 == 0 {
				fail(name, fmt.Errorf("id=NULL or 0"))
				return 0
			}
			return nid.Int64

		case kindMSSQL:
			s, a := mustBuild(name, "users#insertUserMssql", params)
			var id int64
			if err := db.QueryRow(s, a...).Scan(&id); err != nil {
				fail(name, err)
				return 0
			}
			if id == 0 {
				fail(name, fmt.Errorf("id=0"))
			}
			return id
		}
		return 0
	}

	// insertOrder: DB별 INSERT 후 ID 획득
	insertOrder := func(name string, params query.OrderInsertParams) int64 {
		switch kind {
		case kindMySQL:
			s, a := mustBuild(name, "orders#insertOrder", params)
			res, err := db.Exec(s, a...)
			if err != nil {
				fail(name, err)
				return 0
			}
			id, err := res.LastInsertId()
			if err != nil {
				fail(name+" (LastInsertId)", err)
				return 0
			}
			return id

		case kindPG:
			s, a := mustBuild(name, "orders#insertOrderPg", params)
			var id int64
			if err := db.QueryRow(s, a...).Scan(&id); err != nil {
				fail(name, err)
				return 0
			}
			return id

		case kindOracle:
			s, a := mustBuild(name, "orders#insertOrder", params)
			if _, err := db.Exec(s, a...); err != nil {
				fail(name, err)
				return 0
			}
			ms, ma := mustBuild(name+" (selectMaxId)", "orders#selectOrderMaxId", params)
			var nid sql.NullInt64
			if err := db.QueryRow(ms, ma...).Scan(&nid); err != nil {
				fail(name+" (selectMaxId)", err)
				return 0
			}
			if !nid.Valid || nid.Int64 == 0 {
				fail(name, fmt.Errorf("id=NULL or 0"))
				return 0
			}
			return nid.Int64

		case kindMSSQL:
			s, a := mustBuild(name, "orders#insertOrderMssql", params)
			var id int64
			if err := db.QueryRow(s, a...).Scan(&id); err != nil {
				fail(name, err)
				return 0
			}
			return id
		}
		return 0
	}

	countRows := func(name, sqlStr string, args []interface{}) int {
		rows, err := db.Query(sqlStr, args...)
		if err != nil {
			fail(name+" (Query)", err)
			return -1
		}
		defer rows.Close()
		n := 0
		for rows.Next() {
			n++
		}
		if err := rows.Err(); err != nil {
			fail(name+" (rows.Err)", err)
			return -1
		}
		return n
	}

	// ── CREATE-1: users INSERT ──────────────────────────────────────────────
	userID1 := insertUser("CREATE-1 users INSERT", query.InsertUserParams{
		Name: "alice", Status: "active", Score: 85, Age: 30,
	})
	if userID1 != 0 {
		pass("CREATE-1 users INSERT")
	}

	// ── CREATE-2: orders INSERT ─────────────────────────────────────────────
	orderID := insertOrder("CREATE-2 orders INSERT", query.OrderInsertParams{
		UserId: userID1, Item: "keyboard", Amount: 100,
	})
	if orderID != 0 {
		pass("CREATE-2 orders INSERT")
	}

	// READ-4용 추가 users 2건
	user2ID := insertUser("pre-insert user2", query.InsertUserParams{Name: "bob", Status: "inactive", Score: 70, Age: 25})
	user3ID := insertUser("pre-insert user3", query.InsertUserParams{Name: "carol", Status: "active", Score: 90, Age: 35})

	// ── READ-3: users 동적 WHERE ────────────────────────────────────────────
	name := "alice"
	s, a := mustBuild("READ-3-1 name only", "users#searchUser", query.IfParams{Name: &name})
	if n := countRows("READ-3-1 name only", s, a); n >= 1 {
		pass("READ-3-1 name only")
	} else if n >= 0 {
		fail("READ-3-1 name only", fmt.Errorf("rows=%d", n))
	}

	status := "active"
	s, a = mustBuild("READ-3-2 status only", "users#searchUser", query.IfParams{Status: &status})
	if n := countRows("READ-3-2 status only", s, a); n >= 1 {
		pass("READ-3-2 status only")
	} else if n >= 0 {
		fail("READ-3-2 status only", fmt.Errorf("rows=%d", n))
	}

	s, a = mustBuild("READ-3-3 name+status", "users#searchUser", query.IfParams{Name: &name, Status: &status})
	if n := countRows("READ-3-3 name+status", s, a); n >= 1 {
		pass("READ-3-3 name+status")
	} else if n >= 0 {
		fail("READ-3-3 name+status", fmt.Errorf("rows=%d", n))
	}

	// ── READ-4: users IN 절 for loop ───────────────────────────────────────
	s, a = mustBuild("READ-4 IN ids", "users#selectByIds", query.IdListParams{
		IdList: []int64{userID1, user2ID, user3ID},
	})
	if n := countRows("READ-4 IN ids", s, a); n == 3 {
		pass("READ-4 IN ids")
	} else if n >= 0 {
		fail("READ-4 IN ids", fmt.Errorf("want 3, got %d", n))
	}

	// ── READ-5: orders by user_id ──────────────────────────────────────────
	s, a = mustBuild("READ-5 orders by user_id", "orders#selectOrdersByUser", query.OrderUserIdParams{UserId: userID1})
	if n := countRows("READ-5 orders by user_id", s, a); n == 1 {
		pass("READ-5 orders by user_id")
	} else if n >= 0 {
		fail("READ-5 orders by user_id", fmt.Errorf("want 1, got %d", n))
	}

	// ── UPDATE-6: orders 동적 SET ──────────────────────────────────────────
	s, a = mustBuild("UPDATE-6 orders dynamic SET", "orders#updateOrder", query.OrderUpdateParams{
		Id: orderID,
		Updates: []struct{ Key, Value string }{
			{Key: "item", Value: "monitor"},
			{Key: "amount", Value: "200"},
		},
	})
	if _, err := db.Exec(s, a...); err != nil {
		fail("UPDATE-6 orders dynamic SET", err)
	} else {
		vs, va := mustBuild("UPDATE-6 verify", "orders#selectOrderById", query.OrderIdParams{Id: orderID})
		rows, err := db.Query(vs, va...)
		if err != nil {
			fail("UPDATE-6 orders verify", err)
		} else {
			var id, userID int64
			var item string
			var amount int
			ok := false
			if rows.Next() {
				_ = rows.Scan(&id, &userID, &item, &amount)
				ok = item == "monitor" && amount == 200
			}
			rows.Close()
			if ok {
				pass("UPDATE-6 orders dynamic SET")
			} else {
				fail("UPDATE-6 orders dynamic SET", fmt.Errorf("item=%q amount=%d", item, amount))
			}
		}
	}

	// ── UPDATE-7: users 단일 필드 ──────────────────────────────────────────
	s, a = mustBuild("UPDATE-7 users name", "users#updateUser", query.UpdateParams{
		Id:     userID1,
		Fields: []struct{ Key, Value string }{{Key: "name", Value: "alice-v2"}},
	})
	if _, err := db.Exec(s, a...); err != nil {
		fail("UPDATE-7 users name", err)
	} else {
		vs, va := mustBuild("UPDATE-7 verify", "users#selectUserById", query.UserIdParams{Id: userID1})
		rows, err := db.Query(vs, va...)
		if err != nil {
			fail("UPDATE-7 users verify", err)
		} else {
			var id int64
			var uname, ustatus string
			var score, age int
			ok := false
			if rows.Next() {
				_ = rows.Scan(&id, &uname, &ustatus, &score, &age)
				ok = uname == "alice-v2"
			}
			rows.Close()
			if ok {
				pass("UPDATE-7 users name")
			} else {
				fail("UPDATE-7 users name", fmt.Errorf("name=%q", uname))
			}
		}
	}

	// ── DELETE-8: orders ───────────────────────────────────────────────────
	s, a = mustBuild("DELETE-8 orders", "orders#deleteOrder", query.OrderIdParams{Id: orderID})
	if _, err := db.Exec(s, a...); err != nil {
		fail("DELETE-8 orders", err)
	} else {
		vs, va := mustBuild("DELETE-8 verify", "orders#selectOrderById", query.OrderIdParams{Id: orderID})
		if n := countRows("DELETE-8 verify", vs, va); n == 0 {
			pass("DELETE-8 orders")
		} else if n >= 0 {
			fail("DELETE-8 orders", fmt.Errorf("want 0 rows, got %d", n))
		}
	}

	// ── DELETE-9: users (INSERT한 3건 전부) ────────────────────────────────
	allOK := true
	for _, uid := range []int64{userID1, user2ID, user3ID} {
		s, a = mustBuild(fmt.Sprintf("DELETE-9 users id=%d", uid), "users#deleteUser", query.UserIdParams{Id: uid})
		if _, err := db.Exec(s, a...); err != nil {
			fail(fmt.Sprintf("DELETE-9 users id=%d", uid), err)
			allOK = false
		}
	}
	if allOK {
		vs, va := mustBuild("DELETE-9 verify", "users#selectUserById", query.UserIdParams{Id: userID1})
		if n := countRows("DELETE-9 verify", vs, va); n == 0 {
			pass("DELETE-9 users")
		} else if n >= 0 {
			fail("DELETE-9 users", fmt.Errorf("want 0 rows, got %d", n))
		}
	}
}
