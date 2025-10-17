package api

import (
	"github.com/zondax/fil-parser/actors/cache/impl/common"
)

func GetDataSource(config *Config, node RPCClientInterface) common.DataSource {
	cacheDataSource := common.DataSource{
		Node: node.FullNodeClient(),
		Config: common.DataSourceConfig{
			NetworkName: config.NetworkName,
		},
	}
	return cacheDataSource
}
