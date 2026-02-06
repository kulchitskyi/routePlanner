package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockTagExtractor struct {
	mock.Mock
}

func (m *MockTagExtractor) ExtractTags(text string, allowedTags []string) ([]string, error) {
	args := m.Called(text, allowedTags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockTagExtractor) ExtractMultiTags(segments, allowedTags []string) ([][]string, error) {
	args := m.Called(segments, allowedTags)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([][]string), args.Error(1)
}
