/*
This package is the embeded version of 'github.com/Sioro-Neoku/go-peerflix/'.
We did some modifications on it in order to let it fit into 'torrodle'
*/
package player

import (
	"fmt"
	"log"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	// "github.com/sirupsen/logrus"
	"github.com/stl3/torrodle/config"
)

var u, _ = user.Current()
var home = u.HomeDir
var configFile = filepath.Join(home, ".torgo.json")

// var configurations config.TorrodleConfig

// func init() {
// 	// Load the configuration during package initialization
// 	loadConfig()
// }

// // Function to load the configuration
// func loadConfig() {
// 	configurations, err := config.LoadConfig(configFile)
// 	if err != nil {
// 		fmt.Println("Error loading config:", err)
// 		configurations = config.TorrodleConfig{}
// 	}
// 	fmt.Printf("Loaded configuration: %+v\n", configurations)
// }

// Declare Mpv_params as a package-level variable
var MpvParams string

// var configurations config.TorrodleConfig

// var configurations = config.TorrodleConfig{}

// func init() {
// 	// Load the configuration
// 	u, _ := user.Current()
// 	home := u.HomeDir
// 	configFile := filepath.Join(home, ".torgo.json")

// 	// configurations, err := config.LoadConfig(configFile)
// 	// if err != nil {
// 	// 	fmt.Println("Error loading config:", err)
// 	// 	configurations = config.TorrodleConfig{}
// 	// }
// 	// fmt.Printf("Loaded configuration: %+v\n", configurations)
// }

// func init() {
// 	// Load the configuration during package initialization
// 	configurations, err := config.LoadConfig(configFile)
// 	if err != nil {
// 		fmt.Println("Error loading config:", err)
// 		configurations = config.TorrodleConfig{}
// 	}
// 	fmt.Printf("Loaded configuration: %+v\n", configurations)
// }

// func init() {
// 	// Load the configuration during package initialization
// 	loadConfig()
// }

// // Function to load the configuration
func loadConfig() {
	configurations, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		configurations = config.TorrodleConfig{}
	}
	// Shows used config options from json
	// fmt.Printf("Loaded configuration: %+v\n", configurations)
	MpvParams = configurations.Mpv_params
}

// Players holds structs of all supported players.
var Players = []Player{
	{
		Name:          "mpv",
		DarwinCommand: []string{"mpv"},
		LinuxCommand:  []string{"mpv"},
		// AndroidCommand: []string{},
		AndroidCommand:  []string{"am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d"},
		SubtitleCommand: "--sub-file=",
		TitleCommand:    "--force-media-title=", // Shows the movie folder name as title instead of http://localhost:port
		// Check if configurations.Mpv_params is not empty, and adjust WindowsCommand accordingly
		WindowsCommand: func() []string {
			// Load the configuration
			// // configurations, err := config.LoadConfig(configFile)
			// // if err != nil {
			// // 	fmt.Println("Error loading config:", err)
			// // 	configurations = config.TorrodleConfig{}
			// // }
			loadConfig()
			// // fmt.Println("Mpv_params:", configurations.Mpv_params)
			fmt.Println("Mpv_params:", MpvParams)
			// fmt.Printf(configurations.Mpv_params)
			// logrus.Infof("Mpv_params: %s", configurations.Mpv_params)

			// if configurations.Mpv_params != "" {
			// 	return []string{"mpv", configurations.Mpv_params, "--no-resume-playback", "--no-terminal"}
			// }
			if MpvParams != "" {
				return []string{"mpv", MpvParams, "--no-resume-playback", "--no-terminal"}
			}
			fmt.Println("Mpv_params is empty")
			return []string{"mpv", "--no-resume-playback", "--no-terminal"}
		}(),
	},
	{
		Name:          "vlc",
		DarwinCommand: []string{"/Applications/VLC.app/Contents/MacOS/VLC"},
		LinuxCommand:  []string{"vlc"},
		// AndroidCommand: []string{},
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
		Name:           "Chromecast",
		WindowsCommand: []string{""},
	},
}

// Player manages the execution of a media player.
type Player struct {
	Name string
	// Type            PlayerType // New field to indicate the player type
	DarwinCommand   []string
	LinuxCommand    []string
	WindowsCommand  []string
	AndroidCommand  []string
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

	// configurations, err := config.LoadConfig(configFile)
	// if err != nil {
	// 	fmt.Println("Error loading config:", err)
	// 	configurations = config.TorrodleConfig{}
	// }
	// fmt.Printf("Loaded configuration: %+v\n", configurations)

	// configurations, err := config.LoadConfig(configFile)
	// if err != nil {
	// 	fmt.Println("Error loading config:", err)
	// 	configurations = config.TorrodleConfig{}
	// }
	// fmt.Printf(configurations.Mpv_params)

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
