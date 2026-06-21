# Dokdo v2.1 테스트 케이스 문서

**Author: Moonsh**
**Status: Draft**
**v1.0 — 최초 작성**

---

## 개요

이 문서는 Dokdo v2.1의 수동 출력 확인용 테스트 케이스다.
각 테스트는 SQL 출력과 args를 콘솔에 출력하여 눈으로 확인한다.

**테스트 방식:**
- `Build()` 호출 후 SQL + args 콘솔 출력
- 예상 SQL과 비교하여 정상 여부 확인
- 에러 케이스는 에러 타입 + 메시지 출력

**변경사항 (스펙 대비):**
- leading UNION ALL 자동 제거 → 제거 안 함 (그대로 출력)

---

## 프로젝트 구조

```
testapp/
  query/
    users.go     ← 타입 선언
    users.kx     ← SQL 쿼리
  main.go        ← 테스트 실행
  go.mod
```

---

## 1. if 조건 분기

### users.kx
```kx
<users>
  <selectUser set:{"users#UserParams"}>
    SELECT * FROM users
    <where>
      [[ if name != nil :{
        AND name = #{name}
      }]]
      [[ if status != nil :{
        AND status = #{status}
      }]]
    </>
  </>
</>
```

### users.go
```go
package query

type UserParams struct {
    Name   *string
    Status *string
}
```

### 케이스 1-A — name만 있을 때
```
params: Name: &"kim", Status: nil

예상 SQL: SELECT * FROM users WHERE name = ?
예상 args: ["kim"]
```

### 케이스 1-B — status만 있을 때
```
params: Name: nil, Status: &"active"

예상 SQL: SELECT * FROM users WHERE status = ?
예상 args: ["active"]
```

### 케이스 1-C — 둘 다 있을 때
```
params: Name: &"kim", Status: &"active"

예상 SQL: SELECT * FROM users WHERE name = ? AND status = ?
예상 args: ["kim", "active"]
```

### 케이스 1-D — 둘 다 nil → WHERE 생략
```
params: Name: nil, Status: nil

예상 SQL: SELECT * FROM users
예상 args: []
```

---

## 2. else if 체인

### users.kx 추가
```kx
<selectGrade set:{"users#GradeParams"}>
  SELECT * FROM users
  [[ if grade == "A" :{
    AND score >= 90
  } else if grade == "B" :{
    AND score >= 80
  } else :{
    AND score >= 70
  }]]
</>
```

### users.go 추가
```go
type GradeParams struct {
    Grade *string
}
```

### 케이스 2-A — grade=A
```
params: Grade: &"A"

예상 SQL: SELECT * FROM users AND score >= 90
예상 args: []
```

### 케이스 2-B — grade=B
```
params: Grade: &"B"

예상 SQL: SELECT * FROM users AND score >= 80
예상 args: []
```

### 케이스 2-C — grade=C → else 분기
```
params: Grade: &"C"

예상 SQL: SELECT * FROM users AND score >= 70
예상 args: []
```

---

## 3. for scalar — trailing comma 제거

### users.kx 추가
```kx
<selectByIds set:{"users#IdsParams"}>
  SELECT * FROM users
  WHERE id IN (
    [[ for id in ids :{
      #{id},
    }]]
  )
</>
```

### users.go 추가
```go
type IdsParams struct {
    Ids []int64
}
```

### 케이스 3-A — 3건
```
params: Ids: []int64{1, 2, 3}

예상 SQL: SELECT * FROM users WHERE id IN ( ?, ?, ? )
예상 args: [1, 2, 3]
확인: 마지막 , 없음
```

### 케이스 3-B — 1건
```
params: Ids: []int64{1}

예상 SQL: SELECT * FROM users WHERE id IN ( ? )
예상 args: [1]
확인: , 없음
```

### 케이스 3-C — 빈 슬라이스
```
params: Ids: []int64{}

예상 SQL: SELECT * FROM users WHERE id IN (  )
예상 args: []
확인: 빈 IN절 출력
```

### 케이스 3-D — nil 슬라이스
```
params: Ids: nil

예상 SQL: SELECT * FROM users WHERE id IN (  )
예상 args: []
```

---

## 4. for []string — 컬럼 나열 + trailing comma 제거

### users.kx 추가
```kx
<selectColumns set:{"users#ColumnParams"}>
  SELECT
  [[ for col in columns :{
    ${col},
  }]]
  FROM users
</>
```

### users.go 추가
```go
type ColumnParams struct {
    Columns []string
}
```

### 케이스 4-A — 3컬럼
```
params: Columns: []string{"id", "name", "status"}

예상 SQL: SELECT id, name, status FROM users
예상 args: []
확인: 마지막 , 없음
```

### 케이스 4-B — 1컬럼
```
params: Columns: []string{"id"}

예상 SQL: SELECT id FROM users
예상 args: []
```

### 케이스 4-C — 빈 슬라이스
```
params: Columns: []string{}

예상 SQL: SELECT  FROM users
예상 args: []
```

---

## 5. for struct — UPDATE SET절

### users.kx 추가
```kx
<updateUser set:{"users#UpdateParams"}>
  UPDATE users SET
  [[ for field in updates :{
    ${field.Key} = #{field.Value},
  }]]
  WHERE id = #{id}
</>
```

### users.go 추가
```go
type UpdateParams struct {
    Id      int64
    Updates []struct {
        Key   string
        Value string
    }
}
```

### 케이스 5-A — 2개 필드
```
params:
  Id: 1
  Updates: []struct{Key, Value string}{{"name", "kim"}, {"status", "active"}}

예상 SQL: UPDATE users SET name = ?, status = ? WHERE id = ?
예상 args: ["kim", "active", 1]
확인: 마지막 , 없음
```

### 케이스 5-B — 빈 슬라이스
```
params:
  Id: 1
  Updates: []struct{Key, Value string}{}

예상 SQL: UPDATE users SET  WHERE id = ?
예상 args: [1]
```

---

## 6. for struct — 벌크 INSERT

### users.kx 추가
```kx
<bulkInsert set:{"users#BulkParams"}>
  INSERT INTO users (name, status)
  VALUES
  [[ for user in users :{
    (#{user.Name}, #{user.Status}),
  }]]
</>
```

### users.go 추가
```go
type BulkParams struct {
    Users []struct {
        Name   string
        Status string
    }
}
```

### 케이스 6-A — 3건
```
params:
  Users: []struct{Name, Status string}{
    {"kim", "active"},
    {"lee", "inactive"},
    {"park", "active"},
  }

예상 SQL: INSERT INTO users (name, status) VALUES (?, ?), (?, ?), (?, ?)
예상 args: ["kim", "active", "lee", "inactive", "park", "active"]
확인: 마지막 , 없음
```

---

## 7. UNION ALL — 자동 제거 없음

### users.kx 추가
```kx
<selectUnion set:{"users#UnionParams"}>
  [[ for id in ids :{
    UNION ALL SELECT * FROM users WHERE id = #{id}
  }]]
</>
```

### users.go 추가
```go
type UnionParams struct {
    Ids []int64
}
```

### 케이스 7-A — 3건
```
params: Ids: []int64{1, 2, 3}

예상 SQL:
  UNION ALL SELECT * FROM users WHERE id = ?
  UNION ALL SELECT * FROM users WHERE id = ?
  UNION ALL SELECT * FROM users WHERE id = ?

예상 args: [1, 2, 3]
확인: 첫 번째 UNION ALL 제거 안 함 — 그대로 출력
```

---

## 8. 이스케이프

### users.kx 추가
```kx
<selectEscape set:{"users#EscapeParams"}>
  SELECT * FROM users
  <where>
    [[ if minScore != nil :{
      AND score \>= #{minScore}
    }]]
    [[ if maxScore != nil :{
      AND score \<= #{maxScore}
    }]]
    [[ if minAmount != nil :{
      AND amount \> #{minAmount}
    }]]
    [[ if maxAmount != nil :{
      AND amount \< #{maxAmount}
    }]]
  </>
</>
```

### users.go 추가
```go
type EscapeParams struct {
    MinScore  *int
    MaxScore  *int
    MinAmount *int64
    MaxAmount *int64
}
```

### 케이스 8-A — 전체 활성
```
params:
  MinScore:  &80
  MaxScore:  &100
  MinAmount: &int64(1000)
  MaxAmount: &int64(9999)

예상 SQL: SELECT * FROM users WHERE score >= ? AND score <= ? AND amount > ? AND amount < ?
예상 args: [80, 100, 1000, 9999]
확인: \>= → >=, \<= → <=, \> → >, \< → <
```

---

## 9. 복수 `<where>` — 서브쿼리

### users.kx 추가
```kx
<selectSubquery set:{"users#SubParams"}>
  SELECT * FROM (
    SELECT * FROM users
    <where>
      [[ if status != nil :{
        AND status = #{status}
      }]]
    </>
  ) sub
  <where>
    [[ if minScore != nil :{
      AND sub.score \>= #{minScore}
    }]]
  </>
</>
```

### users.go 추가
```go
type SubParams struct {
    Status   *string
    MinScore *int
}
```

### 케이스 9-A — 전체 활성
```
params: Status: &"active", MinScore: &80

예상 SQL: SELECT * FROM ( SELECT * FROM users WHERE status = ? ) sub WHERE sub.score >= ?
예상 args: ["active", 80]
```

### 케이스 9-B — 전체 nil → 양쪽 WHERE 모두 생략
```
params: Status: nil, MinScore: nil

예상 SQL: SELECT * FROM ( SELECT * FROM users ) sub
예상 args: []
```

---

## 10. switch/case

### users.kx 추가
```kx
<selectByGrade set:{"users#GradeParams"}>
  SELECT * FROM users
  [[ switch (grade) :{
    case ("A") :{
      WHERE score >= 90
    }
    case ("B") :{
      WHERE score >= 80
    }
  }]]
</>
```

### 케이스 10-A — grade=A
```
params: Grade: &"A"

예상 SQL: SELECT * FROM users WHERE score >= 90
예상 args: []
```

### 케이스 10-B — grade=B
```
params: Grade: &"B"

예상 SQL: SELECT * FROM users WHERE score >= 80
예상 args: []
```

### 케이스 10-C — grade=D (매칭 없음, default 없음)
```
params: Grade: &"D"

예상 SQL: SELECT * FROM users
예상 args: []
```

---

## 11. 중첩 — if 안에 for

### users.kx 추가
```kx
<selectNested set:{"users#NestedParams"}>
  SELECT * FROM users
  [[ if ids != nil :{
    WHERE id IN (
      [[ for id in ids :{
        #{id},
      }]]
    )
  }]]
</>
```

### users.go 추가
```go
type NestedParams struct {
    Ids []int64
}
```

### 케이스 11-A — ids 있음
```
params: Ids: []int64{1, 2, 3}

예상 SQL: SELECT * FROM users WHERE id IN ( ?, ?, ? )
예상 args: [1, 2, 3]
```

### 케이스 11-B — ids nil → if 스킵
```
params: Ids: nil

예상 SQL: SELECT * FROM users
예상 args: []
```

---

## 12. 에러 케이스

### 케이스 12-A — map 직접 전달 → InvalidParamsError
```
Build("users#selectUser", map[string]interface{}{"name": "kim"})

예상 에러: InvalidParamsError
메시지: map is not allowed as params. Use a struct instead.
```

### 케이스 12-B — 존재하지 않는 태그 → TagNotFoundError
```
Build("users#notExist", params)

예상 에러: TagNotFoundError
메시지: tag not found: users#notExist
```

### 케이스 12-C — 외부 타입 참조 → BuildError (Load 시점)
```
// users.go
type BadParams struct {
    Items []CustomType  // 외부 타입 참조
}

Load("query/") 호출 시

예상 에러: BuildError
메시지: field 'Items' uses unsupported type
```

### 케이스 12-D — set:{} 타입 미존재 → BuildError (Load 시점)
```
// users.kx
<selectUser set:{"users#NotExistType"}>

Load("query/") 호출 시

예상 에러: BuildError
메시지: type not found: users#NotExistType
```

### 케이스 12-E — ${}  차단목록 히트 → RuntimeError
```
// Updates[0].Key = "DROP"
Build("users#updateUser", params)

예상 에러: RuntimeError
메시지: blocked SQL keyword: DROP
```

### 케이스 12-F — ${} 허용목록 백틱 없음 → RuntimeError
```
// Updates[0].Key = "STATUS"  (허용목록 예약어, 백틱 없음)
Build("users#updateUser", params)

예상 에러: RuntimeError
메시지: reserved word requires backtick: `STATUS`
```

### 케이스 12-G — 필드 타입 불일치 → TypeMismatchError
```
// users.go: Name *string
// params: Name int (타입 불일치)

Build("users#selectUser", params)

예상 에러: TypeMismatchError
```

### 케이스 12-H — ${}  for 밖 사용 → ParseError (Load 시점)
```
// users.kx
<selectUser set:{"users#UserParams"}>
  SELECT ${col} FROM users   ← for 밖에서 ${}
</>

Load("query/") 호출 시

예상 에러: ParseError
메시지: '${}' is only allowed inside a 'for' statement
```

---

## Claude Code 지시사항

위 케이스를 기반으로 `main.go` 작성:

1. 각 케이스별로 `fmt.Println("=== 케이스 N-X ===")` 출력
2. `Build()` 호출 후 `SQL:`, `Args:` 출력
3. 에러 케이스는 `Error:` + 에러 타입 + 메시지 출력
4. 정상/에러 구분 없이 전체 순서대로 실행
5. 에러 발생해도 panic 없이 다음 케이스 계속 진행
