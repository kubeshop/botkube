//go:build integration

package e2e

import (
	"fmt"
	"time"

	"github.com/pmezard/go-difflib/difflib"
)

// Original source: https://github.com/stretchr/testify/blob/181cea6eab8b2de7071383eca4be32a424db38dd/assert/assertions.go#L1685-L1695
// Copyright (c) 2012-2020 Mat Ryer, Tyler Bunnell and contributors. Licensed under MIT License.
// return diff string is expect and actual are different, otherwise return empty string
func diff(expect string, actual string) string {
	if expect == actual {
		return ""
	}
	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(expect),
		B:        difflib.SplitLines(actual),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})

	return "\n\nDiff:\n" + diff
}

// countMatchBlock count the number of lines matched between two strings
func countMatchBlock(expect string, actual string) int {
	matcher := difflib.NewMatcher(difflib.SplitLines(expect), difflib.SplitLines(actual))
	matches := matcher.GetMatchingBlocks()
	count := 0
	for _, match := range matches {
		count += match.Size
	}
	return count
}

func timeWithinDuration(expected, actual time.Time, delta time.Duration) error {
	dt := expected.Sub(actual)
	if dt < -delta || dt > delta {
		return fmt.Errorf("max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, dt)
	}

	return nil
}
