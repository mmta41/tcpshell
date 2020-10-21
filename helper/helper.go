package helper

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var WriterIO io.Writer
var clientList map[net.Conn]struct{}
var clientLock sync.Locker
var once = sync.Once{}

func init() {
	clientList = make(map[net.Conn]struct{})
	clientLock = new(sync.Mutex)
}

func execApp(runApp string) (io.ReadCloser, io.ReadCloser) {
	appArgs := strings.Split(runApp, " ")
	args, err := parseArgs(strings.Join(appArgs[1:], " "))
	if err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command(appArgs[0], args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	WriterIO, err = cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	return stdoutPipe, stderrPipe
}

func HandleStdin(runApp string, conn net.Conn) {
	if len(runApp) > 0 {
		stdoutPipe, stderrPipe := execApp(runApp)
		go Dispatch(stdoutPipe, conn)
		go Dispatch(stderrPipe, conn)
		return
	}
	once.Do(func() {
		WriterIO = os.Stdout
		go Dispatch(os.Stdin, nil)
	})
}

func Dispatch(reader io.Reader, writer io.Writer) {
	bufferReader := bufio.NewReader(reader)
	if writer != nil {
		for {
			data := make([]byte, 1)
			_, err := bufferReader.Read(data)
			if err != nil {
				fmt.Println("err occurred while reading: ", err)
				break
			}
			_, err = fmt.Fprint(writer, string(data))
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "error writing to pipe: ", err)
				break
			}
		}
	} else {
		for {
			data := make([]byte, 1)
			_, err := bufferReader.Read(data)
			if err != nil {
				fmt.Println("err occurred while reading: ", err)
				break
			}
			clientLock.Lock()
			for conn := range clientList {
				_, err := fmt.Fprint(conn, string(data))
				if err != nil {
					fmt.Println(err)
					delete(clientList, conn)
				}
			}
			clientLock.Unlock()
		}
	}
}

func AddClient(conn net.Conn) {
	clientLock.Lock()
	clientList[conn] = struct{}{}
	clientLock.Unlock()
}

func RemoveClient(conn net.Conn) {
	clientLock.Lock()
	delete(clientList, conn)
	clientLock.Unlock()
}

func parseArgs(command string) ([]string, error) {
	var args []string
	const (
		StateStart = iota
		StateQuote
		StateArg
	)
	state := StateStart
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == StateQuote {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = StateStart
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = StateQuote
			quote = string(c)
			continue
		}

		if state == StateArg {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = StateStart
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = StateArg
			current += string(c)
		}
	}

	if state == StateQuote {
		return []string{}, fmt.Errorf("unclosed quote in command line: %s", command)
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}
