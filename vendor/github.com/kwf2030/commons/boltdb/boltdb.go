package boltdb

import (
  "bytes"
  "errors"
  "os"
  "go.etcd.io/bbolt"
)

var (
  ErrInvalidArgs    = errors.New("invalid args")
  ErrBucketNotFound = errors.New("bucket not found")
  ErrKeyNotFound    = errors.New("key not found")
)

type Store struct {
  DB *bbolt.DB
}

func Open(path string, buckets ...string) (*Store, error) {
  db, e := bbolt.Open(path, os.ModePerm, nil)
  if e != nil {
    return nil, e
  }
  if len(buckets) > 0 {
    e = db.Update(func(tx *bbolt.Tx) error {
      for _, v := range buckets {
        if v == "" {
          continue
        }
        _, e := tx.CreateBucketIfNotExists([]byte(v))
        if e != nil {
          return e
        }
      }
      return nil
    })
    if e != nil {
      return nil, e
    }
  }
  return &Store{DB: db}, nil
}

func (s *Store) Close() error {
  return s.DB.Close()
}

func (s *Store) QueryAndUpdateV(bucket, k []byte, f func(k, v []byte, n int) ([]byte, error)) error {
  if bucket == nil || k == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.Update(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    ov := b.Get(k)
    if ov == nil {
      return ErrKeyNotFound
    }
    nv, e := f(k, ov, b.Stats().KeyN)
    if e != nil {
      return e
    }
    if nv != nil {
      return b.Put(k, nv)
    }
    return nil
  })
}

func (s *Store) QueryAndUpdateVPrefix(bucket, prefix []byte, f func(k, v []byte, n int) ([]byte, error)) error {
  if bucket == nil || prefix == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.Update(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    n := b.Stats().KeyN
    c := b.Cursor()
    for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
      if !bytes.HasPrefix(k, prefix) {
        break
      }
      nv, e := f(k, v, n)
      if e != nil {
        return e
      }
      if nv != nil {
        return b.Put(k, nv)
      }
    }
    return nil
  })
}

func (s *Store) UpdateV(bucket, k, v []byte) error {
  if bucket == nil || k == nil || v == nil {
    return ErrInvalidArgs
  }
  return s.DB.Update(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    return b.Put(k, v)
  })
}

func (s *Store) UpdateB(bucket []byte, f func(b *bbolt.Bucket) error) error {
  if bucket == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.Update(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    return f(b)
  })
}

func (s *Store) Update(f func(tx *bbolt.Tx) error) error {
  if f == nil {
    return ErrInvalidArgs
  }
  return s.DB.Update(f)
}

func (s *Store) Get(bucket, k []byte) []byte {
  if bucket == nil || k == nil {
    return nil
  }
  var ret []byte
  e := s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    v := b.Get(k)
    if v == nil {
      return ErrKeyNotFound
    }
    ret = make([]byte, len(v))
    copy(ret, v)
    return nil
  })
  if e != nil {
    return nil
  }
  return ret
}

func (s *Store) QueryV(bucket, k []byte, f func(k, v []byte, n int) error) error {
  if bucket == nil || k == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    v := b.Get(k)
    if v == nil {
      return ErrKeyNotFound
    }
    return f(k, v, b.Stats().KeyN)
  })
}

func (s *Store) QueryB(bucket []byte, f func(b *bbolt.Bucket) error) error {
  if bucket == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    return f(b)
  })
}

func (s *Store) Query(f func(tx *bbolt.Tx) error) error {
  if f == nil {
    return ErrInvalidArgs
  }
  return s.DB.View(f)
}

func (s *Store) EachKV(bucket []byte, f func(k, v []byte, n int) error) error {
  if bucket == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    n := b.Stats().KeyN
    c := b.Cursor()
    for k, v := c.First(); k != nil; k, v = c.Next() {
      if e := f(k, v, n); e != nil {
        return e
      }
    }
    return nil
  })
}

func (s *Store) EachKVPrefix(bucket, prefix []byte, f func(k, v []byte, n int) error) error {
  if bucket == nil || prefix == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    n := b.Stats().KeyN
    c := b.Cursor()
    for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
      if !bytes.HasPrefix(k, prefix) {
        break
      }
      if e := f(k, v, n); e != nil {
        return e
      }
    }
    return nil
  })
}

func (s *Store) EachB(bucket []byte, f func(b *bbolt.Bucket) error) error {
  if bucket == nil || f == nil {
    return ErrInvalidArgs
  }
  return s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    return f(b)
  })
}

func (s *Store) CountKV(bucket []byte) (int, error) {
  if bucket == nil {
    return 0, ErrInvalidArgs
  }
  n := 0
  e := s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    n = b.Stats().KeyN
    return nil
  })
  if e != nil {
    return 0, e
  }
  return n, nil
}

func (s *Store) CountKVPrefix(bucket, prefix []byte) (int, error) {
  if bucket == nil {
    return 0, ErrInvalidArgs
  }
  n := 0
  e := s.DB.View(func(tx *bbolt.Tx) error {
    b := tx.Bucket(bucket)
    if b == nil {
      return ErrBucketNotFound
    }
    c := b.Cursor()
    for k, _ := c.Seek(prefix); k != nil; k, _ = c.Next() {
      if !bytes.HasPrefix(k, prefix) {
        break
      }
      n++
    }
    return nil
  })
  if e != nil {
    return 0, e
  }
  return n, nil
}
