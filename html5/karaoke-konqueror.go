package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	// Idea: this design will block if a single function takes long
	// Solution: Have a protocol where each command is issued with a GUID
	// Fire off command in goroutine such that other commands can be executed
	// and return immediatley GUID with a queued flag
	// When result commands back read result from channel and send to STDOUT
	// along with the matching GUID, this way clients can match up calls
	// Interesting solution.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		inCommand := scanner.Text()

		if inCommand == `{"command":"go"}` {
			fmt.Println("Going...")
		} else if inCommand == `{"command":"bark"}` {
			fmt.Println("Barking...")
		} else if inCommand == `{"command":"quit"}` {
			fmt.Println("Quiting...")
			break
		} else {
			fmt.Println("No command...")
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}
