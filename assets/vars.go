package assets

import _ "embed"

const (
	AppName = "myapp"
)

var (
	//go:embed build_commit_hash.txt
	BuildCommitID string

	//go:embed build_time.txt
	BuildTime string
)
