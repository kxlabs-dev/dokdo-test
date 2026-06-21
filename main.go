package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

func fmtArgs(args []interface{}) string {
	if len(args) == 0 {
		return "[]"
	}
	parts := make([]string, len(args))
	for i, a := range args {
		rv := reflect.ValueOf(a)
		if rv.IsValid() && rv.Kind() == reflect.Ptr && !rv.IsNil() {
			parts[i] = fmt.Sprintf("%v", rv.Elem().Interface())
		} else {
			parts[i] = fmt.Sprintf("%v", a)
		}
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// extractKXQuery reads the .kx file for a given target ("file#tag") and returns
// the indented body content of that tag block.
func extractKXQuery(queryDir, target string) string {
	parts := strings.SplitN(target, "#", 2)
	if len(parts) != 2 {
		return ""
	}
	kxFile := filepath.Join(queryDir, parts[0]+".kx")
	tagName := parts[1]

	data, err := os.ReadFile(kxFile)
	if err != nil {
		return ""
	}
	src := string(data)

	// Find opening tag: <tagName followed by space, >, or newline
	openTag := "<" + tagName
	start := strings.Index(src, openTag)
	if start == -1 {
		return ""
	}
	after := start + len(openTag)
	if after < len(src) && src[after] != ' ' && src[after] != '>' && src[after] != '\n' {
		return "" // false match (e.g. <searchUserExtra when looking for <searchUser)
	}

	// Find end of opening tag line (the closing '>')
	gtIdx := strings.Index(src[start:], ">")
	if gtIdx == -1 {
		return ""
	}
	bodyStart := start + gtIdx + 1
	if bodyStart < len(src) && src[bodyStart] == '\n' {
		bodyStart++
	}

	// Find matching </> by tracking nesting depth.
	// <letter...> increments depth; </> decrements depth.
	depth := 1
	pos := bodyStart
	endIdx := -1

	for pos < len(src) {
		nextLT := strings.Index(src[pos:], "<")
		if nextLT == -1 {
			break
		}
		nextLT += pos
		rest := src[nextLT:]

		if strings.HasPrefix(rest, "</>") {
			depth--
			if depth == 0 {
				endIdx = nextLT
				break
			}
			pos = nextLT + 3
		} else if len(rest) > 1 && ((rest[1] >= 'a' && rest[1] <= 'z') || (rest[1] >= 'A' && rest[1] <= 'Z')) {
			depth++
			pos = nextLT + 1
		} else {
			pos = nextLT + 1
		}
	}

	if endIdx == -1 {
		return ""
	}

	body := strings.TrimRight(src[bodyStart:endIdx], " \t\n")
	lines := strings.Split(body, "\n")

	// Compute minimum indentation across non-empty lines
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent < 0 {
		minIndent = 0
	}

	// Strip minimum indent and re-indent with 2 spaces
	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, "")
			continue
		}
		stripped := line
		if len(line) >= minIndent {
			stripped = line[minIndent:]
		} else {
			stripped = strings.TrimLeft(line, " \t")
		}
		result = append(result, "  "+stripped)
	}
	return strings.Join(result, "\n")
}

func printKX(queryDir, target string) {
	kx := extractKXQuery(queryDir, target)
	if kx != "" {
		fmt.Printf("KX:\n%s\n", kx)
	}
}

func printParams(params interface{}) {
	b, err := json.Marshal(params)
	if err != nil {
		fmt.Printf("Params: (marshal error: %v)\n", err)
		return
	}
	fmt.Printf("Params: %s\n", b)
}

func run(dq *dokdo.Dokdo, queryDir, label, target string, params interface{}) {
	fmt.Printf("=== %s ===\n", label)
	printKX(queryDir, target)
	printParams(params)
	sql, args, err := dq.Build(target, params)
	if err != nil {
		fmt.Printf("Error: %T %v\n\n", err, err)
		return
	}
	fmt.Printf("SQL: %s\n", strings.TrimSpace(sql))
	fmt.Printf("Args: %s\n\n", fmtArgs(args))
}

func main() {
	const queryDir = "query"
	dq, err := dokdo.Load(queryDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Load error: %v\n", err)
		os.Exit(1)
	}

	// ─── 케이스 1: if 분기 ───────────────────────────────────────────
	name := "kim"
	status := "active"

	run(dq, queryDir, "케이스 1-1 name만", "users#searchUser",
		query.IfParams{Name: &name})
	run(dq, queryDir, "케이스 1-2 status만", "users#searchUser",
		query.IfParams{Status: &status})
	run(dq, queryDir, "케이스 1-3 둘다", "users#searchUser",
		query.IfParams{Name: &name, Status: &status})
	run(dq, queryDir, "케이스 1-4 둘다nil", "users#searchUser",
		query.IfParams{})

	// ─── 케이스 2: else if 체인 ──────────────────────────────────────
	run(dq, queryDir, "케이스 2-1 grade A", "users#gradeUser",
		query.ElseIfParams{Grade: "A"})
	run(dq, queryDir, "케이스 2-2 grade B", "users#gradeUser",
		query.ElseIfParams{Grade: "B"})
	run(dq, queryDir, "케이스 2-3 grade C (else)", "users#gradeUser",
		query.ElseIfParams{Grade: "C"})

	// ─── 케이스 3: for scalar trailing comma ─────────────────────────
	run(dq, queryDir, "케이스 3-1 []int64 3건", "users#selectByIds",
		query.IdListParams{IdList: []int64{1, 2, 3}})
	run(dq, queryDir, "케이스 3-2 []int64 1건", "users#selectByIds",
		query.IdListParams{IdList: []int64{42}})
	run(dq, queryDir, "케이스 3-3 빈슬라이스", "users#selectByIds",
		query.IdListParams{IdList: []int64{}})
	run(dq, queryDir, "케이스 3-4 nil", "users#selectByIds",
		query.IdListParams{IdList: nil})

	// ─── 케이스 4: for []string 컬럼나열 ─────────────────────────────
	run(dq, queryDir, "케이스 4-1 3컬럼", "users#selectCols",
		query.ColumnParams{Columns: []string{"id", "name", "email"}})
	run(dq, queryDir, "케이스 4-2 1컬럼", "users#selectCols",
		query.ColumnParams{Columns: []string{"id"}})
	run(dq, queryDir, "케이스 4-3 빈슬라이스", "users#selectCols",
		query.ColumnParams{Columns: []string{}})

	// ─── 케이스 5: for struct UPDATE SET절 ───────────────────────────
	run(dq, queryDir, "케이스 5-1 2필드", "users#updateUser",
		query.UpdateParams{
			Id: 1,
			Fields: []struct {
				Key   string
				Value string
			}{{"name", "kim"}, {"age", "30"}},
		})
	run(dq, queryDir, "케이스 5-2 빈슬라이스", "users#updateUser",
		query.UpdateParams{
			Id: 1,
			Fields: []struct {
				Key   string
				Value string
			}{},
		})

	// ─── 케이스 6: for struct 벌크 INSERT ────────────────────────────
	run(dq, queryDir, "케이스 6-1 3건", "users#bulkInsert",
		query.BulkInsertParams{
			Rows: []struct {
				Name  string
				Email string
				Age   int
			}{
				{"Alice", "alice@example.com", 30},
				{"Bob", "bob@example.com", 25},
				{"Carol", "carol@example.com", 35},
			},
		})

	// ─── 케이스 7: UNION ALL (leading 제거 없음) ──────────────────────
	run(dq, queryDir, "케이스 7-1 UNION ALL 3건", "users#unionAll",
		query.IdListParams{IdList: []int64{10, 20, 30}})

	// ─── 케이스 8: 이스케이프 \>= \<= \> \< ─────────────────────────
	minScore, maxScore, minAge, maxAge := 80, 90, 20, 60
	run(dq, queryDir, "케이스 8-1 전체활성", "users#rangeSearch",
		query.RangeParams{
			MinScore: &minScore,
			MaxScore: &maxScore,
			MinAge:   &minAge,
			MaxAge:   &maxAge,
		})

	// ─── 케이스 9: 복수 where 서브쿼리 ──────────────────────────────
	statusVal := "active"
	minAmount := int64(1000)
	run(dq, queryDir, "케이스 9-1 전체활성", "users#subSearch",
		query.SubqueryParams{Status: &statusVal, MinAmount: &minAmount})
	run(dq, queryDir, "케이스 9-2 전체nil", "users#subSearch",
		query.SubqueryParams{})

	// ─── 케이스 10: switch/case ───────────────────────────────────────
	gradeA, gradeB, gradeC := "A", "B", "C"
	run(dq, queryDir, "케이스 10-1 grade A", "users#switchGrade",
		query.SwitchParams{Grade: &gradeA})
	run(dq, queryDir, "케이스 10-2 grade B", "users#switchGrade",
		query.SwitchParams{Grade: &gradeB})
	run(dq, queryDir, "케이스 10-3 매칭없음 (C)", "users#switchGrade",
		query.SwitchParams{Grade: &gradeC})

	// ─── 케이스 11: 중첩 for-in-if ────────────────────────────────────
	nestedStatus := "active"
	run(dq, queryDir, "케이스 11-1 ids있음", "users#nestedIfFor",
		query.NestedParams{Ids: []int64{1, 2, 3}, Status: &nestedStatus})
	run(dq, queryDir, "케이스 11-2 ids nil", "users#nestedIfFor",
		query.NestedParams{})

	// ─── 케이스 12: 에러케이스 ────────────────────────────────────────

	// 12-1: map 전달
	mapParams := map[string]interface{}{"name": "kim"}
	fmt.Println("=== 케이스 12-1 map전달 ===")
	printKX(queryDir, "users#searchUser")
	printParams(mapParams)
	_, _, err = dq.Build("users#searchUser", mapParams)
	fmt.Printf("Error: %T %v\n\n", err, err)

	// 12-2: 태그없음
	fmt.Println("=== 케이스 12-2 태그없음 ===")
	printKX(queryDir, "users#ghost")
	printParams(nil)
	_, _, err = dq.Build("users#ghost", nil)
	fmt.Printf("Error: %T %v\n\n", err, err)

	// 12-3: 차단목록히트 (SELECT → blocklist)
	blockParams := query.ColumnParams{Columns: []string{"SELECT"}}
	fmt.Println("=== 케이스 12-3 차단목록히트 ===")
	printKX(queryDir, "users#selectCols")
	printParams(blockParams)
	_, _, err = dq.Build("users#selectCols", blockParams)
	fmt.Printf("Error: %T %v\n\n", err, err)

	// 12-4: 허용목록 백틱없음 (status → allowlist, 백틱 없이 사용)
	allowParams := query.ColumnParams{Columns: []string{"status"}}
	fmt.Println("=== 케이스 12-4 허용목록백틱없음 ===")
	printKX(queryDir, "users#selectCols")
	printParams(allowParams)
	_, _, err = dq.Build("users#selectCols", allowParams)
	fmt.Printf("Error: %T %v\n\n", err, err)

	// 12-5: 타입불일치 (StrictParams.Name=string, 전달=*string)
	type mismatchParams struct{ Name *string }
	mismatchName := "kim"
	mmParams := mismatchParams{Name: &mismatchName}
	fmt.Println("=== 케이스 12-5 타입불일치 ===")
	printKX(queryDir, "users#strictUser")
	printParams(mmParams)
	_, _, err = dq.Build("users#strictUser", mmParams)
	fmt.Printf("Error: %T %v\n\n", err, err)

	// 12-6: ${} for밖사용 → parse error at Load time
	fmt.Println("=== 케이스 12-6 ${}for밖사용 ===")
	tmpDir, _ := os.MkdirTemp("", "dokdo-err-*")
	defer os.RemoveAll(tmpDir)
	kxBad := "<errtest>\n  <badRaw>\n    SELECT ${col} FROM users\n  </>\n</>"
	_ = os.WriteFile(filepath.Join(tmpDir, "errtest.kx"), []byte(kxBad), 0o644)
	_, loadErr := dokdo.Load(tmpDir)
	fmt.Printf("Error: %T %v\n\n", loadErr, loadErr)
}
