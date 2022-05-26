package crypto

import (
	"encoding/json"

	"data-sync-agent/helper"

	cm "data-sync-agent/config"
	ep "data-sync-agent/crypto/enc"
	cryptomodel "data-sync-agent/crypto/model"

	"go.uber.org/zap"
)

var encProvider ep.Provider
var cryptoSecretProvider *cryptomodel.CryptoSecretProvider

//EncType is
type EncType int

const (
	//W1EncType less weightage enc type
	W1EncType EncType = iota + 1
	//W2EncType medium weightage enc type
	W2EncType
	//W3EncType high weightage enc type
	W3EncType
)

//InitializeCryptoProvider initializing the crypto provider....
func InitializeCryptoProvider(log *zap.Logger) {

	//init/load crypto keys(secrets)
	loadCryptoSecrets(log)

	eprovider, err := ep.NewAesProvider()
	if err != nil {
		log.With(zap.Error(err)).Error("Enc Provider initialize error")
		return
	}
	log.Info("Enc Provider initialized")

	//assigning to global variable(must)
	encProvider = eprovider
}

func loadCryptoSecrets(log *zap.Logger) {
	log.Info("Loading crypto secrets starts")

	cryptoSecretConfigKey := helper.GetEnv(helper.RedisKeyForCryptoSecret)

	cryptoSecretHashKeyField := helper.GetEnv(helper.RedisHashFieldForCryptoSecret)

	jsonData, err := cm.HGet(cryptoSecretConfigKey, cryptoSecretHashKeyField)
	if err != nil {
		log.With(zap.Error(err)).Error("LoadCryptoSecrets initialize error")
	}

	//deserializing data
	secretProvider := &cryptomodel.CryptoSecretProvider{}
	err = json.Unmarshal([]byte(jsonData), secretProvider)
	if err != nil {
		log.With(zap.Error(err)).Error("LoadCryptoSecrets(Unmarshal) initialize error")
	}

	log.Info("Loading crypto secrets ends")
	//assigning to global var
	cryptoSecretProvider = secretProvider
}

//initializeInternally used to initalize internally
func initializeInternally() {
	tmpLog, _ := zap.NewDevelopment()
	InitializeCryptoProvider(tmpLog)
}

//Encrypt is
func Encrypt(rawData string, encType EncType) (string, error) {

	var res string
	var err error

	//if provider not found, manully init the provider...
	if encProvider == nil {
		initializeInternally()
	}

	switch encType {
	case W1EncType:
		res, err = encProvider.EncryptT1(rawData, cryptoSecretProvider.EncryptionKeyW1, cryptoSecretProvider.EncryptionIVW1)
	case W2EncType:
		res, err = encProvider.EncryptT1(rawData, cryptoSecretProvider.EncryptionKeyW2, cryptoSecretProvider.EncryptionIVW2)
	case W3EncType:
		res, err = encProvider.EncryptT2(rawData, cryptoSecretProvider.EncryptionKeyW3)
	default:
		res, err = encProvider.EncryptT1(rawData, cryptoSecretProvider.EncryptionKeyW1, cryptoSecretProvider.EncryptionIVW1)
	}

	return res, err
}

//Decrypt is
func Decrypt(cipherData string, encType EncType) (string, error) {
	var res string
	var err error

	//if provider not found, manully init the provider...
	if encProvider == nil {
		initializeInternally()
	}

	switch encType {
	case W1EncType:
		res, err = encProvider.DecryptT1(cipherData, cryptoSecretProvider.EncryptionKeyW1, cryptoSecretProvider.EncryptionIVW1)
	case W2EncType:
		res, err = encProvider.DecryptT1(cipherData, cryptoSecretProvider.EncryptionKeyW2, cryptoSecretProvider.EncryptionIVW2)
	case W3EncType:
		res, err = encProvider.DecryptT2(cipherData, cryptoSecretProvider.EncryptionKeyW3)
	default:
		res, err = encProvider.DecryptT1(cipherData, cryptoSecretProvider.EncryptionKeyW1, cryptoSecretProvider.EncryptionIVW1)
	}

	return res, err
}
