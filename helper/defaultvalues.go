package helper

import (
	"os"
)

//Env. variable names
const (
	ConnectAsRedisClusterMode     = "CONNECTASREDISCLUSTERMODE"
	RedisKeyForServers            = "REDISKEYFORSERVERS"
	RedisKeyForOnBoardedServers   = "REDISKEYFORONBOARDEDSERVERS"
	RedisKeyForRegisteredDevice   = "REDISKEYFORREGISTEREDDEVICE"
	RedisKeyForTestDevice         = "REDISKEYFORTESTDEVICE"
	RedisKeyForCryptoSecret       = "REDISKEYFORCRYPTOSECRET"
	RedisHashFieldForCryptoSecret = "REDISHASHFIELDFORCRYPTOSECRET"
	RedisKeyForCommunicationGroup = "REDISKEYFORCOMMUNICATIONGROUP"
	DeviceCountPerPartition       = "DEVICECOUNTPERPARTITION"
	RedisDeviceCommandChannel     = "REDISDEVICECOMMANDCHANNEL"

	KafkaBrokers     = "KAFKABROKERS"
	KafkaUserName    = "KAFKAUSERNAME"
	KafkaPassword    = "KAFKAPASSWORD"
	KafkaConfigTopic = "KAFKACONFIGTOPIC"

	MongoEndPoint = "MONGOENDPOINT"
	MongoUserName = "MONGOUSERNAME"
	MongoPassword = "MONGOPASSWORD"
	MongoAuthDB   = "MONGOAUTHDB"
	MongoDBName   = "MONGODBNAME"

	PGSHosts    = "PGSHOSTS"
	PGSPort     = "PGSPORT"
	PGSUserName = "PGSUSERNAME"
	PGSPassword = "PGSPASSWORD"
	PGSDBName   = "PGSDBNAME"

	ConfigServerEndPoint = "CONFIGSERVERENDPOINT"
	ConfigServerUserName = "CONFIGSERVERUSERNAME"
	ConfigServerPassword = "CONFIGSERVERPASSWORD"
	LoggerLogFormat      = "LOGGERLOGFORMAT"
	LoggerLogLevel       = "LOGGERLOGLEVEL"
	JobIntervalInSec     = "JOBINTERVALINSEC"
)


//GetEnv ...
func GetEnv(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	//not exist default val...
	return ""
}

func HasEmpty(values... string) (bool){

	for _,v:= range values {
		if v == "" {
			return true
		}
	}

	return false;
}
