package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	_ "go.uber.org/automaxprocs"

	"github.com/teslamotors/fleet-telemetry/config"
	"github.com/teslamotors/fleet-telemetry/server/monitoring"
	"github.com/teslamotors/fleet-telemetry/server/streaming"
)

func main() {
	var err error

	config, logger, err := config.LoadApplicationConfiguration()
	if err != nil {
		// logger is not available yet
		panic(fmt.Sprintf("error=load_service_config value=\"%s\"", err.Error()))
	}

	if config.Monitoring != nil && config.Monitoring.ProfilingPath != "" {
		if config.Monitoring.ProfilerFile, err = os.Create(config.Monitoring.ProfilingPath); err != nil {
			logger.Errorf("profiling_file_error %v", err)
			config.Monitoring.ProfilingPath = ""
		}

		defer func() {
			config.MetricCollector.Shutdown()
			_ = config.Monitoring.ProfilerFile.Close()
		}()
	}

	panic(startServer(config, logger))
}

func startServer(config *config.Config, logger *logrus.Logger) (err error) {
	logger.Infoln("starting")
	registry := streaming.NewSocketRegistry()

	if config.StatusPort > 0 {
		monitoring.StartStatusServer(config, logger)
	}
	if config.Monitoring != nil {
		monitoring.StartServerMetrics(config, logger, registry)
	}

	producerRules, err := config.ConfigureProducers(logger)
	if err != nil {
		return err
	}

	server, _, err := streaming.InitServer(config, producerRules, logger, registry)
	if err != nil {
		return err
	}

	if server.TLSConfig, err = config.ExtractServiceTLSConfig(); err != nil {
		return err
	}
	return server.ListenAndServeTLS(config.TLS.ServerCert, config.TLS.ServerKey)
}
