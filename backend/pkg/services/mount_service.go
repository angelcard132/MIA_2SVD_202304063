package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"os"
	"strings"
)

// Mapa de particiones montadas: ID -> MountedPartition
var mountedPartitions = make(map[string]*models.MountedPartition)

// Contador de particiones por disco: Letra -> Contador
var diskPartitionCounters = make(map[byte]int)

// Carnet del estudiante (últimos 2 dígitos en hex)
const CARNET = "202304063"

// MOUNT - Monta una partición en el sistema
func MOUNT(params map[string]string) (string, error) {
	// Validar parámetros obligatorios
	diskPath, ok := params["path"]
	if !ok {
		return "", fmt.Errorf("Error: El parámetro -path es obligatorio")
	}

	partName, ok := params["name"]
	if !ok {
		return "", fmt.Errorf("Error: El parámetro -name es obligatorio")
	}

	// Agregar extensión si no la tiene
	if !strings.HasSuffix(diskPath, ".mia") {
		diskPath += ".mia"
	}

	// Verificar que el disco existe
	if !utils.FileExists(diskPath) {
		return "", fmt.Errorf("Error: El disco %s no existe", diskPath)
	}

	// Abrir el disco
	file, err := os.OpenFile(diskPath, os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf(" Error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Leer el MBR
	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", fmt.Errorf(" Error al leer el MBR: %v", err)
	}

	// Buscar la partición
	partIndex := -1
	for i := 0; i < 4; i++ {
		existingName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if existingName == partName && mbr.MbrPartitions[i].PartStart != -1 {
			partIndex = i
			break
		}
	}

	if partIndex == -1 {
		return "", fmt.Errorf(" Error: No existe una partición con el nombre '%s' en el disco", partName)
	}

	// Verificar que sea primaria (según el documento, solo primarias se montan)
	if mbr.MbrPartitions[partIndex].PartType != 'P' {
		return "", fmt.Errorf(" Error: Solo se pueden montar particiones primarias")
	}

	// Verificar si ya está montada
	existingID := utils.ByteToString(mbr.MbrPartitions[partIndex].PartId[:])
	if existingID != "" && mbr.MbrPartitions[partIndex].PartStatus == 1 {
		return "", fmt.Errorf(" Error: La partición '%s' ya está montada con ID: %s", partName, existingID)
	}

	// Extraer letra del disco (VDIC-X.mia)
	diskLetter := byte('A')
	if strings.HasPrefix(diskPath, "VDIC-") && len(diskPath) >= 7 {
		diskLetter = diskPath[5]
	}

	// Incrementar contador de particiones para este disco
	diskPartitionCounters[diskLetter]++
	partNum := diskPartitionCounters[diskLetter]

	// Generar ID de la partición
	// Formato: XX + N + L (XX = últimos 2 dígitos del carnet en hex, N = número partición, L = letra disco)
	id := utils.GeneratePartitionID(CARNET, partNum, diskLetter)

	// Actualizar la partición en el MBR
	mbr.MbrPartitions[partIndex].PartStatus = 1
	mbr.MbrPartitions[partIndex].PartCorrelative = int32(partNum)
	copy(mbr.MbrPartitions[partIndex].PartId[:], []byte(id))

	// Escribir el MBR actualizado
	err = utils.WriteStruct(file, &mbr, 0)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir el MBR: %v", err)
	}

	// Agregar a la lista de particiones montadas
	mounted := &models.MountedPartition{
		Id:         id,
		Path:       diskPath,
		Name:       partName,
		Part:       partNum,
		DiskLetter: diskLetter,
	}
	mountedPartitions[id] = mounted

	output := fmt.Sprintf("       PARTICIÓN MONTADA EXITOSAMENTE     \n")

	output += fmt.Sprintf(" ID:         %s\n", id)
	output += fmt.Sprintf(" Nombre:     %s\n", partName)
	output += fmt.Sprintf(" Disco:      %s\n", diskPath)
	output += fmt.Sprintf(" Partición:  %d\n", partNum)
	output += fmt.Sprintf(" Letra:      %c\n", diskLetter)
	output += fmt.Sprintf("\n Partición montada correctamente\n")
	output += fmt.Sprintf(" Usa el ID '%s' para operaciones futuras\n", id)

	return output, nil
}

// MOUNTED - Muestra todas las particiones montadas
func MOUNTED(params map[string]string) (string, error) {
	if len(mountedPartitions) == 0 {
		return "\n No hay particiones montadas actualmente\n", nil
	}

	output := fmt.Sprintf("       PARTICIONES MONTADAS                \n")

	output += fmt.Sprintf("%-8s %-15s %-20s %-6s\n", "ID", "Nombre", "Disco", "Part#")
	output += fmt.Sprintf("─────────────────────────────────────────────\n")

	for _, mp := range mountedPartitions {
		output += fmt.Sprintf("%-8s %-15s %-20s %-6d\n",
			mp.Id,
			mp.Name,
			mp.Path,
			mp.Part)
	}

	output += fmt.Sprintf("\n Total de particiones montadas: %d\n", len(mountedPartitions))

	return output, nil
}

// GetMountedPartition - Obtiene una partición montada por su ID
func GetMountedPartition(id string) (*models.MountedPartition, error) {
	mp, exists := mountedPartitions[id]
	if !exists {
		return nil, fmt.Errorf(" Error: No existe una partición montada con el ID '%s'", id)
	}
	return mp, nil
}

// GetMountedPartitions - Retorna todas las particiones montadas
func GetMountedPartitions() map[string]*models.MountedPartition {
	return mountedPartitions
}

// UnmountPartition - Desmonta una partición (para uso futuro)
func UnmountPartition(id string) error {
	mp, exists := mountedPartitions[id]
	if !exists {
		return fmt.Errorf(" Error: No existe una partición montada con el ID '%s'", id)
	}

	// Abrir el disco
	file, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf(" Error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Leer el MBR
	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return fmt.Errorf(" Error al leer el MBR: %v", err)
	}

	// Buscar y desmontar la partición
	for i := 0; i < 4; i++ {
		existingName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if existingName == mp.Name {
			mbr.MbrPartitions[i].PartStatus = 0
			break
		}
	}

	// Escribir el MBR actualizado
	err = utils.WriteStruct(file, &mbr, 0)
	if err != nil {
		return fmt.Errorf(" Error al escribir el MBR: %v", err)
	}

	// Remover de la lista
	delete(mountedPartitions, id)

	return nil
}
