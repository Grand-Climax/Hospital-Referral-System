package seeder

import (
	"github.com/google/uuid"

	"Hospital-Referral-System/internal/domain/entity"
)

func ptrStr(s string) *string {
	return &s
}

var initialDepartments = []entity.Department{
	{Name: "Internal Medicine", Description: ptrStr("General internal medicine")},
	{Name: "General Surgery", Description: ptrStr("General surgical procedures")},
	{Name: "Pediatrics", Description: ptrStr("Care of infants, children, and adolescents")},
	{Name: "Obstetrics and Gynecology", Description: ptrStr("Pregnancy, childbirth, and female reproductive system")},
	{
		ID:          uuid.MustParse("dfc2b777-a5d5-424b-911a-976b2e8d8614"),
		Name:        "Cardiology", 
		Description: ptrStr("Heart and blood vessel disorders"),
	},
	{Name: "Neurology", Description: ptrStr("Disorders of the nervous system")},
	{Name: "Oncology", Description: ptrStr("Diagnosis and treatment of cancer")},
	{Name: "Orthopedics", Description: ptrStr("Conditions involving the musculoskeletal system")},
	{Name: "Psychiatry", Description: ptrStr("Mental health disorders")},
	{Name: "Emergency Medicine", Description: ptrStr("Acute illnesses or injuries requiring immediate care")},
	{Name: "Dermatology", Description: ptrStr("Skin, hair, and nail conditions")},
	{Name: "Ophthalmology", Description: ptrStr("Eye and vision care")},
}

var initialHospitals = []entity.Hospital{
	// Tertiary/Specialized level (Federal/University Hospitals)
	{
		Name:         "Tikur Anbessa Specialized Hospital",
		TierLevel:    entity.TertiaryHosp,
		Region:       "Addis Ababa",
		Address:      ptrStr("Zambia St, Addis Ababa, Ethiopia"),
		ContactPhone: ptrStr("+251 11 551 1211"),
	},
	{
		Name:         "St. Paul's Hospital Millennium Medical College",
		TierLevel:    entity.TertiaryHosp,
		Region:       "Addis Ababa",
		Address:      ptrStr("Swaziland St, Addis Ababa, Ethiopia"),
		ContactPhone: ptrStr("+251 11 275 0122"),
	},
	{
		Name:         "Jimma University Medical Center",
		TierLevel:    entity.SpecializedHosp,
		Region:       "Oromia",
		Address:      ptrStr("Jimma, Oromia, Ethiopia"),
		ContactPhone: ptrStr("+251 47 111 1458"),
	},
	{
		Name:         "Ayder Comprehensive Specialized Hospital",
		TierLevel:    entity.SpecializedHosp,
		Region:       "Tigray",
		Address:      ptrStr("Mekelle, Tigray, Ethiopia"),
		ContactPhone: ptrStr("+251 34 441 6662"),
	},
	{
		Name:         "Hawassa University Comprehensive Specialized Hospital",
		TierLevel:    entity.SpecializedHosp,
		Region:       "Sidama",
		Address:      ptrStr("Hawassa, Sidama, Ethiopia"),
		ContactPhone: ptrStr("+251 46 220 5311"),
	},
	{
		Name:         "Felege Hiwot Comprehensive Specialized Hospital",
		TierLevel:    entity.SpecializedHosp,
		Region:       "Amhara",
		Address:      ptrStr("Bahir Dar, Amhara, Ethiopia"),
		ContactPhone: ptrStr("+251 58 220 1666"),
	},
	
	// General Hospitals (Regional)
	{
		ID:           uuid.MustParse("0f74f069-d52d-4482-9ba5-41b007fdc1e5"),
		Name:         "Adama General Hospital",
		TierLevel:    entity.GeneralHosp,
		Region:       "Oromia",
		Address:      ptrStr("Adama, Oromia, Ethiopia"),
		ContactPhone: ptrStr("+251 22 112 1111"), // Mock
	},
	{
		Name:         "Dessie Referral Hospital",
		TierLevel:    entity.GeneralHosp,
		Region:       "Amhara",
		Address:      ptrStr("Dessie, Amhara, Ethiopia"),
		ContactPhone: ptrStr("+251 33 111 1111"), // Mock
	},
	{
		Name:         "Dil Chora General Hospital",
		TierLevel:    entity.GeneralHosp,
		Region:       "Dire Dawa",
		Address:      ptrStr("Dire Dawa, Ethiopia"),
		ContactPhone: ptrStr("+251 25 111 2222"), // Mock
	},
	
	// Primary Hospitals (District level)
	{
		ID:           uuid.MustParse("cd323204-bfb7-4583-88e9-bb5cbed68af0"),
		Name:         "Bishoftu Primary Hospital",
		TierLevel:    entity.PrimaryHosp,
		Region:       "Oromia",
		Address:      ptrStr("Bishoftu, Oromia, Ethiopia"),
		ContactPhone: ptrStr("+251 44 111 0000"), // Mock
	},
	{
		Name:         "Debre Berhan Primary Hospital",
		TierLevel:    entity.PrimaryHosp,
		Region:       "Amhara",
		Address:      ptrStr("Debre Berhan, Amhara, Ethiopia"),
		ContactPhone: ptrStr("+251 33 222 0000"), // Mock
	},
}

// System-wide users independent of hospitals
var systemTestUsers = []entity.User{
	{
		NationalID: "SUPERADMIN-001",
		Email:      "superadmin@moh.gov.et",
		FirstName:  "System",
		LastName:   "Super Administrator",
		Role:       entity.RoleSystemSuperAdmin,
	},
	{
		NationalID: "MOH-001",
		Email:      "analyst@moh.gov.et",
		FirstName:  "MoH",
		LastName:   "Analyst",
		Role:       entity.RoleMohAnalyst,
	},
	// Liaison Officer handles regional coordination, optionally attached to hopital, but let's test a global one
	{
		NationalID: "LIAISON-001",
		Email:      "liaison@moh.gov.et",
		FirstName:  "Regional",
		LastName:   "Liaison",
		Role:       entity.RoleLiaisonOfficer,
	},
}

// Helper structs for building users dynamically mapped to inserted hospitals
type hospitalUserTemplate struct {
	NationalID   string
	Email        string
	FirstName    string
	LastName     string
	Role         entity.UserRole
	DepartmentID *uuid.UUID // nil for roles that don't need a department (Admin, Liaison, Receptionist)
}

var (
	cardiologyID = uuid.MustParse("dfc2b777-a5d5-424b-911a-976b2e8d8614")
)

// Primary Hospital template users (Bishoftu Primary Hospital = cd323204-bfb7-4583-88e9-bb5cbed68af0)
var primaryHospUsers = []hospitalUserTemplate{
	{
		NationalID:   "DOC-PRI-001",
		Email:        "doc.primary@hospital.et",
		FirstName:    "Primary",
		LastName:     "Doctor",
		Role:         entity.RoleReferringDoctor,
		DepartmentID: &cardiologyID, // Doctors are linked to their clinical department
	},
	{
		NationalID: "LIAISON-PRI-001",
		Email:      "liaison.primary@hospital.et",
		FirstName:  "Primary",
		LastName:   "Liaison Officer",
		Role:       entity.RoleLiaisonOfficer,
		// No department
	},
	{
		NationalID: "RECEPT-PRI-001",
		Email:      "reception.primary@hospital.et",
		FirstName:  "Primary",
		LastName:   "Receptionist",
		Role:       entity.RoleReceptionist,
		// No department
	},
}

// Specialized Hospital template users (Jimma University Medical Center)
var specializedHospUsers = []hospitalUserTemplate{
	{
		NationalID:   "SPEC-001",
		Email:        "specialist.cardio@hospital.et",
		FirstName:    "Cardio",
		LastName:     "Specialist",
		Role:         entity.RoleReceivingSpecialist,
		DepartmentID: &cardiologyID,
	},
	{
		NationalID: "HOSPADMIN-001",
		Email:      "admin.specialized@hospital.et",
		FirstName:  "Specialized",
		LastName:   "Hospital Admin",
		Role:       entity.RoleHospitalAdmin,
	},
	{
		NationalID:   "HEAD-001",
		Email:        "head.cardio@hospital.et",
		FirstName:    "Cardio",
		LastName:     "Dept Head",
		Role:         entity.RoleDeptHead,
		DepartmentID: &cardiologyID,
	},
	{
		NationalID: "LIAISON-SPEC-001",
		Email:      "liaison.specialized@hospital.et",
		FirstName:  "Specialized",
		LastName:   "Liaison Officer",
		Role:       entity.RoleLiaisonOfficer,
	},
	{
		NationalID: "RECEPT-SPEC-001",
		Email:      "reception.spec@hospital.et",
		FirstName:  "Specialized",
		LastName:   "Receptionist",
		Role:       entity.RoleReceptionist,
	},
}

// General Hospital template users (Adama General Hospital = 0f74f069-d52d-4482-9ba5-41b007fdc1e5)
var generalHospUsers = []hospitalUserTemplate{
	{
		NationalID:   "SPEC-GENERAL-001",
		Email:        "specialist.general@hospital.et",
		FirstName:    "General",
		LastName:     "Specialist",
		Role:         entity.RoleReceivingSpecialist,
		DepartmentID: &cardiologyID,
	},
	{
		NationalID: "LIAISON-GEN-001",
		Email:      "liaison.general@hospital.et",
		FirstName:  "General",
		LastName:   "Liaison Officer",
		Role:       entity.RoleLiaisonOfficer,
	},
	{
		NationalID: "HOSPADMIN-GEN-001",
		Email:      "admin.general@hospital.et",
		FirstName:  "General",
		LastName:   "Hospital Admin",
		Role:       entity.RoleHospitalAdmin,
	},
}
