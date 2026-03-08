package seeder

import "Hospital-Referral-System/internal/domain/entity"

func ptrStr(s string) *string {
	return &s
}

var initialDepartments = []entity.Department{
	{Name: "Internal Medicine", Description: ptrStr("General internal medicine")},
	{Name: "General Surgery", Description: ptrStr("General surgical procedures")},
	{Name: "Pediatrics", Description: ptrStr("Care of infants, children, and adolescents")},
	{Name: "Obstetrics and Gynecology", Description: ptrStr("Pregnancy, childbirth, and female reproductive system")},
	{Name: "Cardiology", Description: ptrStr("Heart and blood vessel disorders")},
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
	NationalID string
	Email      string
	FirstName  string
	LastName   string
	Role       entity.UserRole
	DeptName   string // For mapping to a seeded department if required
}

// Primary Hospital template users
var primaryHospUsers = []hospitalUserTemplate{
	{
		NationalID: "DOC-PRI-001",
		Email:      "doc.primary@hospital.et",
		FirstName:  "Primary",
		LastName:   "Doctor",
		Role:       entity.RoleReferringDoctor,
	},
	{
		NationalID: "RECEPT-PRI-001",
		Email:      "reception.primary@hospital.et",
		FirstName:  "Primary",
		LastName:   "Receptionist",
		Role:       entity.RoleReceptionist,
	},
}

// Specialized Hospital template users
var specializedHospUsers = []hospitalUserTemplate{
	{
		NationalID: "SPEC-001",
		Email:      "specialist.cardio@hospital.et",
		FirstName:  "Cardio",
		LastName:   "Specialist",
		Role:       entity.RoleReceivingSpecialist,
	},
	{
		NationalID: "HOSPADMIN-001",
		Email:      "admin.specialized@hospital.et",
		FirstName:  "Specialized",
		LastName:   "Hospital Admin",
		Role:       entity.RoleHospitalAdmin,
	},
	{
		NationalID: "HEAD-001",
		Email:      "head.cardio@hospital.et",
		FirstName:  "Cardio",
		LastName:   "Dept Head",
		Role:       entity.RoleDeptHead,
		DeptName:   "Cardiology",
	},
	{
		NationalID: "RECEPT-SPEC-001",
		Email:      "reception.spec@hospital.et",
		FirstName:  "Specialized",
		LastName:   "Receptionist",
		Role:       entity.RoleReceptionist,
	},
}
