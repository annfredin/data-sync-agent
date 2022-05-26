package model

//CryptoSecretProvider is used to hold crypto secrets
type CryptoSecretProvider struct {
	HashingSecret   string `json:"hashing_secret"`
	EncryptionKeyW1 string `json:"encryption_key_w1"`
	EncryptionIVW1  string `json:"encryption_iv_w1"`
	EncryptionKeyW2 string `json:"encryption_key_w2"`
	EncryptionIVW2  string `json:"encryption_iv_w2"`
	EncryptionKeyW3 string `json:"encryption_key_w3"`
}
