package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func WithSpinner(msg string, fn func()) {
	if !term.IsTerminal(os.Stderr.Fd()) {
		fn()
		return
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	defer wg.Wait()
	defer close(stop)
	wg.Go(func() {
		t := time.NewTicker(80 * time.Millisecond)
		defer t.Stop()
		i := 0
		fmt.Fprintf(os.Stderr, "\r%s %s", accentStyle.Render(spinnerFrames[0]), msg)
		for {
			select {
			case <-stop:
				fmt.Fprint(os.Stderr, "\r\033[K")
				return
			case <-t.C:
				i++
				fmt.Fprintf(os.Stderr, "\r%s %s", accentStyle.Render(spinnerFrames[i%len(spinnerFrames)]), msg)
			}
		}
	})

	fn()
}
