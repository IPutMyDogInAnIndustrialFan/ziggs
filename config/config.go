package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

func init() {
	if home, err = os.UserHomeDir(); err != nil {
		panic(err)
	}
	prefConfigLocation = home + "/.config/" + Title
	Snek = viper.New()
}

func writeConfig() {
	if runtime.GOOS == "windows" {
		newconfig := Title
		Snek.SetConfigName(newconfig)
		if err = Snek.MergeInConfig(); err != nil {
			if err = Snek.SafeWriteConfigAs(newconfig + ".toml"); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		}
		return
	}

	if _, err := os.Stat(prefConfigLocation); os.IsNotExist(err) {
		if err = os.MkdirAll(prefConfigLocation, 0o755); err != nil {
			println("error writing new config: " + err.Error())
			os.Exit(1)
		}
	}

	newconfig := prefConfigLocation + "/" + "config.toml"
	if err = Snek.SafeWriteConfigAs(newconfig); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	Filename = newconfig
}

// Init will initialize our toml configuration engine and define our default values.
func Init() {
	Snek.SetConfigType("toml")
	Snek.SetConfigName("config")

	argParse()

	if customconfig {
		associateExportedVariables()
		return
	}

	setConfigFileLocations()
	setDefaults()

	for _, loc := range configLocations {
		Snek.AddConfigPath(loc)
	}

	if err = Snek.MergeInConfig(); err != nil {
		writeConfig()
	}

	if len(Filename) < 1 {
		Filename = Snek.ConfigFileUsed()
	}

	associateExportedVariables()
}

func setDefaults() {
	var (
		configSections = []string{"logger", "lights", "http", "ssh", "bridges"}
		deflogdir      = home + "/.config/" + Title + "/logs/"
		defNoColor     = false
	)
	if runtime.GOOS == "windows" {
		deflogdir = "logs/"
		defNoColor = true
	}
	Opt := make(map[string]map[string]interface{})
	Opt["logger"] = map[string]interface{}{
		"debug":             true,
		"trace":             true,
		"directory":         deflogdir,
		"nocolor":           defNoColor,
		"use_date_filename": true,
	}
	Opt["bridges"] = map[string]interface{}{
		"hostname": "192.168.6.100",
		"username": "",
		"proxy":    "socks5://127.0.0.1:8060",
	}

	Opt["http"] = map[string]interface{}{
		"bind_addr": "127.0.0.1:9090",
	}

	Opt["ssh"] = map[string]interface{}{
		"bind_addr":    "127.0.0.1:2222",
		"host_key_dir": "~/.config/" + Title + "/.ssh",
	}

	for _, def := range configSections {
		Snek.SetDefault(def, Opt[def])
	}
	if GenConfig {
		if err = Snek.SafeWriteConfigAs("./config.toml"); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func setConfigFileLocations() {
	configLocations = append(configLocations, "./")

	if runtime.GOOS != "windows" {
		configLocations = append(configLocations,
			prefConfigLocation,
			"/etc/"+Title+"/",
			"../", "../../")
	}
}

func loadCustomConfig(path string) {
	if f, err = os.Open(path); err != nil {
		println("Error opening specified config file: " + path)
		panic("config file open fatal error: " + err.Error())
	}
	buf, err := io.ReadAll(f)
	err2 := Snek.ReadConfig(bytes.NewBuffer(buf))
	switch {
	case err != nil:
		fmt.Println("config file read fatal error: ", err.Error())
	case err2 != nil:
		fmt.Println("config file read fatal error: ", err2.Error())
	default:
		break
	}
	customconfig = true
}

func printUsage() {
	println("\n" + Title + " v" + Version + " Usage\n")
	println("-c <config.toml> - Specify config file")
	println("--nocolor - disable color and banner ")
	println("--banner - show banner + version and exit")
	println("--genconfig - write default config to 'default.toml' then exit")
	os.Exit(0)
}

// TODO: should probably just make a proper CLI with flags or something
func argParse() {
	for i, arg := range os.Args {
		switch arg {
		case "-h":
			printUsage()
		case "--genconfig":
			GenConfig = true
		case "--config", "-c":
			if len(os.Args) <= i-1 {
				panic("syntax error! expected file after -c")
			}
			loadCustomConfig(os.Args[i+1])
		default:
			continue
		}
	}
}

func processOpts() {
	// string options and their exported variables
	stringOpt := map[string]*string{
		"http.bind_addr":   &HTTPBind,
		"logger.directory": &logDir,
		"http.api_key":     &APIKey,
	}
	// bool options and their exported variables
	boolOpt := map[string]*bool{
		"logger.nocolor": &NoColor,
		"logger.debug":   &Debug,
		"logger.trace":   &Trace,
	}
	// int options and their exported variables
	intOpt := map[string]*int{
		"http.bind_port": &HTTPPort,
	}

	err := Snek.UnmarshalKey("bridges", &KnownBridges)
	if err != nil {
		println(err.Error())
	}

	for key, opt := range intOpt {
		*opt = Snek.GetInt(key)
	}
	for key, opt := range stringOpt {
		*opt = Snek.GetString(key)
	}
	for key, opt := range boolOpt {
		*opt = Snek.GetBool(key)
	}

	switch {
	case Trace:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		logger.Trace().Msg("trace verbosity enabled")
	case Debug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logger.Trace().Msg("debug verbosity enabled")
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

}

func associateExportedVariables() {
	processOpts()
}
