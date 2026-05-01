package agent

import (
	"fmt"
	"os"
)

// writeOSC133 writes an OSC 133 semantic-prompt escape to /dev/tty so the
// surrounding terminal emulator can navigate between Claude Code prompts.
//
// Sequences:
//
//	A       prompt start (before input renders)
//	B       input ready (cursor in input box)
//	C       execution start (user submitted)
//	D;<n>   execution end with exit status n
func writeOSC133(seq string) {
	f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "\x1b]133;%s\x07", seq)
}
