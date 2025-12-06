package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"os"
	"strings"
	"time"
	"unsafe"
)

// MKFS - Formatea una partición con el sistema de archivos EXT2
func MKFS(params map[string]string) (string, error) {
	// Validar parámetro obligatorio
	id, ok := params["id"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -id es obligatorio")
	}

	// Obtener type (opcional, default full)
	formatType := "full"
	if t, ok := params["type"]; ok {
		formatType = strings.ToLower(t)
		if formatType != "full" {
			return "", fmt.Errorf(" Error: El tipo debe ser 'full'")
		}
	}

	// Obtener la partición montada
	mp, err := GetMountedPartition(id)
	if err != nil {
		return "", err
	}

	// Abrir el disco
	file, err := os.OpenFile(mp.Path, os.O_RDWR, 0644)
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
	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == mp.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	if partition == nil {
		return "", fmt.Errorf(" Error: No se encontró la partición")
	}

	// Calcular número de estructuras
	numInodes, numBlocks := utils.CalculateEXT2Structures(partition.PartSize)

	if numInodes < 2 {
		return "", fmt.Errorf(" Error: La partición es muy pequeña para formatear")
	}

	// Tamaños de estructuras
	superblockSize := int64(unsafe.Sizeof(models.Superblock{}))
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	blockSize := int64(64)

	// Calcular posiciones
	bitmapInodeStart := partition.PartStart + superblockSize
	bitmapBlockStart := bitmapInodeStart + int64(numInodes)
	inodeStart := bitmapBlockStart + int64(numBlocks)
	blockStart := inodeStart + (int64(numInodes) * inodeSize)

	// Crear Superbloque
	now := time.Now()
	superblock := models.Superblock{
		S_filesystem_type:   2,
		S_inodes_count:      numInodes,
		S_blocks_count:      numBlocks,
		S_free_blocks_count: numBlocks - 2, // Se usan 2 bloques iniciales
		S_free_inodes_count: numInodes - 2, // Se usan 2 inodos iniciales
		S_mtime:             now.Unix(),
		S_umtime:            0,
		S_mnt_count:         1,
		S_magic:             0xEF53,
		S_inode_s:           int32(inodeSize),
		S_block_s:           int32(blockSize),
		S_first_ino:         2,
		S_first_blo:         2,
		S_bm_inode_start:    bitmapInodeStart,
		S_bm_block_start:    bitmapBlockStart,
		S_inode_start:       inodeStart,
		S_block_start:       blockStart,
	}

	// Escribir el superbloque
	err = utils.WriteStruct(file, &superblock, partition.PartStart)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir el superbloque: %v", err)
	}

	// Inicializar bitmap de inodos (todos en 0)
	bitmapInodes := make([]byte, numInodes)
	bitmapInodes[0] = 1 // Inodo 0 ocupado (raíz)
	bitmapInodes[1] = 1 // Inodo 1 ocupado (users.txt)
	_, err = file.WriteAt(bitmapInodes, bitmapInodeStart)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir bitmap de inodos: %v", err)
	}

	// Inicializar bitmap de bloques
	bitmapBlocks := make([]byte, numBlocks)
	bitmapBlocks[0] = 1 // Bloque 0 ocupado (carpeta raíz)
	bitmapBlocks[1] = 1 // Bloque 1 ocupado (archivo users.txt)
	_, err = file.WriteAt(bitmapBlocks, bitmapBlockStart)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir bitmap de bloques: %v", err)
	}

	// Crear inodo raíz (directorio /)
	rootInode := models.Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: now.Unix(),
		I_ctime: now.Unix(),
		I_mtime: now.Unix(),
		I_type:  0, // Carpeta
		I_perm:  [3]byte{'7', '7', '7'},
	}
	for i := 0; i < 15; i++ {
		rootInode.I_block[i] = -1
	}
	rootInode.I_block[0] = 0 // Apunta al bloque 0

	err = utils.WriteStruct(file, &rootInode, inodeStart)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir inodo raíz: %v", err)
	}

	// Crear bloque de carpeta raíz
	rootFolderBlock := models.FolderBlock{}
	// Entrada 0: .
	copy(rootFolderBlock.B_content[0].B_name[:], ".")
	rootFolderBlock.B_content[0].B_inodo = 0
	// Entrada 1: ..
	copy(rootFolderBlock.B_content[1].B_name[:], "..")
	rootFolderBlock.B_content[1].B_inodo = 0
	// Entrada 2: users.txt
	copy(rootFolderBlock.B_content[2].B_name[:], "users.txt")
	rootFolderBlock.B_content[2].B_inodo = 1
	// Entrada 3: vacía
	rootFolderBlock.B_content[3].B_inodo = -1

	err = utils.WriteStruct(file, &rootFolderBlock, blockStart)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir bloque raíz: %v", err)
	}

	// Crear archivo users.txt
	usersContent := "1,G,root\n1,U,1,root,123\n"
	usersInode := models.Inode{
		I_uid:   1,
		I_gid:   1,
		I_size:  int64(len(usersContent)),
		I_atime: now.Unix(),
		I_ctime: now.Unix(),
		I_mtime: now.Unix(),
		I_type:  1, // Archivo
		I_perm:  [3]byte{'6', '6', '4'},
	}
	for i := 0; i < 15; i++ {
		usersInode.I_block[i] = -1
	}
	usersInode.I_block[0] = 1 // Apunta al bloque 1

	err = utils.WriteStruct(file, &usersInode, inodeStart+inodeSize)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir inodo users.txt: %v", err)
	}

	// Crear bloque de contenido de users.txt
	usersBlock := models.FileBlock{}
	copy(usersBlock.B_content[:], usersContent)

	err = utils.WriteStruct(file, &usersBlock, blockStart+blockSize)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir bloque users.txt: %v", err)
	}

	output := fmt.Sprintf("      PARTICIÓN FORMATEADA (EXT2)          \n")
	output += fmt.Sprintf(" ID:              %s\n", id)
	output += fmt.Sprintf("  Partición:       %s\n", mp.Name)
	output += fmt.Sprintf("  Disco:           %s\n", mp.Path)
	output += fmt.Sprintf("  Sistema:         EXT2\n")
	output += fmt.Sprintf("  Inodos:          %d\n", numInodes)
	output += fmt.Sprintf("  Bloques:         %d\n", numBlocks)
	output += fmt.Sprintf("  Estructura:\n")
	output += fmt.Sprintf("   - Superbloque:   %d bytes\n", superblockSize)
	output += fmt.Sprintf("   - Bitmap Inodos: %d bytes\n", numInodes)
	output += fmt.Sprintf("   - Bitmap Bloques:%d bytes\n", numBlocks)
	output += fmt.Sprintf("   - Tabla Inodos:  %d bytes\n", int64(numInodes)*inodeSize)
	output += fmt.Sprintf("   - Tabla Bloques: %d bytes\n", int64(numBlocks)*blockSize)
	output += fmt.Sprintf("\n Formateo completado exitosamente\n")
	output += fmt.Sprintf(" Archivo users.txt creado con usuario root\n")

	return output, nil
}
