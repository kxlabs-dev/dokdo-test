# dokdo — Integration Test Suite

`github.com/kxlabs-dev/dokdo v0.2.0-alpha` 기준 통합 검증 레포지토리.

4개 DB(MySQL · PostgreSQL · Oracle · SQL Server)와 GORM 인터페이스를 대상으로  
CRUD, 동적 SQL, 인젝션 방어, 동시성을 실제 DB 커넥션으로 검증한다.

---

## 테스트 대상 라이브러리

| 항목 | 값 |
|---|---|
| 라이브러리 | `github.com/kxlabs-dev/dokdo` |
| 검증 버전 | `v0.2.0-alpha` |
| Go 버전 | `1.26.4` |

---

## 검증 DB

| DB | 드라이버 | DSN 환경변수 |
|---|---|---|
| MySQL | `github.com/go-sql-driver/mysql` | `DOKDO_TEST_MYSQL_DSN` |
| PostgreSQL | `github.com/jackc/pgx/v5/stdlib` | `DOKDO_TEST_PG_DSN` |
| Oracle | `github.com/sijms/go-ora/v2` | `DOKDO_TEST_ORACLE_DSN` |
| SQL Server | `github.com/microsoft/go-mssqldb` | `DOKDO_TEST_MSSQL_DSN` |

DSN을 하나도 설정하지 않으면 DB 연동 테스트는 자동으로 skip된다.

---

## 쿼리 파일 구조

```
query/
  users.kx     # users 테이블 CRUD + 동적 SQL
  orders.kx    # orders 테이블 CRUD + 동적 SQL
  query.go     # 파라미터 타입 정의
```

### users.kx 주요 쿼리

| 태그 | 기능 |
|---|---|
| `searchUser` | `if` 분기로 name/status 동적 WHERE |
| `gradeUser` | `else if` 체인으로 등급별 score 조건 |
| `selectByIds` | `for` 루프로 IN 절 생성 |
| `selectCols` | `for` + `${}` (RawParam)으로 동적 컬럼 나열 |
| `updateUser` | `for` + `${}` (RawParam)으로 동적 SET 절 |
| `bulkInsert` | `for struct`로 다건 VALUES 생성 |
| `rangeSearch` | `\>=` `\<=` 이스케이프 + 다중 `if` |
| `subSearch` | `<where>` 중첩으로 서브쿼리 동적 조건 |
| `switchGrade` | `switch/case`로 등급 분기 |
| `nestedIfFor` | `if` 안에 `for` 중첩 |
| `insertUser` | 기본 INSERT (MySQL·Oracle·MSSQL 공통) |
| `insertUserPg` | INSERT + `RETURNING id` (PostgreSQL) |
| `insertUserMssql` | INSERT + `OUTPUT INSERTED.id` (SQL Server) |
| `deleteUser` | `#{id}` 단건 DELETE |

### orders.kx 주요 쿼리

| 태그 | 기능 |
|---|---|
| `insertOrder` | 기본 INSERT |
| `insertOrderPg` | INSERT + `RETURNING id` |
| `insertOrderMssql` | INSERT + `OUTPUT INSERTED.id` |
| `updateOrder` | `for` + `${}` 동적 SET + `WHERE id` |
| `selectOrdersByUser` | user_id 기준 SELECT |
| `deleteOrder` | 단건 DELETE |

---

## 테스트 파일

### `db_verify_test.go` — database/sql 직접 CRUD

`database/sql` 표준 인터페이스로 4개 DB 전체를 검증한다.

- dialect별 별도 `Dokdo` 인스턴스 로드 (플레이스홀더 자동 적용)
- DB별 INSERT ID 획득 방식 분기:

  | DB | 방식 |
  |---|---|
  | MySQL | `res.LastInsertId()` |
  | PostgreSQL | `RETURNING id` + `QueryRow.Scan` |
  | Oracle | INSERT 후 `SELECT MAX(id)` (dokdo 쿼리) |
  | SQL Server | `OUTPUT INSERTED.id` + `QueryRow.Scan` |

**검증 케이스 (DB당 11건)**

| 케이스 | 내용 |
|---|---|
| CREATE-1 | users INSERT |
| CREATE-2 | orders INSERT |
| READ-3-1 | searchUser — name 조건만 |
| READ-3-2 | searchUser — status 조건만 |
| READ-3-3 | searchUser — name + status 동시 |
| READ-4 | selectByIds — IN 절 슬라이스 |
| READ-5 | selectOrdersByUser — JOIN 없이 user_id 검색 |
| UPDATE-6 | updateOrder — 동적 SET (2 컬럼) |
| UPDATE-7 | updateUser — name 단일 컬럼 |
| DELETE-8 | deleteOrder |
| DELETE-9 | deleteUser |

---

### `gorm_verify_test.go` — GORM 인터페이스 CRUD

`gorm.DB`의 `db.Raw(sql, args...).Scan()` / `db.Exec(sql, args...)`가  
dokdo가 생성한 SQL + args를 그대로 받아 동작하는지 검증한다.

**GORM dialect 선택 기준**

| DB | dokdo 인스턴스 | 이유 |
|---|---|---|
| MySQL | `DialectMySQL` (`?`) | 기본 |
| PostgreSQL | `DialectPostgres` (`$1,$2...`) | 기본 |
| Oracle | `DialectOracle` (`:1,:2...`) | 기본 |
| **SQL Server** | **`DialectMySQL` (`?`)** | GORM 우회 — 아래 참조 |

> **GORM + SQL Server 주의사항**
>
> GORM은 SQL에 `@`가 포함되면 `clause.NamedExpr` 경로로 분기한다
> (`finisher_api.go:773`, `chainable_api.go:465`).  
> `DialectSQLServer`가 생성하는 `@p1, @p2, ...` 플레이스홀더는  
> `NamedExpr.Build`의 namedMap 조회에서 매핑되지 않아 `stmt.Vars`가  
> 비어진 채로 `ExecContext`에 전달되므로  
> SQL Server가 `"Must declare the scalar variable '@p1'"` 오류를 반환한다.
>
> **해결**: `DialectMySQL`(`?` 플레이스홀더)을 사용하면 GORM이  
> `clause.Expr` 경로로 처리해 args가 정상 바인딩된다.  
> SQL Server INSERT ID는 `OUTPUT INSERTED.id` + `db.Raw().Scan()`으로  
> 단일 쿼리에서 원자적으로 획득한다 (SCOPE_IDENTITY는 커넥션 풀에서  
> 다른 커넥션이 배정될 경우 NULL을 반환하므로 사용하지 않는다).

**검증 케이스 (DB당 4건)**

| 케이스 | 내용 |
|---|---|
| CASE-1 | `searchUser` — 동적 WHERE + `Raw().Scan()` |
| CASE-2 | `selectByIds` — IN 절 슬라이스 Scan |
| CASE-3 | `insertUser` — `db.Exec` RowsAffected 검증 |
| CASE-4 | `updateOrder` — 동적 SET + `Raw().Scan()`으로 결과 검증 |

---

### `injection_test.go` — RawParam (`${}`) 인젝션 방어 검증

`${}` (RawParam) 경로에 악의적 페이로드를 주입해 `validateRaw`가  
모두 차단하는지 검증한다. PASS-THROUGH는 결함으로 처리(`t.Errorf`).

**컨텍스트**

| 컨텍스트 | 경로 | 쿼리 |
|---|---|---|
| `selectCols` | `${col}` | `users#selectCols` |
| `updateUser` | `${field.Key}` | `users#updateUser` |

**페이로드 분류 (44건)**

| 분류 | 페이로드 예시 | 차단 기준 |
|---|---|---|
| C1 대소문자 변형 | `SeLeCt`, `dRoP` | blocklist 대소문자 무관 |
| C2 공백·탭·개행 | `SE LECT`, `SE\tLECT` | rawPattern 불일치 |
| C3 SQL 주석 | `SELECT--`, `SELECT/**/` | `--` 연속 하이픈 / rawPattern |
| C4 멀티 스테이트먼트 | `id; DROP TABLE users;--` | rawPattern 불일치 |
| C5 allowlist 덧붙임 | `status OR 1=1` | rawPattern 불일치 |
| C6 백틱 멀티페어 | `` `status` OR `1`=`1` `` | btCount ≠ 2 |
| C7 전각 유니코드 | `ＤＲＯＰ`, `` `ＤＲＯＰ` `` | non-ASCII 거부 |
| C8 경계값 | `\x00`, `""`, 10000자 | rawPattern / 길이 제한(128) |
| C9 주석 분할 키워드 | `SE--LECT`, `id.DROP` | `--` 검사 / 세그먼트 blocklist |

**최종 결과: BLOCKED 44 / PASSED-THROUGH 0**

---

### `concurrent_test.go` — 동시성

여러 goroutine이 동시에 `Build`를 호출해도 경쟁 조건이 없는지 검증.

---

## 실행

```bash
# 환경변수 설정 (예: .env 파일)
export DOKDO_TEST_MYSQL_DSN="user:pass@tcp(host:3306)/dbname"
export DOKDO_TEST_PG_DSN="postgres://user:pass@host:5432/dbname?sslmode=disable"
export DOKDO_TEST_ORACLE_DSN="oracle://user:pass@host:1521/FREEPDB1"
export DOKDO_TEST_MSSQL_DSN="sqlserver://user:pass@host:1433?database=dbname"

# 전체 테스트
go test -v ./...

# 특정 테스트만
go test -v -run TestDBVerify ./...
go test -v -run TestGORMVerify ./...
go test -v -run TestInjection ./...
go test -v -run TestConcurrentBuild ./...
```

---

## 최종 결과 요약

| 테스트 | 항목 | 결과 |
|---|---|---|
| TestConcurrentBuild | 동시성 빌드 | PASS |
| TestDBVerify | MySQL 11 케이스 | PASS |
| TestDBVerify | PostgreSQL 11 케이스 | PASS |
| TestDBVerify | Oracle 11 케이스 | PASS |
| TestDBVerify | SQL Server 11 케이스 | PASS |
| TestGORMVerify | MySQL 4 케이스 | PASS |
| TestGORMVerify | PostgreSQL 4 케이스 | PASS |
| TestGORMVerify | Oracle 4 케이스 | PASS |
| TestGORMVerify | SQL Server 4 케이스 | PASS |
| TestInjection | 44 페이로드 전량 차단 | PASS |
