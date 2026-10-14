package idgen

import (
	"fmt"
	"strings"
	"testing"
)

func TestNextID(t *testing.T) {
	// 1. Create a slice to hold the value blocks
	var valueStrings []string

	for i := 0; i < 10; i++ {
		res, err := sf().NextID()
		if err != nil {
			t.Fatalf("idgen nextID: %v", err)
		}

		// Mock data for the rest of your columns
		userID := fmt.Sprintf("user_%d", i+1)
		balance := "1000000000"
		currency := "IDR"

		// 2. Build the value row component
		valueStr := fmt.Sprintf("(%d, '%s', '%s', '%s')", res, userID, balance, currency)
		valueStrings = append(valueStrings, valueStr)
	}

	// 3. Combine everything with a comma followed by a newline (\n)
	bulkSQL := fmt.Sprintf(
		"INSERT INTO wallets(id, user_id, balance, currency) VALUES \n%s;",
		strings.Join(valueStrings, ",\n"),
	)

	// Print or execute your final query
	t.Logf("Generated Bulk SQL:\n%s\n", bulkSQL)
}
