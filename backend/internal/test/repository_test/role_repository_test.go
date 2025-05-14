package repositorytest

import (
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	
	var err error
	if os.Getenv("CI") != "true" {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Fatalf("Error loading .env file")
		}
	}
	

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to test database: %v", err)
	}

	// Migrate schema
	err = testDB.AutoMigrate(&entity.Role{})
	
	if err != nil {
		log.Fatalf("Failed to auto-migrate: %v", err)
	}

	code := m.Run()
	os.Exit(code)
}

func TestCreateRole(t *testing.T) {
	transaction := testDB.Begin()
	defer transaction.Rollback()
	var err error

	roleRepo := repository.NewRoleRepository(transaction)
	role := &entity.Role{Name: "Tester"}
	

	created, err := roleRepo.CreateRole(role)

	if err != nil {
		t.Fatalf("failed to create role: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected role ID to be set")
	}
	if created.Name != "Tester" {
		t.Errorf("expected role name to be 'Tester', got '%s'", created.Name)
	}

}

func TestGetRole(t *testing.T) {
	// Start a transaction to isolate test data
	transaction := testDB.Begin()
	defer transaction.Rollback()

	
	// Create test data
	testRole := &entity.Role{Name: "Tester"}
	if err := transaction.Create(testRole).Error; err != nil {
		t.Fatalf("Failed to insert test role: %v", err)
	}

	// Create repository with the test transaction
	roleRepo := repository.NewRoleRepository(transaction)

	// Act: Call GetRole
	roles := roleRepo.GetRole()

	// Assert: Check if at least one role exists
	if len(roles) == 0 {
		t.Fatal("Expected at least one role, got none")
	}

	// Assert: Check if our test role is in the list
	found := false
	for _, role := range roles {
		if role.Name == "Tester" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find role with name 'Tester', but it was not found")
	}
}

func TestUpdateRole(t *testing.T){
	transaction := testDB.Begin()
	defer transaction.Rollback()

	existingRole := &entity.Role{Name:"Tester"}

	roleRepo := repository.NewRoleRepository(transaction)
	createdRole, err := roleRepo.CreateRole(existingRole)
	if err != nil {
		t.Fatalf("Couldn't create a role %v", err)
	}

	updatedRole := &entity.Role{ID: createdRole.ID, Name: "updated"}
	updated, err := roleRepo.UpdateRole(updatedRole)
	if err != nil {
		t.Fatal("Couldn't update the role")
	}

	if updated.Name != "updated" {
		t.Errorf("Expected updated but found %v", updated.Name)
	}
}

func TestDeleteRole(t *testing.T) {
	transaction := testDB.Begin()
	defer transaction.Rollback()

	role := &entity.Role{Name: "Tester"}
	roleRepo := repository.NewRoleRepository(transaction)
	createdRole, err := roleRepo.CreateRole(role)

	if err != nil {
		t.Fatalf("Failed to create a role %v", err)
	}

	err = roleRepo.DeleteRole(createdRole.ID)
	if err != nil {
		t.Fatal("Failed to delete it")
	}
	var deletedRole entity.Role
	if err = transaction.Find(&deletedRole, createdRole.ID).Error; err != nil {
		t.Error("Successfully deleted")
	}
}

