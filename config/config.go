package config

import "path"

const (
	DEFAULT_REGENT_DATADIR  = "/Users/prestonevans/sovereign/regent"
	DEFAULT_ERIGON_DATADIR  = "/Users/prestonevans/sovereign/"
	JWT_SECRET_FILENAME     = "jwt.hex"
	DEFAULT_ENGINE_RPC_PORT = "8551"
)

var ErigonDatadir = DEFAULT_ERIGON_DATADIR
var EngineRpcPort string = DEFAULT_ENGINE_RPC_PORT

type Config struct {
	ErigonDatadir string
	EngineRpcPort string
	JwtSecretPath string
	RegentDatadir string
}

func New() *Config {
	return &Config{
		ErigonDatadir: DEFAULT_ERIGON_DATADIR,
		JwtSecretPath: path.Join(DEFAULT_ERIGON_DATADIR, JWT_SECRET_FILENAME),
		EngineRpcPort: DEFAULT_ENGINE_RPC_PORT,
		RegentDatadir: DEFAULT_REGENT_DATADIR,
	}
}
