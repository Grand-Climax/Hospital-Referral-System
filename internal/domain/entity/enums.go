package entity

type EthiopianRegion string

const (
	RegionAddisAbaba        EthiopianRegion = "Addis Ababa"
	RegionAfar              EthiopianRegion = "Afar"
	RegionAmhara            EthiopianRegion = "Amhara"
	RegionBenishangulGumuz  EthiopianRegion = "Benishangul-Gumuz"
	RegionDireDawa          EthiopianRegion = "Dire Dawa"
	RegionGambela           EthiopianRegion = "Gambela"
	RegionHarari            EthiopianRegion = "Harari"
	RegionOromia            EthiopianRegion = "Oromia"
	RegionSidama            EthiopianRegion = "Sidama"
	RegionSomali            EthiopianRegion = "Somali"
	RegionSouthEthiopia     EthiopianRegion = "South Ethiopia"
	RegionSouthWestEthiopia EthiopianRegion = "South West Ethiopia"
	RegionCentralEthiopia   EthiopianRegion = "Central Ethiopia"
	RegionTigray            EthiopianRegion = "Tigray"
)

func (r EthiopianRegion) IsValid() bool {
	switch r {
	case RegionAddisAbaba, RegionAfar, RegionAmhara, RegionBenishangulGumuz,
		RegionDireDawa, RegionGambela, RegionHarari, RegionOromia,
		RegionSidama, RegionSomali, RegionSouthEthiopia,
		RegionSouthWestEthiopia, RegionCentralEthiopia, RegionTigray:
		return true
	}
	return false
}
