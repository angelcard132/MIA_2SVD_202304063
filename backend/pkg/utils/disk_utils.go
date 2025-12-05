package utils

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

// ConvertToBytes - Convierte tamaño según la unidad especificada
func ConvertToBytes(size int64, unit string) int64 {
	unit = strings.ToUpper(unit)
	switch unit {
	case "K":
		return size * 1024
	case "M":
		return size * 1024 * 1024
	default:
		return size * 1024 * 1024 // Default: Megabytes
	}
}

// GenerateRandomSignature - Genera un número random para identificar el disco
func GenerateRandomSignature() int32 {
	rand.Seed(time.Now().UnixNano())
	return rand.Int31()
}

// GetFitByte - Convierte string de ajuste a byte
func GetFitByte(fit string) byte {
	fit = strings.ToUpper(fit)
	switch fit {
	case "BF", "B":
		return 'B'
	case "FF", "F":
		return 'F'
	case "WF", "W":
		return 'W'
	default:
		return 'F' // Default: First Fit
	}
}

// ByteToString - Convierte array de bytes a string eliminando bytes nulos
func ByteToString(b []byte) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n])
}

// StringToBytes - Convierte string a array de bytes de tamaño fijo
func StringToBytes(s string, size int) []byte {
	b := make([]byte, size)
	copy(b, []byte(s))
	return b
}

// CreateDirectory - Crea un directorio si no existe
func CreateDirectory(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// FileExists - Verifica si un archivo existe
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// WriteStruct - Escribe una estructura en un archivo en la posición especificada
func WriteStruct(file *os.File, data interface{}, position int64) error {
	_, err := file.Seek(position, 0)
	if err != nil {
		return err
	}
	return binary.Write(file, binary.LittleEndian, data)
}

// ReadStruct - Lee una estructura desde un archivo en la posición especificada
func ReadStruct(file *os.File, data interface{}, position int64) error {
	_, err := file.Seek(position, 0)
	if err != nil {
		return err
	}
	return binary.Read(file, binary.LittleEndian, data)
}

// GetStructSize - Obtiene el tamaño de una estructura
func GetStructSize(data interface{}) int64 {
	return int64(unsafe.Sizeof(data))
}

// GeneratePartitionID - Genera el ID de una partición montada
// Formato: XX + N + L (XX = últimos 2 dígitos del carnet en hex, N = número partición, L = letra disco)
func GeneratePartitionID(carnet string, partNum int, diskLetter byte) string {
	// Extraer últimos 2 dígitos del carnet
	if len(carnet) < 2 {
		carnet = "00"
	}
	lastTwo := carnet[len(carnet)-2:]

	// Convertir a hexadecimal
	var num int
	fmt.Sscanf(lastTwo, "%d", &num)
	hexValue := fmt.Sprintf("%02X", num)

	// Formato: HEX + PartNum + DiskLetter
	return fmt.Sprintf("%s%d%c", hexValue, partNum, diskLetter)
}

// ParseCommand - Analiza un comando y extrae sus parámetros
func ParseCommand(command string) (string, map[string]string, error) {
	command = strings.TrimSpace(command)

	// Ignorar comentarios y líneas vacías
	if strings.HasPrefix(command, "#") || command == "" {
		return "", nil, nil
	}

	parts := []string{}
	current := ""
	inQuotes := false

	for _, char := range command {
		if char == '"' {
			inQuotes = !inQuotes
		} else if char == ' ' && !inQuotes {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) == 0 {
		return "", nil, nil
	}

	cmdName := strings.ToLower(parts[0])
	params := make(map[string]string)

	for i := 1; i < len(parts); i++ {
		param := parts[i]
		if strings.Contains(param, "=") {
			kv := strings.SplitN(param, "=", 2)
			key := strings.ToLower(strings.TrimPrefix(kv[0], "-"))
			value := strings.Trim(kv[1], "\"")
			params[key] = value
		}
	}

	return cmdName, params, nil
}

// FormatTime - Formatea un time.Time para mostrar
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// CalculateEXT2Structures - Calcula el número de inodos y bloques para EXT2
func CalculateEXT2Structures(partitionSize int64) (int32, int32) {
	// Tamaño de las estructuras
	superblockSize := int64(unsafe.Sizeof(struct {
		S_filesystem_type   int32
		S_inodes_count      int32
		S_blocks_count      int32
		S_free_blocks_count int32
		S_free_inodes_count int32
		S_mtime             int64
		S_umtime            int64
		S_mnt_count         int32
		S_magic             int32
		S_inode_s           int32
		S_block_s           int32
		S_first_ino         int32
		S_first_blo         int32
		S_bm_inode_start    int64
		S_bm_block_start    int64
		S_inode_start       int64
		S_block_start       int64
	}{}))

	inodeSize := int64(unsafe.Sizeof(struct {
		I_uid   int32
		I_gid   int32
		I_size  int64
		I_atime int64
		I_ctime int64
		I_mtime int64
		I_block [15]int32
		I_type  byte
		I_perm  [3]byte
	}{}))
	blockSize := int64(64) // Tamaño del bloque

	// Fórmula: partitionSize = superblock + n + 3*n + n*inodeSize + 3*n*blockSize
	// partitionSize = superblock + n(1 + 3 + inodeSize + 3*blockSize)
	// n = (partitionSize - superblock) / (1 + 3 + inodeSize + 3*blockSize)

	denominator := 1 + 3 + inodeSize + (3 * blockSize)
	n := float64(partitionSize-superblockSize) / float64(denominator)

	numStructures := int32(n)

	if numStructures < 1 {
		numStructures = 1
	}

	numInodes := numStructures
	numBlocks := numStructures * 3

	return numInodes, numBlocks
}

// GetNextAvailableDiskLetter - Obtiene la siguiente letra disponible para un disco
func GetNextAvailableDiskLetter(existingDisks []string) byte {
	used := make(map[byte]bool)

	for _, disk := range existingDisks {
		// Extraer letra del nombre del disco (VDIC-X.mia)
		if len(disk) >= 7 {
			letter := disk[5]
			used[letter] = true
		}
	}

	// Buscar primera letra disponible desde A
	for letter := byte('A'); letter <= byte('Z'); letter++ {
		if !used[letter] {
			return letter
		}
	}

	return 'A'
}
