package mock_storage

import (
	"github.com/golang/mock/gomock"
	"testing"
)

func TestMockStorageKeeper(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock storage keeper
	mockStorage := NewMockStorageKeeper(ctrl)

	// Verify the mock was created correctly
	if mockStorage == nil {
		t.Error("Expected mock storage keeper to be created, but got nil")
	}

	// Verify that we can call methods on the mock
	// This is just a basic test to ensure the mock works
	mockStorage.EXPECT().Ping().Return(nil).AnyTimes()

	// Call the method
	err := mockStorage.Ping()

	// Verify the result
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
}

func TestMockStorageKeeperAdd(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock storage keeper
	mockStorage := NewMockStorageKeeper(ctrl)

	// Set up expectations
	mockStorage.EXPECT().Add("test", "url").Return(nil)

	// Call the method
	err := mockStorage.Add("test", "url")

	// Verify the result
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}
}

func TestMockStorageKeeperGetURL(t *testing.T) {
	// Create a mock controller
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create a mock storage keeper
	mockStorage := NewMockStorageKeeper(ctrl)

	// Set up expectations
	mockStorage.EXPECT().GetURL("test").Return("url", nil)

	// Call the method
	url, err := mockStorage.GetURL("test")

	// Verify the result
	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	if url != "url" {
		t.Errorf("Expected url to be 'url', but got %v", url)
	}
}
