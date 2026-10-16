package logx

import (
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	entries []string
	max     = 60
)

func Printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Print(msg)

	entry := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)

	mu.Lock()
	entries = append(entries, entry)
	if len(entries) > max {
		entries = entries[len(entries)-max:]
	}
	mu.Unlock()
}

func Tail() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, len(entries))
	copy(out, entries)
	return out
}
