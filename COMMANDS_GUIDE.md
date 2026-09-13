# 📖 Panduan Lengkap Perintah & Fitur WhatsApp AI Assistant

Bot ini dirancang agar Anda bisa berinteraksi dengan dua cara:
1. **Bahasa Natural / Bebas** (Ketik santai atau kirim **Voice Note**) 🎙️
2. **Perintah Slash (Command)** (Lebih presisi dan cepat) ⚡

---

## 💰 1. Pencatatan Keuangan

### A. Menggunakan Teks Santai / Voice Note (Otomatis Dikenali AI)
Cukup kirim pesan teks biasa atau kirim **Voice Note (VN)** di WhatsApp:
- `"Beli nasi padang 25rb"` *(Otomatis masuk Pengeluaran - Kategori: Makanan)*
- `"Bensin motor 35k"` *(Otomatis masuk Pengeluaran - Kategori: Transportasi)*
- `"Gaji bulan ini cair 7.5jt"` *(Otomatis masuk Pemasukan - Kategori: Gaji)*
- `"Terima transfer freelance 1.200.000"` *(Otomatis masuk Pemasukan - Kategori: Freelance)*
- `"Tadi ngopi sama temen 45rb"` *(Otomatis masuk Pengeluaran)*

> **Format Nominal yang Didukung:**
> - `10k` = Rp 10.000
> - `50rb` = Rp 50.000
> - `1.5jt` = Rp 1.500.000
> - `25.000` / `25000` = Rp 25.000

---

### B. Edit & Hapus Transaksi
Setiap transaksi yang berhasil dicatat akan memiliki **ID Unik** (contoh: `20260313-001`).

| Perintah | Contoh | Penjelasan |
| :--- | :--- | :--- |
| `/edit [ID] [field] [nilai]` | `/edit 20260313-001 jumlah 30000` | Mengubah nominal transaksi |
| | `/edit 20260313-001 kategori Belanja` | Mengubah kategori transaksi |
| | `/edit 20260313-001 deskripsi Kopi Susu` | Mengubah catatan deskripsi |
| `/hapus [ID]` | `/hapus 20260313-001` | Menghapus baris transaksi dari Google Sheet |

---

## 📊 2. Laporan Finansial & Grafik

| Perintah | Variasi / Teks Alami | Keterangan & Periode |
| :--- | :--- | :--- |
| `/laporan hari ini` | *"minta laporan hari ini"* | Rekap transaksi & pengeluaran hari ini |
| `/laporan minggu ini` | *"laporan mingguan"* | Rekap 7 hari terakhir |
| `/laporan bulan ini` | *"laporan bulanan"* | Rekap bulan berjalan (contoh: Maret 2026) |
| `/laporan gajian` | *"laporan siklus gajian"* | **Siklus Gajian:** Tanggal 23 bulan lalu s.d. 23 bulan ini (atau s.d. hari ini jika belum tgl 23) |
| `/laporan total` | `/laporan semua`, *"laporan keseluruhan"* | **All-Time:** Total pemasukan, pengeluaran, & sisa saldo dari awal pertama kali input sampai sekarang |

---

## 🧠 3. Evaluasi & Saran Finansial AI

Minta AI mengevaluasi pola belanja Anda dan memberikan tips hemat:

| Perintah | Teks Alami | Fungsi |
| :--- | :--- | :--- |
| `/evaluasi` | *"evaluasi keuanganku dong"* | AI menganalisis pos pengeluaran terbesar, kebocoran dana, dan memberi **3 saran konkret** |
| `/saran` | *"menurutmu bulan ini aku boros gak?"* | Konsultasi finansial interaktif berdasarkan data transaksi Anda |

---

## 🎯 4. Budgeting & Batas Pengeluaran

Atur batas maksimal pengeluaran bulanan per kategori. Bot akan otomatis memberi peringatan jika pengeluaran sudah mencapai 80% atau melebihi budget.

| Perintah | Contoh | Penjelasan |
| :--- | :--- | :--- |
| `/budget [kategori] [jumlah]` | `/budget Makanan 1500000` | Mengatur batas pengeluaran kategori Makanan Rp 1.500.000/bulan |
| | `/budget Hiburan 500000` | Mengatur batas kategori Hiburan |

---

## ⏰ 5. Pengingat Terjadwal & To-Do List (*Reminders*)

### A. Membuat Pengingat
Bisa lewat perintah atau cukup chat santai:
- `/reminder tanggal 26 maret bayar cicilan laptop`
- `/reminder besok jam 10 pagi ada zoom meeting`
- *"ingetin ntar malem jam 8 bayar wifi"*
- *"jangan lupa tgl 25 kirim uang ke ortu"*

> 💡 **Fitur Reminder Berkala:**  
> Jika pengingat dibuat tanpa menyebutkan jam spesifik (contoh: *"ingetin beli obat"*), bot otomatis mengingatkan Anda **3x sehari** (pagi, siang, sore) sampai Anda menandainya selesai.

### B. Menyelesaikan Pengingat
Setiap pengingat memiliki ID unik (contoh: `REM-1234`).
- Ketik: `/done REM-1234` atau chat *"udah selesai zoom meeting"*.

---

## 📝 6. Catatan Cepat (*Quick Notes*)

Simpan catatan ide atau memo penting langsung ke tab `Notes` di Google Sheets:
- `/notes ukuran baju kemeja kerja: L, celana: 32`
- `/notes rekomendasi tempat liburan: Bandung, Jogja, Malang`
- Atau chat santai: *"catat ide project baru..."*

---

## 📂 7. Navigasi & Info Bantuan

| Perintah | Penjelasan |
| :--- | :--- |
| `/help` atau `/menu` | Menampilkan ringkasan menu bantuan di chat WhatsApp |
| `/kategori` | Melihat daftar kategori pengeluaran dan pemasukan resmi |
| `/export` | Mendapatkan link langsung ke Google Spreadsheet Anda |
| `/start` | Pesan sambutan & info status bot |

---

## 🤖 8. Jadwal Laporan Otomatis dari Bot

Tanpa perlu diminta, bot akan secara otomatis mengirimkan pesan rekap terjadwal ke WhatsApp Anda:
1. 🌙 **Laporan Harian Rutin**: Setiap malam pukul **00:00 WIB** (Rekap pengeluaran harian + grafik).
2. 📅 **Laporan Mingguan Rutin**: Setiap hari **Senin pukul 00:01 WIB** (Rekap mingguan + grafik).
3. 🗓️ **Laporan Bulanan Rutin**: Setiap **tanggal 1 pukul 00:01 WIB** (Rekap bulan sebelumnya + grafik).
