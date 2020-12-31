package confinator

import (
	"fmt"
	"hash/crc32"
	"os"
	"strings"
)

const (
	EnvBuildUniqueID = "BUILD_UNIQUE_ID"
)

type BuildInfo struct {
	BuildName   string `json:"build_name"`
	BuildDate   string `json:"build_date"`
	BuildBranch string `json:"build_branch"`
	BuildNumber string `json:"build_number"`

	Version     string `json:"version"`
	VersionHash int64  `json:"version_hash"`

	BuildUniqueID string `json:"unique_id"`
}

func NewBuildInfo(name, date, branch, number string) BuildInfo {
	var (
		version       string
		buildUniqueID string
	)

	if strings.HasPrefix(branch, "release") {
		version = strings.TrimLeft(branch, "release/")
	} else {
		version = fmt.Sprintf("dev-%s", branch)
	}

	// try to get build unique id, ok if we don't for local dev.
	buildUniqueID = os.Getenv(EnvBuildUniqueID)

	vh := crc32.ChecksumIEEE([]byte(fmt.Sprintf("%s%s%s", date, branch, buildUniqueID)))

	return BuildInfo{
		BuildName:     name,
		BuildDate:     date,
		BuildBranch:   branch,
		BuildNumber:   number,
		Version:       version,
		VersionHash:   int64(vh),
		BuildUniqueID: buildUniqueID,
	}
}
