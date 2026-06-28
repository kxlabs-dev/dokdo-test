package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kxlabs-dev/dokdo"
	"testapp/query"
)

func TestInjection(t *testing.T) {
	dq, err := dokdo.Load("query", dokdo.DialectMySQL)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	type tc struct {
		label   string
		payload string
	}

	// selectCols 용 페이로드: ${col} 경로
	colPayloads := []tc{
		// ── Case 1: 대소문자 변형 ─────────────────────────────────────────
		{"C1 SeLeCt", "SeLeCt"},
		{"C1 DROP", "DROP"},
		{"C1 dRoP", "dRoP"},
		{"C1 DeLeTe", "DeLeTe"},

		// ── Case 2: 공백/탭/개행 ─────────────────────────────────────────
		{"C2 SE LECT (space)", "SE LECT"},
		{"C2 SE\\tLECT (tab)", "SE\tLECT"},
		{"C2 SE\\nLECT (newline)", "SE\nLECT"},
		{"C2 DR OP (space)", "DR OP"},

		// ── Case 3: SQL 주석 ──────────────────────────────────────────────
		{"C3 SELECT-- (hyphen comment)", "SELECT--"},
		{"C3 SELECT/**/ (block comment)", "SELECT/**/"},
		{"C3 /*!50000DROP*/ (mysql vers comment)", "/*!50000DROP*/"},
		// hyphen은 rawPattern에 허용됨 — 이 조합이 blocklist를 우회하는지 확인
		{"C3 SELECT- (single hyphen suffix)", "SELECT-"},
		{"C3 DROP-x (hyphen + char)", "DROP-x"},

		// ── Case 4: 멀티 스테이트먼트 ────────────────────────────────────
		{"C4 id; DROP TABLE users;--", "id; DROP TABLE users;--"},
		{"C4 name;DELETE FROM users", "name;DELETE FROM users"},

		// ── Case 5: allowlist 항목에 페이로드 덧붙임 ─────────────────────
		{"C5 status; DROP TABLE users", "status; DROP TABLE users"},
		{"C5 status OR 1=1", "status OR 1=1"},
		{"C5 status' OR '1'='1", "status' OR '1'='1"},

		// ── Case 6: 백틱 멀티-페어 (backtick outer pair bypass) ──────────
		// validateRaw는 HasPrefix(`) && HasSuffix(`) 이면 inner만 blocklist 검사
		// 중간에 추가 백틱+페이로드가 있어도 outer pair로만 판단됨
		{"C6 `status` OR `1`=`1`", "`status` OR `1`=`1`"},
		{"C6 `status` UNION SELECT * FROM users--`", "`status` UNION SELECT * FROM users--`"},
		{"C6 `id` DROP TABLE users--`", "`id` DROP TABLE users--`"},
		// 블락되어야 하는 정상 케이스도 함께 확인
		{"C6 `SELECT` (should block)", "`SELECT`"},
		{"C6 `DROP` (should block)", "`DROP`"},

		// ── Case 7: 유니코드/전각 문자로 키워드 위장 ─────────────────────
		// 전각: U+FF24 U+FF32 U+FF2F U+FF30 = ＤＲＯＰ
		{"C7 fullwidth DROP (ＤＲＯＰ)", "ＤＲＯＰ"},
		// 전각 SELECT
		{"C7 fullwidth SELECT (ＳＥＬＥＣＴ)", "ＳＥＬＥＣＴ"},
		// 백틱으로 감싼 전각
		{"C7 `ＤＲＯＰ` backtick fullwidth", "`ＤＲＯＰ`"},

		// ── Case 8: NULL 바이트 / 빈 문자열 / 매우 긴 문자열 ─────────────
		{"C8 null byte", "\x00"},
		{"C8 empty string", ""},
		{"C8 long string 10000 chars", strings.Repeat("a", 10000)},

		// ── Case 9: 주석으로 끊은 blocklist 키워드 ───────────────────────
		{"C9 SELE/**/CT", "SELE/**/CT"},
		{"C9 SE--LECT", "SE--LECT"},
		// dot은 rawPattern에 허용됨 — dot-notation으로 키워드 우회 시도
		{"C9 id.DROP (dot notation)", "id.DROP"},
		{"C9 id.SELECT (dot notation)", "id.SELECT"},
		{"C9 SELECT.-- (dot+hyphen)", "SELECT.--"},
		// 연속 하이픈으로 SELECT 뒤에 주석 시작
		{"C9 SELECT--- (triple hyphen)", "SELECT---"},
		// 숫자 단독 (literal 1 in SELECT context)
		{"C9 1 (numeric literal)", "1"},
		{"C9 1.5 (dotted numeric)", "1.5"},
	}

	// updateUser 용 페이로드: ${field.Key} 경로
	fieldPayloads := []tc{
		{"F6 `status` OR `1`=`1`", "`status` OR `1`=`1`"},
		{"F6 `status` UNION SELECT * FROM users--`", "`status` UNION SELECT * FROM users--`"},
		{"F9 SELECT--", "SELECT--"},
		{"F9 DROP-", "DROP-"},
		{"F9 id.DROP", "id.DROP"},
		{"F9 id.SELECT", "id.SELECT"},
		{"F7 fullwidth `ＤＲＯＰ`", "`ＤＲＯＰ`"},
	}

	type result struct {
		label       string
		payload     string
		context     string
		blocked     bool
		errMsg      string
		generatedSQL string
	}
	var results []result

	// ── selectCols 컨텍스트 ───────────────────────────────────────────────
	for _, tc := range colPayloads {
		sql, _, err := dq.Build("users#selectCols", query.ColumnParams{
			Columns: []string{tc.payload},
		})
		r := result{label: tc.label, payload: tc.payload, context: "selectCols"}
		if err != nil {
			r.blocked = true
			r.errMsg = err.Error()
		} else {
			r.blocked = false
			r.generatedSQL = strings.TrimSpace(sql)
		}
		results = append(results, r)
	}

	// ── updateUser 컨텍스트 ───────────────────────────────────────────────
	for _, tc := range fieldPayloads {
		sql, _, err := dq.Build("users#updateUser", query.UpdateParams{
			Id:     1,
			Fields: []query.UpdateField{{Key: tc.payload, Value: "x"}},
		})
		r := result{label: tc.label, payload: tc.payload, context: "updateUser"}
		if err != nil {
			r.blocked = true
			r.errMsg = err.Error()
		} else {
			r.blocked = false
			r.generatedSQL = strings.TrimSpace(sql)
		}
		results = append(results, r)
	}

	// ── 결과 출력 ─────────────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println("  RawParam (${}) 인젝션 테스트 결과")
	fmt.Println("══════════════════════════════════════════════════════════════")

	blockedCount := 0
	passedCount := 0

	for _, r := range results {
		if r.blocked {
			fmt.Printf("[BLOCKED] [%-10s] %-45q  → %v\n", r.context, r.payload, r.errMsg)
			blockedCount++
		} else {
			fmt.Printf("[⚠ PASS ] [%-10s] %-45q  → SQL: %s\n", r.context, r.payload, r.generatedSQL)
			passedCount++
		}
	}

	fmt.Println("──────────────────────────────────────────────────────────────")
	fmt.Printf("  BLOCKED: %d  /  PASSED-THROUGH: %d  /  TOTAL: %d\n",
		blockedCount, passedCount, blockedCount+passedCount)
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println()

	// t.Errorf: PASS-THROUGH 된 것들만 실패 표시
	for _, r := range results {
		if !r.blocked {
			t.Errorf("NOT BLOCKED [%s] payload=%q → SQL: %s", r.context, r.payload, r.generatedSQL)
		}
	}
}
