package slsPgx

import "errors"

type slsConnConfig struct {
	MaxConnectionsFreqMs     float32
	ManualMaxConnections     bool
	MaxConnections           int
	MinConnectionIdleTimeSec float32
	MaxIdleConnectionsToKill *int // this can be nil
	ConnUtilization          float32
	Debug                    bool
	BackoffCapMs             float32
	BackoffBaseMs            float32
	BackoffDelayMs           float32
	BackoffMaxRetries        int
}

type SlsConnConfigParams struct {
	MaxConnectionsFreqMs     *float32
	ManualMaxConnections     *bool
	MaxConnections           *int
	MinConnectionIdleTimeSec *float32
	MaxIdleConnectionsToKill *int
	ConnUtilization          *float32
	Debug                    *bool
	BackoffCapMs             *float32
	BackoffBaseMs            *float32
	BackoffDelayMs           *float32
	BackoffMaxRetries        *int
}

func newDefaultConfig() slsConnConfig {
	return slsConnConfig{
		MaxConnectionsFreqMs:     60000,
		ManualMaxConnections:     false,
		MaxConnections:           100,
		MinConnectionIdleTimeSec: 0.5,
		MaxIdleConnectionsToKill: nil,
		ConnUtilization:          0.8,
		Debug:                    false,
		BackoffCapMs:             1000,
		BackoffBaseMs:            2,
		BackoffDelayMs:           1000,
		BackoffMaxRetries:        3,
	}
}

func (s *slsConnConfig) mergeAndValidate(c SlsConnConfigParams) error {
	if c.Debug != nil {
		s.Debug = *c.Debug
	}
	if c.MaxConnections != nil {
		if err := s.validateInt("MaxConnections", *c.MaxConnections); err != nil {
			return err
		}
		s.MaxConnections = *c.MaxConnections
	}
	if c.BackoffMaxRetries != nil {
		if err := s.validateInt("BackoffMaxRetries", *c.BackoffMaxRetries); err != nil {
			return err
		}
		s.BackoffMaxRetries = *c.BackoffMaxRetries
	}
	if c.ConnUtilization != nil {
		if err := s.validateConnectionsUtilization(*c.ConnUtilization); err != nil {
			return err
		}
		s.ConnUtilization = *c.ConnUtilization
	}
	if c.MaxConnectionsFreqMs != nil {
		if err := s.validateFloat("MaxConnectionsFreqMs", *c.MaxConnectionsFreqMs); err != nil {
			return err
		}
		s.MaxConnectionsFreqMs = *c.MaxConnectionsFreqMs
	}
	if c.MaxIdleConnectionsToKill != nil {
		if err := s.validateInt("MaxIdleConnectionsToKill", *c.MaxIdleConnectionsToKill); err != nil {
			return err
		}
		s.MaxIdleConnectionsToKill = c.MaxIdleConnectionsToKill
	}
	if c.MinConnectionIdleTimeSec != nil {
		if err := s.validateFloat("MinConnectionIdleTimeSec", *c.MinConnectionIdleTimeSec); err != nil {
			return err
		}
		s.MinConnectionIdleTimeSec = *c.MinConnectionIdleTimeSec
	}
	if c.BackoffBaseMs != nil {
		if err := s.validateFloat("backoffBaseMs", *c.BackoffBaseMs); err != nil {
			return err
		}
		s.BackoffBaseMs = *c.BackoffBaseMs
	}
	if c.BackoffCapMs != nil {
		if err := s.validateFloat("backoffCapMs", *c.BackoffCapMs); err != nil {
			return err
		}
		s.BackoffCapMs = *c.BackoffCapMs
	}
	if c.BackoffDelayMs != nil {
		if err := s.validateFloat("backoffDelayMs", *c.BackoffDelayMs); err != nil {
			return err
		}
		s.BackoffDelayMs = *c.BackoffDelayMs
	}
	if c.ManualMaxConnections != nil {
		s.ManualMaxConnections = *c.ManualMaxConnections
	}

	return nil
}

func (s slsConnConfig) validateInt(name string, value int) error {
	if value < 0 {
		return errors.New(name+" should not be negative")
	}

	return nil
}

func (s slsConnConfig) validateFloat(name string, value float32) error {
	if value < 0 {
		return errors.New(name+" should not be negative")
	}

	return nil
}

func (s slsConnConfig) validateConnectionsUtilization(value float32) error {
	if value < 0 {
		return errors.New("connectionsUtilization should not be negative")
	}

	if value > 1 {
		return errors.New("connectionsUtilization should not be bigger than 1")
	}

	return nil
}
