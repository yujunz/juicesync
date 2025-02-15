// Copyright (C) 2018-present Juicedata Inc.

package object

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/juicedata/juicesync/utils"
)

var logger = utils.GetLogger("juicesync")

type Object struct {
	Key   string
	Size  int64
	Ctime int // Unix seconds
	Mtime int // Unix seconds
}

type MultipartUpload struct {
	MinPartSize int
	MaxCount    int
	UploadID    string
}

type Part struct {
	Num  int
	Size int
	ETag string
}

type PendingPart struct {
	Key      string
	UploadID string
	Created  time.Time
}

type ObjectStorage interface {
	String() string
	Create() error
	Get(key string, off, limit int64) (io.ReadCloser, error)
	Put(key string, in io.Reader) error
	Copy(dst, src string) error
	Exists(key string) error
	Delete(key string) error
	List(prefix, marker string, limit int64) ([]*Object, error)
	CreateMultipartUpload(key string) (*MultipartUpload, error)
	UploadPart(key string, uploadID string, num int, body []byte) (*Part, error)
	AbortUpload(key string, uploadID string)
	CompleteUpload(key string, uploadID string, parts []*Part) error
	ListUploads(marker string) ([]*PendingPart, string, error)
}

var notSupported = errors.New("not supported")

type defaultObjectStorage struct{}

func (s defaultObjectStorage) Create() error {
	return nil
}

func (s defaultObjectStorage) CreateMultipartUpload(key string) (*MultipartUpload, error) {
	return nil, notSupported
}

func (s defaultObjectStorage) UploadPart(key string, uploadID string, num int, body []byte) (*Part, error) {
	return nil, notSupported
}

func (s defaultObjectStorage) AbortUpload(key string, uploadID string) {}

func (s defaultObjectStorage) CompleteUpload(key string, uploadID string, parts []*Part) error {
	return notSupported
}

func (s defaultObjectStorage) ListUploads(marker string) ([]*PendingPart, string, error) {
	return nil, "", nil
}

func (s defaultObjectStorage) List(prefix, marker string, limit int64) ([]*Object, error) {
	return nil, notSupported
}

type Register func(endpoint, accessKey, secretKey string) ObjectStorage

var storages = make(map[string]Register)

func register(name string, register Register) {
	storages[name] = register
}

func CreateStorage(name, endpoint, accessKey, secretKey string) ObjectStorage {
	f, ok := storages[name]
	if ok {
		return f(endpoint, accessKey, secretKey)
	}
	panic(fmt.Sprintf("invalid storage: %s", name))
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func DoTesting(store ObjectStorage) error {
	rand.Seed(int64(time.Now().UnixNano()))
	key := "testing/" + randSeq(10)
	data := make([]byte, 100)
	rand.Read(data)
	if err := store.Put(key, bytes.NewBuffer(data)); err != nil {
		if err := store.Create(); err != nil {
			return fmt.Errorf("Failed to create %s: %s", store, err.Error())
		}
		if err := store.Put(key, bytes.NewBuffer(data)); err != nil {
			time.Sleep(time.Second * 3)
			err = store.Put(key, bytes.NewBuffer(data))
			return fmt.Errorf("Failed to put: %v", err)
		}
	}
	p, err := store.Get(key, 0, -1)
	if err != nil {
		return fmt.Errorf("Failed to get: %v", err)
	}
	data2, err := ioutil.ReadAll(p)
	p.Close()
	if !bytes.Equal(data, data2) {
		return fmt.Errorf("Read wrong data")
	}
	if err = store.Delete(key); err != nil {
		return fmt.Errorf("Failed to delete: %v", err)
	}
	return nil
}
