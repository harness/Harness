package builtin

import (
	"bytes"
	"github.com/boltdb/bolt"
	"io"
	"io/ioutil"
	"strconv"
)

// SetLogs inserts or updates a task logs for the
// named repository and build number.
func (db *DB) SetLogs(repo string, build int, task int, rd io.Reader) error {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(task))
	t, err := db.Begin(true)
	if err != nil {
		return err
	}
	//Added
	//rdBuff := new(bytes.Buffer)
	//rdBuff.ReadFrom(rd)
	//log := rdBuff.Bytes()
	log, err := ioutil.ReadAll(rd)
	if err != nil {
		return err
	}

	err = t.Bucket(bucketBuildLogs).Put(key, log)
	if err != nil {
		t.Rollback()
		return err
	}
	return t.Commit()
}

// LogReader gets the task logs at index N for
// the named repository and build number.
func (db *DB) LogReader(repo string, build int, task int) (io.Reader, error) {
	key := []byte(repo + "/" + strconv.Itoa(build) + "/" + strconv.Itoa(task))

	var log []byte
	err := db.View(func(t *bolt.Tx) error {
		var err error
		log, err = raw(t, bucketBuildLogs, key)
		return err
	})
	buf := bytes.NewBuffer(log)
	return buf, err
}
