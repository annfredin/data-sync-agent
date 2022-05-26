package config

import (
	"fmt"
	"time"

	cm "data-sync-agent/config/conman"
	"data-sync-agent/helper"
	"data-sync-agent/utils/logger"

	"go.uber.org/zap"
)

var configProvider cm.Provider

func init() {
	connectAsRedisClusterMode := helper.GetEnv(helper.ConnectAsRedisClusterMode)

	configServerEndPoint := helper.GetEnv(helper.ConfigServerEndPoint)

	configServerUserName := helper.GetEnv(helper.ConfigServerUserName)

	configServerPassword := helper.GetEnv(helper.ConfigServerPassword)

	
	InitializeConfigProvider(configServerEndPoint, configServerUserName, configServerPassword, connectAsRedisClusterMode)
}

//InitializeConfigProvider initializing the config provider....
func InitializeConfigProvider(configServerEndPoint, configServerUserName, configServerPassword, connectAsCluster string) cm.Provider {

	var confProvider cm.Provider
	var err error

	if nOK := helper.HasEmpty(configServerEndPoint,configServerUserName,configServerPassword,connectAsCluster); nOK {

		logger.Log().Fatal("Redis Connection Secret Error")
		return nil
	}

	if connectAsCluster == "1" {
		confProvider, err = cm.NewRedisProvider(cm.ConnRequest{Endpoint: configServerEndPoint, UserName: configServerUserName, Password: configServerPassword})
	} else {
		confProvider, err = cm.NewRedisProviderClient(cm.ConnRequest{Endpoint: configServerEndPoint})
	}

	if err != nil {
		logger.Log().With(zap.Error(err)).Error(fmt.Sprintf("Config Server Connection Error EndPoint %s", configServerEndPoint))

		return nil
	}

	logger.Log().Info(fmt.Sprintf("Config Server started with EndPoint %s", configServerEndPoint))

	//assigning to global variable(must)
	configProvider = confProvider
	return confProvider
}

//Get ...
func Get(key string) (string, error) {
	return configProvider.Get(key)
}

//MGet ...
func MGet(keys []string) ([]interface{}, error) {
	return configProvider.MGet(keys)
}

//Set ...
func Set(key string, value interface{}) error {
	return configProvider.Set(key, value)
}

//MSet ...
func MSet(kvPairs map[string]interface{}) error {
	return configProvider.MSet(kvPairs)
}

//SetWithExpiry ...
func SetWithExpiry(key string, value interface{}, expiry time.Duration) error {
	return configProvider.SetWithExpiry(key, value, expiry)

}

//Del ...
func Del(keys []string) (int64, error) {
	return configProvider.Del(keys)
}

//Scan ...
func Scan(key string) ([]string, error) {
	return configProvider.Scan(key)
}

//GetWithPrefix ...
func GetWithPrefix(key string) ([]string, []interface{}, error) {

	keys, err := configProvider.Scan(key)
	if err != nil {
		return nil, nil, err
	}

	values, err := configProvider.MGet(keys)
	if err != nil {
		return nil, nil, err
	}

	return keys, values, nil

}

//DelWithPrefix ...
func DelWithPrefix(key string) (int64, error) {

	keys, err := configProvider.Scan(key)
	if err != nil {
		return 0, err
	}

	return configProvider.Del(keys)
}

//HGet ...
func HGet(key string, field string) (string, error) {
	return configProvider.HGet(key, field)

}

//HMGet ...
func HMGet(key string, fields []string) ([]interface{}, error) {
	return configProvider.HMGet(key, fields)
}

//HGetAll ...
func HGetAll(key string) (map[string]string, error) {
	return configProvider.HGetAll(key)
}

//HSet ...
func HSet(key string, field string, value interface{}) error {
	return configProvider.HSet(key, field, value)

}

//HMSet ...
func HMSet(key string, fieldValPair map[string]interface{}) error {
	return configProvider.HMSet(key, fieldValPair)
}

//HDel ...
func HDel(key string, fields []string) (int64, error) {
	return configProvider.HDel(key, fields)

}

//HLen ...
func HLen(key string) (int64, error) {
	return configProvider.HLen(key)

}

//Expire ...
func Expire(key string, expiration time.Duration) error {
	return configProvider.Expire(key, expiration)
}

//ExpireAt ...
func ExpireAt(key string, tm time.Time) error {
	return configProvider.ExpireAt(key, tm)
}

//TTL ...
func TTL(key string) (time.Duration, error) {
	return configProvider.TTL(key)
}

//Exists ...
func Exists(key string) (int64, error) {
	return configProvider.Exists(key)
}

//SMembers ...
func SMembers(key string) ([]string, error) {
	return configProvider.SMembers(key)
}

//Publish -> publisg msg to channel...
func Publish(channelName string, msg interface{}) (int64, error) {
	return configProvider.Publish(channelName, msg)
}

//Close ...
func Close() error {
	return configProvider.Close()
}
