package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Estructura para trackear discos creados
type DiskInfo struct {
	Name   string
	Letter byte
	Path   string
}

// Lista de discos creados en esta sesión
var createdDisksMap = make(map[byte]DiskInfo)

// LoadExistingDisks - Carga todos los discos .mia existentes en el directorio actual
func LoadExistingDisks() {
	files, err := filepath.Glob("*.mia")
	if err != nil {
		return
	}

	for _, file := range files {
		// Extraer letra del nombre del disco (VDIC-X.mia)
		if strings.HasPrefix(file, "VDIC-") && len(file) >= 7 {
			letter := file[5]
			if letter >= 'A' && letter <= 'Z' {
				createdDisksMap[letter] = DiskInfo{
					Name:   file,
					Letter: letter,
					Path:   file,
				}
			}
		}
	}
}

// GetNextAvailableLetter - Obtiene la siguiente letra disponible
func GetNextAvailableLetter() byte {
	for letter := byte('A'); letter <= byte('Z'); letter++ {
		if _, exists := createdDisksMap[letter]; !exists {
			return letter
		}
	}
	return 'A' // Fallback
}

// MKDISK - Crea un nuevo disco virtual
func MKDISK(params map[string]string) (string, error) {
	// Validar parámetros obligatorios
	sizeStr, ok := params["size"]
	if !ok {
		return "", fmt.Errorf("Error: El parámetro -size es obligatorio")
	}

	// Convertir size a int64
	var size int64
	_, err := fmt.Sscanf(sizeStr, "%d", &size)
	if err != nil || size <= 0 {
		return "", fmt.Errorf("Error: El tamaño debe ser un número positivo mayor que 0")
	}

	// Obtener unit (opcional, default M)
	unit := "M"
	if u, ok := params["unit"]; ok {
		unit = strings.ToUpper(u)
		if unit != "K" && unit != "M" {
			return "", fmt.Errorf("Error: La unidad debe ser K o M")
		}
	}

	// Obtener fit (opcional, default FF)
	fit := "FF"
	if f, ok := params["fit"]; ok {
		fit = strings.ToUpper(f)
		if fit != "BF" && fit != "FF" && fit != "WF" {
			return "", fmt.Errorf("Error: El ajuste debe ser BF, FF o WF")
		}
	}

	// Calcular tamaño en bytes
	sizeBytes := utils.ConvertToBytes(size, unit)

	// Generar nombre del disco automáticamente con la siguiente letra disponible
	diskLetter := GetNextAvailableLetter()
	diskName := fmt.Sprintf("VDIC-%c.mia", diskLetter)
	diskPath := filepath.Join(".", diskName)

	// Verificar si el disco ya existe
	if utils.FileExists(diskPath) {
		return "", fmt.Errorf("Error: El disco %s ya existe", diskName)
	}

	// Crear el archivo del disco
	file, err := os.Create(diskPath)
	if err != nil {
		return "", fmt.Errorf("Error al crear el archivo del disco: %v", err)
	}
	defer file.Close()

	// Llenar el disco con ceros usando un buffer de 1024 bytes
	buffer := make([]byte, 1024)
	remaining := sizeBytes

	for remaining > 0 {
		writeSize := int64(1024)
		if remaining < writeSize {
			writeSize = remaining
		}

		n, err := file.Write(buffer[:writeSize])
		if err != nil {
			return "", fmt.Errorf("Error al escribir en el disco: %v", err)
		}
		remaining -= int64(n)
	}

	// Crear el MBR
	now := time.Now()
	mbr := models.MBR{
		MbrTamano:        sizeBytes,
		MbrFechaCreacion: now.Unix(), // Guardar como Unix timestamp
		MbrDskSignature:  utils.GenerateRandomSignature(),
		DskFit:           utils.GetFitByte(fit),
	}

	// Inicializar particiones vacías
	for i := 0; i < 4; i++ {
		mbr.MbrPartitions[i] = models.Partition{
			PartStatus:      0,
			PartType:        0,
			PartFit:         0,
			PartStart:       -1,
			PartSize:        0,
			PartCorrelative: -1,
		}
	}

	// Escribir el MBR al inicio del disco
	_, err = file.Seek(0, 0)
	if err != nil {
		return "", fmt.Errorf("❌ Error al posicionar el puntero del archivo: %v", err)
	}

	err = binary.Write(file, binary.LittleEndian, &mbr)
	if err != nil {
		return "", fmt.Errorf("❌ Error al escribir el MBR: %v", err)
	}

	// Registrar el disco creado
	createdDisksMap[diskLetter] = DiskInfo{
		Name:   diskName,
		Letter: diskLetter,
		Path:   diskPath,
	}

	// Construir mensaje de éxito
	output := fmt.Sprintf("         DISCO CREADO EXITOSAMENTE         \n")
	output += fmt.Sprintf("Nombre:     %s\n", diskName)
	output += fmt.Sprintf("Tamaño:     %d %s (%d bytes)\n", size, unit, sizeBytes)
	output += fmt.Sprintf("Ajuste:     %s\n", fit)
	output += fmt.Sprintf("Signature:  %d\n", mbr.MbrDskSignature)
	output += fmt.Sprintf("Creado:     %s\n", now.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("Ubicación:  %s\n", diskPath)
	output += fmt.Sprintf("\nDisco creado y MBR inicializado correctamente\n")

	return output, nil
}

// RMDISK - Elimina un disco virtual
func RMDISK(params map[string]string) (string, error) {
	// Validar parámetro obligatorio
	diskName, ok := params["diskname"]
	if !ok {
		return "", fmt.Errorf("Error: El parámetro -diskName es obligatorio")
	}

	// Agregar extensión si no la tiene
	if !strings.HasSuffix(diskName, ".mia") {
		diskName += ".mia"
	}

	diskPath := filepath.Join(".", diskName)

	// Verificar si el disco existe
	if !utils.FileExists(diskPath) {
		return "", fmt.Errorf("	Error: El disco %s no existe", diskName)
	}

	// Eliminar el archivo
	err := os.Remove(diskPath)
	if err != nil {
		return "", fmt.Errorf("Error al eliminar el disco: %v", err)
	}

	// Remover del mapa de discos creados
	// Extraer letra del disco
	if strings.HasPrefix(diskName, "VDIC-") && len(diskName) >= 7 {
		letter := diskName[5]
		delete(createdDisksMap, letter)
	}

	output := fmt.Sprintf("        DISCO ELIMINADO EXITOSAMENTE       \n")

	output += fmt.Sprintf("Disco eliminado: %s\n", diskName)
	output += fmt.Sprintf("Operación completada correctamente\n")

	return output, nil
}

// GetCreatedDisks - Retorna la lista de discos creados
func GetCreatedDisks() []DiskInfo {
	disks := make([]DiskInfo, 0, len(createdDisksMap))
	for _, disk := range createdDisksMap {
		disks = append(disks, disk)
	}
	return disks
}

// GetDiskCount - Retorna el número de discos creados
func GetDiskCount() int {
	return len(createdDisksMap)
}
