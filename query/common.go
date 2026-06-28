package query

// set:{} 파일 경계 검증용 (common.go)
type CommonParams struct {
	Id int64
}

type PageParams struct {
	Limit  int
	Offset int
}
