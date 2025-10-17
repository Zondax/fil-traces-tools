package api

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {

	// NetworkName is the network name
	NetworkName string `mapstructure:"network_name"`
	// NodeURL is the url of the blockchain node's rpc server
	NodeURL string `mapstructure:"node_url"`
	// NetworkSymbol is the token symbol for this network
	NetworkSymbol string `mapstructure:"network_name"`
	NodeToken     string `mapstructure:"node_token"`

	S3Bucket      string `mapstructure:"s3_bucket"`
	S3URL         string `mapstructure:"s3_url"`
	S3AccessKey   string `mapstructure:"s3_access_key"`
	S3SecretKey   string `mapstructure:"s3_secret_key"`
	S3SSL         bool   `mapstructure:"s3_ssl"`
	S3RawDataPath string `mapstructure:"s3_raw_data_path"`
}

func (c *Config) SetDefaults() {}
func (c *Config) Validate() error {
	return nil
}

func GetGlobalConfigs() Config {
	viper := viper.New()
	viper.SetConfigName("config") // config file name without extension
	viper.AddConfigPath(".")      // search path
	viper.AutomaticEnv()          // read value ENV variables

	err := viper.ReadInConfig()
	if err != nil {
		zap.L().Debug("Config not found. Using env variables")
	}

	return Config{
		// Network
		NodeURL:       viper.GetString("node_url"),
		NetworkName:   viper.GetString("network_name"),
		NetworkSymbol: viper.GetString("network_symbol"),
		NodeToken:     viper.GetString("node_token"),

		// Raw data download S3
		S3URL:         viper.GetString("s3_url"),
		S3SSL:         viper.GetBool("s3_ssl"),
		S3AccessKey:   viper.GetString("s3_access_key"),
		S3SecretKey:   viper.GetString("s3_secret_key"),
		S3Bucket:      viper.GetString("s3_bucket"),
		S3RawDataPath: viper.GetString("s3_raw_data_path"),
	}
}
