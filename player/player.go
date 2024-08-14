/*
This package is the embeded version of 'github.com/Sioro-Neoku/go-peerflix/'.
We did some modifications on it in order to let it fit into 'torgo'
*/
package player

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	// "github.com/sirupsen/logrus"
	"github.com/stl3/torgo/config"
)

// // Get the current working directory
// currentDir, err := os.Getwd()
// if err != nil {
//     fmt.Println("Error getting current directory:", err)
//     return
// }

// // Construct the path to the .torgo.json file in the current directory
// var configFile = filepath.Join(currentDir, ".torgo.json")

var u, _ = user.Current()
var home = u.HomeDir
var configFile = filepath.Join(home, ".torgo.json")

// Declare Mpv_params as a package-level variable
var MpvParams string

// Function to load the configuration
func loadConfig() {
	configurations, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		configurations = config.TorgoConfig{}
	}
	// Shows used config options from json
	// fmt.Printf("Loaded configuration: %+v\n", configurations)
	MpvParams = configurations.Mpv_params
}

// Players holds structs of all supported players.
var Players = func() []Player {
	// fmt.Printf("Loaded configuration: %+v\n", configurations)
	loadConfig()
	// Initialize the Players slice
	return []Player{
		{
			Name:            "mpv",
			DarwinCommand:   []string{"mpv"},
			LinuxCommand:    []string{"mpv"},
			AndroidCommand:  []string{"am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d"},
			SubtitleCommand: "--sub-file=",
			TitleCommand:    "--force-media-title=", // Shows the movie folder name as title instead of http://localhost:port
			WindowsCommand: func() []string {
				fmt.Println("mpv params loaded:", MpvParams)
				if MpvParams != "" {
					return []string{"mpv", MpvParams, "--no-resume-playback", "--no-terminal"}
				}
				fmt.Println("Mpv_params is empty")
				return []string{"mpv", "--no-resume-playback", "--no-terminal"}
			}(),
		},
		{
			Name:           "vlc",
			DarwinCommand:  []string{"/Applications/VLC.app/Contents/MacOS/VLC"},
			LinuxCommand:   []string{"vlc"},
			AndroidCommand: []string{"am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d"},
			// WindowsCommand:  []string{"%ProgramFiles%\\VideoLAN\\VLC\\vlc.exe"},
			WindowsCommand:  []string{"vlc.exe"}, // vlc player should be in users env path in case installed in non-default path
			SubtitleCommand: "--sub-file=",
			TitleCommand:    "--meta-title=", //
		},
		{
			Name:           "KMPlayer",
			WindowsCommand: []string{"KMPlayer.exe"}, // Do people use this?
		},
		{
			Name: "chromecast",
			// ChromecastCommand: []string{"F:\\go\\bin\\go-chromecast.exe", "-a", "10.0.0.107", "load"},
			WindowsCommand: []string{"go-chromecast.exe", "-a", "10.0.0.107", "load"}, // shit don't work
		},
	}
}()

// Player manages the execution of a media player.
type Player struct {
	Name string
	// Type            PlayerType // New field to indicate the player type
	DarwinCommand  []string
	LinuxCommand   []string
	WindowsCommand []string
	AndroidCommand []string
	// ChromecastCommand []string
	SubtitleCommand string
	TitleCommand    string
	started         bool
}

// Start launches the Player with the given command and arguments in subprocess.
func (player *Player) Start(url string, subtitlePath string, title string) {
	// if player.started == true {
	if player.started {
		// prevent multiple calls
		return
	}

	var command []string
	switch runtime.GOOS {
	case "darwin":
		command = player.DarwinCommand
	case "linux":
		command = player.LinuxCommand
	case "windows":
		command = player.WindowsCommand
	case "android":
		command = player.AndroidCommand
	}

	// Wait for server to be ready
	timeout := 1250 * time.Millisecond // Adjust the timeout as needed
	err := waitForServer(url, timeout)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Append the video URL to the command for non-Android cases
	command = append(command, url)

	if player.Name == "mpv" && runtime.GOOS == "android" {
		fmt.Println("Using mpv")
		command = append(command, "-n", "is.xyz.mpv/.MPVActivity")

	} else if player.Name == "vlc" && runtime.GOOS == "android" {
		fmt.Println("Using VLC")
		command = append(command, "-n", "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity")
	}

	if subtitlePath != "" && runtime.GOOS != "android" {
		command = append(command, player.SubtitleCommand+subtitlePath)
	}
	if title != "" && runtime.GOOS != "android" {
		command = append(command, player.TitleCommand+title)
	}

	log.Printf("\x1b[36mLaunching player:\x1b[0m \x1b[33m%v\x1b[0m\n", command)
	// logrus.Debugf("command: %v\n", command)

	cmd := exec.Command(command[0], command[1:]...)
	player.started = true

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting player: %v\n", err)
		return
	}
	// Wait for the player process to complete
	if err := cmd.Wait(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			log.Printf("Player exited with non-zero status: %v\n", exitErr.ExitCode())
		} else {
			log.Printf("Error waiting for player: %v\n", err)
		}
	}

	// Reset the started flag to allow for subsequent calls
	player.started = false

}

// GetPlayer returns the Player struct of the given player name.
func GetPlayer(name string) *Player {
	for _, player := range Players {
		if strings.EqualFold(player.Name, name) {
			return &player
		}
	}
	return nil
}

func waitForServer(url string, timeout time.Duration) error {
	startTime := time.Now()

	for {
		// Attempt to make a request to the server
		_, err := http.Get(url)
		if err == nil {
			// Server is reachable
			fmt.Println("Server is running!")
			return nil
		}

		// Check if the timeout has been reached
		if time.Since(startTime) >= timeout {
			return fmt.Errorf("timed out waiting for the server to start")
		}

		// Introduce a short delay before checking again
		time.Sleep(1 * time.Second)
	}
}
