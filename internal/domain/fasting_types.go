package domain

import "fmt"

type FastingType struct {
	ID          int
	Name        string
	Description string
	FastHours   int
}

var FastingTypes = []FastingType{
	{ID: 1, Name: "IF 12:12", Description: "Puasa 12 jam, makan 12 jam", FastHours: 12},
	{ID: 2, Name: "IF 14:10", Description: "Puasa 14 jam, makan 10 jam", FastHours: 14},
	{ID: 3, Name: "IF 16:8", Description: "Puasa 16 jam, makan 8 jam", FastHours: 16},
	{ID: 4, Name: "IF 18:6", Description: "Puasa 18 jam, makan 6 jam", FastHours: 18},
	{ID: 5, Name: "IF 20:4", Description: "Puasa 20 jam, makan 4 jam", FastHours: 20},
	{ID: 6, Name: "OMAD-1", Description: "One Meal A Day - Puasa 22 jam", FastHours: 22},
	{ID: 7, Name: "OMAD-2", Description: "One Meal A Day - Puasa 23 jam", FastHours: 23},
	{ID: 8, Name: "Water Fasting", Description: "Puasa air - 24, 36, 48, atau 72 jam", FastHours: 0},
	{ID: 9, Name: "Water Fasting (Bebas)", Description: "Puasa air bebas (minimal 24 jam)", FastHours: 0},
	{ID: 10, Name: "Dry Fasting", Description: "Puasa kering - bebas tentukan durasi", FastHours: 0},
}

func GetFastingTypeByID(id int) (*FastingType, error) {
	for _, ft := range FastingTypes {
		if ft.ID == id {
			return &ft, nil
		}
	}
	return nil, fmt.Errorf("jenis puasa tidak ditemukan")
}

func GetFastingTypesList() string {
	result := "📋 *Daftar Jenis Puasa*\n\n"
	for _, ft := range FastingTypes {
		if ft.ID <= 7 {
			result += fmt.Sprintf("*%d. %s*\n   %s\n\n", ft.ID, ft.Name, ft.Description)
		} else if ft.ID == 8 {
			result += fmt.Sprintf("*%d. %s*\n   %s\n   _Pilih: 24, 36, 48, atau 72 jam_\n\n", ft.ID, ft.Name, ft.Description)
		} else if ft.ID == 9 {
			result += fmt.Sprintf("*%d. %s*\n   %s\n   _Tentukan durasi sendiri (minimal 24 jam)_\n\n", ft.ID, ft.Name, ft.Description)
		} else {
			result += fmt.Sprintf("*%d. %s*\n   %s\n   _Tentukan durasi sendiri_\n\n", ft.ID, ft.Name, ft.Description)
		}
	}
	result += "*Cara menggunakan:*\n"
	result += "`/set-puasa <nomor> <jam_mulai>`\n"
	result += "Contoh: `/set-puasa 3 05:00`\n\n"
	result += "Untuk Water/Dry Fasting:\n"
	result += "`/set-puasa 8 05:00 48` (Water Fasting 48 jam)\n"
	result += "`/set-puasa 10 05:00 18` (Dry Fasting 18 jam)"
	return result
}
