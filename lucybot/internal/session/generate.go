package session

import (
	"crypto/md5"
	"fmt"
	"time"
)

// GenerateSessionID creates a unique 32-character session ID using MD5 hash.
// The ID is generated from timestamp + agentName + first 128 chars of query.
func GenerateSessionID(agentName string, firstQuery string) string {
	timestamp := fmt.Sprintf("%f", float64(time.Now().UnixNano()) / 1e9)

	// Truncate query to first 128 characters
	queryPart := firstQuery
	if len(queryPart) > 128 {
		queryPart = queryPart[:128]
	}

	// Combine: timestamp:agentName:query
	data := fmt.Sprintf("%s:%s:%s", timestamp, agentName, queryPart)

	// Generate MD5 hash and return as hex string
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
