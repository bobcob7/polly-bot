package db

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"
)

func NewFile(filePath string) (*File, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0777)
	if err != nil {
		return nil, err
	}
	return &File{
		file: file,
	}, nil
}

type File struct {
	sync.RWMutex
	file  io.ReadWriteCloser
	Users map[uuid.UUID]*User `json:"users"`
}

func (m *File) Close() error {
	return m.file.Close()
}

func (m *File) readFile() error {
	return json.NewDecoder(m.file).Decode(m)
}

func (m *File) writeFile() error {
	return json.NewEncoder(m.file).Encode(m)
}

func (m *File) GetUsers(ctx context.Context) ([]*User, error) {
	m.RLock()
	defer m.RUnlock()
	if err := m.readFile(); err != nil {
		return nil, err
	}
	return maps.Values(m.Users), nil
}

func (m *File) CreateUser(ctx context.Context, name string) (*User, error) {
	m.Lock()
	defer m.Unlock()
	if err := m.readFile(); err != nil {
		return nil, err
	}
	// Check if user name is already there
	for _, user := range m.Users {
		if user.Name == name {
			return nil, fmt.Errorf("user %s is already present", name)
		}
	}
	user := &User{
		ID:        uuid.New(),
		Name:      name,
		AuthLevel: AuthLevelNone,
		Created:   time.Now(),
	}
	m.Users[user.ID] = user
	if err := m.writeFile(); err != nil {
		return nil, err
	}
	return user, nil
}

func (m *File) UpdateUser(ctx context.Context, user *User) error {
	m.Lock()
	defer m.Unlock()
	if err := m.readFile(); err != nil {
		return err
	}
	// Check that user exists and that name matches
	existingUser, ok := m.Users[user.ID]
	if !ok {
		return fmt.Errorf("user %s is not present", user.ID)
	} else if existingUser.Name != user.Name {
		return fmt.Errorf("user name cannot be changed")
	}
	now := time.Now()
	existingUser.Updated = &now
	existingUser.AuthLevel = user.AuthLevel
	m.Users[user.ID] = existingUser
	if err := m.writeFile(); err != nil {
		return err
	}
	return nil
}

func (m *File) DeleteUser(ctx context.Context, id uuid.UUID) error {
	m.Lock()
	defer m.Unlock()
	delete(m.Users, id)
	if err := m.writeFile(); err != nil {
		return err
	}
	return nil
}
