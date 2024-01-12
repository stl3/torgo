package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/oz/osdb"
	"github.com/sirupsen/logrus"
	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/stl3/torrodle"
	"github.com/stl3/torrodle/client"
	"github.com/stl3/torrodle/config"
	"github.com/stl3/torrodle/models"
	"github.com/stl3/torrodle/player"
)

const version = "0.1-beta"

var u, _ = user.Current()
var home = u.HomeDir
var configFile = filepath.Join(home, ".torgo.json")
var configurations config.TorrodleConfig

var dataDir string
var subtitlesDir string

func errorPrint(arg ...interface{}) {
	c := color.New(color.FgHiRed).Add(color.Bold)
	_, _ = c.Print("ERROR: ")
	_, _ = c.Println(arg...)
}

func infoPrint(arg ...interface{}) {
	c := color.New(color.FgHiYellow)
	_, _ = c.Print("[i] ")
	_, _ = c.Println(arg...)
}

func pickCategory() string {
	category := ""
	prompt := &survey.Select{
		Message: "Choose a category:",
		Options: []string{"All", "Movie", "TV", "Anime", "Porn"},
	}
	err := survey.AskOne(prompt, &category, nil)
	if err != nil {
		// Handle the error (e.g., print a message or return a default category)
		fmt.Println("Error selecting category:", err)
		return "All" // Default to "All" category in case of error
	}
	return category
}

func pickProviders(options []string) []interface{} {
	var providers []interface{}

	for {
		var chosen []string
		prompt := &survey.MultiSelect{
			Message: "Choose providers [use ? for help]:",
			Options: options,
			Help:    "[Use arrows to move, space to select checkbox, type to filter, enter when done]",
		}
		_ = survey.AskOne(prompt, &chosen, nil)

		if len(chosen) > 0 {
			for _, choice := range chosen {
				for _, provider := range torrodle.AllProviders {
					if provider.GetName() == choice {
						providers = append(providers, provider)
					}
				}
			}
			break // Exit the loop if choices are made
		} else {
			fmt.Println("Please select at least one provider.")
		}
	}

	return providers
}

func inputQuery() string {
	query := ""
	// prompt := &survey.Input{Message: "Search Torrents:"}
	prompt := &survey.Input{
		Message: color.HiMagentaString("Valid searches:\n") +
			color.HiYellowString("- Avatar the Way of Water\n"+
				"- (if no results use)=> Avatar.the.Way.of.Water\n"+
				"- Do the same for TV Shows.\n"+
				"- tv.show.s01e01\n"+
				"- movie.name.2023\n") +
			color.HiGreenString("Search Torrents:"),
	}
	_ = survey.AskOne(prompt, &query, nil)

	// // Properly encode the search query
	// query = url.QueryEscape(query)
	// fmt.Println(query)
	return query
}

func pickSortBy() string {
	sortBy := ""
	prompt := &survey.Select{
		Message: "Sort by:",
		Default: "default",
		Options: []string{"default", "seeders", "leechers", "size"},
	}
	_ = survey.AskOne(prompt, &sortBy, nil)
	return sortBy
}

func pickPlayer() string {
	options := []string{"None"}
	// options := []string{"None", "mpv-android", "vlc-android"}
	playerChoice := ""
	for _, p := range player.Players {
		options = append(options, p.Name)
	}
	// fmt.Println("Select None for standalone/mpv-android/vlc-android options")
	// fmt.Println(color.HiYellowString("Select None for standalone/mpv-android/vlc-android options"))
	fmt.Println(color.HiYellowString("Select None for standalone server"))
	// fmt.Println(color.GreenString("Select None for standalone/mpv-android/vlc-android options"))

	prompt := &survey.Select{
		Message: "Player:",
		Options: options,
	}
	_ = survey.AskOne(prompt, &playerChoice, nil)
	return playerChoice
}

func chooseResults(results []models.Source) string {
	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "S", "L", "Size"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.BgHiYellowColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.BgHiGreenColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.BgHiRedColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.BgHiCyanColor, tablewriter.FgBlackColor},
	)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor},
		tablewriter.Colors{},
		tablewriter.Colors{tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.FgHiRedColor},
		tablewriter.Colors{tablewriter.FgHiCyanColor},
	)

	for i, result := range results {
		title := strings.TrimSpace(result.Title)

		if result.Title != "" || result.Seeders > 0 || result.Leechers > 0 || result.FileSize > 0 {
			isEng := utf8.RuneCountInString(title) == len(title)

			// Adjusted truncation limits for titles
			if isEng {
				if len(title) > 65 { // Increase the limit for English titles
					title = title[:55] + "..."
				}
			} else {
				if utf8.RuneCountInString(title) > 45 { // Increase the limit for non-English titles
					title = string([]rune(title)[:42]) + "..."
				}
			}

			table.Append([]string{
				strconv.Itoa(i + 1),
				title,
				strconv.Itoa(result.Seeders),
				strconv.Itoa(result.Leechers),
				humanize.Bytes(uint64(result.FileSize)),
			})
		}
	}

	table.Render()

	// Prompt choice
	choice := ""
	question := &survey.Question{
		Prompt: &survey.Input{Message: "Choice(#):"},
		Validate: func(val interface{}) error {
			index, err := strconv.Atoi(val.(string))
			if err != nil {
				return fmt.Errorf("input must be numbers")
			} else if index < 1 || index > len(results) {
				return fmt.Errorf("input range exceeded (1-%d)", len(results))
			}
			return nil
		},
	}
	_ = survey.Ask([]*survey.Question{question}, &choice)
	return choice
}

func pickLangs() []string {
	languagesMap := map[string]string{
		"English":               "eng",
		"Chinese (traditional)": "zht",
		"Chinese (simplified)":  "chi",
		"Arabic":                "ara",
		"Hindi":                 "hin",
		"Dutch":                 "dut",
		"French":                "fre",
		"Portuguese":            "por",
		"Russian":               "rus",
	}
	var languagesOpts []string
	for k := range languagesMap {
		languagesOpts = append(languagesOpts, k)
	}

	var chosen []string
	prompt := &survey.MultiSelect{
		Message: "Choose subtitles languages:",
		Default: []string{"English"},
		Options: languagesOpts,
	}
	_ = survey.AskOne(prompt, &chosen, nil)

	var languages []string
	for _, choice := range chosen {
		languages = append(languages, languagesMap[choice])
	}
	return languages
}

func chooseSubtitles(subtitles osdb.Subtitles) string {
	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetAutoWrapText(false)
	table.SetHeader([]string{"#", "Name", "Lang", "Fmt", "HI", "Size"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.BgHiYellowColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold},
		tablewriter.Colors{tablewriter.BgHiCyanColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.BgHiMagentaColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.BgHiBlueColor, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.BgYellowColor, tablewriter.FgBlackColor},
	)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiYellowColor},
		tablewriter.Colors{}, // name
		tablewriter.Colors{tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.FgHiMagentaColor},
		tablewriter.Colors{}, // hi
		tablewriter.Colors{tablewriter.FgYellowColor},
	)
	for i, sub := range subtitles {
		// hearing impaired
		var hiSymbol string
		hi, _ := strconv.Atoi(sub.SubHearingImpaired)
		if hi != 0 {
			hiSymbol = color.HiGreenString("Y")
		} else {
			hiSymbol = color.HiRedString("N")
		}
		// name
		name := strings.TrimSpace(sub.MovieReleaseName)
		if len(name) > 42 {
			name = name[:39] + "..."
		}
		// size
		size, _ := strconv.ParseUint(sub.SubSize, 10, 0)

		table.Append([]string{strconv.Itoa(i + 1), name, sub.SubLanguageID, sub.SubFormat, hiSymbol, humanize.Bytes(size)})
	}
	table.Render()

	// Prompt choice
	choice := ""
	question := &survey.Question{
		Prompt: &survey.Input{Message: "Choice(#):"},
		Validate: func(val interface{}) error {
			index, err := strconv.Atoi(val.(string))
			if err != nil {
				return fmt.Errorf("input must be numbers")
			} else if index < 1 || index > len(subtitles) {
				return fmt.Errorf("input range exceeded (1-%d)", len(subtitles))
			}
			return nil
		},
	}
	_ = survey.Ask([]*survey.Question{question}, &choice)
	return choice
}

func getSubtitles(query string) (subtitlePath string) {
	// yes or no
	need := false
	prompt := &survey.Confirm{
		Message: "Need subtitles?",
	}
	_ = survey.AskOne(prompt, &need, nil)
	// if need == false {
	// 	return
	// }
	if !need {
		return
	}

	// pick subtitle languages
	langs := pickLangs()
	c, _ := osdb.NewClient()
	if err := c.LogIn("", "", ""); err != nil {
		errorPrint(err)
		os.Exit(1)
	}
	// search subtitles
	langstr := strings.Join(langs, ",")
	params := []interface{}{
		c.Token,
		[]struct {
			Query string `xmlrpc:"query"`
			Langs string `xmlrpc:"sublanguageid"`
		}{{
			query,
			langstr,
		}},
	}
	subtitles, err := c.SearchSubtitles(&params)
	if err != nil {
		errorPrint(err)
		os.Exit(1)
	}
	if len(subtitles) == 0 {
		errorPrint("No subtitles found")
		return
	}
	// choose subtitles
	choice := chooseSubtitles(subtitles)
	index, _ := strconv.Atoi(choice)
	subtitle := subtitles[index-1]
	// download
	fmt.Println(color.HiYellowString("[i] Downloading subtitle to"), subtitlesDir)
	subtitlePath = filepath.Join(subtitlesDir, subtitle.SubFileName)
	err = c.DownloadTo(&subtitle, subtitlePath)
	if err != nil {
		errorPrint(err)
		os.Exit(1)
	}
	// cleanup
	_ = c.LogOut()
	_ = c.Close()
	return
}

func startClient(player *player.Player, source models.Source, subtitlePath string) {
	// Play the video
	infoPrint("Streaming torrent...")
	// create client
	c, err := client.NewClient(dataDir, configurations.TorrentPort, configurations.HostPort)

	if err != nil {
		errorPrint(err)
		os.Exit(1)
	}
	_, err = c.SetSource(source)
	if err != nil {
		errorPrint(err)
		os.Exit(1)
	}

	// start client
	c.Start()

	// handle exit signals
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func(interruptChannel chan os.Signal) {
		for range interruptChannel {
			c.Close()
			fmt.Print("\n")
			infoPrint("Exiting...")
			// Delete the directory
			dirPath := filepath.Join(dataDir, c.Torrent.Name())
			infoPrint("Deleting downloads...", filepath.Join(dataDir, c.Torrent.Name()))
			if err := os.RemoveAll(dirPath); err != nil {
				errorPrint("Error deleting directory:", err)
			}
			// Delete files inside subtitlesDir
			subtitleFiles, err := filepath.Glob(filepath.Join(subtitlesDir, "*"))
			if err != nil {
				errorPrint("Error getting subtitle files:", err)
			} else {
				for _, subtitlePath := range subtitleFiles {
					infoPrint("Deleting subtitles...", subtitlePath)
					if err := os.Remove(subtitlePath); err != nil {
						errorPrint("Error deleting subtitle file:", err)
					}
				}
			}
			os.Exit(0)
		}
	}(interruptChannel)
	if player != nil {
		// serve via HTTP
		c.Serve()

		fmt.Println(color.HiYellowString("[i] Serving on"), c.URL)
		// goroutine ticker loop to update PrintProgress
		go func() {
			// Delay for ticker update time. Use whatever sane values you want. I use 500-1500
			ticker := time.NewTicker(1500 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				c.PrintProgress()
				fmt.Print("\r")
				os.Stdout.Sync() // Flush the output buffer to ensure immediate display
			}
		}()

		if subtitlePath != "" { // With subs
			if runtime.GOOS != "android" {

				// open player with subtitle
				player.Start(c.URL, subtitlePath, c.Torrent.Name())
				// Just for debugging:
				// fmt.Println(color.HiYellowString("[i] Launched player with subtitle"), subtitlePath)
			} else if runtime.GOOS == "android" {
				if player.Name == "mpv" {
					cmd := exec.Command("am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", c.URL, "-n", "is.xyz.mpv/.MPVActivity")
					logCmd(cmd)
					err_cmd := cmd.Run()
					if err_cmd != nil {
						fmt.Println("Error:", err)
					}
					gofuncTicker(c)
				} else if player.Name == "vlc" {
					cmd := exec.Command("am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", c.URL, "-n", "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity")
					logCmd(cmd)
					err_cmd := cmd.Run()
					if err_cmd != nil {
						fmt.Println("Error:", err)
					}
					gofuncTicker(c)
				// }
				} else if player.Name == "Chromecast" {
					cmd := exec.Command("go-chromecast", "-a", "10.0.0.107", "load", c.URL )
					logCmd(cmd)
					err_cmd := cmd.Run()
					if err_cmd != nil {
						fmt.Println("Error:", err)
					}
					gofuncTicker(c)
				}
			}
		} else { // Without subs
			if runtime.GOOS == "android" {
				if player.Name == "mpv" {
					cmd := exec.Command("am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", c.URL, "-n", "is.xyz.mpv/.MPVActivity")
					logCmd(cmd)
					err_cmd := cmd.Run()
					if err_cmd != nil {
						fmt.Println("Error:", err)
					}
					gofuncTicker(c)
				} else if player.Name == "vlc" {
					cmd := exec.Command("am", "start", "--user", "0", "-a", "android.intent.action.VIEW", "-d", c.URL, "-n", "org.videolan.vlc/org.videolan.vlc.gui.video.VideoPlayerActivity")
					logCmd(cmd)
					err_cmd := cmd.Run()
					if err_cmd != nil {
						fmt.Println("Error:", err)
					}
					gofuncTicker(c)
				}
			} else {
				// open player without subtitle
				player.Start(c.URL, "", c.Torrent.Name())
				// Just for debugging:
				fmt.Println(color.HiYellowString("[i] Launched player without subtitle"), player.Name)
			}
		}
	} else {
		c.Serve()
		fmt.Println(color.HiYellowString("[i] Serving on"), c.URL)
		// gofuncTicker(c) // No player command for this case
	}

	fmt.Print("\n")
	infoPrint("Exiting...")
	// Delete the directory
	dirPath := filepath.Join(dataDir, c.Torrent.Name())
	infoPrint("Deleting downloads from: ", filepath.Join(dataDir, c.Torrent.Name()))
	if err := os.RemoveAll(dirPath); err != nil {
		errorPrint("Error deleting directory:", err)
	}
	// Delete files inside subtitlesDir
	subtitleFiles, err := filepath.Glob(filepath.Join(subtitlesDir, "*"))
	if err != nil {
		errorPrint("No subtitles to delete", err)
	} else {
		for _, subtitlePath := range subtitleFiles {
			infoPrint("Deleting subtitles...", subtitlePath)
			if err := os.Remove(subtitlePath); err != nil {
				errorPrint("Error deleting subtitle file:", err)
			}
		}
	}
	os.Exit(0)

}

func logCmd(cmd *exec.Cmd) {
	log.Printf("\x1b[36mLaunching player:\x1b[0m \x1b[33m%v\x1b[0m\n", cmd)
}

func gofuncTicker(c *client.Client) {
	go func() {
		ticker := time.NewTicker(1500 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			c.PrintProgress()
			fmt.Print("\r")
			os.Stdout.Sync() // Flush the output buffer to ensure immediate display
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig // Wait for Ctrl+C
}

func init() {
	var err error

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		err = config.InitConfig(configFile)
		if err != nil {
			fmt.Printf("Error initializing config (%v): %v\n", configFile, err)
			os.Exit(1)
		}
	}

	configurations, err = config.LoadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		os.Exit(1)
	}

	dataDir = configurations.DataDir
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "torrodle")
	} else if strings.HasPrefix(dataDir, "~/") {
		dataDir = filepath.Join(home, dataDir[2:]) // expand user home directoy for path in configurations file
	}
	configurations.DataDir = dataDir
	subtitlesDir = filepath.Join(dataDir, "subtitles")

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		_ = os.Mkdir(dataDir, 0700)
	}
	if _, err := os.Stat(subtitlesDir); os.IsNotExist(err) {
		_ = os.Mkdir(subtitlesDir, 0700)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		DisableTimestamp:       true,
		DisableLevelTruncation: false,
	})
	logrus.SetOutput(os.Stdout)

	if configurations.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.ErrorLevel)
	}

}

func main() {
	name := color.HiYellowString("[torgo v%s]", version)
	banner :=
		`
	boop~
	,-.___,-.
	\_/_ _\_/
	  )O_O(
	 { (_) }
	  ` + "`" + `-^-' 
    You are using %v
`
	heart := color.HiRedString("<3")
	bold := color.New(color.Bold)
	// Startup
	fmt.Printf(banner, name)
	_, _ = bold.Print("    Made with ")
	fmt.Print(heart)
	_, _ = bold.Print(" by someone ")
	fmt.Print("(https://github.com/stl3/torrodle)\n\n")
	logrus.Debug(configurations)

	// Stream torrent from magnet provided in command-line
	if len(os.Args) > 1 {
		// make source
		source := models.Source{
			From:   "User Provided",
			Title:  "Unknown",
			Magnet: os.Args[1],
		}
		// player
		playerChoice := pickPlayer()
		if playerChoice == "" {
			errorPrint("Operation aborted")
			return
		}
		var p *player.Player
		if playerChoice == "None" {
			p = nil
		} else {
			p = player.GetPlayer(playerChoice)
		}
		// start
		startClient(p, source, "")
	}

	// Prepare options and query for searching torrents
	category := pickCategory()
	if category == "" {
		errorPrint("Operation aborted")
		return
	}
	cat := torrodle.Category(strings.ToUpper(category))
	var options []string
	// check for availibility of each category for each provider
	for _, provider := range torrodle.AllProviders {
		if torrodle.GetCategoryURL(cat, provider.GetCategories()) != "" {
			options = append(options, provider.GetName())
		}
	}
	providers := pickProviders(options)
	if len(providers) == 0 {
		errorPrint("Operation aborted")
		return
	}
	query := inputQuery()
	query = strings.TrimSpace(query)
	// Replace spaces with dots
	// query = strings.ReplaceAll(query, " ", ".")

	if query == "" {
		errorPrint("Operation aborted")
		return
	}
	sortBy := pickSortBy()
	if sortBy == "" {
		errorPrint("Operation aborted")
		return
	}
	sb := torrodle.SortBy(strings.ToLower(sortBy))

	// Call torrodle API to search for torrents
	limit := configurations.ResultsLimit
	results := torrodle.ListResults(providers, query, limit, cat, sb)
	if len(results) == 0 {
		errorPrint("No torrents found")
		return
	}
	// Prompt for source choosing
	fmt.Print("\033c") // reset screen
	choice := chooseResults(results)
	if choice == "" {
		errorPrint("Operation aborted")
		return
	}
	index, _ := strconv.Atoi(choice)
	source := results[index-1]

	// Print source information
	fmt.Print("\033c") // reset screen
	boldYellow := color.New(color.Bold, color.FgBlue)
	_, _ = boldYellow.Print("Title: ")
	fmt.Println(source.Title)
	_, _ = boldYellow.Print("From: ")
	fmt.Println(source.From)
	_, _ = boldYellow.Print("URL: ")
	fmt.Println(source.URL)
	_, _ = boldYellow.Print("Seeders: ")
	color.Green(strconv.Itoa(source.Seeders))
	_, _ = boldYellow.Print("Leechers: ")
	color.Red(strconv.Itoa(source.Leechers))
	_, _ = boldYellow.Print("FileSize: ")
	humanFileSize := humanize.Bytes(uint64(source.FileSize))
	fmt.Println(color.CyanString(humanFileSize))
	_, _ = boldYellow.Print("Magnet: ")
	truncatedMagnet := truncateMagnet(source.Magnet, 60) // Adjust the length as needed
	fmt.Println(truncatedMagnet)

	// Player
	playerChoice := pickPlayer()
	if playerChoice == "" {
		errorPrint("Operation aborted")
		return
	}
	var p *player.Player
	var subtitlePath string
	if playerChoice == "None" {
		p = nil
		// Asks for subtitles when using no player
		// subtitlePath = getSubtitles(source.Title)
	} else {
		// Get subtitles
		subtitlePath = getSubtitles(source.Title)
		p = player.GetPlayer(playerChoice)
	}

	// Start playing video...
	startClient(p, source, subtitlePath)

}

func truncateMagnet(magnet string, maxLength int) string {
	if len(magnet) > maxLength {
		return magnet[:maxLength]
	}
	return magnet
}
