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
	"github.com/nosebit/act/utils"
)

//############################################################
// Types
//############################################################

/**
 * This is the main struct which implements io.Writer interface
 * to be used as stdout/stderr for commands.
 */
type LogWriter struct {
	Detached  bool
	ctx       *ActRunCtx
	buf       *bytes.Buffer
	readLines string
	logFile   *os.File
}

/**
 * This function implements io.Writer interface.
 */
func (l LogWriter) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.OutputLines()
	return
}

/**
 * This finction close the writer.
 */
func (l *LogWriter) Close() error {
	l.Flush()
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

/**
 * Flush all buffered bytes to screen/file.
 */
func (l *LogWriter) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

/**
 * This function going to output line by line from buffer.
 */
func (l *LogWriter) OutputLines() (err error) {
	for {
		line, err := l.buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		l.readLines += line
		l.out(line)
	}

	return nil
}

/**
 * Output string to screen/file.
 */
func (l *LogWriter) out(str string) (err error) {
	// Get time to log.
	now := time.Now().Format("2006-01-02 15:04:05.000000")

	/**
	 * If this act process was invoked by other act then
	 * prevent double info logging.
	 */
	logPrefix := l.ctx.RunCtx.Info.NameId

	if l.ctx.ActFile.Namespace != "" {
		logPrefix = fmt.Sprintf("%s.%s", l.ctx.ActFile.Namespace, l.ctx.Act.Name)
	}

	var strToLog string

	/**
	 * If act process is detached from another parent act process then
	 * we going to prevent add prefix info.
	 */
	if l.Detached {
		strToLog = str
	} else {
		strToLog = fmt.Sprintf("%s | %s %s", aurora.Yellow(logPrefix).Bold(), aurora.Cyan(now), str)
	}

	/**
	 * Log both to stdout and to file.
	 */
	fmt.Print(strToLog)
	l.logFile.Write([]byte(strToLog))

	return nil
}

//############################################################
// Exported Functions
//############################################################

/**
 * This function going to create a new log writer.
 */
func NewLogWriter(ctx *ActRunCtx) *LogWriter {
	logFilePath := ctx.RunCtx.Info.GetLogFilePath()
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)

	if err != nil {
	  utils.FatalError(fmt.Sprintf("cannot open log file at %s", logFilePath), err)
	}

	l := &LogWriter{
		buf:     bytes.NewBuffer([]byte("")),
		ctx:     ctx,
		logFile: logFile,
	}

	return l
}