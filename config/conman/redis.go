package conman

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
)

type (

	//Provider ...
	Provider interface {
		Get(key string) (string, error)
		MGet(keys []string) ([]interface{}, error)
		Set(key string, value interface{}) error
		MSet(kvPairs map[string]interface{}) error
		SetWithExpiry(key string, value interface{}, expiry time.Duration) error
		Del(keys []string) (int64, error)
		Scan(key string) ([]string, error)

		//hash fns
		HGet(key string, field string) (string, error)
		HMGet(key string, fields []string) ([]interface{}, error)
		HGetAll(key string) (map[string]string, error)
		HSet(key string, field string, value interface{}) error
		HMSet(key string, fieldValPair map[string]interface{}) error
		HDel(key string, fields []string) (int64, error)
		HLen(key string) (int64, error)

		Expire(key string, expiration time.Duration) error
		ExpireAt(key string, expiration time.Time) error
		TTL(key string) (time.Duration, error)

		Exists(key string) (int64, error)

		SMembers(key string) ([]string, error)

		Publish(channelName string, msg interface{}) (int64, error)

		Close() error
	}

	//ConnRequest -> Conn Request type
	ConnRequest struct {
		Endpoint string
		UserName string
		Password string
	}

	//RadisProvider -> store client
	RadisProvider struct {
		client *redis.ClusterClient
	}
)

//NewRedisProvider ...
func NewRedisProvider(request ConnRequest) (Provider, error) {

	clusterOptions := &redis.ClusterOptions{
		Addrs: strings.Split(request.Endpoint, ","),
	}

	if len(request.UserName) > 0 {
		clusterOptions.Username = request.UserName
	}

	if len(request.Password) > 0 {
		clusterOptions.Password = request.Password
	}

	rdb := redis.NewClusterClient(clusterOptions)
	err := rdb.Ping().Err()
	if err != nil {
		return nil, err
	}

	// returning redis client...
	return &RadisProvider{
		client: rdb,
	}, nil
}

//Get ...
func (rp *RadisProvider) Get(key string) (string, error) {

	value, err := rp.client.Get(key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

//MGet ...
func (rp *RadisProvider) MGet(keys []string) ([]interface{}, error) {

	value, err := rp.client.MGet(keys...).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

//Set ...
func (rp *RadisProvider) Set(key string, value interface{}) error {

	return rp.client.Set(key, value, 0).Err()
}

//MSet ...
func (rp *RadisProvider) MSet(kvPairs map[string]interface{}) error {

	flatPair := make([]interface{}, 0)
	for key, value := range kvPairs {
		flatPair = append(flatPair, key)
		flatPair = append(flatPair, value)
	}

	return rp.client.MSet(flatPair...).Err()
}

//SetWithExpiry ...
func (rp *RadisProvider) SetWithExpiry(key string, value interface{}, expiry time.Duration) error {

	return rp.client.Set(key, value, expiry).Err()
}

//Del ...
func (rp *RadisProvider) Del(keys []string) (int64, error) {
	return rp.client.Del(keys...).Result()
}

//Scan -> get keys with prefix
func (rp *RadisProvider) Scan(key string) ([]string, error) {

	var cursor uint64
	var err error
	var result []string

	for {
		var keys []string
		keys, cursor, err = rp.client.Scan(cursor, fmt.Sprintf("%s*", key), 10).Result()
		if err != nil {
			break
		}

		result = append(result, keys...)
		if cursor == 0 {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

//HGet ...
func (rp *RadisProvider) HGet(key string, field string) (string, error) {

	value, err := rp.client.HGet(key, field).Result()
	if err != nil {
		return "", err
	}

	return value, nil
}

//HMGet ...
func (rp *RadisProvider) HMGet(key string, fields []string) ([]interface{}, error) {

	value, err := rp.client.HMGet(key, fields...).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

//HGetAll ...
func (rp *RadisProvider) HGetAll(key string) (map[string]string, error) {

	value, err := rp.client.HGetAll(key).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

//HSet ...
func (rp *RadisProvider) HSet(key string, field string, value interface{}) error {

	return rp.client.HSet(key, field, value).Err()
}

//HMSet ...
func (rp *RadisProvider) HMSet(key string, fieldValPair map[string]interface{}) error {

	return rp.client.HMSet(key, fieldValPair).Err()
}

//HDel ...
func (rp *RadisProvider) HDel(key string, fields []string) (int64, error) {
	return rp.client.HDel(key, fields...).Result()
}

//HLen ...
func (rp *RadisProvider) HLen(key string) (int64, error) {
	return rp.client.HLen(key).Result()
}

//Expire ...
func (rp *RadisProvider) Expire(key string, expiration time.Duration) error {
	return rp.client.Expire(key, expiration).Err()
}

//ExpireAt ...
func (rp *RadisProvider) ExpireAt(key string, tm time.Time) error {
	return rp.client.ExpireAt(key, tm).Err()
}

//TTL ...
func (rp *RadisProvider) TTL(key string) (time.Duration, error) {
	return rp.client.TTL(key).Result()
}

//Exists ...
func (rp *RadisProvider) Exists(key string) (int64, error) {
	return rp.client.Exists(key).Result()
}

//SMembers ...
func (rp *RadisProvider) SMembers(key string) ([]string, error) {
	return rp.client.SMembers(key).Result()
}

//Publish -> publisg msg to channel...
func (rp *RadisProvider) Publish(channelName string, msg interface{}) (int64, error) {
	return rp.client.Publish(channelName, msg).Result()
}

//Close -> cloes the redis cluster conn...
func (rp *RadisProvider) Close() error {
	return rp.client.Close()
}
