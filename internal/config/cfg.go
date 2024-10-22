// Copyright (C) 2022 AlgoNode Org.
//
// spambot is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// spambot is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with spambot.  If not, see <https://www.gnu.org/licenses/>.

package config

import (
	"flag"
	"fmt"

	"github.com/algonode/spambot/internal/utils"
)

var cfgFile = flag.String("f", "config.jsonc", "config file")

type NodeConfig struct {
	Address string `json:"address"`
	Token   string `json:"token"`
}

type SPAMConfig struct {
	Threads   int    `json:"threads"`
	Rate      int    `json:"rate"`
	CleanFile string `json:"cleanfile"`
}

type KV map[string]string
type KB map[string]bool

type BotConfig struct {
	Algod  *NodeConfig `json:"algod-api"`
	SPAM   *SPAMConfig `json:"spam"`
	PKeys  KV          `json:"pkeys"`
	WSnglt KB          `json:"singletons"`
}

var defaultConfig = BotConfig{}

// loadConfig loads the configuration from the specified file, merging into the default configuration.
func LoadConfig() (cfg BotConfig, err error) {
	flag.Parse()
	cfg = defaultConfig
	err = utils.LoadJSONCFromFile(*cfgFile, &cfg)

	if cfg.Algod == nil {
		return cfg, fmt.Errorf("[CFG] Missing algod-api config")
	}

	if cfg.PKeys == nil {
		return cfg, fmt.Errorf("[CFG] Missing pkeys config")
	}

	if cfg.WSnglt == nil || len(cfg.WSnglt) == 0 {
		return cfg, fmt.Errorf("[CFG] Singleton config missing")
	}

	return cfg, err
}
