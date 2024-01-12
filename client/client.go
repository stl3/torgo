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
	"net"
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
	// signal to indicate download completion
	downloadComplete chan struct{}
}

var u, _ = user.Current()
var home = u.HomeDir
var configFile = filepath.Join(home, ".torgo.json")
var configurations config.TorrodleConfig

// Function to find an available port starting from the given port
func findAvailablePort(startPort int) int {
	for port := startPort; port <= 65535; port++ {
		// Check if the port is available
		if isPortAvailable(port) {
			logrus.Infof("Using port: %s", port)
			return port
		}
	}
	// Return an error value if no available port is found
	// Chances are, there will be an available port but we
	// save that problem for another rainy day
	return -1
}

// Function to check if a port is available
func isPortAvailable(port int) bool {
	// Attempt to bind to the specified port
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		// Port is in use, return false
		return false
	}
	// Close the listener if the port is available
	_ = listener.Close()
	return true
}

func NewClient(dataDir string, torrentPort int, hostPort int) (*Client, error) {
	var client Client
	// Attempt to find an available port starting from the specified port
	torrentPort = findAvailablePort(torrentPort)

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
		return nil, err
	}
	client.Client = c
	client.HostPort = hostPort

	// Create channel for signaling download completion
	client.downloadComplete = make(chan struct{})

	// return a pointer to the client instance
	return &client, nil
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

			if t.BytesCompleted() == t.Length() {
				// Signal download completion only if the channel is still open
				if client.downloadComplete != nil {
					close(client.downloadComplete)
					client.downloadComplete = nil // set to nil to avoid closing it again
				}
				// Print the final progress information before exiting
				client.PrintProgress()
				return // exit the loop if download is complete
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

// Stop is a new method to stop the client and associated resources
func (client *Client) Stop() {
	// Perform cleanup operations here
	// Close the client, release resources, etc.
	client.Torrent.Drop()
	client.Client.Close()

	// Add any other cleanup steps as needed
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
		<-client.downloadComplete

		// Stop the client when the download is complete
		client.Stop()
	}()

	// Add a brief delay to ensure server setup before returning
	time.Sleep(100 * time.Millisecond)
}

// Add a field to store the previous bytes completed
var previousBytesCompleted int64

func (client *Client) PrintProgress() {
	t := client.Torrent
	// Do not run PrintProgress anymore when download completes
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

}
