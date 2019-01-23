package cmd

import (
	"io/ioutil"
	"strings"
)

func getLastLSNFromFile(fileName string) (string, error) {
	checkpointFileContents, openErr := ioutil.ReadFile(fileName)

	if openErr != nil {
		return "", openErr
	}

	checkpointFileLines := strings.Split(string(checkpointFileContents), "\n")
	for _, line := range checkpointFileLines {
		if strings.Contains(line, "to_lsn") {
			statementPieces := strings.Split(line, "=")
			if len(statementPieces) == 2 {
				return strings.TrimSpace(statementPieces[1]), nil
			}
			break
		}
	}

	return "", nil
}
