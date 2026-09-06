package usecase

import (
	"errors"
	"fmt"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

func (u *fastingUsecase) RegisterUser(phone, jid, name string) (string, error) {
	if name == "" {
		return "❌ Nama harus diisi. Gunakan: /daftar <nama>\nContoh: /daftar kyomel", nil
	}

	existingUser, err := u.userRepo.FindByPhone(phone)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if existingUser != nil && existingUser.ID != "" {
		registeredName := existingUser.Name
		if registeredName == "" {
			registeredName = existingUser.Phone
		}
		return fmt.Sprintf("✅ Akun sudah terdaftar!\nID: %s\nNama: %s\nNomor: %s\n\nGunakan /setname <nama> untuk mengubah nama.", existingUser.ID, registeredName, existingUser.Phone), nil
	}

	user := &domain.User{
		Phone: phone,
		Name:  name,
		JID:   jid,
	}
	if err := u.userRepo.Create(user); err != nil {
		return "", fmt.Errorf("gagal mendaftar: %w", err)
	}

	return fmt.Sprintf("🎉 *Selamat datang, %s!*\n"+
		"ID: %s\nNomor: %s\n\n"+
		"Bot ini bakal nemenin kamu tracking puasa — IF, OMAD, Water/Dry/Prolonged Fasting — plus ngirim notifikasi otomatis pas mulai & waktunya buka.\n\n"+
		"🚀 *Mulai dalam 30 detik:*\n"+
		"1️⃣ /panduan — baca ringkas jenis puasa & cara pakai bot\n"+
		"2️⃣ /puasa 14 atau /puasa 16 — mulai dari sekarang\n"+
		"3️⃣ /puasa 16 05:00 — mulai jam 5, durasi 16 jam\n"+
		"4️⃣ /puasa 16 23-05-2026 05:00 — jadwalkan ke tanggal tertentu\n\n"+
		"Saran buat pemula: mulai dari IF 14:10 atau IF 16:8. Body adaptation lebih halus. 💪", name, user.ID, phone), nil
}

func (u *fastingUsecase) SetName(phone, name string) (string, error) {
	if name == "" {
		return "❌ Nama harus diisi. Gunakan: /setname <nama baru>\nContoh: /setname kyomel baru", nil
	}

	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return msgNotRegistered, nil
		}
		return "", fmt.Errorf(errCheckDataFormat, err)
	}

	if err := u.userRepo.UpdateName(user.ID, name); err != nil {
		return "", fmt.Errorf("gagal mengubah nama: %w", err)
	}

	return fmt.Sprintf("✅ Nama berhasil diubah menjadi: %s", name), nil
}

func (u *fastingUsecase) lookupUser(phone string) (*domain.User, error) {
	user, err := u.userRepo.FindByPhone(phone)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf(errCheckDataFormat, err)
	}
	return user, nil
}

func displayUserName(user *domain.User) string {
	if user.Name != "" {
		return user.Name
	}
	return user.Phone
}
