package mock

type Tx struct{}

func (tx *Tx) Commit() error {
	return nil
}

func (tx *Tx) Rollback() error {
	return nil
}
