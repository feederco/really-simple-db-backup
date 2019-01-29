package pkg

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ParseCommandLineFlags parsed flags defined by `flag` package. Required to work with sub-commands
func ParseCommandLineFlags(args []string) error {
	// If a commandline app works like this: ./app subcommand -flag -flag2
	// `flag.Parse` won't parse anything after `subcommand`.
	// To still be able to use `flag.String/flag.Int64` etc without creating
	// a new `flag.FlagSet`, we need this hack to find the first arg that has a dash
	// so we know when to start parsing
	firstArgWithDash := 0
	for i := 0; i < len(args); i++ {
		firstArgWithDash = i

		if len(args[i]) > 0 && args[i][0] == '-' {
			break
		}
	}

	return flag.CommandLine.Parse(args[firstArgWithDash:])
}

// PerformCommand performs a command line command with nice helpers
func PerformCommand(cmdArgs ...string) (string, error) {
	if VerboseMode {
		Log.Printf("== `%s`\n", strings.Join(cmdArgs, " "))
	}

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	output := ""

	scanner := bufio.NewScanner(cmdReader)
	go func() {
		for scanner.Scan() {
			chunk := scanner.Text()
			if VerboseMode {
				fmt.Printf("%s\n", chunk)
			}
			output += chunk + "\n"
		}
	}()

	err = cmd.Start()
	if err != nil {
		if VerboseMode {
			ErrorLog.Println("Error starting Cmd", err)
		}
		return "", err
	}

	err = cmd.Wait()
	if err != nil {
		return "", err
	}

	return output, nil
}

// PerformCommandWithFileOutput performs a command with output to a file
func PerformCommandWithFileOutput(outputFilename string, cmd string, cmdArgs ...string) error {
	var err error
	var outputFile *os.File
	outputFile, err = os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	execCmd := exec.Command(cmd, cmdArgs...)
	execCmd.Stdout = outputFile

	err = execCmd.Start()
	if err != nil {
		return err
	}

	err = execCmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
