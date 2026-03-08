package interfaces


type PatientRepository interface {
	BaseRepository[Patient]
	FindByNationalID(ctx context.Context, nationalID string) (*Patient, error)
	FindByPhoneNumber(ctx context.Context, phoneNumber string) (*Patient, error)
	FindByName(ctx context.Context, name string) ([]Patient, error)
}