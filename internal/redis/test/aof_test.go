package redis_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ng-namanh/redis-go/internal/commands"
)

func TestAOF(t *testing.T) {
	// Setup a temporary directory for AOF tests
	tempDir := t.TempDir()

	// Configure AOF settings
	commands.Dir = tempDir
	commands.AppendOnly = true
	commands.AppendDirName = "test_aof_dir"
	commands.AppendFileName = "appendonly.aof"
	commands.AppendFsync = "always"

	t.Run("AOF Logging and Replay", func(t *testing.T) {
		// 1. Initialize AOF (creates directory and manifest)
		err := commands.InitializeAOF()
		if err != nil {
			t.Fatalf("InitializeAOF failed: %v", err)
		}

		// 2. Execute some write commands
		commands.SET([]string{"aof_key", "aof_value"})
		commands.INCR([]string{"aof_counter"})
		commands.RPUSH([]string{"aof_list", "item1", "item2"})

		// Verify file exists
		aofIncrPath := filepath.Join(tempDir, "test_aof_dir", "appendonly.aof.1.incr.aof")
		if _, err := os.Stat(aofIncrPath); os.IsNotExist(err) {
			t.Fatal("AOF incremental file was not created")
		}

		// 3. Reset the internal state to simulate a server restart
		commands.ResetForTesting()

		// 4. Re-initialize AOF (this should trigger ReplayAOF)
		err = commands.InitializeAOF()
		if err != nil {
			t.Fatalf("Re-initialization/Replay failed: %v", err)
		}

		// 5. Verify the data was restored
		res, _ := commands.GET([]string{"aof_key"})
		if !bytes.Contains(res, []byte("aof_value")) {
			t.Errorf("Expected restored value 'aof_value', got %q", res)
		}

		res, _ = commands.GET([]string{"aof_counter"})
		if !bytes.Contains(res, []byte(":1")) { // RESP integer 1 is :1\r\n
			t.Errorf("Expected restored counter '1', got %q", res)
		}

		res, _ = commands.LLEN([]string{"aof_list"})
		if !bytes.Contains(res, []byte(":2")) { // RESP integer 2 is :2\r\n
			t.Errorf("Expected restored list length '2', got %q", res)
		}
	})

	t.Run("AOF ignores non-modifying commands", func(t *testing.T) {
		// Get current file size
		aofIncrPath := filepath.Join(tempDir, "test_aof_dir", "appendonly.aof.1.incr.aof")
		info, _ := os.Stat(aofIncrPath)
		initialSize := info.Size()

		// Execute read-only commands
		commands.GET([]string{"aof_key"})
		commands.INFO([]string{"replication"})
		commands.CONFIG([]string{"GET", "dir"})

		// Check if file size changed
		info, _ = os.Stat(aofIncrPath)
		if info.Size() != initialSize {
			t.Errorf("AOF file grew after read-only commands! (Initial: %d, Final: %d)", initialSize, info.Size())
		}
	})
}
