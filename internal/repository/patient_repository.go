package repository

type patientRepository struct {
	interfaces.PatientRepository
	db *gorm.DB
}

func NewPatientRepository(db *gorm.DB) interfaces.PatientRepository {
	return &patientRepository{
		PatientRepository: NewBaseRepository[entity.Patient](db),
		db:                db,
	}
}

func hashNationalID(nationalID string) string {
	hash := sha256.Sum256([]byte(nationalID))
	return hex.EncodeToString(hash[:])
}

func (r *patientRepository) FindByNationalID(ctx context.Context, nationalID string) (*entity.Patient, error) {
	var patient entity.Patient
	hashedID := hashNationalID(nationalID)
	if err := r.db.WithContext(ctx).Where("national_id_hash = ?", hashedID).First(&patient).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("patient not found")
		}
		return nil, err
	}
	return &patient, nil
}