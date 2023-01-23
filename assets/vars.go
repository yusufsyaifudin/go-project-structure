package assets

import (
	_ "embed"
	"strconv"
	"strings"
	"time"
)

const (
	AppName = "myapp"
)

var (
	//go:embed build_commit_hash.txt
	buildCommitID string

	//go:embed build_time.txt
	buildTime string
)

func BuildCommitID() string {
	return strings.TrimSpace(buildCommitID)
}

func BuildTime() time.Time {
	t := strings.TrimSpace(buildTime)

	buildTimeInt, err := strconv.Atoi(t)
	if err != nil {
		return time.Time{}
	}

	return time.Unix(int64(buildTimeInt), 0)
}
