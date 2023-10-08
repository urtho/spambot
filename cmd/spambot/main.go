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

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/algonode/spambot/internal/algodapi"
	"github.com/algonode/spambot/internal/config"
	"github.com/algonode/spambot/internal/worker"

	log "github.com/sirupsen/logrus"
)

func checkBalanceAndToggleSpammer(ctx context.Context, apis *worker.WorkerAPIs, cfg *config.BotConfig, spamWorkers []*worker.SPAMWorker) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get account information
			accountInfo, err := apis.Aapi.Client.AccountInformation(cfg.Monitored.Address).Do(ctx)
			if err != nil {
				log.Error(err)
				continue
			}
			// Start or stop spamming based on balance
			for _, spamWorker := range spamWorkers {
				if accountInfo.Amount > cfg.Monitored.SpamThreshold {
					spamWorker.StartSpamming()
				} else if accountInfo.Amount <= cfg.Monitored.StopSpamThreshold {
					spamWorker.StopSpamming()
				}
			}
		}
	}
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)
}

func main() {

	slog := log.StandardLogger()

	//load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Error()
		return
	}

	//make us a nice cancellable context
	//set Ctrl-C as the cancell trigger
	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	{
		cancelCh := make(chan os.Signal, 1)
		signal.Notify(cancelCh, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			<-cancelCh
			log.Error("stopping streamer")
			cf()
		}()
	}

	aapi, err := algodapi.Make(ctx, cfg.Algod, slog)
	if err != nil {
		log.Panic(err)
	}

	apis := &worker.WorkerAPIs{
		Aapi: aapi,
	}

	workers := []worker.Worker{
		worker.SPAMWorkerNew(ctx, apis, slog, &cfg),
	}

	var spamWorkers []*worker.SPAMWorker

	for _, w := range workers {
		if err := w.Config(ctx); err != nil {
			log.Panic(err)
		}
		if spamWorker, ok := w.(*worker.SPAMWorker); ok {
			spamWorkers = append(spamWorkers, spamWorker)
		}
	}

	for _, w := range workers {
		if err := w.Spawn(ctx); err != nil {
			log.Panic(err)
		}
	}

	go checkBalanceAndToggleSpammer(ctx, apis, &cfg, spamWorkers)

	<-ctx.Done()
	time.Sleep(time.Second)

	log.Error("Bye!")

}
