package bpi

import (
	"bufio"
	"strings"

	"github.com/pkg/errors"
)

func patchbreak(line string) bool {
	if strings.HasPrefix(line, "diff -") {
		return true
	}

	if strings.HasPrefix(line, "Index: ") {
		return true
	}

	// "--- <filename>" starts patches without headers
	// "---<sp>*" is a manual separator
	if len(line) < 3 {
		return false
	}

	if strings.HasPrefix(line, "---") {
		// space followed by a filename?
		if len(line) >= 4 && line[3] == ' ' && line[4] != ' ' { // FIXME: check because isspace is a function in git
			return true
		}

		// just whitespace?
		for i := 3; i < len(line); i++ {
			if line[i] != ' ' { // FIXME isspace
				return false
			}
		}

		return true
	}

	return false
}

// ParseDiff parses git diff into list of diffs per file
//
// TODO: some other git headers
// "old mode "
// "new mode "
// "deleted file mode "
// "new file mode "
// "copy from "
// "copy to "
// "rename old "
// "rename new "
// "rename from "
// "rename to "
// "similarity index "
// "dissimilarity index "
//
// FIXME: GNU diff add "\n" lines as empty context lines
func ParseDiff(input string) ([]string, error) {
	if input == "" {
		return nil, nil
	}

	var diffs [][]string
	var currentDiff []string

	inHeader := "inHeader"
	inHunk := "inHunk"
	inFooter := "inFooter"
	state := ""

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()

		switch true {
		case state == "" && line == "---":
			// skip header line
		case strings.HasPrefix(line, "diff -"):
			// new diff
			state = inHeader

			if len(currentDiff) > 0 {
				diffs = append(diffs, currentDiff)
			}
			currentDiff = []string{}

			currentDiff = append(currentDiff, line)
		case state == "":
			// statistic skip for now
		case state == inHeader && strings.HasPrefix(line, "index "):
			// commit information
			currentDiff = append(currentDiff, line)
		case state == inHeader && strings.HasPrefix(line, "--- "):
			// file from
			currentDiff = append(currentDiff, line)
		case state == inHeader && strings.HasPrefix(line, "+++ "):
			// file to
			currentDiff = append(currentDiff, line)
		case (state == inHeader || state == inHunk) && strings.HasPrefix(line, "@@ "):
			// start new chunk
			state = inHunk
			currentDiff = append(currentDiff, line)
		case state == inHunk && strings.HasPrefix(line, "-- "):
			// footer
			state = inFooter
		case state == inHunk && (len(line) < 1 || (line[0] != ' ' && line[0] != '+' && line[0] != '-')):
			return nil, errors.Errorf("incorrect line in hunk: %s", line)
		case state == inHunk:
			currentDiff = append(currentDiff, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(currentDiff) > 0 {
		diffs = append(diffs, currentDiff)
	}

	result := make([]string, len(diffs))
	for i, diff := range diffs {
		result[i] = strings.Join(diff, "\n")
	}

	return result, nil
}
