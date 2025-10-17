package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/boltdb/bolt"
)

const (
	AddressHeightSeparator = "_"
)

type DB struct {
	bucket string
	db     *bolt.DB
}

func NewDB(path, bucket string) (*DB, error) {
	db, err := bolt.Open(filepath.Join(path, bucket+".db"), 0600, nil)
	if err != nil {
		return nil, err
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to create bucket: %s", err)
	}
	return &DB{
		bucket: bucket,
		db:     db,
	}, nil
}

func (d *DB) Insert(key string, data any) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return d.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(d.bucket))
		return bucket.Put([]byte(key), dataBytes)
	})
}

func (d *DB) Get(key string, value any) error {
	if err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(d.bucket))
		dataBytes := bucket.Get([]byte(key))
		if dataBytes == nil {
			return nil
		}
		if err := json.Unmarshal(dataBytes, value); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// func (d *DB) GetLatestHeight() (int64, error) {
// 	var height int64
// 	if err := d.db.View(func(tx *bolt.Tx) error {
// 		bucket := tx.Bucket([]byte(d.bucket))
// 		cursor := bucket.Cursor()
// 		for k, _ := cursor.First(); k != nil; k, _ = cursor.Next() {
// 			if strings.Contains(string(k), AddressHeightSeparator) {
// 				continue
// 			}
// 			tmp, err := strconv.ParseInt(string(k), 10, 64)
// 			if err != nil {
// 				return err
// 			}
// 			height = tmp

// 		}
// 		return nil
// 	}); err != nil {
// 		return 0, err
// 	}
// 	return height, nil
// }

func (d *DB) GetLatestHeight() (int64, error) {
	var height int64
	if err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(d.bucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", d.bucket)
		}

		cursor := bucket.Cursor()

		// Start from the last key and work backwards
		for k, _ := cursor.Last(); k != nil; k, _ = cursor.Prev() {
			// Skip keys containing the separator
			if strings.Contains(string(k), AddressHeightSeparator) {
				continue
			}

			// Try to parse the key as a number
			tmp, err := strconv.ParseInt(string(k), 10, 64)
			if err != nil {
				// Skip non-numeric keys
				continue
			}

			// Found the last valid numeric key
			height = tmp
			return nil
		}

		return nil // No valid numeric key found
	}); err != nil {
		return 0, err
	}

	return height, nil
}

// get all keys and values and print as json
func (d *DB) GetAllKVAsJSON() ([]byte, error) {
	data := map[string]any{}
	// the value will already be json string
	if err := d.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(d.bucket))
		cursor := bucket.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			value := map[string]any{}
			if err := json.Unmarshal(v, &value); err != nil {
				return err
			}
			data[string(k)] = value
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func (d *DB) Close() error {
	return d.db.Close()
}
