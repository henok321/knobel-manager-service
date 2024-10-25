package integration_tests

import (
	"testing"

	_ "github.com/lib/pq"
)

func teamTestCases(t *testing.T) []testCase {
	return []testCase{}
}

func TestTeams(t *testing.T) {
	RunTestGroup(t, teamTestCases)
}
