package discovery

import (
	"fmt"

	"github.com/mcorbin/cabourotte/healthcheck"
	"go.uber.org/zap"

	"gopkg.in/yaml.v2"
)

func addCheck(healthcheckComponent *healthcheck.Component, logger *zap.Logger, newChecks map[string]bool, healthcheckType string, stringConfig string, target string, source string, labels map[string]string) error {
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
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		for k, v := range labels {
			config.Labels[k] = v
		}
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
		config.Base.Source = source
		for k, v := range labels {
			config.Labels[k] = v
		}
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
		for k, v := range labels {
			config.Labels[k] = v
		}
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
		for k, v := range labels {
			config.Labels[k] = v
		}
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
		var config healthcheck.CommandHealthcheckConfiguration
		err := yaml.Unmarshal([]byte(stringConfig), &config)
		if err != nil {
			return err
		}
		config.Base.Source = source
		if config.Labels == nil {
			config.Labels = make(map[string]string)
		}
		for k, v := range labels {
			config.Labels[k] = v
		}
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
