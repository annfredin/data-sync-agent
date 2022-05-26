package conman

import (
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
)

//RadisProviderClient ...
type RadisProviderClient struct {
	client *redis.Client
}

//NewRedisProviderClient ...
func NewRedisProviderClient(request ConnRequest) (Provider, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     request.Endpoint,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := client.Ping().Err()
	if err != nil {
		return nil, err
	}

	// returning redis client...
	return &RadisProviderClient{
		client: client,
	}, nil
}

//Get ...
func (rp RadisProviderClient) Get(key string) (string, error) {

	value, err := rp.client.Get(key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

//MGet ...
func (rp RadisProviderClient) MGet(keys []string) ([]interface{}, error) {

	value, err := rp.client.MGet(keys...).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

//Set ...
func (rp RadisProviderClient) Set(key string, value interface{}) error {

	return rp.client.Set(key, value, 0).Err()
}

//MSet ...
func (rp RadisProviderClient) MSet(kvPairs map[string]interface{}) error {

	flatPair := make([]interface{}, 0)
	for key, value := range kvPairs {
		flatPair = append(flatPair, key)
		flatPair = append(flatPair, value)
	}

	return rp.client.MSet(flatPair...).Err()
}

//SetWithExpiry ...
func (rp RadisProviderClient) SetWithExpiry(key string, value interface{}, expiry time.Duration) error {

	return rp.client.Set(key, value, expiry).Err()
}

//Del ...
func (rp RadisProviderClient) Del(keys []string) (int64, error) {
	return rp.client.Del(keys...).Result()
}

//Scan -> get keys with prefix
func (rp RadisProviderClient) Scan(key string) ([]string, error) {

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
func (rp RadisProviderClient) HGet(key string, field string) (string, error) {

	value, err := rp.client.HGet(key, field).Result()
	if err != nil {
		return "", err
	}

	return value, nil
}

//HMGet ...
func (rp RadisProviderClient) HMGet(key string, fields []string) ([]interface{}, error) {

	value, err := rp.client.HMGet(key, fields...).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

//HGetAll ...
func (rp RadisProviderClient) HGetAll(key string) (map[string]string, error) {

	value, err := rp.client.HGetAll(key).Result()
	if err != nil {
		return nil, err
	}

	return value, nil
}

//HSet ...
func (rp RadisProviderClient) HSet(key string, field string, value interface{}) error {

	return rp.client.HSet(key, field, value).Err()
}

//HMSet ...
func (rp RadisProviderClient) HMSet(key string, fieldValPair map[string]interface{}) error {

	return rp.client.HMSet(key, fieldValPair).Err()
}

//HDel ...
func (rp RadisProviderClient) HDel(key string, fields []string) (int64, error) {
	return rp.client.HDel(key, fields...).Result()
}

//HLen ...
func (rp RadisProviderClient) HLen(key string) (int64, error) {
	return rp.client.HLen(key).Result()
}

//Expire ...
func (rp RadisProviderClient) Expire(key string, expiration time.Duration) error {
	return rp.client.Expire(key, expiration).Err()
}

//ExpireAt ...
func (rp RadisProviderClient) ExpireAt(key string, tm time.Time) error {
	return rp.client.ExpireAt(key, tm).Err()
}

//TTL ...
func (rp RadisProviderClient) TTL(key string) (time.Duration, error) {
	return rp.client.TTL(key).Result()
}

//Exists ...
func (rp RadisProviderClient) Exists(key string) (int64, error) {
	return rp.client.Exists(key).Result()
}

//SMembers ...
func (rp RadisProviderClient) SMembers(key string) ([]string, error) {
	return rp.client.SMembers(key).Result()
}

//Publish -> publisg msg to channel...
func (rp RadisProviderClient) Publish(channelName string, msg interface{}) (int64, error) {
	return rp.client.Publish(channelName, msg).Result()
}

//Close -> cloes the redis cluster conn...
func (rp RadisProviderClient) Close() error {
	return rp.client.Close()
}
