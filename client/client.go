package main

import (
	"os"
	"os/signal"
	"syscall"
	"fortio.org/terminal/ansipixels"
)

var loggedin bool = false

func drawLogin(ap *ansipixels.AnsiPixels) {
	ap.ClearScreen()
	ap.DrawSquareBox(0, 0, ap.W, ap.H)
}

func drawBase(ap *ansipixels.AnsiPixels) {
	ap.ClearScreen()
	ap.DrawSquareBox(0, 0, ap.W/3, ap.H)
	ap.DrawSquareBox(ap.W/3, 0, ap.W/3*2, ap.H)
}

func main() {
	var input []byte
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

	ap.GetSize()

	drawLogin(ap)
	ap.Out.Flush()

	for {
		n, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil { break }
		if(n>0) {
			input = ap.Data[:n]
			if input[0] == 3  || input[0] == 17 {
				break
			}
		}


		if(!loggedin) {
			drawLogin(ap)
			ap.WriteAt(5, 6, "%d", input)
		}
		ap.Out.Flush()
	}


}
