package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/zserge/webview"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	width  int
	height int
)

const (
	appName  = "papago-translator-linux"
	selector = "document.getElementById('txtSource')"
)

func init() {
	flag.IntVar(&width, "w", 800, "window width")
	flag.IntVar(&height, "h", 600, "window height")
	flag.Parse()
}

func main() {
	address := filepath.Join(os.TempDir(), fmt.Sprintf("%s.sock", appName))
	log.Printf("socket file: <%s>\n", address)

	l := translateListener(address)
	defer l.Close()
	defer syscall.Unlink(address)

	go signalHandle(address)

	w := webview.New(true)
	defer w.Destroy()

	w.SetTitle("Papago-linux")
	w.SetSize(width, height, webview.HintNone)

	cbContent, err := getClipboard()
	if err != nil || strings.TrimSpace(cbContent) == "" {
		log.Printf("[err] clipboard readall fail: %+v\n", err)
		return
	}

	w.Navigate("https://papago.naver.com/?sk=auto&tk=vi&st=" + cbContent)

	log.Printf("navigate to: %s\n", "https://papago.naver.com/?sk=auto&tk=vi&st="+cbContent)

	w.Run()
}

func checkJapChar(c rune) bool {
	// Japanese-style punctuation ( 3000 - 303f)
	// Hiragana ( 3040 - 309f)
	// Katakana ( 30a0 - 30ff)
	// Full-width roman characters and half-width katakana ( ff00 - ffef)
	// CJK unifed ideographs - Common and uncommon kanji ( 4e00 - 9faf)
	
	return (c >= 0x3001 && c <= 0x303f) ||
		(c >= 0x3040 && c <= 0x309f) ||
		(c >= 0x30a0 && c <= 0x30ff) ||
		(c >= 0xff00 && c <= 0xffef) ||
		(c >= 0x4e00 && c <= 0x9faf)
}

func getClipboard() (string, error) {
	clipboardContent, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	log.Printf("got clipboard text: [%s]\n", clipboardContent)

	// Convert to hex for URL
	cbContent := ""
	firstChar := false
	for _, c := range clipboardContent {
		// If character is not a letter or number or a Japanese/Kanji character, convert to hex
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && !checkJapChar(c) {
			if firstChar {
				// If the character is large, convert to 3 bytes
				// Example: 0x3000 -> %e3%80%80
				if c > 0xff {
					bytes := []byte(string(c))
					for _, b := range bytes {
						cbContent += fmt.Sprintf("%%%x", b)
					}
				} else if c < 0x10 {
					cbContent += fmt.Sprintf("%%0%x", c)
				} else {
					cbContent += fmt.Sprintf("%%%x", c)
				}
			}
		} else {
			cbContent += string(c)
			firstChar = true
		}
	}
	return cbContent, nil
}

func translateListener(address string) net.Listener {
	tryStart := 0

start:
	if tryStart > 1 {
		log.Fatal("tried too many times")
	}

	l, err := net.Listen("unix", address)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			unixAddr, err := net.ResolveUnixAddr("unix", address)
			if err != nil {
				log.Fatal(err)
			}

			conn, err := net.DialUnix("unix", nil, unixAddr)
			if err != nil {
				if errors.Is(err, syscall.ECONNREFUSED) {
					syscall.Unlink(address) // SIGKILL and SIGSTOP may not be caught
					tryStart += 1
					goto start
				}
				log.Fatal(err)
			}
			defer conn.Close()
			return nil
		}
		log.Fatal(err)
	}

	return l
}

func signalHandle(address string) {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGABRT,
	) // SIGKILL and SIGSTOP may not be caught

	go func() {
		for {
			sig := <-sc
			log.Printf("got signal %s\n", sig)
			syscall.Unlink(address)
			os.Exit(0)
		}
	}()
}
