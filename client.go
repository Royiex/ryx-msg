package main

import (
	"os"
	"os/signal"
	"syscall"

	"fortio.org/terminal/ansipixels"
	// "ryxmsg/backend"
)

var sendstr string = ""
var chatheight int = 1
// func drawLogin(ap *ansipixels.AnsiPixels) {
//	ap.ClearScreen()
//	ap.DrawSquareBox(0, 0, ap.W, ap.H)
// }

func drawSend(ap *ansipixels.AnsiPixels) {
	if len(sendstr) == 0 { return }
	linelen := ap.W/3*2-10
	line := -1
	start := 0

	for start < len(sendstr) {
		end := start + linelen
		if end > len(sendstr) {
			end = len(sendstr)
		} else {
			// Try to wrap line at ' '(space)
			for end > start && sendstr[end-1] != ' ' {
				end--
			}
			// If not possible wrap at line end
			if end == start {
				end = start + linelen
				if end > len(sendstr) {
					end = len(sendstr)
				}
			}
		}

		ap.WriteAt(ap.W/3+8, ap.H-chatheight+line, "%s", sendstr[start:end])
		start = end
		line++
	}
	chatheight = line + 1;
}

func drawBase(ap *ansipixels.AnsiPixels) {
	pad := chatheight+2

	ap.ClearScreen()
	drawSend(ap)
	ap.DrawSquareBox(0, 0, ap.W/3, ap.H)
	ap.DrawSquareBox(ap.W/3, 0, ap.W-ap.W/3, ap.H)
	ap.DrawSquareBox(ap.W/3, ap.H-pad, ap.W-ap.W/3, pad)
	ap.WriteAtStr(ap.W/3, ap.H-pad, "├")
	ap.WriteAtStr(ap.W, ap.H-pad, "┤")
	ap.WriteAtStr(ap.W/3+1, ap.H-pad+1, "Send >")
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

	drawBase(ap)
	ap.Out.Flush()

	for {
		n, err := ap.ReadOrResizeOrSignalOnce()
		if err != nil { break }
		if(n>0) {
			input = ap.Data[:n]
			if input[0] == 3  || input[0] == 17 {
				break
			}
			if input[0] == 127 {
				if(len(sendstr)>0) {
					sendstr = sendstr[:len(sendstr)-1]
				}
				continue
			}
			if input[0] == 13 {
				sendstr=""
				chatheight = 1
				continue
			}
			sendstr+=string(input[0])
		}

		drawBase(ap)
		ap.WriteAt(5, 6, "%d", input)
		ap.Out.Flush()
	}

}
