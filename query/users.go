package query

// Case 1: if 분기
type IfParams struct {
	Name   *string
	Status *string
}

// Case 2: else if 체인
type ElseIfParams struct {
	Grade string
}

// Case 3, 7: for scalar ([]int64)
type IdListParams struct {
	IdList []int64
}

// Case 4: for []string 컬럼나열
type ColumnParams struct {
	Columns []string
}

type UpdateField struct {
	Key   string
	Value string
}

type BulkInsertRow struct {
	Name  string
	Email string
	Age   int
}

// Case 5: for struct UPDATE SET절
type UpdateParams struct {
	Id     int64
	Fields []UpdateField
}

// Case 6: for struct 벌크 INSERT
type BulkInsertParams struct {
	Rows []BulkInsertRow
}

// Case 8: 이스케이프 \>= \<= \> \<
type RangeParams struct {
	MinScore *int
	MaxScore *int
	MinAge   *int
	MaxAge   *int
}

// Case 9: 복수 where 서브쿼리
type SubqueryParams struct {
	Status    *string
	MinAmount *int64
}

// Case 10: switch/case
type SwitchParams struct {
	Grade *string
}

// Case 11: 중첩 for-in-if
type NestedParams struct {
	Ids    []int64
	Status *string
}

// Case 12-5: 타입불일치 검증용
type StrictParams struct {
	Name string
}

// DB 검증용: INSERT users
type InsertUserParams struct {
	Name   string
	Status string
	Score  int
	Age    int
}

// DB 검증용: single-id SELECT / DELETE
type UserIdParams struct {
	Id int64
}

// set:{} 파일 경계 검증용 (users.go)
type UserSearchParams struct {
	Name *string
}

// struct 드릴링 테스트용
type AddressInfo struct {
	City    string
	Country string
}

type UserDetailItem struct {
	Score   int64
	Address AddressInfo
}

type DrillParams struct {
	Items []UserDetailItem
}

// switch → for 테스트용
type SwitchForParams struct {
	Type  string
	Items []BulkInsertRow
}

// for → switch 테스트용
type StatusItem struct {
	Id     int64
	Status string
}

type ForSwitchParams struct {
	Items []StatusItem
}

// for → switch → for 테스트용
type IdGroup struct {
	Type string
	Ids  []int64
}

type ForSwitchForParams struct {
	Groups []IdGroup
}

