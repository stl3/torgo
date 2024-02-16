package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type TorrodleConfig struct {
	DataDir      string `json:"DataDir"`
	ResultsLimit int    `json:"ResultsLimit"`
	TorrentPort  int    `json:"TorrentPort"`
	HostPort     int    `json:"HostPort"`
	Proxy        string `json:"Proxy"`
	Eztv_cookie  string `json:"eztv_cookie"`
	Ext_cookie   string `json:"ext_cookie"`
	Mpv_params   string `json:"mpv_params"`
	ECPT         int    `json:"EstablishedConnsPerTorrent"`
	HOCPT        int    `json:"HalfOpenConnsPerTorrent"`
	THOC         int    `json:"TotalHalfOpenConns"`
	Debug        bool   `json:"Debug"`
}

// This function is for debug purposes
// It shows config parameters used in ~/.torgo.json
func (t TorrodleConfig) String() string {
	return fmt.Sprintf(
		`TorrentDir: %v | ResultsLimit: %d | TorrentPort: %d | HostPort: %d | Debug: %v`,
		t.DataDir, t.ResultsLimit, t.TorrentPort, t.HostPort, t.Debug,
	)
}

// Just for future reference
// OS PATHS
// Config Paths
// Linux
// ${XDG_CONFIG_HOME:-${HOME}/.config}/example/config
// MacOS
// ${HOME}/Library/Application Support/example/config
// Termux
// $HOME/.config/example/config
// Windows
// %APPDATA%\example\config
// %APPDATA%\.config\example\config

// Temp File Paths
// Linux/macOS/Termux
// $TMPDIR/torgo
// Windows - %TEMP%/torgo

func InitConfig(path string) error {
	config := TorrodleConfig{
		DataDir:      getTempDir(),
		ResultsLimit: 100,
		// TorrentPort:  10800,
		TorrentPort: 36663,
		HostPort:    8789,
		ECPT:        45,
		HOCPT:       25,
		THOC:        50,
	}
	data, _ := json.MarshalIndent(config, "", "\t")
	err := os.WriteFile(path, data, 0644)
	return err
}

func LoadConfig(path string) (TorrodleConfig, error) {
	var config TorrodleConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(data, &config)
	return config, err
}

func getTempDir() string {
	tempDir := os.TempDir()
	return filepath.Join(tempDir, "torgo")
}
