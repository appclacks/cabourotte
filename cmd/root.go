package cmd

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cabourotte/daemon"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// Main the main entrypoint
func Main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "daemon",
				Usage: "starts the Cabourotte daemon",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "config",
						Usage:    "Path to the configuration file",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					file, err := ioutil.ReadFile(c.String("config"))
					if err != nil {
						return errors.Wrapf(err, "fail to read the configuration file")
					}
					var config daemon.Configuration
					if err := yaml.Unmarshal(file, &config); err != nil {
						return errors.Wrapf(err, "Fail to read the yaml config file")
					}
					logger, err := zap.NewProduction()
					if err != nil {
						return errors.Wrapf(err, "Fail to start the logger")
					}
					// nolint
					defer logger.Sync()
					daemonComponent, err := daemon.New(logger, &config)
					if err != nil {
						return errors.Wrapf(err, "Fail to creae the daemon")
					}
					signals := make(chan os.Signal, 1)
					errChan := make(chan error)

					signal.Notify(
						signals,
						syscall.SIGINT,
						syscall.SIGTERM,
						syscall.SIGHUP)
					go func() {
						for sig := range signals {
							switch sig {
							case syscall.SIGINT, syscall.SIGTERM:
								logger.Info(fmt.Sprintf("Received signal %s, shutdown", sig))
								signal.Stop(signals)
								err := daemonComponent.Stop()
								if err != nil {
									logger.Error(fmt.Sprintf("Fail to stop: %s", err.Error()))
									errChan <- err
								}
								errChan <- nil
							case syscall.SIGHUP:
								logger.Info(fmt.Sprintf("Received signal %s, reload", sig))
								newFile, err := ioutil.ReadFile(c.String("config"))
								if err != nil {
									errChan <- err
								} else {
									var newConfig daemon.Configuration
									if err := yaml.Unmarshal(newFile, &newConfig); err != nil {
										errChan <- err
									} else {
										err := daemonComponent.Reload(&newConfig)
										if err != nil {
											logger.Error(fmt.Sprintf("Fail to reload: %s", err.Error()))
											errChan <- err
										}
									}
								}
							}

						}
					}()
					exitErr := <-errChan
					return exitErr
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
