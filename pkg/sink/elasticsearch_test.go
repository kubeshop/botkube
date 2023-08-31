package sink

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElasticsearchVersion(t *testing.T) {
	type versions struct {
		clusterVersion       string
		expectedMajorVersion int
		err                  error
	}

	tests := []versions{
		{clusterVersion: "8.10.2", expectedMajorVersion: 8, err: nil},
		{clusterVersion: "7.2", expectedMajorVersion: 7, err: nil},
		{clusterVersion: "6.0.2-dev", expectedMajorVersion: 6, err: nil},
		{clusterVersion: "", expectedMajorVersion: 0, err: errors.New("cluster version is not valid")},
	}

	for _, test := range tests {
		version, err := esMajorClusterVersion(test.clusterVersion)
		assert.Equal(t, test.expectedMajorVersion, version)
		assert.Equal(t, test.err, err)
	}
}
