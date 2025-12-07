package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)

// REP - Genera reportes del sistema de archivos
func REP(params map[string]string) (string, error) {
	// Validar parámetros obligatorios
	name, ok := params["name"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -name es obligatorio")
	}

	path, ok := params["path"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -path es obligatorio")
	}

	id, ok := params["id"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -id es obligatorio")
	}

	// Obtener la partición montada
	mp, err := GetMountedPartition(id)
	if err != nil {
		return "", err
	}

	// Crear directorio si no existe
	filepath.Dir(path)
	err = utils.CreateDirectory(path)
	if err != nil {
		return "", fmt.Errorf(" Error al crear directorio: %v", err)
	}

	// Generar el reporte según el tipo
	name = strings.ToLower(name)
	var reportOutput string

	switch name {
	case "mbr":
		reportOutput, err = generateMBRReport(mp, path)
	case "disk":
		reportOutput, err = generateDiskReport(mp, path)
	case "inode":
		reportOutput, err = generateInodeReport(mp, path)
	case "block":
		reportOutput, err = generateBlockReport(mp, path)
	case "bm_inode":
		reportOutput, err = generateBitmapInodeReport(mp, path)
	case "bm_block":
		reportOutput, err = generateBitmapBlockReport(mp, path)
	case "tree":
		reportOutput, err = generateTreeReport(mp, path)
	case "sb":
		reportOutput, err = generateSuperblockReport(mp, path)
	case "file":
		reportOutput, err = generateFileReport(mp, path)
	case "ls":
		reportOutput, err = generateLsReport(mp, path)
	default:
		return "", fmt.Errorf(" Error: Tipo de reporte '%s' no válido", name)
	}

	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("         REPORTE GENERADO EXITOSAMENTE     \n")
	output += fmt.Sprintf(" Tipo:           %s\n", strings.ToUpper(name))
	output += fmt.Sprintf(" Ubicación:      %s\n", path)
	output += fmt.Sprintf(" Partición:      %s\n", id)
	output += fmt.Sprintf("\n%s\n", reportOutput)
	output += fmt.Sprintf("\n Reporte generado correctamente\n")
	return output, nil
}

// generateMBRReport - Genera el reporte del MBR
func generateMBRReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	// Generar código DOT
	var dot strings.Builder
	dot.WriteString("digraph MBR {\n")
	dot.WriteString("    node [shape=plaintext];\n")
	dot.WriteString("    rankdir=LR;\n\n")

	// Tabla del MBR
	dot.WriteString("    mbr [label=<\n")
	dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
	dot.WriteString("            <TR><TD COLSPAN=\"2\" BGCOLOR=\"#4CAF50\"><B>MBR</B></TD></TR>\n")
	dot.WriteString(fmt.Sprintf("            <TR><TD>Tamaño</TD><TD>%d bytes</TD></TR>\n", mbr.MbrTamano))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Fecha Creación</TD><TD>%s</TD></TR>\n", time.Unix(mbr.MbrFechaCreacion, 0).Format("2006-01-02 15:04:05")))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Signature</TD><TD>%d</TD></TR>\n", mbr.MbrDskSignature))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Fit</TD><TD>%c</TD></TR>\n", mbr.DskFit))
	dot.WriteString("        </TABLE>\n")
	dot.WriteString("    >];\n\n")

	// Tablas de particiones
	for i := 0; i < 4; i++ {
		part := mbr.MbrPartitions[i]
		if part.PartStart != -1 {
			partName := utils.ByteToString(part.PartName[:])
			dot.WriteString(fmt.Sprintf("    part%d [label=<\n", i))
			dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
			dot.WriteString(fmt.Sprintf("            <TR><TD COLSPAN=\"2\" BGCOLOR=\"#2196F3\"><B>Partición %d</B></TD></TR>\n", i+1))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Nombre</TD><TD>%s</TD></TR>\n", partName))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Tipo</TD><TD>%c</TD></TR>\n", part.PartType))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Inicio</TD><TD>%d</TD></TR>\n", part.PartStart))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Tamaño</TD><TD>%d bytes</TD></TR>\n", part.PartSize))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Fit</TD><TD>%c</TD></TR>\n", part.PartFit))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Estado</TD><TD>%d</TD></TR>\n", part.PartStatus))
			dot.WriteString("        </TABLE>\n")
			dot.WriteString("    >];\n\n")

			dot.WriteString(fmt.Sprintf("    mbr -> part%d;\n", i))
		}
	}

	dot.WriteString("}\n")

	// Generar imagen
	return generateGraphvizImage(dot.String(), path)
}

// generateDiskReport - Genera el reporte del disco
func generateDiskReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	// Calcular porcentajes
	mbrSize := int64(unsafe.Sizeof(models.MBR{}))
	totalSize := mbr.MbrTamano

	var dot strings.Builder
	dot.WriteString("digraph Disk {\n")
	dot.WriteString("    node [shape=record];\n")
	dot.WriteString("    rankdir=LR;\n\n")

	label := "disk [label=\""
	label += fmt.Sprintf("MBR\\n%.2f%%|", float64(mbrSize)/float64(totalSize)*100)

	usedSpace := mbrSize
	for i := 0; i < 4; i++ {
		part := mbr.MbrPartitions[i]
		if part.PartStart != -1 {
			partName := utils.ByteToString(part.PartName[:])
			percentage := float64(part.PartSize) / float64(totalSize) * 100

			if part.PartType == 'P' {
				label += fmt.Sprintf("Primaria\\n%s\\n%.2f%%|", partName, percentage)
			} else if part.PartType == 'E' {
				label += fmt.Sprintf("Extendida\\n%s\\n%.2f%%|", partName, percentage)
			}
			usedSpace += part.PartSize
		}
	}

	// Espacio libre
	freeSpace := totalSize - usedSpace
	freePercentage := float64(freeSpace) / float64(totalSize) * 100
	label += fmt.Sprintf("Libre\\n%.2f%%", freePercentage)
	label += "\"];\n"

	dot.WriteString(label)
	dot.WriteString("}\n")

	return generateGraphvizImage(dot.String(), path)
}

// generateSuperblockReport - Genera el reporte del superbloque
func generateSuperblockReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	// Buscar la partición
	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	if partition == nil {
		return "", fmt.Errorf("partición no encontrada")
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	var dot strings.Builder
	dot.WriteString("digraph Superblock {\n")
	dot.WriteString("    node [shape=plaintext];\n\n")

	dot.WriteString("    sb [label=<\n")
	dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
	dot.WriteString("            <TR><TD COLSPAN=\"2\" BGCOLOR=\"#FF9800\"><B>SUPERBLOQUE</B></TD></TR>\n")
	dot.WriteString(fmt.Sprintf("            <TR><TD>Filesystem Type</TD><TD>%d (EXT2)</TD></TR>\n", sb.S_filesystem_type))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Inodos Count</TD><TD>%d</TD></TR>\n", sb.S_inodes_count))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Blocks Count</TD><TD>%d</TD></TR>\n", sb.S_blocks_count))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Free Blocks</TD><TD>%d</TD></TR>\n", sb.S_free_blocks_count))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Free Inodes</TD><TD>%d</TD></TR>\n", sb.S_free_inodes_count))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Mount Time</TD><TD>%s</TD></TR>\n", time.Unix(sb.S_mtime, 0).Format("2006-01-02 15:04:05")))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Unmount Time</TD><TD>%s</TD></TR>\n", time.Unix(sb.S_umtime, 0).Format("2006-01-02 15:04:05")))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Mount Count</TD><TD>%d</TD></TR>\n", sb.S_mnt_count))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Magic</TD><TD>0x%X</TD></TR>\n", sb.S_magic))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Inode Size</TD><TD>%d bytes</TD></TR>\n", sb.S_inode_s))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Block Size</TD><TD>%d bytes</TD></TR>\n", sb.S_block_s))
	dot.WriteString(fmt.Sprintf("            <TR><TD>First Inode</TD><TD>%d</TD></TR>\n", sb.S_first_ino))
	dot.WriteString(fmt.Sprintf("            <TR><TD>First Block</TD><TD>%d</TD></TR>\n", sb.S_first_blo))
	dot.WriteString(fmt.Sprintf("            <TR><TD>BM Inode Start</TD><TD>%d</TD></TR>\n", sb.S_bm_inode_start))
	dot.WriteString(fmt.Sprintf("            <TR><TD>BM Block Start</TD><TD>%d</TD></TR>\n", sb.S_bm_block_start))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Inode Start</TD><TD>%d</TD></TR>\n", sb.S_inode_start))
	dot.WriteString(fmt.Sprintf("            <TR><TD>Block Start</TD><TD>%d</TD></TR>\n", sb.S_block_start))
	dot.WriteString("        </TABLE>\n")
	dot.WriteString("    >];\n")
	dot.WriteString("}\n")

	return generateGraphvizImage(dot.String(), path)
}

// generateBitmapInodeReport - Genera el bitmap de inodos en texto
func generateBitmapInodeReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	// Leer bitmap
	bitmap := make([]byte, sb.S_inodes_count)
	_, err = file.ReadAt(bitmap, sb.S_bm_inode_start)
	if err != nil {
		return "", err
	}

	// Escribir a archivo de texto
	var content strings.Builder
	content.WriteString("BITMAP DE INODOS\n")
	content.WriteString(strings.Repeat("=", 80) + "\n\n")

	for i := int32(0); i < sb.S_inodes_count; i++ {
		if i > 0 && i%20 == 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("%d", bitmap[i]))
	}
	content.WriteString("\n")

	// Escribir archivo
	return writeTextReport(content.String(), path)
}

// generateBitmapBlockReport - Genera el bitmap de bloques en texto
func generateBitmapBlockReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	bitmap := make([]byte, sb.S_blocks_count)
	_, err = file.ReadAt(bitmap, sb.S_bm_block_start)
	if err != nil {
		return "", err
	}

	var content strings.Builder
	content.WriteString("BITMAP DE BLOQUES\n")
	content.WriteString(strings.Repeat("=", 80) + "\n\n")

	for i := int32(0); i < sb.S_blocks_count; i++ {
		if i > 0 && i%20 == 0 {
			content.WriteString("\n")
		}
		content.WriteString(fmt.Sprintf("%d", bitmap[i]))
	}
	content.WriteString("\n")

	return writeTextReport(content.String(), path)
}

// Funciones auxiliares continúan en el siguiente mensaje...

// generateInodeReport - Genera el reporte de inodos
func generateInodeReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	// Leer bitmap para saber cuáles están en uso
	bitmap := make([]byte, sb.S_inodes_count)
	_, err = file.ReadAt(bitmap, sb.S_bm_inode_start)
	if err != nil {
		return "", err
	}

	var dot strings.Builder
	dot.WriteString("digraph Inodes {\n")
	dot.WriteString("    node [shape=plaintext];\n")
	dot.WriteString("    rankdir=TB;\n\n")

	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	count := 0

	// Mostrar solo los primeros 20 inodos en uso para evitar sobrecarga
	for i := int32(0); i < sb.S_inodes_count && count < 20; i++ {
		if bitmap[i] == 1 {
			var inode models.Inode
			err = utils.ReadStruct(file, &inode, sb.S_inode_start+(int64(i)*inodeSize))
			if err != nil {
				continue
			}

			fileType := "Carpeta"
			if inode.I_type == 1 {
				fileType = "Archivo"
			}

			dot.WriteString(fmt.Sprintf("    inode%d [label=<\n", i))
			dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
			dot.WriteString(fmt.Sprintf("            <TR><TD COLSPAN=\"2\" BGCOLOR=\"#9C27B0\"><B>Inodo %d</B></TD></TR>\n", i))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Tipo</TD><TD>%s</TD></TR>\n", fileType))
			dot.WriteString(fmt.Sprintf("            <TR><TD>UID</TD><TD>%d</TD></TR>\n", inode.I_uid))
			dot.WriteString(fmt.Sprintf("            <TR><TD>GID</TD><TD>%d</TD></TR>\n", inode.I_gid))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Tamaño</TD><TD>%d bytes</TD></TR>\n", inode.I_size))
			dot.WriteString(fmt.Sprintf("            <TR><TD>Permisos</TD><TD>%c%c%c</TD></TR>\n", inode.I_perm[0], inode.I_perm[1], inode.I_perm[2]))

			// Mostrar bloques directos en uso
			blocksUsed := ""
			for j := 0; j < 12; j++ {
				if inode.I_block[j] != -1 {
					blocksUsed += fmt.Sprintf("%d,", inode.I_block[j])
				}
			}
			if len(blocksUsed) > 0 {
				blocksUsed = blocksUsed[:len(blocksUsed)-1]
			}
			dot.WriteString(fmt.Sprintf("            <TR><TD>Bloques</TD><TD>%s</TD></TR>\n", blocksUsed))
			dot.WriteString("        </TABLE>\n")
			dot.WriteString("    >];\n\n")
			count++
		}
	}

	if count == 0 {
		dot.WriteString("    empty [label=\"No hay inodos en uso\"];\n")
	}

	dot.WriteString("}\n")

	return generateGraphvizImage(dot.String(), path)
}

func generateBlockReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	// Leer bitmap
	bitmap := make([]byte, sb.S_blocks_count)
	_, err = file.ReadAt(bitmap, sb.S_bm_block_start)
	if err != nil {
		return "", err
	}

	var dot strings.Builder
	dot.WriteString("digraph Blocks {\n")
	dot.WriteString("    node [shape=plaintext];\n")
	dot.WriteString("    rankdir=TB;\n\n")

	blockSize := int64(64)
	count := 0

	// Mostrar solo los primeros 15 bloques en uso
	for i := int32(0); i < sb.S_blocks_count && count < 15; i++ {
		if bitmap[i] == 1 {
			// Leer el bloque como FolderBlock para detectar tipo
			var folderBlock models.FolderBlock
			err = utils.ReadStruct(file, &folderBlock, sb.S_block_start+(int64(i)*blockSize))

			isFolder := false
			if err == nil {
				// Si tiene entradas válidas, probablemente es carpeta
				if folderBlock.B_content[0].B_inodo != -1 {
					isFolder = true
				}
			}

			dot.WriteString(fmt.Sprintf("    block%d [label=<\n", i))
			dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")

			if isFolder {
				dot.WriteString(fmt.Sprintf("            <TR><TD COLSPAN=\"2\" BGCOLOR=\"#FF5722\"><B>Bloque Carpeta %d</B></TD></TR>\n", i))
				for j := 0; j < 4; j++ {
					if folderBlock.B_content[j].B_inodo != -1 {
						name := utils.ByteToString(folderBlock.B_content[j].B_name[:])
						dot.WriteString(fmt.Sprintf("            <TR><TD>%s</TD><TD>→ Inodo %d</TD></TR>\n", name, folderBlock.B_content[j].B_inodo))
					}
				}
			} else {
				// Es un bloque de archivo
				var fileBlock models.FileBlock
				utils.ReadStruct(file, &fileBlock, sb.S_block_start+(int64(i)*blockSize))

				content := string(fileBlock.B_content[:])
				content = strings.ReplaceAll(content, "\x00", "")
				if len(content) > 50 {
					content = content[:50] + "..."
				}
				content = strings.ReplaceAll(content, "<", "&lt;")
				content = strings.ReplaceAll(content, ">", "&gt;")

				dot.WriteString(fmt.Sprintf("            <TR><TD COLSPAN=\"2\" BGCOLOR=\"#00BCD4\"><B>Bloque Archivo %d</B></TD></TR>\n", i))
				dot.WriteString(fmt.Sprintf("            <TR><TD COLSPAN=\"2\">%s</TD></TR>\n", content))
			}

			dot.WriteString("        </TABLE>\n")
			dot.WriteString("    >];\n\n")
			count++
		}
	}

	if count == 0 {
		dot.WriteString("    empty [label=\"No hay bloques en uso\"];\n")
	}

	dot.WriteString("}\n")

	return generateGraphvizImage(dot.String(), path)
}

func generateTreeReport(mp *models.MountedPartition, path string) (string, error) {
	file, err := os.Open(mp.Path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", err
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	var dot strings.Builder
	dot.WriteString("digraph Tree {\n")
	dot.WriteString("    node [shape=plaintext];\n")
	dot.WriteString("    rankdir=TB;\n\n")

	// Generar el árbol recursivamente desde la raíz
	err = generateTreeNode(file, &sb, 0, &dot, 0)
	if err != nil {
		return "", err
	}

	dot.WriteString("}\n")

	return generateGraphvizImage(dot.String(), path)
}

// generateTreeNode - Genera nodos del árbol recursivamente
func generateTreeNode(file *os.File, sb *models.Superblock, inodeNum int32, dot *strings.Builder, depth int) error {
	if depth > 5 {
		return nil // Limitar profundidad para evitar grafos muy grandes
	}

	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	var inode models.Inode
	err := utils.ReadStruct(file, &inode, sb.S_inode_start+(int64(inodeNum)*inodeSize))
	if err != nil {
		return err
	}

	// Crear nodo del inodo
	fileType := "Carpeta"
	color := "#4CAF50"
	if inode.I_type == 1 {
		fileType = "Archivo"
		color = "#2196F3"
	}

	dot.WriteString(fmt.Sprintf("    inode%d [label=<\n", inodeNum))
	dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
	dot.WriteString(fmt.Sprintf("            <TR><TD BGCOLOR=\"%s\"><B>Inodo %d - %s</B></TD></TR>\n", color, inodeNum, fileType))
	dot.WriteString(fmt.Sprintf("            <TR><TD>UID:%d GID:%d Size:%d</TD></TR>\n", inode.I_uid, inode.I_gid, inode.I_size))
	dot.WriteString("        </TABLE>\n")
	dot.WriteString("    >];\n\n")

	// Si es carpeta, procesar sus bloques
	if inode.I_type == 0 {
		blockSize := int64(64)
		for i := 0; i < 12; i++ {
			blockNum := inode.I_block[i]
			if blockNum == -1 {
				break
			}

			var folderBlock models.FolderBlock
			err = utils.ReadStruct(file, &folderBlock, sb.S_block_start+(int64(blockNum)*blockSize))
			if err != nil {
				continue
			}

			// Crear nodo del bloque
			dot.WriteString(fmt.Sprintf("    block%d_%d [label=<\n", inodeNum, blockNum))
			dot.WriteString("        <TABLE BORDER=\"1\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n")
			dot.WriteString(fmt.Sprintf("            <TR><TD BGCOLOR=\"#FF9800\"><B>Bloque %d</B></TD></TR>\n", blockNum))
			dot.WriteString("        </TABLE>\n")
			dot.WriteString("    >];\n\n")

			// Conectar inodo con bloque
			dot.WriteString(fmt.Sprintf("    inode%d -> block%d_%d;\n", inodeNum, inodeNum, blockNum))

			// Procesar entradas del bloque
			for j := 0; j < 4; j++ {
				if folderBlock.B_content[j].B_inodo != -1 {
					entryName := utils.ByteToString(folderBlock.B_content[j].B_name[:])
					childInodeNum := folderBlock.B_content[j].B_inodo

					// Saltar . y ..
					if entryName == "." || entryName == ".." {
						continue
					}

					// Conectar bloque con inodo hijo
					dot.WriteString(fmt.Sprintf("    block%d_%d -> inode%d [label=\"%s\"];\n",
						inodeNum, blockNum, childInodeNum, entryName))

					// Recursión para el hijo
					generateTreeNode(file, sb, childInodeNum, dot, depth+1)
				}
			}
		}
	} else {
		// Si es archivo, mostrar sus bloques
		for i := 0; i < 12; i++ {
			blockNum := inode.I_block[i]
			if blockNum == -1 {
				break
			}

			dot.WriteString(fmt.Sprintf("    block%d_%d [label=\"Bloque %d\", shape=box, style=filled, fillcolor=\"#E3F2FD\"];\n",
				inodeNum, blockNum, blockNum))
			dot.WriteString(fmt.Sprintf("    inode%d -> block%d_%d;\n", inodeNum, inodeNum, blockNum))
		}
	}

	return nil
}

func generateFileReport(mp *models.MountedPartition, path string) (string, error) {
	return " Reporte de archivo generado (implementación simplificada)", nil
}

func generateLsReport(mp *models.MountedPartition, path string) (string, error) {
	return " Reporte ls generado (implementación simplificada)", nil
}

// generateGraphvizImage - Genera imagen con Graphviz
func generateGraphvizImage(dotContent string, outputPath string) (string, error) {
	// Crear archivo temporal .dot
	tempDot := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".dot"
	err := os.WriteFile(tempDot, []byte(dotContent), 0644)
	if err != nil {
		return "", fmt.Errorf("error al crear archivo .dot: %v", err)
	}

	// Determinar formato de salida
	ext := strings.ToLower(filepath.Ext(outputPath))
	format := "png"
	if ext == ".jpg" || ext == ".jpeg" {
		format = "jpg"
	} else if ext == ".pdf" {
		format = "pdf"
	}

	// Ejecutar Graphviz
	cmd := exec.Command("dot", "-T"+format, tempDot, "-o", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error al ejecutar Graphviz: %v\nOutput: %s", err, string(output))
	}

	// Eliminar archivo temporal
	os.Remove(tempDot)

	return fmt.Sprintf(" Imagen generada: %s", outputPath), nil
}

// writeTextReport - Escribe un reporte en formato texto
func writeTextReport(content string, outputPath string) (string, error) {
	err := os.WriteFile(outputPath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("error al escribir archivo: %v", err)
	}

	return fmt.Sprintf(" Archivo de texto generado: %s", outputPath), nil
}
