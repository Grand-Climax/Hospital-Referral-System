package repositorytest

import (
	"Hospital-Referral-System/internal/domain/entity"
	repositoryinterface "Hospital-Referral-System/internal/domain/contract/repository_interface"
	customerrors "Hospital-Referral-System/internal/errors"
	"Hospital-Referral-System/internal/repository"
	"errors"
	"strings"
	"testing"
)


func withTransaction(t *testing.T, testFunc func(repo repositoryinterface.DepartmentRepositoryInterface)) {
	t.Helper()
	tx := testDB.Begin()
	defer tx.Rollback()
	repo := repository.NewDepartmentRepository(tx)
	testFunc(repo)
}

func TestDepartmentRepository(t *testing.T) {
	t.Run("CreateDepartment", func(t *testing.T) {
		withTransaction(t, func(repo repositoryinterface.DepartmentRepositoryInterface) {
			dept := &entity.Department{Name: "Cashier"}
			dept.Name = strings.ToLower(dept.Name)

			created, err := repo.CreateDepartment(dept)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if created.ID == 0 {
				t.Fatal("Expected department ID to be set")
			}
			if created.Name != "cashier" {
				t.Errorf("Expected name 'cashier', got '%s'", created.Name)
			}
		})
	})

	t.Run("GetDepartments", func(t *testing.T) {
		withTransaction(t, func(repo repositoryinterface.DepartmentRepositoryInterface) {
			_, _ = repo.CreateDepartment(&entity.Department{Name: "finance"})
			departments, err := repo.GetDepartments()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if len(departments) == 0 {
				t.Fatal("Expected non-empty list of departments")
			}
		})
	})

	t.Run("GetDepartmentByID", func(t *testing.T) {
		withTransaction(t, func(repo repositoryinterface.DepartmentRepositoryInterface) {
			created, _ := repo.CreateDepartment(&entity.Department{Name: "admin"})
			found, err := repo.GetDepartmentByID(created.ID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if found.Name != "admin" {
				t.Errorf("Expected 'admin', got '%s'", found.Name)
			}

			_, err = repo.GetDepartmentByID(99999) // likely non-existent
			if !errors.Is(err, customerrors.ErrNotFound) {
				t.Errorf("Expected ErrNotFound for non-existent ID, got %v", err)
			}
		})
	})

	t.Run("GetDepartmentByName", func(t *testing.T) {
		withTransaction(t, func(repo repositoryinterface.DepartmentRepositoryInterface) {
			name := "marketing"
			_, _ = repo.CreateDepartment(&entity.Department{Name: name})

			found, err := repo.GetDepartmentByName(name)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if found.Name != name {
				t.Errorf("Expected '%s', got '%s'", name, found.Name)
			}

			_, err = repo.GetDepartmentByName("nonexistent")
			if !errors.Is(err, customerrors.ErrNotFound) {
				t.Errorf("Expected ErrNotFound for unknown name, got %v", err)
			}
		})
	})

	t.Run("UpdateDepartment", func(t *testing.T) {
		withTransaction(t, func(repo repositoryinterface.DepartmentRepositoryInterface) {
			created, _ := repo.CreateDepartment(&entity.Department{Name: "sales"})
			created.Name = "revenue"
			updated, err := repo.UpdateDepartment(created)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if updated.Name != "revenue" {
				t.Errorf("Expected 'revenue', got '%s'", updated.Name)
			}
		})
	})

	t.Run("DeleteDepartment", func(t *testing.T) {
		withTransaction(t, func(repo repositoryinterface.DepartmentRepositoryInterface) {
			created, _ := repo.CreateDepartment(&entity.Department{Name: "legal"})
			if err := repo.DeleteDepartment(created.ID); err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Try to delete again
			err := repo.DeleteDepartment(created.ID)
			if err != nil {
				t.Errorf("Expected no error on deleting nonexistent record (idempotent), got: %v", err)
			}
		})
	})
}
