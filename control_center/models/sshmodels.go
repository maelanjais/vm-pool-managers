package models

type Student struct {
	ID     uint `gorm:"primaryKey;autoIncrement"`
	ListId uint `gorm:"index"`
	Name   string
	SshKey string
}

type ListStudents struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	PoolId   uint      `gorm:"uniqueIndex"`
	Students []Student `gorm:"foreignKey:ListId;constraint:OnDelete:CASCADE"`
}
