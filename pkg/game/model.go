package game

type Game struct {
	ID     uint    `json:"id" gorm:"primaryKey"`
	Name   string  `json:"name" gorm:"not null"`
	Owners []Owner `json:"owners" gorm:"many2many:game_owners;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

type Owner struct {
	ID  uint   `json:"id" gorm:"primaryKey"`
	Sub string `json:"sub" gorm:"not null,unique"`
}
