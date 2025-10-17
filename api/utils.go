package api

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/klauspost/compress/s2"
	parserV1 "github.com/zondax/fil-parser/parser/v1"
	parserV2 "github.com/zondax/fil-parser/parser/v2"
	"github.com/zondax/fil-parser/types"
	"golang.org/x/mod/semver"
)

const (
	nv20UpgradeHeight = 489094
)

func decompress(data []byte) ([]byte, error) {
	// Decompress data using s2
	b := bytes.NewBuffer(data)
	var out bytes.Buffer
	r := s2.NewReader(b)

	if _, err := io.Copy(&out, r); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func processNodeVersion(fullVersion string) (*types.NodeInfo, error) {
	var majorMinor string
	splitVersion := strings.Split(fullVersion, "+")
	if len(splitVersion) < 2 {
		return nil, fmt.Errorf("could not get node version, invalid version format detected: %s", fullVersion)
	} else {
		majorMinor = fmt.Sprintf("v%s", splitVersion[0])
		majorMinor = semver.MajorMinor(majorMinor)
		if majorMinor == "" {
			return nil, fmt.Errorf("could not get node version, invalid version format detected: %s", fullVersion)
		}
	}

	return &types.NodeInfo{
		NodeFullVersion:       fullVersion,
		NodeMajorMinorVersion: majorMinor,
	}, nil
}

// HeightToNodeVersion returns the maximum node version for a given height.
// This is used to enable fil-parser to use the correct StateCompute format for parsing.
func HeightToNodeVersion(height int64) *types.NodeInfo {
	if height <= nv20UpgradeHeight {
		return &types.NodeInfo{
			NodeFullVersion:       "v1.22.0",
			NodeMajorMinorVersion: "v1.22",
		}
	}
	return &types.NodeInfo{
		NodeFullVersion:       "v1.34.0",
		NodeMajorMinorVersion: "v1.34",
	}
}

// HeightToParserVersion returns the parser version for a given height.
func HeightToParserVersion(height int64) string {
	if height <= nv20UpgradeHeight {
		return parserV1.Version
	}
	return parserV2.Version
}
