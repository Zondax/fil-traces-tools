package api

import (
	"fmt"

	"github.com/Zondax/zindexer/components/connections/data_store"
	"github.com/filecoin-project/lotus/api"
	lotusChainTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/fil-parser/types"
)

type RawData struct {
	Tipset         *types.ExtendedTipSet
	Trace          *api.ComputeStateOutput
	EthLogs        []types.EthLog
	NativeLogs     []*lotusChainTypes.ActorEvent
	TipsetMetadata types.BlockMetadata
}

func GetDataStoreClient(config *Config) (*data_store.DataStoreClient, error) {
	dsConf := data_store.DataStoreConfig{
		Url:      config.S3URL,
		UseHttps: config.S3SSL,
		User:     config.S3AccessKey,
		Password: config.S3SecretKey,
		Service:  "s5",
		DataPath: config.S3RawDataPath,
	}

	client, err := data_store.NewDataStoreClient(dsConf)
	if err != nil {
		return nil, fmt.Errorf("could not create data store: %v", err)
	}
	return &client, nil
}

func GetTraceFromDataStore(height int64, dsClient *data_store.DataStoreClient, config *Config) ([]byte, error) {
	storePath := fmt.Sprintf("%s/%s", config.S3Bucket, config.S3RawDataPath)
	name := fmt.Sprintf("traces_%012d.json.s2", height)

	data, err := dsClient.Client.GetFile(name, storePath)
	if err != nil {
		return nil, err
	}

	decompressed, err := decompress(data)
	if err != nil {
		return nil, err
	}

	return decompressed, nil
}
