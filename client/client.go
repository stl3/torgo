/*
This package is the embeded version of 'github.com/Sioro-Neoku/go-peerflix/'.
We did some modifications on it in order to let it fit into 'Torrodle'
*/
package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"

	"github.com/stl3/torrodle/config"
	"github.com/stl3/torrodle/models"
)

// Client manages the torrent downloading.
type Client struct {
	Client        *torrent.Client
	ClientConfig  *torrent.ClientConfig
	Torrent       *torrent.Torrent
	Source        models.Source
	URL           string
	HostPort      int
	lastPrintTime time.Time
}

var u, _ = user.Current()
var home = u.HomeDir
var configFile = filepath.Join(home, ".torrodle.json")
var configurations config.TorrodleConfig

// NewClient initializes a new torrent client.
func NewClient(dataDir string, torrentPort int, hostPort int) (Client, error) {
	var client Client

	// Initialize Config
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = dataDir
	clientConfig.ListenPort = torrentPort
	clientConfig.NoUpload = true
	clientConfig.Seed = false
	clientConfig.Debug = false
	clientConfig.EstablishedConnsPerTorrent = configurations.ECPT
	clientConfig.HalfOpenConnsPerTorrent = configurations.HOCPT
	clientConfig.TotalHalfOpenConns = configurations.THOC
	client.ClientConfig = clientConfig

	clientConfig.HTTPProxy = func(req *http.Request) (*url.URL, error) {

		proxyURL, err := url.Parse(configurations.Proxy)
		if err != nil {
			return nil, err
		}
		return proxyURL, nil
	}

	// Create Client
	c, err := torrent.NewClient(clientConfig)
	if err != nil {
		return client, err
	}
	client.Client = c
	client.HostPort = hostPort

	return client, err
}

// SetSource sets the source (magnet uri) which the client is based on.
// * must be called before `Client.Start()`
func (client *Client) SetSource(source models.Source) (*Client, error) {
	client.Source = source
	t, err := client.Client.AddMagnet(source.Magnet)
	if err == nil {
		t.SetDisplayName(source.Title)
		client.Torrent = t
	}
	return client, err
}

func (client *Client) getLargestFile() *torrent.File {
	var largestFile *torrent.File
	var lastFileSize int64
	for _, file := range client.Torrent.Files() {
		if file.Length() > lastFileSize {
			lastFileSize = file.Length()
			largestFile = file
		}
	}
	return largestFile
}

func (client *Client) download() {
	t := client.Torrent
	t.DownloadAll()
	// Set priorities of file (5% ahead)
	for {
		largestFile := client.getLargestFile()
		firstPieceIndex := largestFile.Offset() * int64(t.NumPieces()) / t.Length()
		endPieceIndex := (largestFile.Offset() + largestFile.Length()) * int64(t.NumPieces()) / t.Length()
		for idx := firstPieceIndex; idx <= endPieceIndex*10/100; idx++ {
			t.Piece(int(idx)).SetPriority(torrent.PiecePriorityNow)
			// Check if the download is complete
			if t.BytesCompleted() == t.Length() {
				break // exit the loop if download is complete
			}
			// Sleep for a short duration before checking again
			time.Sleep(1 * time.Second)
		}
	}
}

// Start starts the client by getting the torrent information and allocating the priorities of each piece.
func (client *Client) Start() {
	<-client.Torrent.GotInfo() // blocks until it got the info
	go client.download()       // download file
}

func (client *Client) streamHandler(w http.ResponseWriter, r *http.Request) {
	file := client.getLargestFile()
	entry, err := NewFileReader(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.DisplayPath()+"\"")
	http.ServeContent(w, r, file.DisplayPath(), time.Now(), entry)
}

// Serve serves the torrent via HTTP localhost:{port}.
func (client *Client) Serve() {
	p := strconv.Itoa(client.HostPort)
	client.URL = "http://localhost:" + p

	// Setup logging
	logPrefix := fmt.Sprintf("[Serve:%s] \n", client.Torrent.Name())
	logger := log.New(logrus.StandardLogger().Writer(), logPrefix, log.LstdFlags)

	// Set up HTTP server
	server := &http.Server{
		Addr:    ":" + p,
		Handler: http.HandlerFunc(client.streamHandler),
	}

	// Start serving in a separate goroutine
	go func() {
		// logger.Printf("Serving on http://localhost:%s\n", p)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Printf("Error serving: %v\n", err)
		}
	}()

	// Add a brief delay to ensure server setup before returning
	time.Sleep(100 * time.Millisecond)
}

// Add a field to store the previous bytes completed
var previousBytesCompleted int64

func (client *Client) PrintProgress() {
	t := client.Torrent
	if t.Info() == nil {
		return
	}
	total := t.Length()
	currentProgress := t.BytesCompleted()
	complete := humanize.Bytes(uint64(currentProgress))
	size := humanize.Bytes(uint64(total))
	percentage := float64(currentProgress) / float64(total) * 100

	// Calculate download speed
	currentTime := time.Now()
	elapsedTime := currentTime.Sub(client.lastPrintTime)
	downloadedBytes := currentProgress - previousBytesCompleted
	downloadSpeed := float64(downloadedBytes) / elapsedTime.Seconds()
	downloadSpeedFormatted := humanize.Bytes(uint64(downloadSpeed))

	output := bufio.NewWriter(os.Stdout)

	// Choose colors for different parts of the output
	completeColor := color.New(color.FgGreen).SprintFunc()
	sizeColor := color.New(color.FgBlue).SprintFunc()
	percentageColor := color.New(color.FgYellow).SprintFunc()
	speedColor := color.New(color.FgCyan).SprintFunc()

	// used \033[K at eol because previous line may extend over the current line
	_, _ = fmt.Fprintf(output, "Progress: %s / %s  %s%%  Download Speed: %s/s\033[K",
		// 			os.Stdout.Sync()
		completeColor(complete),
		sizeColor(size),
		percentageColor(fmt.Sprintf("%.2f", percentage)),
		speedColor(downloadSpeedFormatted))
	_ = output.Flush()

	// Update previousBytesCompleted and lastPrintTime for the next calculation
	previousBytesCompleted = currentProgress
	client.lastPrintTime = currentTime
}

// Close cleans up the connections of the client.
func (client *Client) Close() {
	client.Torrent.Drop()
	client.Client.Close()
}

// SeekableContent describes an io.ReadSeeker that can be closed as well.
type SeekableContent interface {
	io.ReadSeeker
	io.Closer
}

// FileEntry helps reading a torrent file.
type FileEntry struct {
	*torrent.File
	torrent.Reader
}

// Seek seeks to the correct file position, paying attention to the offset.
func (f *FileEntry) Seek(offset int64, whence int) (int64, error) {
	return f.Reader.Seek(offset+f.File.Offset(), whence)
}

// NewFileReader sets up a torrent file for streaming reading.
func NewFileReader(f *torrent.File) (SeekableContent, error) {
	t := f.Torrent()
	reader := t.NewReader()

	// We read ahead 1% of the file continuously.
	reader.SetReadahead(f.Length() / 100)
	reader.SetResponsive()
	_, err := reader.Seek(f.Offset(), io.SeekStart)

	return &FileEntry{File: f, Reader: reader}, err
}

func init() {
	// var err error

	loadedConfig, err := config.LoadConfig(configFile)
	if err != nil {
		// handle error
		fmt.Printf("Error initializing config (%v): %v\n", configFile, err)
	}
	configurations = loadedConfig

	// if _, err := os.Stat(configFile); os.IsNotExist(err) {
	// 	err = config.InitConfig(configFile)
	// 	if err != nil {
	// 		fmt.Printf("Error initializing config (%v): %v\n", configFile, err)
	// 		os.Exit(1)
	// 	}
	// }

	// configurations, err = config.LoadConfig(configFile)
	// if err != nil {
	// 	fmt.Println("Error loading config:", err)
	// 	os.Exit(1)
	// }

	// if configurations.Debug {
	// 	logrus.SetLevel(logrus.DebugLevel)
	// } else {
	// 	logrus.SetLevel(logrus.ErrorLevel)
	// }

}
