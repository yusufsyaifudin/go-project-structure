package assets

import _ "embed"

var (
	//go:embed build_commit_hash.txt
	BuildCommitID string

	//go:embed build_time.txt
	BuildTime string
)
