package solidq

import (
	"encoding/json"
	"errors"

	"go.etcd.io/bbolt"
)

type Work[T any] struct {
	Id   string
	Data T
}

type Que[T any] struct {
	db *bbolt.DB
}

func OpenQue[T any](path string) (*Que[T], error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Que[T]{db: db}, nil
}

func (q *Que[T]) Close() error {
	if q.db != nil {
		return q.db.Close()
	}
	return nil
}

func (q *Que[T]) Push(channel string, work Work[T]) error {
	if q.db == nil {
		return errors.New("database is not open")
	}

	if work.Id == "" {
		return errors.New("work ID cannot be empty")
	}

	// if work.Data ==  || len(work.Data) == 0 {
	// 	work.Data = make(Payload)
	// }

	return q.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(channel))
		if err != nil {
			return err
		}

		dataBytes, err := json.Marshal(work.Data)
		if err != nil {
			return err
		}

		return b.Put([]byte(work.Id), dataBytes)
	})
}

func (q *Que[T]) ListChannels() ([]string, error) {
	if q.db == nil {
		return nil, errors.New("database is not open")
	}

	var channels []string
	err := q.db.View(func(tx *bbolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
			channels = append(channels, string(name))
			return nil
		})
	})

	return channels, err
}

func (q *Que[T]) ListChannelsWithCount() (map[string]int, error) {
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

func (q *Que[T]) ResetChannel(channel string) error {
	if q.db == nil {
		return errors.New("database is not open")
	}

	return q.db.Update(func(tx *bbolt.Tx) error {
		return tx.DeleteBucket([]byte(channel))
	})
}

func (q *Que[T]) Count(channel string) (int, error) {
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

func (q *Que[T]) PopWithCount(channel string, count int) ([]*Work[T], error) {
	if q.db == nil {
		return nil, errors.New("database is not open")
	}

	var works []*Work[T]
	err := q.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(channel))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for i := 0; i < count; i++ {
			k, v := c.First()
			if k == nil {
				break
			}

			work := &Work[T]{}
			work.Id = string(k)
			err := json.Unmarshal(v, &work.Data)
			if err != nil {
				return err
			}

			works = append(works, work)
			err = b.Delete(k)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return works, err
}

func (q *Que[T]) Pop(channel string) (*Work[T], error) {
	if q.db == nil {
		return &Work[T]{}, errors.New("database is not open")
	}

	var work *Work[T]
	err := q.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(channel))
		if b == nil {
			return nil
		}

		if b.Stats().KeyN == 0 {
			return nil
		}

		c := b.Cursor()
		k, v := c.First()
		if k == nil {
			return nil
		}

		work = &Work[T]{}
		work.Id = string(k)
		err := json.Unmarshal(v, &work.Data)
		if err != nil {
			return err
		}

		return b.Delete(k)
	})

	return work, err
}
