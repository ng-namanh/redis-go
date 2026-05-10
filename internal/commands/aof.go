package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

var aofFile *os.File
var aofMutex sync.Mutex

func InitializeAOF() error {
	aofMutex.Lock()
	defer aofMutex.Unlock()

	aofDir := filepath.Join(Dir, AppendDirName)
	if err := os.MkdirAll(aofDir, 0755); err != nil {
		return err
	}

	manifestPath := filepath.Join(aofDir, AppendFileName+".manifest")
	aofPath := filepath.Join(aofDir, AppendFileName+".1.incr.aof")

	// Check if manifest exists and replay if so
	if _, err := os.Stat(manifestPath); err == nil {
		if err := ReplayAOF(aofDir, manifestPath); err != nil {
			return err
		}
	} else {
		// Create new manifest
		manifestContent := fmt.Sprintf("file %s.1.incr.aof seq 1 type i\n", AppendFileName)
		if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
			return err
		}
	}

	// Open AOF file for appending
	f, err := os.OpenFile(aofPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	aofFile = f

	if AppendFsync == "everysec" {
		go func() {
			for {
				time.Sleep(time.Second)
				aofMutex.Lock()
				if aofFile != nil {
					_ = aofFile.Sync()
				}
				aofMutex.Unlock()
			}
		}()
	}

	return nil
}

func ReplayAOF(aofDir, manifestPath string) error {
	isReplaying = true
	defer func() { isReplaying = false }()

	manifestFile, err := os.Open(manifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()

	var aofFilename string
	scanner := bufio.NewScanner(manifestFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "type i") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "file" && i+1 < len(parts) {
					aofFilename = parts[i+1]
					break
				}
			}
		}
	}

	if aofFilename == "" {
		return nil // No incremental file found
	}

	aofPath := filepath.Join(aofDir, aofFilename)
	f, err := os.Open(aofPath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	for {
		v, err := resp.ReadValue(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		cmd, args, err := resp.ParseCommand(v)
		if err != nil {
			continue
		}

		// Execute without acquiring the global lock because we are during startup
		// and nothing else is running yet.
		_, _ = HandleCommand(strings.ToUpper(cmd), args)
	}

	return nil
}

func AppendToAOF(cmd string, args []string) {
	if !AppendOnly || aofFile == nil {
		return
	}

	aofMutex.Lock()
	defer aofMutex.Unlock()

	elems := make([]resp.RESP, len(args)+1)
	elems[0] = resp.RESP{Type: resp.BulkString, Str: cmd}

	for i, arg := range args {
		elems[i+1] = resp.RESP{Type: resp.BulkString, Str: arg}
	}
	if _, err := aofFile.Write(resp.WriteArray(elems)); err != nil {
		fmt.Println("Error writing to AOF file:", err)
		return
	}

	if AppendFsync == "always" {
		_ = aofFile.Sync()
	}
}
