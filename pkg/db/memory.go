package db

func New() *Memory {
	return &Memory{}
}

type Memory struct{}
