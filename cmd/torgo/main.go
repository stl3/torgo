package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"regexp"
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

	// "github.com/stl3/osdb"

	// "gopkg.in/AlecAivazis/survey.v1"
	"github.com/AlecAivazis/survey/v2"

	"github.com/stl3/torgo"
	"github.com/stl3/torgo/client"
	"github.com/stl3/torgo/config"
	"github.com/stl3/torgo/models"
	"github.com/stl3/torgo/player"
)

const version = "0.1-beta"

var u, _ = user.Current()
var home = u.HomeDir
var configFile = filepath.Join(home, ".torgo.json")
var configurations config.TorgoConfig

var dataDir string
var subtitlesDir string
var tmagnet string

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
		Options: []string{"All", "Movie", "TV", "Anime", "Porn", "Audiobook", "Documentaries"},
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
				for _, provider := range torgo.AllProviders {
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
	playerChoice := ""
	for _, p := range player.Players {
		options = append(options, p.Name)
	}
	fmt.Println(color.HiYellowString("Select None for standalone server"))

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
	// Define a regular expression pattern for matching URLs
	urlPattern := regexp.MustCompile(`https://[^\s]+`)

	// Find all matches in the SubDownloadLink field
	matches := urlPattern.FindAllString(subtitle.ZipDownloadLink, -1)

	// Get the last URL
	var lastURL string
	if len(matches) > 0 {
		lastURL = matches[len(matches)-1]
	}
	// Print the result
	fmt.Println("Last URL:", lastURL)

	// download
	fmt.Println(color.HiYellowString("[i] Downloading subtitle to"), subtitlesDir)
	subtitlePath = filepath.Join(subtitlesDir, subtitle.SubFileName)
	// err = c.DownloadTo(&subtitle, subtitlePath)
	err = downloadAndExtractSubtitle(subtitle, subtitlePath)
	// err = parseAndDownloadSubtitle(subtitle, subtitlesDir)
	fmt.Println(color.HiYellowString("[i] SubtitlePath: "), subtitlePath)
	// fmt.Println(color.HiYellowString("[i] Subtitle: "), &subtitle)
	if err != nil {
		// errorPrint(err)
		// os.Exit(1)
		fmt.Println(color.HiRedString("Error downloading subtitle:"), err)
		// Ask the user if they want to try again
		tryAgain := false
		prompt := &survey.Confirm{
			Message: "Do you want to try again?",
		}
		_ = survey.AskOne(prompt, &tryAgain, nil)
		if tryAgain {
			// Recursively call the function again
			return getSubtitles(query)
		}
	}
	// cleanup
	_ = c.LogOut()
	_ = c.Close()
	return
}

func downloadAndExtractSubtitle(subtitle osdb.Subtitle, outputPath string) error {
	// Get the download URL from the last URL in the struct
	// downloadURL := subtitle.SubDownloadLink
	downloadURL := subtitle.ZipDownloadLink
	fmt.Println("Downloading subtitle from:", downloadURL)

	err := downloadWithWget(downloadURL, subtitlesDir)
	if err != nil {
		fmt.Printf("Error downloading file: %s\n", err)
		return err
	}

	// fmt.Printf("File downloaded to: %s\n", outputPath)
	// fmt.Printf("File downloaded to: %s\n", subtitlesDir)

	return nil
}

func downloadWithWget(url, outputPath string) error {
	// // // cmd := exec.Command("wget", "-O", outputPath+"/subtitle.zip", url)
	// // // cmd.Stdout = os.Stdout
	// // // cmd.Stderr = os.Stderr

	// // // err := cmd.Run()
	// // // if err != nil {
	// // // 	return fmt.Errorf("error running wget: %s", err)
	// // // }
	// // // fmt.Printf("File downloaded to: %s\n", subtitlesDir)

	// Create the output file
	outputFile, err := os.Create(filepath.Join(outputPath, "subtitle.zip"))
	if err != nil {
		return err
	}
	defer outputFile.Close()

	// Make the HTTP request
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Check if the response status is OK
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s", response.Status)
	}

	// Copy the response body to the output file
	_, err = io.Copy(outputFile, response.Body)
	if err != nil {
		return err
	}

	fmt.Printf("File downloaded to: %s\n", filepath.Join(outputPath, "subtitle.zip"))

	// Open the zip file
	reader, err := zip.OpenReader(outputPath + "/subtitle.zip")
	if err != nil {
		return err
	}
	defer reader.Close()

	// Extract files from the zip archive
	for _, file := range reader.File {
		filePath := filepath.Join(outputPath, file.Name)
		if strings.HasSuffix(filePath, string(filepath.Separator)) {
			// Skip directory entries
			continue
		}

		// Create the file
		fileWriter, err := os.Create(filePath)
		if err != nil {
			return err
		}

		// Read from the zip file and write to the destination file
		fileReader, err := file.Open()
		if err != nil {
			fileWriter.Close()
			return err
		}
		_, err = io.Copy(fileWriter, fileReader)

		fileReader.Close()
		fileWriter.Close()

		if err != nil {
			return err
		}
	}

	fmt.Println("Subtitle extracted to:", outputPath)

	return nil
}

func startClient(player *player.Player, source models.Source, subtitlePath string) {
	var printProgressEnabled = true

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

	// Create a channel to signal when c.Start() has completed
	done := make(chan struct{})

	// Start c.Start() in a goroutine
	go func() {
		defer close(done)
		c.Start()
	}()

	// Wait for c.Start() to complete
	<-done
	tn := c.Torrent.Name()
	// Introduce a flag to control whether c.PrintProgress() should be executed

	// WatchedDatabase-torgo.db code
	// Check if "WatchedDatabase-torgo.db" exists
	watchedDBPath := "WatchedDatabase-torgo.db"
	if _, err := os.Stat(watchedDBPath); os.IsNotExist(err) {
		// Create the file if it doesn't exist
		createWatchedDB(watchedDBPath)
	} else {
		// File exists, check number of lines
		numLines, err := countLines(watchedDBPath)
		if err != nil {
			errorPrint("Error counting lines in WatchedDatabase:", err)
			os.Exit(1)
		}

		if numLines > 500 {
			// Rename the file to "WatchedDatabase-torgo-{datetime}.db"
			renameWatchedDB(watchedDBPath)

			// Create a new "WatchedDatabase-torgo.db"
			createWatchedDB(watchedDBPath)
		}
	}

	// handle exit signals
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	// Channels to control goroutines
	exitChan := make(chan struct{})
	progressStopChan := make(chan struct{})

	go func(interruptChannel chan os.Signal, exitChan chan struct{}, progressStopChan chan struct{}) {
		for {
			select {
			case <-interruptChannel:
				close(progressStopChan)
				// infoPrint("Stopping processes...")
				// Set the flag to disable PrintProgress
				printProgressEnabled = false
				// c.Stop()
				// c.Close()
				// fmt.Print("\n")
				// infoPrint("Exiting...")
				dirPath := filepath.Join(dataDir, tn)
				fmt.Print("\n")
				// infoPrint("Deleting downloads...", filepath.Join(dataDir, tn))

				// Define the maximum number of retries
				maxRetries := 5
				retryCount := 0

				for {
					time.Sleep(2500 * time.Millisecond)

					if err := os.RemoveAll(dirPath); err != nil {
						errorPrint("Error deleting directory:", err)

						// Check if maximum retries reached
						if retryCount >= maxRetries {
							errorPrint("Maximum retries reached. Exiting...")
							os.Exit(1)
						}

						// Increment the retry count
						retryCount++
						infoPrint("Retrying...", retryCount)
						continue // Retry the deletion
					}
					// Deletion successful, break out of the loop
					break
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

			case <-exitChan:

				close(progressStopChan) // Signal to stop the progress goroutine
				// Set the flag to disable PrintProgress
				printProgressEnabled = false
				return // Exit the goroutine
				// }
			}
		}
	}(interruptChannel, exitChan, progressStopChan)
	if player != nil {
		// serve via HTTP
		c.Serve()
		// Wait until client.LargestFile has a value
		for {
			if c.LargestFile != nil {

				break
			}

			// Introduce a short delay before checking again
			time.Sleep(500 * time.Millisecond)
		}
		selectedTitle := c.LargestFile.DisplayPath()

		// We write watch data to file
		file, err := os.OpenFile(watchedDBPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			errorPrint("Error opening WatchedDatabase for append:", err)
			os.Exit(1)
		}
		defer file.Close()

		// Get current date and time in the desired format
		currentDate := time.Now().Format("2006-01-02 15:04:05")

		// Prepare the data to append
		data := fmt.Sprintf("[%s] Torrent: %s Title: %s Magnet: %s\n", currentDate, tn, selectedTitle, tmagnet)

		// Append the data to the file
		if _, err := file.WriteString(data); err != nil {
			errorPrint("Error appending data to WatchedDatabase:", err)
			// os.Exit(1)
		}

		fmt.Println(color.HiYellowString("[i] Serving on"), c.URL)
		fmt.Println(color.HiYellowString("[i] Torrent Port:"), c.ClientConfig.ListenPort)
		// goroutine ticker loop to update PrintProgress
		go func() {
			// Delay for ticker update time. Use whatever sane values you want. I use 500-1500
			ticker := time.NewTicker(1500 * time.Millisecond)
			defer ticker.Stop()

			// for range ticker.C {
			for {
				select {
				// // case <-ticker.C:
				// // 	c.PrintProgress()
				// // 	fmt.Print("\r")
				// // 	os.Stdout.Sync() // Flush the output buffer to ensure immediate display
				case <-ticker.C:
					if printProgressEnabled {
						c.PrintProgress()
						fmt.Print("\r")
						os.Stdout.Sync() // Flush the output buffer to ensure immediate display
					}
				case <-progressStopChan:
					return // Exit the goroutine when signaled
				}
			}
		}()

		if subtitlePath != "" { // With subs
			if runtime.GOOS != "android" {
				player.Start(c.URL, subtitlePath, selectedTitle)
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
				player.Start(c.URL, "", selectedTitle)
				// Just for debugging:
				fmt.Println(color.HiYellowString("[i] Launched player without subtitle"), player.Name)
			}
		}
	} else {
		// Just host server
		c.Serve()
		// Wait until client.LargestFile has a value
		for {
			if c.LargestFile != nil {

				break
			}

			// Introduce a short delay before checking again
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Println(color.HiYellowString("[i] Serving on"), c.URL)
		gofuncTicker(c)
	}
	// tn := c.Torrent.Name()
	// c.Stop()
	infoPrint("Stopping processes...")
	// Set the flag to disable PrintProgress
	printProgressEnabled = false
	close(exitChan)
	c.Close()
	fmt.Print("\n")
	infoPrint("Exiting...")
	dirPath := filepath.Join(dataDir, tn)
	fmt.Print("\n")
	infoPrint("Deleting downloads...", filepath.Join(dataDir, tn))

	// Define the maximum number of retries
	maxRetries := 5
	retryCount := 0

	for {
		time.Sleep(2500 * time.Millisecond)

		if err := os.RemoveAll(dirPath); err != nil {
			errorPrint("Error deleting directory:", err)

			// Check if maximum retries reached
			if retryCount >= maxRetries {
				errorPrint("Maximum retries reached. Exiting...")
				os.Exit(1)
			}

			// Increment the retry count
			retryCount++
			infoPrint("Retrying...", retryCount)
			continue // Retry the deletion
		}
		// Deletion successful, break out of the loop
		break
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

func createWatchedDB(filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		errorPrint("Error creating WatchedDatabase:", err)
		os.Exit(1)
	}
	defer file.Close()
}

func countLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount, nil
}

func renameWatchedDB(filePath string) {
	// Get current date and time in the desired format
	datetime := time.Now().Format("02-1-2006-150PM")
	newFilePath := fmt.Sprintf("WatchedDatabase-torgo-%s.db", datetime)

	// Rename the file
	err := os.Rename(filePath, newFilePath)
	if err != nil {
		errorPrint("Error renaming WatchedDatabase:", err)
		os.Exit(1)
	}
}

// func appendWatchedData(filePath string) {
// 	// Open the file in append mode
// 	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
// 	if err != nil {
// 		errorPrint("Error opening WatchedDatabase for append:", err)
// 		os.Exit(1)
// 	}
// 	defer file.Close()

// 	// Get current date and time in the desired format
// 	currentDate := time.Now().Format("2006-01-02 15:04:05")

// 	// Prepare the data to append
// 	data := fmt.Sprintf("[%s] Torrent: %s Title: %s Magnet: %s\n", currentDate,  c.client.Torrent.Name(), client.Client.LargestFile.DisplayPath(), tmagnet)

// 	// Append the data to the file
// 	if _, err := file.WriteString(data); err != nil {
// 		errorPrint("Error appending data to WatchedDatabase:", err)
// 		os.Exit(1)
// 	}
// }

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
		dataDir = filepath.Join(os.TempDir(), "torgo")
	} else if strings.HasPrefix(dataDir, "~/") {
		dataDir = filepath.Join(home, dataDir[2:]) // expand user home directory for path in configurations file
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
	fmt.Print("(https://github.com/stl3/torgo)\n\n")
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
	cat := torgo.Category(strings.ToUpper(category))
	var options []string
	// check for availibility of each category for each provider
	for _, provider := range torgo.AllProviders {
		if torgo.GetCategoryURL(cat, provider.GetCategories()) != "" {
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
	sb := torgo.SortBy(strings.ToLower(sortBy))

	// Call torgo API to search for torrents
	limit := configurations.ResultsLimit
	results := torgo.ListResults(providers, query, limit, cat, sb)
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
	tmagnet = truncatedMagnet

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
