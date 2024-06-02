package main

import (
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const NMAX int = 200

type login struct {
	username string
	password string
}

type transaction struct {
	idTransaksi int
	barang      [50]sell
	keuntungan  int
	waktu       time.Time
}

type barang struct {
	id         int
	nama       string
	harga      int
	stock      int
	lokasi     string
	barcode    string
	hargaPokok int
}
type user struct {
	username string
	password string
	role     string
}

type sell struct {
	idBarang int
	nama     string
	jumlah   int
	harga    int
}

var transaksiData [200]transaction
var userData [200]user
var barangData [200]barang
var currUser user
var debug bool
var hourAdjust int

func main() {
	isFresh()
	loadFromDB()
	dashboard()
}

func cetakDashboard() int {
	var usrDashPick int

	fmt.Println("-------------------------------------")
	fmt.Println("|             DASHBOARD             |")
	fmt.Println("-------------------------------------")
	fmt.Println("|  1. LOGIN                         |")
	fmt.Println("|  2. CEK HARGA                     |")
	fmt.Println("|  3. CARI BARANG                   |")
	fmt.Println("|  4. EXIT                          |")
	fmt.Println("-------------------------------------")
	fmt.Print("Silahkan masukkan pilihan anda : ")
	fmt.Scanln(&usrDashPick)
	fmt.Println("-------------------------------------")
	fmt.Println("")
	return usrDashPick
}

func cetakLogin() login {
	var LD login
	var username, password string
	fmt.Println("-----------------------------------------")
	fmt.Println("|                 LOGIN                 |")
	fmt.Println("-----------------------------------------")
	fmt.Print("Masukkan Username Anda   : ")
	fmt.Scanln(&username)
	fmt.Print("Masukkan Password Anda   : ")
	fmt.Scanln(&password)
	fmt.Println("-----------------------------------------")
	fmt.Println("")
	LD.username = username
	LD.password = encrypt(password)
	return LD
}

func encrypt(raw string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}

func dashboard() {
	switch cetakDashboard() {
	case 1:
		loginPage()
	case 2:
		cariBarcode()
		fmt.Println("------------------------------------")
		pressEnter()
		clearScreen()
		dashboard()
	case 3:
		fmt.Println("-----------------------------------")
		fmt.Println("|           CARI BARANG           |")
		fmt.Println("-----------------------------------")
		fmt.Println("Masukkan Nama Barang :")
		fmt.Println("-----------------------------------")
		var name string
		fmt.Scan(&name)
		cariBarang(name)
		pressEnter()
		clearScreen()
		dashboard()
	case 4:
		os.Exit(0)
	default:
		fmt.Println("Pilihan tidak valid")
		pressEnter()
		clearScreen()
		dashboard()
	}
}

func loginPage() {
	var LD login
	var count int
	var found bool
	found = false
	for _, item := range userData {
		if item.username != "" {
			count++
		}
	}

	for i := 3; i >= 0 && !found; i-- {
		LD = cetakLogin()
		for i := 0; i < count && !found; i++ {
			if userData[i].username == LD.username && userData[i].password == LD.password {
				found = true
				currUser = userData[i]
				fmt.Println("Hai", LD.username, "Login Berhasil Sebagai", userData[i].role)
				if userData[i].role == "admin" {
					adminPage()
				} else if userData[i].role == "kasir" {
					kasirPage()
				}
			}
		}
		if !found {
			fmt.Println("Username atau Password yang anda masukkan salah")
			fmt.Println("Tersisa", i, "x percobaan")
		}
	}

	if !found {
		fmt.Println("Terblokir! Program Dihentikan")
	} else {

	}
}

func saveUser(filename string, data [200]user) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, item := range data {
		_, err := file.WriteString(fmt.Sprintf("%s,%s,%s\n", item.username, item.password, item.role))
		if err != nil {
			return err
		}
	}
	return nil
}

func saveBarang(filename string, data [200]barang) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, item := range data {
		_, err := file.WriteString(fmt.Sprintf("%d,%s,%d,%d,%s,%s,%d\n", item.id, item.nama, item.harga, item.stock, item.lokasi, item.barcode, item.hargaPokok))
		if err != nil {
			return err
		}
	}
	return nil
}

func loadBarang(filename string) ([200]barang, error) {
	var data [200]barang
	count := 0

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("ERR Load Data")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) == 7 && count < len(data) {
			id, _ := strconv.Atoi(parts[0])
			harga, _ := strconv.Atoi(parts[2])
			stock, _ := strconv.Atoi(parts[3])
			hargaPokok, _ := strconv.Atoi(parts[6])
			data[count] = barang{id: id, nama: parts[1], harga: harga, stock: stock, lokasi: parts[4], barcode: parts[5], hargaPokok: hargaPokok}
			count++
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("ERR Load Data")
	}
	return data, nil
}

func saveTransaksi(filename string, data [200]transaction) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < 200; i++ {
		item := data[i]
		if item.idTransaksi == 0 {
			continue
		}
		_, err := file.WriteString(fmt.Sprintf("%d,[", item.idTransaksi))
		if err != nil {
			return err
		}
		for j := 0; j < 50; j++ {
			barang := item.barang[j]
			if barang.idBarang == 0 {
				break
			}
			if j != 0 {
				_, err = file.WriteString(",")
				if err != nil {
					return err
				}
			}
			_, err = file.WriteString(fmt.Sprintf("%d:%s:%d:%d", barang.idBarang, barang.nama, barang.jumlah, barang.harga))
			if err != nil {
				return err
			}
		}
		_, err = file.WriteString(fmt.Sprintf("],%d,%s\n", item.keuntungan, item.waktu.Format("2006-01-02 15:04:05")))
		if err != nil {
			return err
		}
	}
	return nil
}

func loadTransaksi(filename string) ([200]transaction, error) {
	var transaksiData [200]transaction
	count := 0

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("ERR Load Data")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) == 4 && count < 200 {
			id, _ := strconv.Atoi(parts[0])
			keuntungan, _ := strconv.Atoi(parts[2])
			waktu, _ := time.Parse("2006-01-02 15:04:05", parts[3])
			transaksiData[count] = transaction{idTransaksi: id, barang: parseBarang(parts[1]), keuntungan: keuntungan, waktu: waktu}
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("ERR Load Data")
	}
	return transaksiData, nil
}

func parseBarang(str string) [50]sell {
	var barang [50]sell
	count := 0
	parts := strings.Split(str[1:len(str)-1], ",")
	for _, part := range parts {
		if count >= 50 {
			break
		}
		items := strings.Split(part, ":")
		if len(items) == 4 {
			id, _ := strconv.Atoi(items[0])
			barang[count] = sell{idBarang: id, nama: items[1], jumlah: atoi(items[2]), harga: atoi(items[3])}
			count++
		}
	}
	return barang
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func adminPage() {
	clearScreen()
	switch cetakAdmin() {
	case 1:
		manageUser()
		pressEnter()
	case 2:
		clearScreen()
		var newBarang barang
		fmt.Println("--------------------------")
		fmt.Println("|    Tambahkan Barang    |")
		fmt.Println("--------------------------")
		fmt.Print("Masukkan Nama : ")
		newBarang.nama = scanKalimat()
		fmt.Print("Masukkan Harga : ")
		fmt.Scanln(&newBarang.harga)
		fmt.Print("Masukkan Stock : ")
		fmt.Scanln(&newBarang.stock)
		fmt.Print("Masukkan Lokasi : ")
		newBarang.lokasi = scanKalimat()
		fmt.Print("Masukkan Barcode : ")
		newBarang.barcode = scanKalimat()
		fmt.Print("Masukkan Harga Pokok : ")
		fmt.Scanln(&newBarang.hargaPokok)
		tambahBarang(newBarang)
		saveBarang("barang.txt", barangData)
		fmt.Println("Barang Berhasil Ditambahkan")
		fmt.Println("")
		pressEnter()
		adminPage()
	case 3:
		clearScreen()
		cetakBarang()
		pressEnter()
		adminPage()
	case 4:
		clearScreen()
		var namaBarang string
		fmt.Println("---------------------------------------------")
		fmt.Println("|                Cari Barang                |")
		fmt.Println("---------------------------------------------")
		fmt.Println("|Masukkan Nama Barang Yang Ingin Di Cari :  |")
		fmt.Println("---------------------------------------------")
		namaBarang = scanKalimat()
		fmt.Println("---------------------------------------------")
		cariBarang(namaBarang)
		fmt.Println("---------------------------------------------")
		pressEnter()
		adminPage()
	case 5:
		clearScreen()
		var namaBarang string
		var idBarang int
		fmt.Println("Hapus Barang")
		fmt.Println("Masukkan Nama Barang Yang Ingin Di Hapus :")
		namaBarang = scanKalimat()
		if cariBarang(namaBarang) == 0 {
		} else {
			fmt.Print("Masukkan ID Barang Yang Ingin Di Hapus :")
			fmt.Scanln(&idBarang)
			if hapusBarang(idBarang) {
				saveBarang("barang.txt", barangData)
				fmt.Printf("Barang ID %d Berhasil Dihapus\n", idBarang)
			}
		}
		pressEnter()
		adminPage()

	case 6:
		clearScreen()
		var upBarang barang
		var namaUpBarang string
		fmt.Println("Update Barang")
		fmt.Println("Masukkan Nama Barang Yang Ingin Di Update :")
		namaUpBarang = scanKalimat()
		if cariBarang(namaUpBarang) == 0 {

		} else {
			fmt.Println("Masukkan ID Barang Yang Ingin Di Update :")
			var idBarang int
			fmt.Scan(&idBarang)
			fmt.Println("Masukkan = atau -1 Jika Tidak Ingin Di Ubah")
			fmt.Print("Masukkan Nama (=) : ")
			var namaBarang string
			namaBarang = scanKalimat()
			fmt.Println("")
			fmt.Print("Masukkan Harga (-1) : ")
			var hargaBarang int
			fmt.Scan(&hargaBarang)
			fmt.Print("Masukkan Stock (-1) : ")
			var stockBarang int
			fmt.Scan(&stockBarang)
			fmt.Print("Masukkan Lokasi (=) : ")
			var lokasiBarang string
			lokasiBarang = scanKalimat()
			fmt.Print("Masukkan Barcode (=) : ")
			var barcodeBarang string
			barcodeBarang = scanKalimat()
			var hargaPokokBarang int
			fmt.Print("Masukkan Harga Pokok (-1) :  ")
			fmt.Scan(&hargaPokokBarang)
			upBarang = barang{id: idBarang, nama: namaBarang, harga: hargaBarang, stock: stockBarang, lokasi: lokasiBarang, barcode: barcodeBarang, hargaPokok: hargaPokokBarang}
			updateBarang(upBarang, idBarang)
			saveBarang("barang.txt", barangData)
			fmt.Println("Barang Berhasil Diupdate")
			fmt.Println("Tekan Enter Untuk Kembali")
		}
		pressEnter()
		adminPage()
	case 7:
		clearScreen()
		ubahPassword()
		fmt.Println("Anda telah mengubah password")

		go func() {
			for i := 3; i >= 0; i-- {
				time.Sleep(1 * time.Second)
				fmt.Printf("\rLogout dalam %d detik", i)
			}
		}()

		time.Sleep(5 * time.Second)
		currUser = user{
			username: "",
			password: "",
			role:     "",
		}
		clearScreen()
		dashboard()
	case 8:
		clearScreen()
		lihatTransaksi()
	case 9:
		currUser = user{
			username: "",
			password: "",
			role:     "",
		}
		clearScreen()
		dashboard()

	case 10:
		debugInsert()
		adminPage()
	default:
		fmt.Println("Pilihan tidak valid")
		fmt.Println("Tekan Enter Untuk Kembali")
		pressEnter()
		adminPage()
	}
}

func cetakBarang() {
	var sort [200]barang
	var urut string
	fmt.Println("Masukkan asc/desc :")
	fmt.Scanln(&urut)
	switch cetakSort() {
	case 1:
		sort = sortBarang("nama", urut, barangData)
		printBarang(sort)
	case 2:
		sort = sortBarang("harga", urut, barangData)
		printBarang(sort)
	case 3:
		sort = sortBarang("stock", urut, barangData)
		printBarang(sort)
	case 4:
		sort = sortBarang("lokasi", urut, barangData)
		printBarang(sort)
	case 5:
		cetakAll()
	case 6:
		adminPage()
	default:
		fmt.Println("Pilihan tidak valid")
		fmt.Println("Tekan Enter Untuk Kembali")
		pressEnter()
		adminPage()
	}
}

func printBarang(b [200]barang) {
	n := panjangBarang(b)
	for i := 0; i < n; i++ {
		fmt.Println("ID : ", b[i].id)
		fmt.Println("NAMA : ", b[i].nama)
		fmt.Println("HARGA : ", b[i].harga)
		fmt.Println("STOCK : ", b[i].stock)
		fmt.Println("LOKASI : ", b[i].lokasi)
		fmt.Println("BARCODE : ", b[i].barcode)
		fmt.Println("====================================")
	}
}

func cetakSort() int {
	clearScreen()
	fmt.Println("-----------------------------")
	fmt.Println("|        CETAK BARANG       |")
	fmt.Println("-----------------------------")
	fmt.Println("|  1. Nama                  |")
	fmt.Println("|  2. Harga                 |")
	fmt.Println("|  3. Stock                 |")
	fmt.Println("|  4. Lokasi                |")
	fmt.Println("|  5. Cetak Semua           |")
	fmt.Println("|  6. Kembali               |")
	fmt.Println("-----------------------------")

	var userSortPick int
	fmt.Print("Silahkan masukkan pilihan anda : ")
	fmt.Scan(&userSortPick)
	return userSortPick
}

func manageUser() {
	clearScreen()
	fmt.Println("---------------------------------")
	fmt.Println("|           MANAGE USER         |")
	fmt.Println("---------------------------------")
	fmt.Println("|  1. Tambah User               |")
	fmt.Println("|  2. Tampilkan Seluruh User    |")
	fmt.Println("|  3. Hapus User                |")
	fmt.Println("|  4. Update User               |")
	fmt.Println("|  5. Kembali                   |")
	fmt.Println("---------------------------------")

	var usrManagePick int
	fmt.Print("Silahkan masukkan pilihan anda : ")
	fmt.Scanln(&usrManagePick)
	switch usrManagePick {
	case 1:
		clearScreen()
		var newUser user
		fmt.Println("----------------------------")
		fmt.Println("|    Tambahkan User Baru    |")
		fmt.Println("----------------------------")
		fmt.Print("Masukkan Username : ")
		fmt.Scanln(&newUser.username)
		fmt.Print("Masukkan Password : ")
		newUser.password = encrypt(scanKalimat())
		fmt.Print("Masukkan Role (admin/kasir) : ")
		fmt.Scanln(&newUser.role)

		var emptyIndex int
		emptyIndex = panjangUser()

		userData[emptyIndex] = newUser
		saveUser("user.txt", userData)
		fmt.Println("User Berhasil Ditambahkan")
		pressEnter()
		manageUser()
	case 2:
		clearScreen()
		cetakAllUser()
		pressEnter()
		manageUser()
	case 3:

		var notEmpty int
		for i := 0; i < len(userData); i++ {
			if userData[i].username != "" {
				notEmpty++
			}
		}

		var usernameDel string
		fmt.Print("Masukkan Username yang ingin dihapus : ")
		fmt.Scanln(&usernameDel)

		found := false
		for i := 0; i < notEmpty && !found; i++ {
			if userData[i].username == usernameDel {
				found = true
				for j := i; j < notEmpty-1; j++ {
					userData[j] = userData[j+1]
				}
				userData[notEmpty-1] = user{}
				notEmpty--
				fmt.Println("User berhasil dihapus")
				saveUser("user.txt", userData)
			}
		}
		if !found {
			fmt.Println("User tidak ditemukan")
		}
		pressEnter()
		manageUser()
	case 4:
		var notEmpty int
		for i := 0; i < len(userData); i++ {
			if userData[i].username != "" {
				notEmpty++
			}
		}

		var usernameUpdate string
		fmt.Print("Masukkan Username yang ingin diupdate : ")
		fmt.Scanln(&usernameUpdate)

		found := false
		for i := 0; i < notEmpty && !found; i++ {
			if userData[i].username == usernameUpdate {
				found = true
				fmt.Print("Masukkan Username Baru : ")
				fmt.Scanln(&userData[i].username)
				fmt.Print("Masukkan Password Baru : ")
				userData[i].password = encrypt(scanKalimat())
				fmt.Print("Masukkan Role Baru (admin/kasir) : ")
				fmt.Scanln(&userData[i].role)

				saveUser("user.txt", userData)
			}
		}
		manageUser()
	case 5:
		adminPage()
	default:
		fmt.Println("Pilihan tidak valid")
		fmt.Println("Tekan Enter Untuk Kembali")
		pressEnter()
		manageUser()
	}
}

func cetakAllUser() {
	var notEmpty int
	found := false
	for i := 0; i < len(userData) && !found; i++ {
		if userData[i].username != "" {
			notEmpty++
		}
	}

	fmt.Println("------------------------------------------------")
	fmt.Println("|            Tampilkan Seluruh User            |")
	fmt.Println("------------------------------------------------")
	fmt.Println("|       Username        |          Role        |")
	fmt.Println("------------------------------------------------")
	for j := 0; j < notEmpty; j++ {
		fmt.Printf("| %-21s | %-20s |\n", userData[j].username, userData[j].role)
	}
	fmt.Println("------------------------------------------------")
}
func cetakAdmin() int {
	var usrAdminPick int
	fmt.Println("--------------------------------")
	fmt.Println("|            ADMIN             |")
	fmt.Println("--------------------------------")
	fmt.Println("|  1. Manage User              |")
	fmt.Println("|  2. Tambah Barang            |")
	fmt.Println("|  3. Tampilkan Seluruh Barang |")
	fmt.Println("|  4. Cari Barang              |")
	fmt.Println("|  5. Hapus Barang             |")
	fmt.Println("|  6. Update Barang            |")
	fmt.Println("|  7. Ubah Password            |")
	fmt.Println("|  8. Tampilkan Transaksi      |")
	fmt.Println("|  9. Logout                   |")
	fmt.Println("--------------------------------")
	fmt.Print("Silahkan masukkan pilihan anda : ")
	fmt.Scanln(&usrAdminPick)
	fmt.Println("--------------------------------")
	fmt.Println("")
	return usrAdminPick
}

func loadUser(filename string) ([200]user, error) {
	var data [200]user
	var count int

	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("ERR Load Data")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) == 3 && count < 200 {
			data[count] = user{username: parts[0], password: parts[1], role: parts[2]}
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("ERR Load Data")
	}
	return data, nil
}

func loadFromDB() {
	if _, err := os.Stat("user.txt"); err == nil {
		data, err := loadUser("user.txt")
		if err != nil {
			fmt.Println(err)
		} else {
			userData = data
			// fmt.Println(userData)
		}
	}

	if _, err := os.Stat("barang.txt"); err == nil {
		data, err := loadBarang("barang.txt")
		if err != nil {
			fmt.Println(err)
		} else {
			barangData = data
			fmt.Println("Sukses Load Barang")
			// fmt.Println(barangData)
		}
	}

	if _, err := os.Stat("transaksi.txt"); err == nil {
		data, err := loadTransaksi("transaksi.txt")
		if err != nil {
			fmt.Println(err)
		} else {
			transaksiData = data
			fmt.Println("Sukses Load Transaksi")
			// fmt.Println(transaksiData)
		}
	}
	loadHourAdjust()

}

func isFresh() {
	var LD login
	if _, err := os.Stat("user.txt"); err != nil {
		fmt.Println("----------------------------")
		fmt.Println("|   Buat User Admin Anda   |")
		fmt.Println("----------------------------")
		fmt.Print("Username : ")
		fmt.Scan(&LD.username)
		fmt.Print("Password : ")
		fmt.Scan(&LD.password)
		LD.password = encrypt(LD.password)
		userData[0] = user{username: LD.username, password: LD.password, role: "admin"}
		saveUser("user.txt", userData)
		fmt.Scan(&hourAdjust)
		saveHourAdjust()
	}
}

func ubahPassword() {
	var passTemp string
	fmt.Print("Masukkan Password Lama : ")
	fmt.Scan(&passTemp)
	if encrypt(passTemp) != currUser.password {
		fmt.Println("Password yang anda masukkan salah")
	} else {
		fmt.Print("Masukkan Password Baru : ")
		fmt.Scan(&currUser.password)
		var found bool = false
		currUser.password = encrypt(currUser.password)
		for i := 0; i < len(userData)&&found == false; i++ {
			if userData[i].username == currUser.username {
				userData[i].password = currUser.password
				saveUser("user.txt", userData)
				found = true
			}
		}
	}
}

func tambahBarang(brg barang) {
	var indexEmpty int
	var found bool
	var idMax int
	idMax = barangData[0].id
	found = false
	for j := 0; j < len(barangData); j++ {
		if barangData[j].id > idMax {
			idMax = barangData[j].id
		}
	}
	for i := 0; i < len(barangData) && !found; i++ {
		if barangData[i].nama == "" {
			indexEmpty = i
			found = true
		}
	}
	barangData[indexEmpty].harga = brg.harga
	barangData[indexEmpty].id = idMax + 1
	barangData[indexEmpty].lokasi = brg.lokasi
	barangData[indexEmpty].stock = brg.stock
	barangData[indexEmpty].nama = brg.nama
	barangData[indexEmpty].barcode = brg.barcode
	barangData[indexEmpty].hargaPokok = brg.hargaPokok
}

// /////Hapus Barang Binary Search/////////
func hapusBarang(idBarang int) bool {
	var n int
	n = panjangBarang(barangData)

	left := 0
	right := n - 1
	for left <= right {
		mid := (left + right) / 2
		if barangData[mid].id == idBarang {
			for i := mid; i < n-1; i++ {
				barangData[i] = barangData[i+1]
			}
			barangData[n-1] = barang{}
			return true
		} else if barangData[mid].id < idBarang {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return false

}

func sortBarang(by string, urutan string, b [200]barang) [200]barang {
	n := panjangBarang(barangData)
	if by == "nama" {
		for i := 0; i < n-1; i++ {
			min := i
			for j := i + 1; j < n; j++ {
				if urutan == "asc" {
					if strings.ToLower(b[j].nama) < strings.ToLower(b[min].nama) {
						min = j
					}
				} else if urutan == "desc" {
					if strings.ToLower(b[j].nama) > strings.ToLower(b[min].nama) {
						min = j
					}
				}
			}
			temp := b[i]
			b[i] = b[min]
			b[min] = temp
		}
	} else if by == "harga" {
		//insertion sort
		if urutan == "asc" {
			for i := 1; i < n; i++ {
				for j := i; j > 0 && b[j].harga < b[j-1].harga; j-- {
					b[j], b[j-1] = b[j-1], b[j]
				}
			}
		} else if urutan == "desc" {
			for i := 1; i < n; i++ {
				for j := i; j > 0 && b[j].harga > b[j-1].harga; j-- {
					b[j], b[j-1] = b[j-1], b[j]
				}
			}
		}
	} else if by == "stock" {
		//selection sort
		if urutan == "asc" {
			for i := 0; i < n-1; i++ {
				min := i
				for j := i + 1; j < n; j++ {
					if b[j].stock < b[min].stock {
						min = j
					}
				}
				temp := b[i]
				b[i] = b[min]
				b[min] = temp
			}
		} else if urutan == "desc" {
			for i := 0; i < n-1; i++ {
				min := i
				for j := i + 1; j < n; j++ {
					if b[j].stock > b[min].stock {
						min = j
					}
				}
				temp := b[i]
				b[i] = b[min]
				b[min] = temp
			}
		}
	} else if by == "lokasi" {
		//bubble sort
		if urutan == "asc" {
			for i := 0; i < n-1; i++ {
				for j := 0; j < n-i-1; j++ {
					if b[j].lokasi > b[j+1].lokasi {
						b[j], b[j+1] = b[j+1], b[j]
					}
				}
			}
		} else if urutan == "desc" {
			for i := 0; i < n-1; i++ {
				for j := 0; j < n-i-1; j++ {
					if b[j].lokasi < b[j+1].lokasi {
						b[j], b[j+1] = b[j+1], b[j]
					}
				}
			}
		}
	}
	return b
}
func updateBarang(brg barang, idBarang int) bool {
	var found bool
	found = false
	for i := 0; i < len(barangData) && !found; i++ {
		if barangData[i].id == idBarang {
			if brg.lokasi != "=" {
			barangData[i].lokasi = brg.lokasi
			}
			if brg.stock != -1 {
				barangData[i].stock = brg.stock
			}
			if brg.harga != -1 {
				barangData[i].harga = brg.harga
			}
			if brg.nama != "=" {
				barangData[i].nama = brg.nama
			}
			if brg.barcode != "=" {
				barangData[i].barcode = brg.barcode
			}
			if brg.hargaPokok != -1 {
				barangData[i].hargaPokok = brg.hargaPokok
			}
			found = true
			return true
		}
	}
	return false
}

func kasirPage() {
	clearScreen()
	switch cetakKasir() {
	case 1:
		clearScreen()
		sellOut()
	case 2:
		clearScreen()
		lihatTransaksi()
		pressEnter()
		kasirPage()
	case 3:
		clearScreen()
		sort := sortBarang("asc", "stock", barangData)
		printBarang(sort)
		pressEnter()
		kasirPage()
	case 4:
		clearScreen()
		fmt.Println("Masukkan Nama Barang Yang Ingin Dicari :")
		var search string
		fmt.Scanln(&search)
		cariBarang(search)
		pressEnter()
		kasirPage()	
	case 5:
		clearScreen()
		currUser = user{}
		dashboard()
	default:
		fmt.Println("Pilihan tidak valid")
	}
}

func sellOut() {
	var sellOutData [50]sell
	var barcode string
	var barangKe, jumlah, uang, idx int
	var end bool = false
	var totalBelanja int = 0
	barangKe = 0
	fmt.Println("--------------------------")
	fmt.Println("|        SELL OUT        |")
	fmt.Println("--------------------------")
	for barangKe < 50 && !end {
		fmt.Println("Masukkan Barcode Barang", barangKe, ":")
		fmt.Scanln(&barcode)
		idx = barcodeCheck(barcode)
		if idx == -1 {
			fmt.Println("Barang tidak ditemukan")
		} else if barangData[idx].stock <= 0 {
			fmt.Println("Stock Barang habis")
		} else {
			fmt.Println("Masukkan Jumlah :")
			fmt.Scanln(&jumlah)
			if jumlah > barangData[idx].stock {
				fmt.Println("Stock Barang Kurang")
			} else {
				fmt.Println("Nama Barang :", barangData[barcodeCheck(barcode)].nama)
				sellOutData[barangKe].idBarang = barangData[idx].id
				sellOutData[barangKe].jumlah = jumlah
				sellOutData[barangKe].harga = barangData[idx].harga * jumlah
				sellOutData[barangKe].nama = barangData[idx].nama
				barangKe++
			}
		}
		fmt.Print("Apakah ingin melanjutkan ? (y/n) ")
		var yn string
		fmt.Scanln(&yn)
		if yn == "n" {
			end = true
		}
	}
	clearScreen()

	fmt.Println("-----------------------------------")
	fmt.Println("| PERIKSA DATA BARANG YANG DIBELI |")
	fmt.Println("-----------------------------------")
	for i := 0; i < len(sellOutData) && sellOutData[i].idBarang != 0; i++ {
		fmt.Println("Barang ke", i+1, ":")
		fmt.Println("ID :", sellOutData[i].idBarang)
		fmt.Println("Jumlah :", sellOutData[i].jumlah)
		totalBelanja = totalBelanja + sellOutData[i].harga
	}
	fmt.Println("-----------------------------------")
	fmt.Println("Apakah ingin melanjutkan sell out (y/n)? ")
	var yn string
	fmt.Scanln(&yn)
	fmt.Println("-----------------------------------")
	if yn == "y" {
		fetchBarangData(sellOutData)
		fmt.Println("Total Belanja :", totalBelanja)
		fmt.Print("Masukkan Uang Pembeli :")
		fmt.Scanln(&uang)
		fmt.Println("Kembalian :", uang-totalBelanja)
		fmt.Println("Terima Kasih")
		pressEnter()
		kasirPage()
	} else {
		kasirPage()
	}
}

func fetchBarangData(sellData [50]sell) {
	var nSell int
	end := false
	for i := 0; i < len(sellData) && !end; i++ {
		if sellData[i].idBarang != 0 {
			nSell++
		} else {
			end = true
		}
	}

	var endInsert bool = false
	var keuntunganTransaksi int
	nBarang := panjangBarang(barangData)
	// binary search
	for i := 0; i < nSell; i++ {
		low := 0
		high := nBarang - 1
		for low <= high && !endInsert {
			mid := (low + high) / 2
			if sellData[i].idBarang == barangData[mid].id {
				barangData[mid].stock -= sellData[i].jumlah
				keuntunganTransaksi = keuntunganTransaksi + ((barangData[mid].harga - barangData[mid].hargaPokok) * sellData[i].jumlah)
				endInsert = true
			} else if sellData[i].idBarang < barangData[mid].id {
				high = mid - 1
			} else {
				low = mid + 1
			}
		}
	}
	lastId := getLastIdTransaksi()
	nTransaksi := panjangTransaksi()
	transaksiData[nTransaksi].idTransaksi = lastId + 1
	transaksiData[nTransaksi].barang = sellData
	transaksiData[nTransaksi].keuntungan = keuntunganTransaksi
	loc, _ := time.LoadLocation("GMT")
	transaksiData[nTransaksi].waktu = time.Now().In(loc).Add(time.Hour + time.Duration(hourAdjust)*time.Hour)
	saveBarang("barang.txt", barangData)
	saveTransaksi("transaksi.txt", transaksiData)
}

func getLastIdTransaksi() int {
	var lastId int
	for i := 0; i < len(transaksiData); i++ {
		if transaksiData[i].idTransaksi > lastId {
			lastId = transaksiData[i].idTransaksi
		}
	}
	return lastId
}
func barcodeCheck(barcode string) int {
	n := panjangBarang(barangData)
	// Sequential Search
	for i := 0; i < n; i++ {
		if barangData[i].barcode == barcode {
			return i
		}
	}
	return -1
}

func cetakKasir() int {
	fmt.Println("---------------------------")
	fmt.Println("|          KASIR          |")
	fmt.Println("---------------------------")
	fmt.Println("|  1. Sell Out            |")
	fmt.Println("|  2. Lihat Transaksi     |")
	fmt.Println("|  3. Lihat Barang        |")
	fmt.Println("|  4. Cari Barang         |")
	fmt.Println("|  5. Logout              |")
	fmt.Println("---------------------------")
	fmt.Print("Silahkan masukkan pilihan anda : ")
	var userPick int
	fmt.Scanln(&userPick)
	fmt.Println("---------------------------")
	fmt.Println("")
	return userPick
}

func scanKalimat() string {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
	}

	// Menghapus karakter newline dari akhir input
	input = strings.TrimSpace(input)
	return input
}

func cetakAll() {
	var isEmpty bool
	isEmpty = false
	for i := 0; i < len(barangData) && !isEmpty; i++ {
		if barangData[i].nama != "" {
			fmt.Println("-----------------------------")
			fmt.Print("ID : ")
			fmt.Println(barangData[i].id)
			fmt.Print("NAMA : ")
			fmt.Println(barangData[i].nama)
			fmt.Print("STOCK : ")
			fmt.Println(barangData[i].stock)
			fmt.Print("LOKASI : ")
			fmt.Println(barangData[i].lokasi)
			fmt.Print("HARGA : Rp.")
			fmt.Println(barangData[i].harga)
			fmt.Print("BARCODE : ")
			fmt.Println(barangData[i].barcode)
			if currUser.role == "admin" {
				fmt.Print("HARGA POKOK : Rp.")
				fmt.Println(barangData[i].hargaPokok)
			}
		} else {
			isEmpty = true
		}
	}
	fmt.Println("-------------------------------")
}

func cariBarang(searchTerm string) int {
	var found bool
	var totalFound int
	searchTerm = strings.ToLower(searchTerm)

	for _, barang := range barangData {
		if barang.nama != "" {
			lowerName := strings.ToLower(barang.nama)
			for i := 0; i <= len(lowerName)-len(searchTerm); i++ {
				if strings.Contains(lowerName[i:i+len(searchTerm)], searchTerm) {
					found = true
					totalFound++
					fmt.Println("-----------------------------")
					if currUser.role == "admin" || currUser.role == "kasir" {
						fmt.Println("ID :", barang.id)
						fmt.Println("NAMA :", barang.nama)
						fmt.Println("STOCK :", barang.stock)
						fmt.Println("LOKASI :", barang.lokasi)
						fmt.Printf("HARGA : Rp.%d\n", barang.harga)
						fmt.Println("BARCODE :", barang.barcode)
					} else {
						fmt.Println("NAMA :", barang.nama)
						fmt.Println("STOCK :", barang.stock)
						fmt.Println("LOKASI :", barang.lokasi)
						fmt.Printf("HARGA : Rp.%d\n", barang.harga)
					}

					if currUser.role == "admin" {
						fmt.Println("HARGA POKOK :", barang.hargaPokok)
					}
					fmt.Println("-----------------------------")
				}
			}
		}
	}

	if !found {
		fmt.Println("Barang tidak ditemukan")
		return 0
	}

	return totalFound
}

func clearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	case "linux":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
	}
}

func debugInsert() {
	dummyMakanan := []string{"Nasi Goreng", "Rendang", "Sate", "Gado-gado", "Soto", "Sop Buntut", "Siomay", "Soto Betawi", "Bakso", "Soto Meken", "Soto Madu", "Soto Ayam", "Soto Pedas", "Soto Rempah-rempah", "Soto Kacang", "Soto Mentega", "Soto Pekora", "Soto Padang", "Soto Betawi"}
	count := 0
	for _, nama := range dummyMakanan {
		if count < len(barangData) {
			barangData[count] = barang{id: count, nama: nama, harga: count * 10000, stock: count * 100, lokasi: "Gedung A", barcode: fmt.Sprintf("%010d", count)}
			count++
		}
	}

}

func pressEnter() {
	var dumpInput string
	fmt.Print("Tekan Enter Untuk Kembali")
	fmt.Scanln(&dumpInput)
}

func cariBarcode() {
	var result string = ""
	var found bool
	var indexBarang int
	n := panjangBarang(barangData)
	fmt.Println("------------------------------------------------")
	fmt.Println("|       Silahkan Masukkan Barcode Produk       |")
	if debug == true {
		fmt.Println("| Atau Buka https://code.zackym.com/proxy/6161 |")
	}
	fmt.Println("------------------------------------------------")
	if debug == true {
		qrResultCh := webDeploy()
		result = <-qrResultCh
	}
	if result == "" {
		fmt.Scanln(&result)
	}

	//Sequential Search
	found = false
	for i := 0; i < n && !found; i++ {
		if barangData[i].barcode == result {
			indexBarang = i
			found = true
		}
	}

	if currUser.role == "admin" && found {
		fmt.Println("Id : ", barangData[indexBarang].id)
		fmt.Println("Nama :", barangData[indexBarang].nama)
		fmt.Println("Stock :", barangData[indexBarang].stock)
		fmt.Println("Lokasi :", barangData[indexBarang].lokasi)
		fmt.Println("Harga :", barangData[indexBarang].harga)
		fmt.Println("Barcode :", barangData[indexBarang].barcode)
		fmt.Println("Harga Pokok :", barangData[indexBarang].hargaPokok)
	} else if currUser.role == "kasir" && found {
		fmt.Println("Id :", barangData[indexBarang].id)
		fmt.Println("Nama :", barangData[indexBarang].nama)
		fmt.Println("Stock :", barangData[indexBarang].stock)
		fmt.Println("Lokasi :", barangData[indexBarang].lokasi)
		fmt.Println("Harga :", barangData[indexBarang].harga)
		fmt.Println("Barcode :", barangData[indexBarang].barcode)
	} else if !found {
		fmt.Println("Barang Tidak Ditemukan")
	} else {
		fmt.Println("Nama Produk :", barangData[indexBarang].nama)
		fmt.Println("Harga : ", barangData[indexBarang].harga)
	}

}

// //////////////////////FOR WEBSITE PURPOSE///////////////////////
func webDeploy() chan string {
	qrResultCh := make(chan string)
	srv := &http.Server{Addr: ":6161"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/qr", func(w http.ResponseWriter, r *http.Request) {
		qrResult := r.URL.Query().Get("url")
		fmt.Println(qrResult)
		qrResultCh <- qrResult
		if err := srv.Shutdown(context.Background()); err != nil {
			fmt.Printf("HTTP server Shutdown: %v", err)
		}
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return qrResultCh
}

func stopServer() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv := &http.Server{}
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server stopped")
}

////////////////////////END OF FOR WEBSITE PURPOSE///////////////////////

func panjangBarang(b [200]barang) int {
	for i := 0; i < len(b); i++ {
		if b[i].nama == "" {
			return i
		}
	}
	return len(barangData)
}

func panjangUser() int {
	for i := 0; i < len(userData); i++ {
		if userData[i].username == "" {
			return i
		}
	}
	return len(userData)
}

func panjangTransaksi() int {
	for i := 0; i < len(transaksiData); i++ {
		if transaksiData[i].idTransaksi == 0 {
			return i
		}
	}
	return len(transaksiData)
}

func lihatTransaksi() {
	var n = panjangTransaksi()
	fmt.Println("-------------------------------------------------")
	fmt.Println("|              Tampilkan Transaksi              |")
	fmt.Println("-------------------------------------------------")
	fmt.Println("1. Tampilkan Seluruh Transaksi")
	fmt.Println("2. Cari Transaksi Berdasarkan Tanggal")
	fmt.Println("3. Cari Transaksi Berdasarkan Range Tanggal")
	if currUser.role == "admin" {
		fmt.Println("4. Tampilkan Omset Berdasarkan Tanggal")
		fmt.Println("5. Tampilkan Omset Berdasarkan Range Tanggal")
		fmt.Println("6. Kembali")
	} else {
		fmt.Println("4. Kembali")
	}
	var userPick int
	fmt.Print("Masukkan Pilihan Anda :")
	fmt.Scanln(&userPick)
	clearScreen()
	fmt.Println("------------------------------------------------")

	switch userPick {
	case 1:
		for i := 0; i < n; i++ {
			fmt.Println("Id Transaksi :", transaksiData[i].idTransaksi)
			fmt.Println("-----------------------------------------------	")
			for j := 0; j < len(transaksiData[i].barang) && transaksiData[i].barang[j].nama != ""; j++ {
				fmt.Println("Nama Barang :", transaksiData[i].barang[j].nama)
				fmt.Println("Jumlah :", transaksiData[i].barang[j].jumlah)
				fmt.Println("Harga :", transaksiData[i].barang[j].harga)
				if currUser.role == "admin" {
					fmt.Println("Omset :", transaksiData[i].keuntungan)
				}
				fmt.Println("------------------------------------------------")
			}
			fmt.Println("Waktu Transaksi :", transaksiData[i].waktu.Format("2006-01-02 15:04:05"))
			fmt.Println("------------------------------------------------")
		}

		pressEnter()
		if currUser.role == "admin" {
			adminPage()
		} else {
			kasirPage()
		}
	case 2:
		var tgl string
		fmt.Print("Masukkan Tanggal Transaksi (YYYY-MM-DD) :")
		fmt.Scanln(&tgl)
		var found bool = false
		for i := 0; i < n; i++ {
			if tgl == transaksiData[i].waktu.Format("2006-01-02") {
				fmt.Println("Id Transaksi :", transaksiData[i].idTransaksi)
				fmt.Println("------------------------------------------------")
				for j := 0; j < len(transaksiData[i].barang) && transaksiData[i].barang[j].nama != ""; j++ {
					fmt.Println("Nama Barang :", transaksiData[i].barang[j].nama)
					fmt.Println("Jumlah :", transaksiData[i].barang[j].jumlah)
					fmt.Println("Harga :", transaksiData[i].barang[j].harga)
					if currUser.role == "admin" {
						fmt.Println("Omset :", transaksiData[i].keuntungan)
					}
					fmt.Println("------------------------------------------------")
				}
				fmt.Println("Waktu Transaksi :", transaksiData[i].waktu.Format("2006-01-02 15:04:05"))
				fmt.Println("------------------------------------------------")
				found = true
				
			}
		}
		if !found {
			fmt.Println("Transaksi Tidak Ditemukan")
		}

		pressEnter()
		if currUser.role == "admin" {
			adminPage()
		} else {
			kasirPage()
		}
	case 3:
		var tgl1 string
		var tgl2 string
		fmt.Print("Masukkan Tanggal Awal (YYYY-MM-DD) : ")
		fmt.Scanln(&tgl1)
		fmt.Print("Masukkan Tanggal Akhir (YYYY-MM-DD) : ")
		fmt.Scanln(&tgl2)
		var found bool = false
		for i := 0; i < n; i++ {
			if tgl1 <= transaksiData[i].waktu.Format("2006-01-02") && tgl2 >= transaksiData[i].waktu.Format("2006-01-02") {
				fmt.Println("Id Transaksi :", transaksiData[i].idTransaksi)
				fmt.Println("------------------------------------------------")
				for j := 0; j < len(transaksiData[i].barang) && transaksiData[i].barang[j].nama != ""; j++ {
					fmt.Println("Nama Barang :", transaksiData[i].barang[j].nama)
					fmt.Println("Jumlah :", transaksiData[i].barang[j].jumlah)
					fmt.Println("Harga :", transaksiData[i].barang[j].harga)
					fmt.Println("------------------------------------------------")
				}
				fmt.Println("Waktu Transaksi :", transaksiData[i].waktu.Format("2006-01-02 15:04:05"))
				fmt.Println("------------------------------------------------")
				found = true
			}
		}
		if !found {
			fmt.Println("-------------------------")
			fmt.Println("Transaksi Tidak Ditemukan")
			fmt.Println("-------------------------")
		}

		pressEnter()
		if currUser.role == "admin" {
			adminPage()
		} else {
			kasirPage()
		}
	case 4:
		if currUser.role == "admin" {
		var keuntungan int
		var tgl string
		fmt.Print("Masukkan Tanggal Transaksi (YYYY-MM-DD) : ")
		fmt.Scanln(&tgl)
		var found bool = false
		for i := 0; i < n; i++ {
			if tgl == transaksiData[i].waktu.Format("2006-01-02") {
				keuntungan = keuntungan + transaksiData[i].keuntungan
				found = true
			}
		}
		if !found {
			fmt.Println("-------------------------")
			fmt.Println("Transaksi Tidak Ditemukan")
			fmt.Println("-------------------------")
		} else {
			fmt.Println("---------------------------------------")
			fmt.Printf("Keuntungan : Rp.%d\n", keuntungan)
			fmt.Println("---------------------------------------")
		}
	}

	pressEnter()
	if currUser.role == "admin" {
		adminPage()
	} else {
		kasirPage()
	}
			case 5:
				if currUser.role == "admin" {
				var keuntungan int
				var tgl1, tgl2 string
				fmt.Print("Masukkan Tanggal Awal (YYYY-MM-DD) : ")
				fmt.Scanln(&tgl1)
				fmt.Print("Masukkan Tanggal Akhir (YYYY-MM-DD) : ")
				fmt.Scanln(&tgl2)
				var found bool = false
				for i := 0; i < n; i++ {
					if tgl1 <= transaksiData[i].waktu.Format("2006-01-02") && tgl2 >= transaksiData[i].waktu.Format("2006-01-02") {
						keuntungan = keuntungan + transaksiData[i].keuntungan
						found = true
					}
				}
				if !found {
					fmt.Println("-------------------------")
					fmt.Println("Transaksi Tidak Ditemukan")
					fmt.Println("-------------------------")
					pressEnter()
				} else {
					fmt.Println("---------------------------------------")
					fmt.Printf("Keuntungan : Rp.%d\n", keuntungan)
					fmt.Println("---------------------------------------")
					
				pressEnter()
				if currUser.role == "admin" {
					adminPage()
				} else {
					kasirPage()
				}
				}}else{
					clearScreen()
					fmt.Println("Pilihan tidak valid")
					pressEnter()
					kasirPage()
				}

			case 6:
				if currUser.role == "admin" {
					clearScreen()
					adminPage()
				}else if currUser.role == "kasir" {
					clearScreen()
					fmt.Println("Pilihan tidak valid")
					pressEnter()
					kasirPage()
		
				}

			default:
				fmt.Println("Pilihan tidak valid")
				pressEnter()
				if currUser.role == "admin" {
					adminPage()
				} else {
					kasirPage()
				}
		}
	}

func saveHourAdjust() {
	f, err := os.OpenFile("hourAdjust.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("Error")
	}
	defer f.Close()

	_, err = f.WriteString(strconv.Itoa(hourAdjust))
	if err != nil {
		fmt.Println("Error")
	}
}

func loadHourAdjust() {
	f, err := os.OpenFile("hourAdjust.txt", os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("Error")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		hourAdjust, _ = strconv.Atoi(scanner.Text())
	}
}
