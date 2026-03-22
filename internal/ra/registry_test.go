package ra

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func tempRADB(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "test_ra.db")
}

func TestResourceRegistry_RegisterAndGet(t *testing.T) {
	reg, err := NewResourceRegistry(tempRADB(t))
	if err != nil {
		t.Fatalf("NewResourceRegistry: %v", err)
	}
	defer reg.Close()

	res := Resource{
		ID:          uuid.New().String(),
		Type:        ResourceCompute,
		Provider:    "local",
		CostPerHour: 0,
		AcquiredAt:  time.Now().UTC(),
		Status:      ResourceStatusActive,
		Capabilities: map[string]interface{}{
			"cpu_cores": 4,
			"ram_gb":    8.0,
		},
	}

	if err := reg.Register(res); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := reg.Get(res.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != res.ID {
		t.Fatalf("ID mismatch: %s != %s", got.ID, res.ID)
	}
	if got.Type != ResourceCompute {
		t.Fatalf("Type mismatch")
	}
}

func TestResourceRegistry_ListByType(t *testing.T) {
	reg, _ := NewResourceRegistry(tempRADB(t))
	defer reg.Close()

	// Register 5 compute + 3 storage resources
	for i := 0; i < 5; i++ {
		reg.Register(Resource{
			ID: uuid.New().String(), Type: ResourceCompute, Provider: "local",
			AcquiredAt: time.Now().UTC(), Status: ResourceStatusActive,
		})
	}
	for i := 0; i < 3; i++ {
		reg.Register(Resource{
			ID: uuid.New().String(), Type: ResourceStorage, Provider: "local",
			AcquiredAt: time.Now().UTC(), Status: ResourceStatusActive,
		})
	}

	compute, _ := reg.ListByType(ResourceCompute)
	storage, _ := reg.ListByType(ResourceStorage)

	if len(compute) != 5 {
		t.Fatalf("expected 5 compute, got %d", len(compute))
	}
	if len(storage) != 3 {
		t.Fatalf("expected 3 storage, got %d", len(storage))
	}
}

func TestResourceRegistry_UpdateStatus(t *testing.T) {
	reg, _ := NewResourceRegistry(tempRADB(t))
	defer reg.Close()

	res := Resource{
		ID: uuid.New().String(), Type: ResourceCompute, Provider: "local",
		AcquiredAt: time.Now().UTC(), Status: ResourceStatusActive,
	}
	reg.Register(res)

	reg.UpdateStatus(res.ID, ResourceStatusReleased)

	got, _ := reg.Get(res.ID)
	if got.Status != ResourceStatusReleased {
		t.Fatalf("expected released, got %s", got.Status)
	}
}

func TestResourceRegistry_Deregister(t *testing.T) {
	reg, _ := NewResourceRegistry(tempRADB(t))
	defer reg.Close()

	res := Resource{
		ID: uuid.New().String(), Type: ResourceCompute, Provider: "local",
		AcquiredAt: time.Now().UTC(), Status: ResourceStatusActive,
	}
	reg.Register(res)
	reg.Deregister(res.ID)

	_, err := reg.Get(res.ID)
	if err == nil {
		t.Fatal("should get error after deregister")
	}
}

func TestEncryptDecryptCreds(t *testing.T) {
	key := make([]byte, 32) // AES-256
	for i := range key {
		key[i] = byte(i)
	}

	original := []byte(`{"host":"10.0.0.1","password":"secret123"}`)

	encrypted, err := EncryptCreds(original, key)
	if err != nil {
		t.Fatalf("EncryptCreds: %v", err)
	}

	// Encrypted should differ from original
	if string(encrypted) == string(original) {
		t.Fatal("encrypted should differ from original")
	}

	decrypted, err := DecryptCreds(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptCreds: %v", err)
	}

	if string(decrypted) != string(original) {
		t.Fatalf("decrypted mismatch: got %s", string(decrypted))
	}
}

func TestEncryptCreds_WrongKeyFails(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	for i := range key1 {
		key1[i] = 1
		key2[i] = 2
	}

	original := []byte("secret data")
	encrypted, _ := EncryptCreds(original, key1)

	_, err := DecryptCreds(encrypted, key2)
	if err == nil {
		t.Fatal("decrypting with wrong key should fail")
	}
}
