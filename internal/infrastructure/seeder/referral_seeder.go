package seeder

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"Hospital-Referral-System/internal/domain/entity"
)

func seedReferrals(ctx context.Context, db *gorm.DB) error {
	log.Println("Seeding mass multi-stage referrals...")

	// 1. Fetch dependencies needed to build the referral relations
	var patients []entity.Patient
	db.WithContext(ctx).Find(&patients)
	if len(patients) == 0 {
		return fmt.Errorf("no patients found to assign referrals to")
	}

	var doctors []entity.User
	db.WithContext(ctx).Where("role = ?", entity.RoleReferringDoctor).Find(&doctors)
	if len(doctors) == 0 {
		return fmt.Errorf("no referring doctors found")
	}

	var specialists []entity.User
	db.WithContext(ctx).Where("role = ?", entity.RoleReceivingSpecialist).Find(&specialists)

	var liaisonOfficers []entity.User
	db.WithContext(ctx).Where("role = ?", entity.RoleLiaisonOfficer).Find(&liaisonOfficers)
	
	// Get target cardiology department for routing mapping
	var cardiologyDept entity.Department
	db.WithContext(ctx).Where("name = ?", "Cardiology").First(&cardiologyDept)
	if cardiologyDept.ID == uuid.Nil {
		return fmt.Errorf("cardiology department not found")
	}

	var targetHosp entity.Hospital
	db.WithContext(ctx).Where("tier_level = ?", entity.SpecializedHosp).First(&targetHosp)
	if targetHosp.ID == uuid.Nil {
		return fmt.Errorf("no specialized hospital found for target routing")
	}

	var icdCodes []entity.ICDCode
	db.WithContext(ctx).Limit(5).Find(&icdCodes)

	// Build exactly 3 distinct referrals per patient (Draft, Submitted, Accepted/Received)
	for i, patient := range patients {
		doc := doctors[i%len(doctors)]
		if doc.HospitalID == nil {
			continue // Skip safely if doctor lacks sender facility
		}
		senderHospID := *doc.HospitalID
		
		var liaisonID *uuid.UUID
		if len(liaisonOfficers) > 0 {
			l := liaisonOfficers[0].ID
			liaisonID = &l
		}

		// Referral 1: DRAFT
		if err := generateFullReferralPackage(ctx, db, patient.ID, doc.ID, senderHospID, targetHosp.ID, cardiologyDept.ID, liaisonID, entity.StatusDraft, icdCodes); err != nil {
			log.Printf("Failed to generate DRAFT referral: %v", err)
		}

		// Referral 2: SUBMITTED (Pending Liaison Review)
		if err := generateFullReferralPackage(ctx, db, patient.ID, doc.ID, senderHospID, targetHosp.ID, cardiologyDept.ID, liaisonID, entity.StatusSubmitted, icdCodes); err != nil {
			log.Printf("Failed to generate SUBMITTED referral: %v", err)
		}

		// Referral 3: FORWARDED (Ready for Specialist Review Queue)
		if err := generateFullReferralPackage(ctx, db, patient.ID, doc.ID, senderHospID, targetHosp.ID, cardiologyDept.ID, liaisonID, entity.StatusForwarded, icdCodes); err != nil {
			log.Printf("Failed to generate FORWARDED referral: %v", err)
		}
	}

	log.Println("Successfully seeded normalized referral structures!")
	return nil
}

func generateFullReferralPackage(
	ctx context.Context, 
	db *gorm.DB, 
	patientID, doctorID, senderID, targetID, deptID uuid.UUID, 
	liaisonID *uuid.UUID, 
	status entity.ReferralStatus,
	icdCodes []entity.ICDCode,
) error {

	// Start Tx
	tx := db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Create Core Referral Entity
	ref := entity.Referral{
		PatientID:         patientID,
		ReferringDoctorID: doctorID,
		SenderHospitalID:  senderID,
		TargetHospitalID:  targetID,
		LiaisonOfficerID:  liaisonID,
		TargetDeptID:      deptID,
		Status:            status,
	}
	if err := tx.Create(&ref).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 2. Create Required Child form Entity (Normalized)
	mockSummary := "Patient presented with chronic chest pain radiating to left arm. Refractory to primary analgesics."
	mockHistory := "Previous history of mild hypertension. Non-smoker."
	physExam := "BP 160/90, HR 105, regular rhythm."
	reason := "Urgent cardiological evaluation for unstable angina."
	
	form := entity.ReferralForm{
		ReferralID:                  ref.ID,
		ClinicalSummary:             mockSummary,
		PatientHistory:              mockHistory,
		PhysicalExaminationFindings: &physExam,
		ReasonOfReferral:            reason,
		ReasonForReferralCategory:   "Evaluation and Management",
		ConditionAtReferral:         "Stable but critical",
	}
	if err := tx.Create(&form).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 3. Bind Vitals mapping
	var sys, dia, hr, rr int16 = 160, 90, 105, 18
	var temp, spo2 float32 = 37.2, 96.0

	vital := entity.Vital{
		ReferralID:      ref.ID,
		SystolicBP:      &sys,
		DiastolicBP:     &dia,
		HeartRate:       &hr,
		Temperature:     &temp,
		RespiratoryRate: &rr,
		SpO2:            &spo2,
	}
	if err := tx.Create(&vital).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 4. Bind ReferralDiagnoses (ICD codes mapping)
	if len(icdCodes) > 0 {
		diag := entity.ReferralDiagnosis{
			ReferralID:         ref.ID,
			ICDCode:            icdCodes[0].Code,
			IsPrimary:          true,
			DiagnosisCertainty: entity.CertaintySuspected,
		}
		if err := tx.Create(&diag).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// 5. Hydrate Historical Status Trajectory (History Log)
	// Base DRAFT state
	hist1 := entity.ReferralStatusHistory{
		ReferralID:  ref.ID,
		ChangedByID: doctorID,
		ToStatus:    entity.StatusDraft,
	}
	tx.Create(&hist1)

	if status == entity.StatusSubmitted || status == entity.StatusForwarded {
		draft := entity.StatusDraft
		hist2 := entity.ReferralStatusHistory{
			ReferralID:  ref.ID,
			ChangedByID: doctorID,
			FromStatus:  &draft,
			ToStatus:    entity.StatusSubmitted,
		}
		tx.Create(&hist2)
	}

	if status == entity.StatusForwarded {
		sub := entity.StatusSubmitted
		// Liaison approves it and forwards to target hospital
		officer := doctorID // Default fallback if no liaison
		if liaisonID != nil {
			officer = *liaisonID
		}
		hist3 := entity.ReferralStatusHistory{
			ReferralID:  ref.ID,
			ChangedByID: officer,
			FromStatus:  &sub,
			ToStatus:    entity.StatusForwarded,
		}
		tx.Create(&hist3)
	}

	return tx.Commit().Error
}
