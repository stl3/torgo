/*
This package is the embeded version of 'github.com/Sioro-Neoku/go-peerflix/'.
We did some modifications on it in order to let it fit into 'torrodle'
*/
package player

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Players holds structs of all supported players.
var Players = []Player{
	{
		Name:          "mpv",
		DarwinCommand: []string{"mpv"},
		LinuxCommand:  []string{"mpv"},
		AndroidCommand: []string{},
		// AndroidCommand: []string{"am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d"},
		// WindowsCommand: []string{"mpv", "--no-resume-playback", "--no-terminal"}, // Default
		WindowsCommand:  []string{"mpv", "--profile=movie-flask", "--no-resume-playback", "--no-terminal"}, // Just for use with my mpv profile
		SubtitleCommand: "--sub-file=",
		TitleCommand:    "--force-media-title=", // Shows the movie folder name as title instead of http://localhost:port
	},
	{
		Name:          "vlc",
		DarwinCommand: []string{"/Applications/VLC.app/Contents/MacOS/VLC"},
		LinuxCommand:  []string{"vlc"},
		AndroidCommand: []string{},
		// AndroidCommand: []string{"am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d"},
		// WindowsCommand:  []string{"%ProgramFiles%\\VideoLAN\\VLC\\vlc.exe"},
		WindowsCommand:  []string{"vlc.exe"}, // vlc player should be in users env path in case installed in non-default path
		SubtitleCommand: "--sub-file=",
		TitleCommand:    "--meta-title=", //
	},
	{
		Name:           "KMPlayer",
		WindowsCommand: []string{"KMPlayer.exe"}, // Do people use this?
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
		// // player.startAndroidMPV(url)
		// mpv_android(url)
		// player.started = true
	}

	// Append the video URL to the command for non-Android cases
	// if runtime.GOOS != "android" {
	command = append(command, url)
	// command = append(command, url+`/`)

	// } else {
	//if player.Name == "mpv" && runtime.GOOS == "android" {
	//	// Do something based on the condition
	//	fmt.Println("Using mpv")
	//	// time.Sleep(3 * time.Second)
	//	command = append(command, "-n", "is.xyz.mpv/.MPVActivity")
	//	// } else if player.Type == vlc {
	//} else if player.Name == "vlc" && runtime.GOOS == "android" {
	//	// Do something else
	//	fmt.Println("Using VLC")
	//	// time.Sleep(3 * time.Second)
	//	command = append(command, "-n", "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity")
	//}
	// }

	if subtitlePath != "" && runtime.GOOS != "android" {
		// if subtitlePath != "" {
		command = append(command, player.TitleCommand+title)
	}
	if title != "" && runtime.GOOS != "android" {
		// if title != "" {
		command = append(command, player.TitleCommand+title)
	}

	log.Printf("\x1b[36mLaunching player:\x1b[0m \x1b[33m%v\x1b[0m\n", command)
	// logrus.Debugf("command: %v\n", command)

	cmd := exec.Command(command[0], command[1:]...)
	time.Sleep(6 * time.Second)
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

// func mpv_android(url string) {
// 	cmd := exec.Command("am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", url, "-n", "is.xyz.mpv/.MPVActivity")
// 	err := cmd.Run()
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	}
// }

// // startAndroidMPV launches mpv on Android using the specific intent.
// func (player *Player) startAndroidMPV(url string) {

// 	// time.Sleep(7 * time.Second)
// 	// cmd0 := exec.Command("mpv-go", url)
// 	// // cmd0 := exec.Command(player.AndroidCommand[0], "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", url, "-n", "is.xyz.mpv/.MPVActivity")
// 	// // err := cmd0.Run()
// 	// // if err != nil {
// 	// // 	log.Fatal(err)
// 	// // }
// 	cmd := exec.Command(player.AndroidCommand[0], "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", url, "-n", "is.xyz.mpv/.MPVActivity")
// 	// // urlWithQuotes := "\"" + url + "\""

// 	// // cmd := exec.Command(player.AndroidCommand[0], "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", urlWithQuotes, "-n", "is.xyz.mpv/.MPVActivity")
// 	player.started = true
// 	log.Printf("\x1b[36mLaunching player:\x1b[0m \x1b[33m%v\x1b[0m\n", cmd.Args)
// 	if err := cmd.Start(); err != nil {
// 		log.Printf("Error starting player: %v\n", err)
// 		return
// 	}
// 	// Wait for the player process to complete
// 	if err := cmd.Wait(); err != nil {
// 		exitErr, ok := err.(*exec.ExitError)
// 		if ok {
// 			log.Printf("Player exited with non-zero status: %v\n", exitErr.ExitCode())
// 		} else {
// 			log.Printf("Error waiting for player: %v\n", err)
// 		}
// 	}
// 	player.started = false
// }
