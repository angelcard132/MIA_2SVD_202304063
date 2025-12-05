package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

// FDISK - Administra particiones en un disco
func FDISK(params map[string]string) (string, error) {
	// Validar parámetros obligatorios
	diskName, ok := params["diskname"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -diskName es obligatorio")
	}

	partName, ok := params["name"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -name es obligatorio")
	}

	sizeStr, ok := params["size"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -size es obligatorio")
	}

	// Convertir size
	var size int64
	_, err := fmt.Sscanf(sizeStr, "%d", &size)
	if err != nil || size <= 0 {
		return "", fmt.Errorf(" Error: El tamaño debe ser un número positivo mayor que 0")
	}

	// Obtener unit (opcional, default K)
	unit := "K"
	if u, ok := params["unit"]; ok {
		unit = strings.ToUpper(u)
		if unit != "B" && unit != "K" && unit != "M" {
			return "", fmt.Errorf(" Error: La unidad debe ser B, K o M")
		}
	}

	// Obtener type (opcional, default P)
	partType := "P"
	if t, ok := params["type"]; ok {
		partType = strings.ToUpper(t)
		if partType != "P" && partType != "E" && partType != "L" {
			return "", fmt.Errorf(" Error: El tipo debe ser P, E o L")
		}
	}

	// Obtener fit (opcional, default WF)
	fit := "WF"
	if f, ok := params["fit"]; ok {
		fit = strings.ToUpper(f)
		if fit != "BF" && fit != "FF" && fit != "WF" {
			return "", fmt.Errorf(" Error: El ajuste debe ser BF, FF o WF")
		}
	}

	// Agregar extensión si no la tiene
	if !strings.HasSuffix(diskName, ".mia") {
		diskName += ".mia"
	}

	// Verificar que el disco existe
	if !utils.FileExists(diskName) {
		return "", fmt.Errorf(" Error: El disco %s no existe", diskName)
	}

	// Calcular tamaño en bytes
	sizeBytes := utils.ConvertToBytes(size, unit)

	// Abrir el disco
	file, err := os.OpenFile(diskName, os.O_RDWR, 0644)
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

	// Verificar que el nombre de la partición no se repita
	for i := 0; i < 4; i++ {
		existingName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if existingName == partName && mbr.MbrPartitions[i].PartStart != -1 {
			return "", fmt.Errorf(" Error: Ya existe una partición con el nombre '%s'", partName)
		}
	}

	// Crear la partición según el tipo
	switch partType {
	case "P":
		return createPrimaryPartition(file, &mbr, partName, sizeBytes, fit)
	case "E":
		return createExtendedPartition(file, &mbr, partName, sizeBytes, fit)
	case "L":
		return createLogicalPartition(file, &mbr, partName, sizeBytes, fit)
	default:
		return "", fmt.Errorf(" Error: Tipo de partición no válido")
	}
}

// createPrimaryPartition - Crea una partición primaria
func createPrimaryPartition(file *os.File, mbr *models.MBR, name string, size int64, fit string) (string, error) {
	// Contar particiones primarias y extendidas
	primCount := 0
	extCount := 0
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartStart != -1 {
			if mbr.MbrPartitions[i].PartType == 'P' {
				primCount++
			} else if mbr.MbrPartitions[i].PartType == 'E' {
				extCount++
			}
		}
	}

	// Validar que no exceda el límite
	if primCount+extCount >= 4 {
		return "", fmt.Errorf(" Error: Ya existen 4 particiones (límite máximo)")
	}

	// Buscar espacio disponible
	mbrSize := int64(unsafe.Sizeof(models.MBR{}))
	start, err := findSpace(mbr, size, fit, mbrSize)
	if err != nil {
		return "", err
	}

	// Buscar slot libre en el MBR
	slotIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartStart == -1 {
			slotIndex = i
			break
		}
	}

	if slotIndex == -1 {
		return "", fmt.Errorf(" Error: No hay slots disponibles en el MBR")
	}

	// Crear la partición
	partition := models.Partition{
		PartStatus:      0,
		PartType:        'P',
		PartFit:         utils.GetFitByte(fit),
		PartStart:       start,
		PartSize:        size,
		PartCorrelative: -1,
	}
	copy(partition.PartName[:], []byte(name))

	// Asignar al MBR
	mbr.MbrPartitions[slotIndex] = partition

	// Escribir el MBR actualizado
	err = utils.WriteStruct(file, mbr, 0)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir el MBR: %v", err)
	}

	output := fmt.Sprintf("      PARTICIÓN PRIMARIA CREADA            \n")
	output += fmt.Sprintf(" Nombre:     %s\n", name)
	output += fmt.Sprintf(" Tamaño:     %d bytes\n", size)
	output += fmt.Sprintf(" Ajuste:     %s\n", fit)
	output += fmt.Sprintf(" Inicio:     %d\n", start)
	output += fmt.Sprintf(" Partición primaria creada exitosamente\n")
	return output, nil
}

// createExtendedPartition - Crea una partición extendida
func createExtendedPartition(file *os.File, mbr *models.MBR, name string, size int64, fit string) (string, error) {
	// Verificar que no exista otra partición extendida
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartType == 'E' && mbr.MbrPartitions[i].PartStart != -1 {
			return "", fmt.Errorf(" Error: Ya existe una partición extendida en el disco")
		}
	}

	// Contar particiones
	partCount := 0
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartStart != -1 {
			partCount++
		}
	}

	if partCount >= 4 {
		return "", fmt.Errorf(" Error: Ya existen 4 particiones (límite máximo)")
	}

	// Buscar espacio disponible
	mbrSize := int64(unsafe.Sizeof(models.MBR{}))
	start, err := findSpace(mbr, size, fit, mbrSize)
	if err != nil {
		return "", err
	}

	// Buscar slot libre
	slotIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartStart == -1 {
			slotIndex = i
			break
		}
	}

	if slotIndex == -1 {
		return "", fmt.Errorf(" Error: No hay slots disponibles en el MBR")
	}

	// Crear la partición extendida
	partition := models.Partition{
		PartStatus:      0,
		PartType:        'E',
		PartFit:         utils.GetFitByte(fit),
		PartStart:       start,
		PartSize:        size,
		PartCorrelative: -1,
	}
	copy(partition.PartName[:], []byte(name))

	mbr.MbrPartitions[slotIndex] = partition

	// Escribir el MBR actualizado
	err = utils.WriteStruct(file, mbr, 0)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir el MBR: %v", err)
	}

	// Crear el primer EBR vacío
	ebr := models.EBR{
		PartMount: 0,
		PartFit:   utils.GetFitByte(fit),
		PartStart: -1,
		PartSize:  0,
		PartNext:  -1,
	}

	err = utils.WriteStruct(file, &ebr, start)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir el EBR: %v", err)
	}

	output := fmt.Sprintf("      PARTICIÓN EXTENDIDA CREADA           \n")
	output += fmt.Sprintf(" Nombre:     %s\n", name)
	output += fmt.Sprintf(" Tamaño:     %d bytes\n", size)
	output += fmt.Sprintf(" Ajuste:     %s\n", fit)
	output += fmt.Sprintf(" Inicio:     %d\n", start)
	output += fmt.Sprintf(" Partición extendida creada exitosamente\n")

	return output, nil
}

// createLogicalPartition - Crea una partición lógica
func createLogicalPartition(file *os.File, mbr *models.MBR, name string, size int64, fit string) (string, error) {
	// Buscar la partición extendida
	extIndex := -1
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartType == 'E' && mbr.MbrPartitions[i].PartStart != -1 {
			extIndex = i
			break
		}
	}

	if extIndex == -1 {
		return "", fmt.Errorf(" Error: No existe una partición extendida donde crear la partición lógica")
	}

	extPartition := mbr.MbrPartitions[extIndex]
	ebrSize := int64(unsafe.Sizeof(models.EBR{}))

	// Leer el primer EBR
	var ebr models.EBR
	err := utils.ReadStruct(file, &ebr, extPartition.PartStart)
	if err != nil {
		return "", fmt.Errorf(" Error al leer el EBR: %v", err)
	}

	// Si el primer EBR está vacío, crear aquí la primera lógica
	if ebr.PartStart == -1 {
		ebr.PartMount = 0
		ebr.PartFit = utils.GetFitByte(fit)
		ebr.PartStart = extPartition.PartStart + ebrSize
		ebr.PartSize = size
		ebr.PartNext = -1
		copy(ebr.PartName[:], []byte(name))

		// Verificar que cabe en la extendida
		if ebr.PartStart+size > extPartition.PartStart+extPartition.PartSize {
			return "", fmt.Errorf(" Error: No hay suficiente espacio en la partición extendida")
		}

		err = utils.WriteStruct(file, &ebr, extPartition.PartStart)
		if err != nil {
			return "", fmt.Errorf(" Error al escribir el EBR: %v", err)
		}

		output := fmt.Sprintf("      PARTICIÓN LÓGICA CREADA           \n")
		output += fmt.Sprintf(" Nombre:     %s\n", name)
		output += fmt.Sprintf(" Tamaño:     %d bytes\n", size)
		output += fmt.Sprintf(" Ajuste:     %s\n", fit)
		output += fmt.Sprintf(" Nombre:     %s\n", name)
		output += fmt.Sprintf(" Tamaño:     %d bytes\n", size)
		output += fmt.Sprintf(" Ajuste:     %s\n", fit)
		output += fmt.Sprintf(" Primera partición lógica creada exitosamente\n")

		return output, nil
	}

	// Recorrer los EBRs hasta encontrar el último
	currentPos := extPartition.PartStart
	for ebr.PartNext != -1 {
		currentPos = ebr.PartNext
		err = utils.ReadStruct(file, &ebr, currentPos)
		if err != nil {
			return "", fmt.Errorf(" Error al leer el EBR: %v", err)
		}
	}

	// Calcular posición del nuevo EBR
	newEBRPos := currentPos + ebrSize + ebr.PartSize

	// Verificar espacio
	if newEBRPos+ebrSize+size > extPartition.PartStart+extPartition.PartSize {
		return "", fmt.Errorf(" Error: No hay suficiente espacio en la partición extendida")
	}

	// Actualizar el EBR actual para apuntar al nuevo
	ebr.PartNext = newEBRPos
	err = utils.WriteStruct(file, &ebr, currentPos)
	if err != nil {
		return "", fmt.Errorf(" Error al actualizar el EBR anterior: %v", err)
	}

	// Crear el nuevo EBR
	newEBR := models.EBR{
		PartMount: 0,
		PartFit:   utils.GetFitByte(fit),
		PartStart: newEBRPos + ebrSize,
		PartSize:  size,
		PartNext:  -1,
	}
	copy(newEBR.PartName[:], []byte(name))

	err = utils.WriteStruct(file, &newEBR, newEBRPos)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir el nuevo EBR: %v", err)
	}

	output := fmt.Sprintf("      PARTICIÓN LÓGICA CREADA           \n")
	output += fmt.Sprintf(" Nombre:     %s\n", name)
	output += fmt.Sprintf(" Tamaño:     %d bytes\n", size)
	output += fmt.Sprintf(" Ajuste:     %s\n", fit)
	output += fmt.Sprintf(" Nombre:     %s\n", name)
	output += fmt.Sprintf(" Tamaño:     %d bytes\n", size)
	output += fmt.Sprintf(" Ajuste:     %s\n", fit)
	output += fmt.Sprintf(" Partición lógica creada exitosamente\n")
	return output, nil
}

// findSpace - Encuentra espacio disponible en el disco
func findSpace(mbr *models.MBR, size int64, fit string, mbrSize int64) (int64, error) {
	// Crear lista de espacios ocupados
	type Space struct {
		start int64
		end   int64
	}

	occupied := []Space{}

	// El MBR siempre ocupa el inicio
	occupied = append(occupied, Space{start: 0, end: mbrSize})

	// Agregar particiones existentes
	for i := 0; i < 4; i++ {
		if mbr.MbrPartitions[i].PartStart != -1 && mbr.MbrPartitions[i].PartSize > 0 {
			occupied = append(occupied, Space{
				start: mbr.MbrPartitions[i].PartStart,
				end:   mbr.MbrPartitions[i].PartStart + mbr.MbrPartitions[i].PartSize,
			})
		}
	}

	// Ordenar espacios ocupados por inicio (bubble sort simple)
	for i := 0; i < len(occupied); i++ {
		for j := i + 1; j < len(occupied); j++ {
			if occupied[j].start < occupied[i].start {
				occupied[i], occupied[j] = occupied[j], occupied[i]
			}
		}
	}

	// Fusionar espacios superpuestos o contiguos
	merged := []Space{occupied[0]}
	for i := 1; i < len(occupied); i++ {
		last := &merged[len(merged)-1]
		current := occupied[i]

		if current.start <= last.end {
			// Espacios superpuestos o contiguos, fusionar
			if current.end > last.end {
				last.end = current.end
			}
		} else {
			// Espacio separado, agregar nuevo
			merged = append(merged, current)
		}
	}

	// Encontrar espacios libres entre particiones
	freeSpaces := []Space{}

	for i := 0; i < len(merged)-1; i++ {
		gapStart := merged[i].end
		gapEnd := merged[i+1].start
		gapSize := gapEnd - gapStart

		if gapSize >= size {
			freeSpaces = append(freeSpaces, Space{
				start: gapStart,
				end:   gapEnd,
			})
		}
	}

	// Agregar espacio al final del disco
	if len(merged) > 0 {
		lastEnd := merged[len(merged)-1].end
		remainingSpace := mbr.MbrTamano - lastEnd

		if remainingSpace >= size {
			freeSpaces = append(freeSpaces, Space{
				start: lastEnd,
				end:   mbr.MbrTamano,
			})
		}
	} else {
		// No hay particiones, todo el espacio después del MBR está libre
		if mbr.MbrTamano-mbrSize >= size {
			freeSpaces = append(freeSpaces, Space{
				start: mbrSize,
				end:   mbr.MbrTamano,
			})
		}
	}

	if len(freeSpaces) == 0 {
		return -1, fmt.Errorf(" Error: No hay suficiente espacio contiguo disponible (se necesitan %d bytes)", size)
	}

	// Aplicar estrategia de ajuste
	selectedSpace := freeSpaces[0]
	fitUpper := strings.ToUpper(fit)

	switch fitUpper {
	case "FF", "F": // First Fit - el primero disponible
		selectedSpace = freeSpaces[0]

	case "BF", "B": // Best Fit - el más pequeño que cabe
		selectedSpace = freeSpaces[0]
		minWaste := (freeSpaces[0].end - freeSpaces[0].start) - size

		for _, space := range freeSpaces[1:] {
			waste := (space.end - space.start) - size
			if waste < minWaste {
				minWaste = waste
				selectedSpace = space
			}
		}

	case "WF", "W": // Worst Fit - el más grande disponible
		selectedSpace = freeSpaces[0]
		maxSize := freeSpaces[0].end - freeSpaces[0].start

		for _, space := range freeSpaces[1:] {
			spaceSize := space.end - space.start
			if spaceSize > maxSize {
				maxSize = spaceSize
				selectedSpace = space
			}
		}
	}

	return selectedSpace.start, nil
}
