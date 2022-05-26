package enc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

var (
	// ErrInvalidBlockSize indicates hash blocksize <= 0.
	ErrInvalidBlockSize = errors.New("invalid blocksize")

	// ErrInvalidPKCS7Data indicates bad input to PKCS7 pad or unpad.
	ErrInvalidPKCS7Data = errors.New("invalid PKCS7 data (empty or not padded)")

	// ErrInvalidPKCS7Padding indicates PKCS7 unpad fails to bad input.
	ErrInvalidPKCS7Padding = errors.New("invalid padding on input")

	//ErrInvalidENCInput indicates ciphertext input too short
	ErrInvalidENCInput = errors.New("ciphertext input too short")

	//ErrCipherInputNotMulOfBlockSize indicates ciphertext is not a multiple of the block size
	ErrCipherInputNotMulOfBlockSize = errors.New("ciphertext is not a multiple of the block size")
)

type (
	//Provider is
	Provider interface {
		EncryptT1(rawData, key, iv string) (string, error)
		DecryptT1(cipherData, key, iv string) (string, error)
		EncryptT2(rawData, key string) (string, error)
		DecryptT2(cipherData, key string) (string, error)
	}

	//AesProvider is
	AesProvider struct{}
)

//NewAesProvider ...
func NewAesProvider() (Provider, error) {
	// returning  client...
	return &AesProvider{}, nil
}

//EncryptT1 is
func (encProvider *AesProvider) EncryptT1(rawData, key, iv string) (string, error) {

	data, err := aesCBCEncrypt([]byte(rawData), []byte(key), []byte(iv))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(data), nil
}

//DecryptT1 is
func (encProvider *AesProvider) DecryptT1(cipherData, key, iv string) (string, error) {
	data, err := hex.DecodeString(cipherData)
	if err != nil {
		return "", err
	}

	dnData, err := aesCBCDecrypt(data, []byte(key), []byte(iv))
	if err != nil {
		return "", err
	}

	return string(dnData), nil
}

//EncryptT2 is
func (encProvider *AesProvider) EncryptT2(rawData, key string) (string, error) {

	data, err := gcmEncrypt([]byte(rawData), []byte(key))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(data), nil
}

//DecryptT2 is
func (encProvider *AesProvider) DecryptT2(cipherData, key string) (string, error) {
	data, err := hex.DecodeString(cipherData)
	if err != nil {
		return "", err
	}

	dnData, err := gcmDecrypt(data, []byte(key))
	if err != nil {
		return "", err
	}

	return string(dnData), nil
}

//AesCBCEncrypt is filling the 16 bits of the key key, 24, 32 respectively corresponding to AES-128, AES-192, or AES-256.
func aesCBCEncrypt(rawData, key, iv []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//fill the original
	blockSize := block.BlockSize()
	rawData = pkcs7Padding(rawData, blockSize)
	cipherText := make([]byte, len(rawData))

	//block size and initial vector size must be the same
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, rawData)

	//returning
	return cipherText, nil
}

//AesCBCDecrypt is
func aesCBCDecrypt(encryptData, key, iv []byte) ([]byte, error) {

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()

	if len(encryptData) < blockSize {
		return nil, ErrInvalidENCInput
	}

	// CBC mode always works in whole blocks.
	if len(encryptData)%blockSize != 0 {
		return nil, ErrCipherInputNotMulOfBlockSize
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(encryptData, encryptData)
	// Unfill
	encryptData = pkcs7UnPadding(encryptData)

	//returning result...
	return encryptData, nil
}

// Use pkcs7Padding to fill
func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

// Use pkcs7UnPadding to unfill
func pkcs7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

//### AES-GCM ENC/DEC section....
//gcmEncrypt is
func gcmEncrypt(rawData, key []byte) ([]byte, error) {

	block, err := aes.NewCipher(getHashKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, rawData, nil)

	return ciphertext, nil
}

//gcmDecrypt ...
func gcmDecrypt(encryptData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(getHashKey(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	nonce, ciphertext := encryptData[:nonceSize], encryptData[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}

	return plaintext, nil
}

//getHashKey ...
func getHashKey(key []byte) []byte {
	hasher := sha256.New()
	hasher.Write(key)
	result := hasher.Sum(nil)

	return result
}
