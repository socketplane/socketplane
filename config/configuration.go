package config

import "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/BurntSushi/toml"

const (
	CFG_DEBUG     = "debug"
	CFG_BOOTSTRAP = "bootstrap"
	CFG_FILE      = "config-file"
	CFG_IFACE     = "iface"
)

type config struct {
	Daemon DaemonCfg
	// Add more Configs such as ClusterCfg, OvsCfg, etc.
}

type DaemonCfg struct {
	Bootstrap bool
	Debug     bool
}

var spConfig config
var Daemon DaemonCfg

func Parse(tomlCfgFile string) error {
	if _, err := toml.DecodeFile(tomlCfgFile, &spConfig); err != nil {
		return err
	}
	Daemon = spConfig.Daemon
	return nil
}
