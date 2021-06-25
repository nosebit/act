/**
 * The implementation here was totally inspired by the following:
 * 
 * https://kvz.io/prefix-streaming-stdout-and-stderr-in-golang.html
 * 
 * @TODO : We need to refactor this to remove record/persist.
 * @TODO : We should add more comments here and jsdocs.
 */

package run

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/logrusorgru/aurora/v3"
)

//############################################################
// Types
//############################################################

type LogStreamer struct {
	buf       *bytes.Buffer
	readLines string
	// If prefix == stdout, colors green
	// If prefix == stderr, colors red
	// Else, prefix is taken as-is, and prepended to anything
	// you throw at Write()
	prefix string
	// if true, saves output in memory
	record  bool
	persist string
}

func (s LogStreamer) Write(p []byte) (n int, err error) {
	if n, err = s.buf.Write(p); err != nil {
		return
	}

	err = s.OutputLines()
	return
}

func (l *LogStreamer) Close() error {
	l.Flush()
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *LogStreamer) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (s *LogStreamer) OutputLines() (err error) {
	for {
		line, err := s.buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		s.readLines += line
		s.out(line)
	}

	return nil
}

func (l *LogStreamer) ResetReadLines() {
	l.readLines = ""
}

func (l *LogStreamer) ReadLines() string {
	return l.readLines
}

func (l *LogStreamer) FlushRecord() string {
	buffer := l.persist
	l.persist = ""
	return buffer
}

func (l *LogStreamer) out(str string) (err error) {
	if l.record {
		l.persist = l.persist + str
	}

	now := time.Now().Format("2006-01-02 15:04:05.000000")

	/**
	 * If this act process was invoked by other act then
	 * prevent double info logging.
	 */
	prefix := l.prefix

	if prefix != "" {
		if actParentPrefix, present := os.LookupEnv("ACT_PARENT_ACT"); present {
			prefix = fmt.Sprintf("%s > %s", actParentPrefix, prefix)
		}

		fmt.Printf("%s | %s %s", aurora.Yellow(prefix).Bold(), aurora.Cyan(now), str)
	} else {
		fmt.Print(str)
	}

	return nil
}

//############################################################
// Exported Functions
//############################################################

/**
 * This function going to create a new log writer.
 */
func NewLogWriter(prefix string, record bool) *LogStreamer {
	streamer := &LogStreamer{
		buf:     bytes.NewBuffer([]byte("")),
		prefix:  prefix,
		record:  record,
		persist: "",
	}

	return streamer
}