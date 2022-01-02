package kubernetes

import (
	"fmt"

	"github.com/mcorbin/cabourotte/healthcheck"
	"go.uber.org/zap"

	"gopkg.in/yaml.v2"
)

func addCheck(healthcheckComponent *healthcheck.Component, logger *zap.Logger, newChecks map[string]bool, healthcheckType string, stringConfig string, target string, source string, labels map[string]string, disableCommandsChecks bool) error {
	if healthcheckType == "http" {
		var config healthcheck.HTTPHealthcheckConfiguration
		err := yaml.Unmarshal([]byte(stringConfig), &config)
		if err != nil {
			return err
		}
		if config.Target == "" {
			config.Target = target
		}
		config.Base.Source = source
		if config.Base.Labels == nil {
			config.Base.Labels = make(map[string]string)
		}
		healthcheck.MergeLabels(&config.Base, labels)
		err = config.Validate()
		if err != nil {
			return err
		}
		healthcheck := healthcheck.NewHTTPHealthcheck(logger, &config)
		err = healthcheckComponent.AddCheck(healthcheck)
		if err != nil {
			return err
		}
		newChecks[config.Base.Name] = true
	} else if healthcheckType == "tcp" {
		var config healthcheck.TCPHealthcheckConfiguration
		err := yaml.Unmarshal([]byte(stringConfig), &config)
		if err != nil {
			return err
		}
		if config.Target == "" {
			config.Target = target
		}
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		healthcheck.MergeLabels(&config.Base, labels)
		config.Base.Source = source
		err = config.Validate()
		if err != nil {
			return err
		}
		healthcheck := healthcheck.NewTCPHealthcheck(logger, &config)
		err = healthcheckComponent.AddCheck(healthcheck)
		if err != nil {
			return err
		}
		newChecks[config.Base.Name] = true
	} else if healthcheckType == "tls" {
		var config healthcheck.TLSHealthcheckConfiguration
		err := yaml.Unmarshal([]byte(stringConfig), &config)
		if err != nil {
			return err
		}
		if config.Target == "" {
			config.Target = target
		}
		config.Base.Source = source
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		healthcheck.MergeLabels(&config.Base, labels)
		err = config.Validate()
		if err != nil {
			return err
		}
		healthcheck := healthcheck.NewTLSHealthcheck(logger, &config)
		err = healthcheckComponent.AddCheck(healthcheck)
		if err != nil {
			return err
		}
		newChecks[config.Base.Name] = true
	} else if healthcheckType == "dns" {
		var config healthcheck.DNSHealthcheckConfiguration
		err := yaml.Unmarshal([]byte(stringConfig), &config)
		if err != nil {
			return err
		}
		if config.Domain == "" {
			config.Domain = target
		}
		config.Base.Source = source
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		healthcheck.MergeLabels(&config.Base, labels)
		err = config.Validate()
		if err != nil {
			return err
		}
		healthcheck := healthcheck.NewDNSHealthcheck(logger, &config)
		err = healthcheckComponent.AddCheck(healthcheck)
		if err != nil {
			return err
		}
		newChecks[config.Base.Name] = true
	} else if healthcheckType == "command" {
		if disableCommandsChecks {
			return fmt.Errorf("Command checks are not allowed")
		}
		var config healthcheck.CommandHealthcheckConfiguration
		err := yaml.Unmarshal([]byte(stringConfig), &config)
		if err != nil {
			return err
		}
		config.Base.Source = source
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		healthcheck.MergeLabels(&config.Base, labels)
		err = config.Validate()
		if err != nil {
			return err
		}
		healthcheck := healthcheck.NewCommandHealthcheck(logger, &config)
		err = healthcheckComponent.AddCheck(healthcheck)
		if err != nil {
			return err
		}
		newChecks[config.Base.Name] = true
	} else {
		return fmt.Errorf("Invalid healthcheck type '%s'", healthcheckType)
	}
	return nil
}
