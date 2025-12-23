package main

import (
	"os"
	"os/signal"
	"syscall"
	"fortio.org/terminal/ansipixels"
)

var loggedin bool =  false

func main() {
	ap := ansipixels.NewAnsiPixels(15)
	if err := ap.Open(); err != nil {
		return
	}
	defer ap.Restore()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		ap.Restore()
		os.Exit(0)
	}()

	ap.ClearScreen()
	ap.WriteAtStr(5, 5, "Hello!")
	ap.Out.Flush()

	for {
		n, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil {
			break
		}
		if n>0 {
			input := ap.Data[:n]
			if input[0] == 3  || input[0] == 17 {
				break
			}

			ap.ClearScreen()
			ap.WriteAt(5, 6, "%d", input)
			ap.Out.Flush()

		}
	}

}
