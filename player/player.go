/*
This package is the embeded version of 'github.com/Sioro-Neoku/go-peerflix/'.
We did some modifications on it in order to let it fit into 'torrodle'
*/
package player

import (
	"log"
	"os/exec"
	"runtime"
	"strings"
)

// Players holds structs of all supported players.
var Players = []Player{
	{
		Name:           "mpv",
		DarwinCommand:  []string{"mpv"},
		LinuxCommand:   []string{"mpv"},
		AndroidCommand: []string{"mpv"},
		// WindowsCommand: []string{"mpv", "--no-resume-playback", "--no-terminal"}, // Default
		WindowsCommand:  []string{"mpv", "--profile=movie-flask", "--no-resume-playback", "--no-terminal"}, // Just for use with my mpv profile
		SubtitleCommand: "--sub-file=",
		TitleCommand:    "--force-media-title=", // Shows the movie folder name as title instead of http://localhost:port
	},
	{
		Name:           "vlc",
		DarwinCommand:  []string{"/Applications/VLC.app/Contents/MacOS/VLC"},
		LinuxCommand:   []string{"vlc"},
		AndroidCommand: []string{"vlc"},
		// WindowsCommand:  []string{"%ProgramFiles%\\VideoLAN\\VLC\\vlc.exe"},
		WindowsCommand:  []string{"vlc.exe"}, // vlc player should be in users env path in case installed in non-default path
		SubtitleCommand: "--sub-file=",
		TitleCommand:    "--meta-title=", //
	},
	// {
	// 	Name: "kmp",
	// 	// DarwinCommand: []string{"/Applications/VLC.app/Contents/MacOS/VLC"},
	// 	// LinuxCommand:  []string{"vlc"},
	// 	// WindowsCommand:  []string{"%ProgramFiles%\\VideoLAN\\VLC\\vlc.exe"},
	// 	WindowsCommand:  []string{"Kmplayer.exe"},
	// 	SubtitleCommand: "--sub-file=",
	// 	// TitleCommand:    "--title=",
	// },
}

// Player manages the execiution of a media player.
type Player struct {
	Name            string
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
	}
	command = append(command, url)
	if subtitlePath != "" {
		command = append(command, player.SubtitleCommand+subtitlePath)
	}
	if title != "" {
		command = append(command, player.TitleCommand+title)
	}
	log.Printf("\x1b[36mLaunching player:\x1b[0m \x1b[33m%v\x1b[0m\n", command)
	// logrus.Debugf("command: %v\n", command)
	cmd := exec.Command(command[0], command[1:]...)
	// cmd.Stdout = os.Stdout // show player output to console
	// cmd.Stdout = os.NewFile(0, os.DevNull)
	// cmd.Stderr = os.Stderr

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
