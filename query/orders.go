package query

type OrderInsertParams struct {
	UserId int64
	Item   string
	Amount int
}

type OrderUpdateParams struct {
	Id      int64
	Updates []struct {
		Key   string
		Value string
	}
}

type OrderIdParams struct {
	Id int64
}

type OrderUserIdParams struct {
	UserId int64
}
