package repositorytest

import (
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
	"testing"

)


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

