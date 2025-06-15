package solidq

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"go.etcd.io/bbolt"
)

var appstore sync.Map

var rootpath string

func init() {
	if runtime.GOOS == "darwin" {
		rootpath = "./"
	} else {
		rootpath = "/var/lib/solidq/"

		_, err := os.Stat(rootpath)
		if os.IsNotExist(err) {
			err = os.MkdirAll(rootpath, 0755)
			if err != nil {
				panic("Failed to create root path: " + err.Error())
			}
		}
	}
}

func listapps(physical bool) ([]string, error) {
	var apps []string

	if physical {
		files, err := os.ReadDir(rootpath)
		if err != nil {
			fmt.Println("Error reading directory:", err)
			return apps, err
		}

		for _, file := range files {
			if !file.IsDir() && filepath.Ext(file.Name()) == ".db" {
				apps = append(apps, file.Name()[:len(file.Name())-3]) // Remove .db extension
			}
		}
		return apps, nil
	}

	appstore.Range(func(key, value interface{}) bool {
		apps = append(apps, key.(string))
		return true
	})
	return apps, nil
}

func enusureQ(appname string) (*Que, error) {
	if que, ok := appstore.Load(appname); ok {
		return que.(*Que), nil
	}

	path := filepath.Join(rootpath, appname+".db")
	q, err := OpenQue(path)
	if err != nil {
		return nil, err
	}

	appstore.Store(appname, q)
	return q, nil
}

type Que struct {
	db *bbolt.DB
}

func OpenQue(path string) (*Que, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	return &Que{db: db}, nil
}

func (q *Que) Close() error {
	if q.db != nil {
		return q.db.Close()
	}
	return nil
}

func (q *Que) Push(channel, id string) error {
	if q.db == nil {
		return errors.New("database is not open")
	}

	if id == "" {
		return errors.New("work ID cannot be empty")
	}

	q.Inc(channel + ":push")
	return q.db.Update(func(tx *bbolt.Tx) error {
		channelbucket, err := tx.CreateBucketIfNotExists([]byte(channel))
		if err != nil {
			return err
		}

		return channelbucket.Put([]byte(id), []byte(""))
	})
}

func (q *Que) ListChannels() ([]string, error) {
	if q.db == nil {
		return nil, errors.New("database is not open")
	}

	var channels []string
	err := q.db.View(func(tx *bbolt.Tx) error {
		channels = make([]string, 0)
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			channels = append(channels, string(name))
			return nil
		})
	})

	return channels, err
}

func (q *Que) ListChannelsWithCount() (map[string]int, error) {
	if q.db == nil {
		return nil, errors.New("database is not open")
	}

	channels := make(map[string]int)
	err := q.db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			channels[string(name)] = b.Stats().KeyN
			return nil
		})
	})

	return channels, err
}

func (q *Que) ResetChannel(channel string) error {
	if q.db == nil {
		return errors.New("database is not open")
	}

	return q.db.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket([]byte(channel))
	})
}

func (q *Que) Inc(chcommand string) error {
	if q.db == nil {
		return errors.New("database is not open")
	}

	return q.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("app_stats"))
		if err != nil {
			return err
		}

		count := fmt.Sprintf("%d", b.Stats().KeyN+1)
		return b.Put([]byte(chcommand), []byte(count))
	})
}

func (q *Que) ListKeysWithValues(bucket string) (map[string]string, error) {
	if q.db == nil {
		return nil, errors.New("database is not open")
	}

	var keyvalues = make(map[string]string)
	err := q.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return errors.New("bucket does not exist") // Bucket does not exist
		}

		return b.ForEach(func(k, v []byte) error {
			keyvalues[string(k)] = string(v)
			return nil
		})
	})

	return keyvalues, err
}

func (q *Que) Count(channel string) (int, error) {
	if q.db == nil {
		return 0, errors.New("database is not open")
	}

	var count int
	err := q.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(channel))
		if b == nil {
			return nil
		}
		count = b.Stats().KeyN
		return nil
	})

	return count, err
}

func (q *Que) PopWithCount(channel string, count int) ([]string, error) {
	if q.db == nil {
		return nil, errors.New("database is not open")
	}

	q.Inc(channel + ":pop")

	var ids []string
	err := q.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(channel))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for i := 0; i < count; i++ {
			k, _ := c.First()
			if k == nil {
				break
			}

			ids = append(ids, string(k))
			err := b.Delete(k)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return ids, err
}
