package pgxServerless

import (
	"reflect"
	"testing"
)

func Test_slsConnConfig_merge(t *testing.T) {
	type args struct {
		c SlsConnConfigParams
	}
	tests := []struct {
		name string
		args args
		want slsConnConfig
	}{
		{
			name: "Should correctly mergeAndValidate the config and override an int default value",
			args: args{c: SlsConnConfigParams{
				MaxConnections: Int(1500),
			}},
			want: slsConnConfig{
				MaxConnectionsFreqMs:     60000,
				ManualMaxConnections:     false,
				MaxConnections:           1500,
				MinConnectionIdleTimeSec: 0.5,
				MaxIdleConnectionsToKill: nil,
				ConnUtilization:          0.8,
				Debug:                    false,
				BackoffCapMs:             1000,
				BackoffBaseMs:            2,
				BackoffDelayMs:           1000,
				BackoffMaxRetries:        3,
			},
		},
		{
			name: "Should correctly mergeAndValidate the config and override a bool default value",
			args: args{c: SlsConnConfigParams{
				ManualMaxConnections: Bool(true),
			}},
			want: slsConnConfig{
				MaxConnectionsFreqMs:     60000,
				ManualMaxConnections:     true,
				MaxConnections:           100,
				MinConnectionIdleTimeSec: 0.5,
				MaxIdleConnectionsToKill: nil,
				ConnUtilization:          0.8,
				Debug:                    false,
				BackoffCapMs:             1000,
				BackoffBaseMs:            2,
				BackoffDelayMs:           1000,
				BackoffMaxRetries:        3,
			},
		},
		{
			name: "Should correctly mergeAndValidate the config and override a pointer default value",
			args: args{c: SlsConnConfigParams{
				 MaxIdleConnectionsToKill: Int(50),
			}},
			want: slsConnConfig{
				MaxConnectionsFreqMs:     60000,
				ManualMaxConnections:     false,
				MaxConnections:           100,
				MinConnectionIdleTimeSec: 0.5,
				MaxIdleConnectionsToKill: Int(50),
				ConnUtilization:          0.8,
				Debug:                    false,
				BackoffCapMs:             1000,
				BackoffBaseMs:            2,
				BackoffDelayMs:           1000,
				BackoffMaxRetries:        3,
			},
		},
		{
			name: "Should correctly mergeAndValidate the config and override a float default value",
			args: args{c: SlsConnConfigParams{
				MinConnectionIdleTimeSec: Float32(2.5),
			}},
			want: slsConnConfig{
				MaxConnectionsFreqMs:     60000,
				ManualMaxConnections:     false,
				MaxConnections:           100,
				MinConnectionIdleTimeSec: 2.5,
				MaxIdleConnectionsToKill: nil,
				ConnUtilization:          0.8,
				Debug:                    false,
				BackoffCapMs:             1000,
				BackoffBaseMs:            2,
				BackoffDelayMs:           1000,
				BackoffMaxRetries:        3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newDefaultConfig()
			_ = s.mergeAndValidate(tt.args.c)

			if !reflect.DeepEqual(s, tt.want) {
				t.Errorf("merge config got = %v, want %v", s, tt.want)
			}
		})
	}
}

func Test_slsConnConfig_validation(t *testing.T) {
	type args struct {
		c SlsConnConfigParams
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Should reject an int value, is less than 0",
			args: args{c: SlsConnConfigParams{
				MaxConnections: Int(-1),
			}},
			want: "MaxConnections should not be negative",
		},
		{
			name: "Should reject an float value, is less than 0",
			args: args{c: SlsConnConfigParams{
				MinConnectionIdleTimeSec: Float32(-1.5),
			}},
			want: "MinConnectionIdleTimeSec should not be negative",
		},
		{
			name: "Should reject ConnUtilization, value is not within range",
			args: args{c: SlsConnConfigParams{
				ConnUtilization: Float32(1.5),
			}},
			want: "connectionsUtilization should not be bigger than 1",
		},
		{
			name: "Should reject ConnUtilization, value is not within range",
			args: args{c: SlsConnConfigParams{
				ConnUtilization: Float32(-0.1),
			}},
			want: "connectionsUtilization should not be negative",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newDefaultConfig()
			err := s.mergeAndValidate(tt.args.c)

			if err.Error() != tt.want {
				t.Errorf("validate config got = %v, want %v", err, tt.want)
			}
		})
	}
}

