// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package config

import (
	"log"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/configs"
	"github.com/spf13/viper"
)

// SIPConfig holds the SIP server configuration
type SIPConfig struct {
	Server            string `mapstructure:"server"`
	ExternalIP        string `mapstructure:"external_ip"` // Public/reachable IP for SDP and SIP Contact headers (defaults to Server if empty)
	Port              int    `mapstructure:"port"`
	Transport         string `mapstructure:"transport"`
	RTPPortRangeStart int    `mapstructure:"rtp_port_range_start"`
	RTPPortRangeEnd   int    `mapstructure:"rtp_port_range_end"`
}

// WebRTCConfig holds WebRTC ICE configuration for production cloud deployments.
// On EC2, Pion only sees the private IP; set PublicIP to the instance's public/elastic IP
// so Pion advertises it in host candidates instead of the unreachable private IP.
type WebRTCConfig struct {
	ExternalIP        string `mapstructure:"external_ip"`
	UDPPortRangeStart int    `mapstructure:"udp_port_range_start"`
	UDPPortRangeEnd   int    `mapstructure:"udp_port_range_end"`
}

type AudioSocketConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type AssistantConfig struct {
	config.AppConfig    `mapstructure:",squash"`
	PostgresConfig      configs.PostgresConfig    `mapstructure:"postgres" validate:"required"`
	RedisConfig         configs.RedisConfig       `mapstructure:"redis" validate:"required"`
	OpenSearchConfig    *configs.OpenSearchConfig `mapstructure:"opensearch"`
	TelemetryConfig     *configs.TelemetryConfig  `mapstructure:"telemetry"`
	WeaviateConfig      configs.WeaviateConfig    `mapstructure:"weaviate"`
	AssetStoreConfig    configs.AssetStoreConfig  `mapstructure:"asset_store" validate:"required"`
	PublicAssistantHost string                    `mapstructure:"public_assistant_host" validate:"required"`
	SIPConfig           *SIPConfig                `mapstructure:"sip"`
	AudioSocketConfig   *AudioSocketConfig        `mapstructure:"audiosocket"`
	WebRTCConfig        *WebRTCConfig             `mapstructure:"webrtc"`
}

// reading config and intializing configs for application
func InitConfig() (*viper.Viper, error) {
	vConfig := viper.NewWithOptions(viper.KeyDelimiter("__"))

	vConfig.AddConfigPath("./env/")
	vConfig.SetConfigName(".assistant.env")
	path := os.Getenv("ENV_PATH")
	if path != "" {
		log.Printf("env path %v", path)
		vConfig.SetConfigFile(path)
	}
	vConfig.SetConfigType("env")
	vConfig.AutomaticEnv()
	if err := vConfig.ReadInConfig(); err != nil {
		log.Printf("Error while reading the config")
	}

	if err := vConfig.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		log.Printf("Reading from env varaibles.")
	}

	return vConfig, nil
}

// Getting application config from viper
func GetApplicationConfig(v *viper.Viper) (*AssistantConfig, error) {
	var config AssistantConfig
	err := v.Unmarshal(&config)
	if err != nil {
		log.Printf("%+v\n", err)
		return nil, err
	}
	// If OpenSearch config is missing any required connection field, treat as not configured
	if config.OpenSearchConfig != nil &&
		(config.OpenSearchConfig.Host == "" || config.OpenSearchConfig.Schema == "") {
		config.OpenSearchConfig = nil
	}
	// valdating the app config
	validate := validator.New()
	err = validate.Struct(&config)
	if err != nil {
		log.Printf("%+v\n", err)
		return nil, err
	}
	return &config, nil
}
