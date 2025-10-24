package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/111222Bomba/Asset-Reuploader/internal/app/config"
	"github.com/111222Bomba/Asset-Reuploader/internal/color"
	"github.com/111222Bomba/Asset-Reuploader/internal/console"
	"github.com/111222Bomba/Asset-Reuploader/internal/files"
	"github.com/111222Bomba/Asset-Reuploader/internal/roblox"
)

var (
	cookieFile = config.Get("cookie_file")
	port       = config.Get("port")
)

func getCookie(c *roblox.Client) {
	color.Info.Print("Enter your .ROBLOSECURITY: ")

	var cookie string
	if _, err := fmt.Scanln(&cookie); err != nil {
		log.Fatal(err)
	}

	if cookie == "" {
		color.Error.Println("Cookie can't be empty.")
		getCookie(c)
		return
	}

	if err := c.Authenticate(cookie); err != nil {
		color.Error.Println(err)
		getCookie(c)
		return
	}
}

func main() {
	console.ClearScreen()

	fmt.Println("Authenticating cookie...")

	cookie, readErr := files.Read(cookieFile)
	cookie = strings.TrimSpace(cookie)

	c, clientErr := roblox.NewClient(cookie)
	console.ClearScreen()

	if readErr != nil || clientErr != nil {
		if readErr != nil && !os.IsNotExist(readErr) {
			color.Error.Println(readErr)
		}

		if clientErr != nil && cookie != "" {
			color.Error.Println(clientErr)
		}

		getCookie(c)
	}

	if err := files.Write(cookieFile, c.Cookie); err != nil {
		color.Error.Println("Failed to save cookie: ", err)
	}

	fmt.Println("localhost started on port " + port + ". Waiting to start reuploading.")
	if err := serve(c); err != nil {
		log.Fatal(err)
	}
}
