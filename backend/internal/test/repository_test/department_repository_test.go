package repositorytest

import (
	"Hospital-Referral-System/internal/domain/entity"
	"Hospital-Referral-System/internal/repository"
	"testing"
)

func TestCreateDepartment(t *testing.T) {
	transaction := testDB.Begin()
	defer transaction.Rollback()

	//create a mock data
	input := &entity.Department{Name: "Cashier"}
	repo := repository.NewDepartmentRepository(transaction)

	created, err := repo.CreateDepartment(input)
	if err != nil {
		t.Fatalf("Failed to insert into database %v", err)
		return
	}

	if created.ID == 0{
		t.Fatal("Expected to set the ID")
		return
	}

	if created.Name != "Cashier"{
		t.Fatal("Wrong name setted")
	}
}

func TestGetDepartments(t *testing.T) {
	transaction := testDB.Begin()
	defer transaction.Rollback()

	repo := repository.NewDepartmentRepository(transaction)

	input := &entity.Department{Name : "Cashier"}
	_, err := repo.CreateDepartment(input)

	if err != nil {
		t.Fatalf("Couldn't insert department %v", err)
	}
	departments, err := repo.GetDepartments()

	if err != nil{
		t.Fatalf("couldn't fetch departments %v", err)
	}

	if len(departments) == 0{
		t.Error("Expected one got zero")
	}

	found := false
	for _, department := range departments {
		if department.Name == "Cashier"{
			found = true
			break
		}
	}
	if !found{
		t.Error("Expected to find a department with name Cashier but couldn't find any")
	}
}

func TestUpdateDepartment(t *testing.T) {
	transaction := testDB.Begin()
	defer transaction.Rollback()

	repo := repository.NewDepartmentRepository(transaction)
	input := &entity.Department{Name: "Cashier"}

	created, err := repo.CreateDepartment(input)
	if err != nil {
		t.Fatalf("Couldn't insert into database %v", err)
	}

	updated := &entity.Department{ID: created.ID, Name: "Doctor"}

	updated_department, err := repo.UpdateDepartment(updated)

	if err != nil {
		t.Fatalf("Couldn't update department %v", err)
	}

	if updated_department.Name != "Doctor"{
		t.Errorf("Expected a name doctor found %v", updated_department.Name)
	}
}

func TestDeleteDepartment(t *testing.T) {
	transaction := testDB.Begin()
	defer transaction.Rollback()

	var err error

	repo := repository.NewDepartmentRepository(transaction)
	input := &entity.Department{Name:"Doctor"}

	created, err := repo.CreateDepartment(input)
	if err != nil {
		t.Fatalf("Failed to create a department %v", err)
	}

	err = repo.DeleteDepartment(created.ID)
	if err != nil {
		t.Fatalf("Failed to delete %v", err)
	}

}