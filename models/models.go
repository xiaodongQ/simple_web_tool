package models

type TableA struct {
	Bid       string `gorm:"primary_key"`
	Bname     string
	User      string
	Partition string
}

type TableADetail struct {
	Fid    string `gorm:"primary_key"`
	Fname  string
	Bid    string
	Fsize  int64
}

type User struct {
	User       string `gorm:"primary_key"`
	Name       string
	Attributes string
}