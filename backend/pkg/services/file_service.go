package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"unsafe"
)

// MKFILE - Crea un archivo en el sistema de archivos
func MKFILE(params map[string]string) (string, error) {
	// Verificar sesión activa
	if activeSession == nil {
		return "", fmt.Errorf(" Error: No hay una sesión activa. Use LOGIN primero")
	}

	// Validar parámetros
	path, ok := params["path"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -path es obligatorio")
	}

	// Parámetros opcionales
	size := int64(0)
	if sizeStr, ok := params["size"]; ok {
		fmt.Sscanf(sizeStr, "%d", &size)
		if size < 0 {
			return "", fmt.Errorf(" Error: El tamaño no puede ser negativo")
		}
	}

	createParents := false
	if _, ok := params["r"]; ok {
		createParents = true
	}

	var content string
	// Prioridad: -cont sobre -size
	if contPath, ok := params["cont"]; ok {
		// Leer archivo del sistema
		data, err := ioutil.ReadFile(contPath)
		if err != nil {
			return "", fmt.Errorf(" Error al leer el archivo: %v", err)
		}
		content = string(data)
		size = int64(len(content))
	} else if size > 0 {
		// Generar contenido con números 0-9
		for i := int64(0); i < size; i++ {
			content += fmt.Sprintf("%d", i%10)
		}
	}

	// Crear el archivo
	err := createFile(path, content, createParents)
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("         ARCHIVO CREADO EXITOSAMENTE       \n")
	output += fmt.Sprintf(" Ruta:           %s\n", path)
	output += fmt.Sprintf(" Tamaño:         %d bytes\n", size)
	output += fmt.Sprintf(" Propietario:    %s (UID: %d)\n", activeSession.User, activeSession.UID)
	output += fmt.Sprintf(" Grupo:          GID: %d\n", activeSession.GID)
	output += fmt.Sprintf(" Permisos:       664\n")
	output += fmt.Sprintf("\n Archivo creado correctamente\n")

	return output, nil
}

// MKDIR - Crea una carpeta en el sistema de archivos
func MKDIR(params map[string]string) (string, error) {
	// Verificar sesión activa
	if activeSession == nil {
		return "", fmt.Errorf(" Error: No hay una sesión activa. Use LOGIN primero")
	}

	// Validar parámetros
	path, ok := params["path"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -path es obligatorio")
	}

	createParents := false
	// Verificar si existe el parámetro p (sin importar el valor)
	if _, ok := params["p"]; ok {
		createParents = true
	}

	// Crear la carpeta
	err := createDirectory(path, createParents)
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("         CARPETA CREADA EXITOSAMENTE       \n")
	output += fmt.Sprintf(" Ruta:           %s\n", path)
	output += fmt.Sprintf(" Propietario:    %s (UID: %d)\n", activeSession.User, activeSession.UID)
	output += fmt.Sprintf(" Grupo:          GID: %d\n", activeSession.GID)
	output += fmt.Sprintf(" Permisos:       664\n")
	output += fmt.Sprintf("\n Carpeta creada correctamente\n")

	return output, nil
}

// CAT - Muestra el contenido de uno o más archivos
func CAT(params map[string]string) (string, error) {
	// Verificar sesión activa
	if activeSession == nil {
		return "", fmt.Errorf(" Error: No hay una sesión activa. Use LOGIN primero")
	}

	// Obtener todos los archivos (file1, file2, file3, ...)
	var files []string
	for i := 1; i <= 10; i++ {
		paramName := fmt.Sprintf("file%d", i)
		if filePath, ok := params[paramName]; ok {
			files = append(files, filePath)
		}
	}

	if len(files) == 0 {
		return "", fmt.Errorf(" Error: Debe especificar al menos un archivo con -file1")
	}

	output := fmt.Sprintf("         CONTENIDO DE ARCHIVO(S)          \n")

	// Leer y concatenar archivos
	for i, filePath := range files {
		content, err := readFile(filePath)
		if err != nil {
			output += fmt.Sprintf(" Error al leer '%s': %v\n\n", filePath, err)
			continue
		}

		if i > 0 {
			output += "\n"
		}
		output += fmt.Sprintf(" Archivo: %s\n", filePath)
		output += fmt.Sprintf("─────────────────────────────────────────────\n")
		output += content + "\n"
		output += fmt.Sprintf("─────────────────────────────────────────────\n")
	}

	return output, nil
}

// createFile - Crea un archivo en el sistema EXT2
func createFile(path string, content string, createParents bool) error {
	// Abrir disco
	file, err := os.OpenFile(activeSession.MountedPart.Path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf(" Error al abrir el disco: %v", err)
	}
	defer file.Close()

	// Leer MBR
	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return err
	}

	// Buscar partición
	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == activeSession.MountedPart.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	if partition == nil {
		return fmt.Errorf(" Error: No se encontró la partición")
	}

	// Leer superbloque
	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return err
	}

	// Separar path en componentes
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	fileName := pathParts[len(pathParts)-1]
	parentPath := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")

	// Crear carpetas padres si es necesario
	if createParents {
		for i := 1; i < len(pathParts); i++ {
			dirPath := "/" + strings.Join(pathParts[:i], "/")
			_, err := findInodeByPath(file, &sb, dirPath)
			if err != nil {
				// Crear la carpeta
				err = createDirectoryRecursive(file, &sb, dirPath, false)
				if err != nil {
					return err
				}
			}
		}
	}

	// Buscar inodo del directorio padre
	parentInodeNum, err := findInodeByPath(file, &sb, parentPath)
	if err != nil {
		return fmt.Errorf(" Error: El directorio padre no existe. Use -r para crear carpetas padres")
	}

	// Buscar inodo libre
	freeInodeNum, err := findFreeInode(file, &sb)
	if err != nil {
		return err
	}

	// Calcular cuántos bloques necesitamos
	blockSize := int64(64)
	contentSize := int64(len(content))
	numBlocksNeeded := (contentSize + blockSize - 1) / blockSize // Redondear hacia arriba

	if numBlocksNeeded > 12 {
		return fmt.Errorf(" Error: El archivo es demasiado grande (máximo 768 bytes)")
	}

	// Buscar bloques libres
	freeBlocks := make([]int32, numBlocksNeeded)
	for i := int64(0); i < numBlocksNeeded; i++ {
		freeBlockNum, err := findFreeBlock(file, &sb)
		if err != nil {
			return err
		}
		freeBlocks[i] = freeBlockNum

		// Marcar el bloque como usado temporalmente
		err = setBitmapBit(file, sb.S_bm_block_start, freeBlockNum, true)
		if err != nil {
			return err
		}
	}

	// Crear el inodo del archivo
	now := time.Now()
	fileInode := models.Inode{
		I_uid:   int32(activeSession.UID),
		I_gid:   int32(activeSession.GID),
		I_size:  contentSize,
		I_atime: now.Unix(),
		I_ctime: now.Unix(),
		I_mtime: now.Unix(),
		I_type:  1, // Archivo
		I_perm:  [3]byte{'6', '6', '4'},
	}
	for i := 0; i < 15; i++ {
		fileInode.I_block[i] = -1
	}

	// Asignar los bloques
	for i := int64(0); i < numBlocksNeeded && i < 12; i++ {
		fileInode.I_block[i] = freeBlocks[i]
	}

	// Escribir el inodo
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	err = utils.WriteStruct(file, &fileInode, sb.S_inode_start+(int64(freeInodeNum)*inodeSize))
	if err != nil {
		return err
	}

	// Escribir el contenido en los bloques
	remainingContent := content
	for i := int64(0); i < numBlocksNeeded; i++ {
		fileBlock := models.FileBlock{}

		// Tomar hasta 64 bytes del contenido
		bytesToWrite := len(remainingContent)
		if bytesToWrite > 64 {
			bytesToWrite = 64
		}

		copy(fileBlock.B_content[:], remainingContent[:bytesToWrite])
		remainingContent = remainingContent[bytesToWrite:]

		err = utils.WriteStruct(file, &fileBlock, sb.S_block_start+(int64(freeBlocks[i])*blockSize))
		if err != nil {
			return err
		}
	}

	// Actualizar bitmap de inodos
	err = setBitmapBit(file, sb.S_bm_inode_start, freeInodeNum, true)
	if err != nil {
		return err
	}

	// Agregar entrada en el directorio padre
	err = addEntryToDirectory(file, &sb, parentInodeNum, fileName, freeInodeNum)
	if err != nil {
		return err
	}

	// Actualizar superbloque
	sb.S_free_inodes_count--
	sb.S_free_blocks_count -= int32(numBlocksNeeded)
	err = utils.WriteStruct(file, &sb, partition.PartStart)
	if err != nil {
		return err
	}

	return nil
}

// createDirectory - Crea una carpeta en el sistema EXT2
func createDirectory(path string, createParents bool) error {
	file, err := os.OpenFile(activeSession.MountedPart.Path, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf(" Error al abrir el disco: %v", err)
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return err
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == activeSession.MountedPart.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	if partition == nil {
		return fmt.Errorf(" Error: No se encontró la partición")
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return err
	}

	return createDirectoryRecursive(file, &sb, path, createParents)
}

// createDirectoryRecursive - Función auxiliar para crear directorios
func createDirectoryRecursive(file *os.File, sb *models.Superblock, path string, createParents bool) error {
	// Normalizar el path
	path = strings.Trim(path, "/")
	if path == "" {
		return nil // La raíz ya existe
	}

	pathParts := strings.Split(path, "/")

	// Si createParents es true, crear todos los directorios en la ruta
	if createParents {
		currentPath := ""
		for _, part := range pathParts {
			if part == "" {
				continue
			}

			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			// Verificar si ya existe
			_, err := findInodeByPath(file, sb, "/"+currentPath)
			if err != nil {
				// No existe, crear
				err = createSingleDirectory(file, sb, "/"+currentPath)
				if err != nil {
					return err
				}
			}
		}
		return nil
	} else {
		// Solo crear el último directorio
		return createSingleDirectory(file, sb, "/"+path)
	}
}

// createSingleDirectory - Crea un solo directorio
func createSingleDirectory(file *os.File, sb *models.Superblock, path string) error {
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	dirName := pathParts[len(pathParts)-1]

	parentPath := "/"
	if len(pathParts) > 1 {
		parentPath = "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
	}

	// Buscar inodo del padre
	parentInodeNum, err := findInodeByPath(file, sb, parentPath)
	if err != nil {
		return fmt.Errorf(" Error: El directorio padre '%s' no existe. Use -p para crear carpetas padres", parentPath)
	}

	// Verificar si el directorio ya existe
	_, err = findInodeInDirectory(file, sb, parentInodeNum, dirName)
	if err == nil {
		return nil // Ya existe, no hacer nada
	}

	// Buscar inodo y bloque libres
	freeInodeNum, err := findFreeInode(file, sb)
	if err != nil {
		return err
	}

	freeBlockNum, err := findFreeBlock(file, sb)
	if err != nil {
		return err
	}

	// Crear el inodo del directorio
	now := time.Now()
	dirInode := models.Inode{
		I_uid:   int32(activeSession.UID),
		I_gid:   int32(activeSession.GID),
		I_size:  0,
		I_atime: now.Unix(),
		I_ctime: now.Unix(),
		I_mtime: now.Unix(),
		I_type:  0, // Carpeta
		I_perm:  [3]byte{'6', '6', '4'},
	}
	for i := 0; i < 15; i++ {
		dirInode.I_block[i] = -1
	}
	dirInode.I_block[0] = freeBlockNum

	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	err = utils.WriteStruct(file, &dirInode, sb.S_inode_start+(int64(freeInodeNum)*inodeSize))
	if err != nil {
		return err
	}

	// Crear bloque de carpeta con . y ..
	blockSize := int64(64)
	dirBlock := models.FolderBlock{}
	copy(dirBlock.B_content[0].B_name[:], ".")
	dirBlock.B_content[0].B_inodo = freeInodeNum
	copy(dirBlock.B_content[1].B_name[:], "..")
	dirBlock.B_content[1].B_inodo = parentInodeNum
	dirBlock.B_content[2].B_inodo = -1
	dirBlock.B_content[3].B_inodo = -1

	err = utils.WriteStruct(file, &dirBlock, sb.S_block_start+(int64(freeBlockNum)*blockSize))
	if err != nil {
		return err
	}

	// Actualizar bitmaps
	err = setBitmapBit(file, sb.S_bm_inode_start, freeInodeNum, true)
	if err != nil {
		return err
	}

	err = setBitmapBit(file, sb.S_bm_block_start, freeBlockNum, true)
	if err != nil {
		return err
	}

	// Agregar entrada al directorio padre
	err = addEntryToDirectory(file, sb, parentInodeNum, dirName, freeInodeNum)
	if err != nil {
		return err
	}

	// Actualizar superbloque
	sb.S_free_inodes_count--
	sb.S_free_blocks_count--

	// Leer MBR para obtener partition start
	var mbr models.MBR
	utils.ReadStruct(file, &mbr, 0)
	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == activeSession.MountedPart.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	err = utils.WriteStruct(file, sb, partition.PartStart)
	if err != nil {
		return err
	}

	return nil
}

// readFile - Lee el contenido de un archivo
func readFile(path string) (string, error) {
	file, err := os.OpenFile(activeSession.MountedPart.Path, os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var mbr models.MBR
	err = utils.ReadStruct(file, &mbr, 0)
	if err != nil {
		return "", fmt.Errorf(" Error al leer el MBR: %v", err)
	}

	var partition *models.Partition
	for i := 0; i < 4; i++ {
		partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
		if partName == activeSession.MountedPart.Name {
			partition = &mbr.MbrPartitions[i]
			break
		}
	}

	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", err
	}

	// Buscar el inodo del archivo
	inodeNum, err := findInodeByPath(file, &sb, path)
	if err != nil {
		return "", fmt.Errorf("archivo no encontrado")
	}

	// Leer el inodo
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	var fileInode models.Inode
	err = utils.ReadStruct(file, &fileInode, sb.S_inode_start+(int64(inodeNum)*inodeSize))
	if err != nil {
		return "", err
	}

	// Verificar que sea un archivo
	if fileInode.I_type != 1 {
		return "", fmt.Errorf("la ruta especificada no es un archivo")
	}

	// Leer el contenido de todos los bloques necesarios
	blockSize := int64(64)
	var content string
	remainingSize := fileInode.I_size

	// Leer bloques directos
	for i := 0; i < 12 && remainingSize > 0; i++ {
		blockNum := fileInode.I_block[i]
		if blockNum == -1 {
			break
		}

		var fileBlock models.FileBlock
		err = utils.ReadStruct(file, &fileBlock, sb.S_block_start+(int64(blockNum)*blockSize))
		if err != nil {
			return "", err
		}

		// Leer solo los bytes necesarios de este bloque
		bytesToRead := remainingSize
		if bytesToRead > 64 {
			bytesToRead = 64
		}

		content += string(fileBlock.B_content[:bytesToRead])
		remainingSize -= bytesToRead
	}

	return content, nil
}

// Funciones auxiliares

func findInodeByPath(file *os.File, sb *models.Superblock, path string) (int32, error) {
	if path == "/" {
		return 0, nil
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentInode := int32(0)

	for _, part := range parts {
		if part == "" {
			continue
		}

		nextInode, err := findInodeInDirectory(file, sb, currentInode, part)
		if err != nil {
			return -1, err
		}
		currentInode = nextInode
	}

	return currentInode, nil
}

func findInodeInDirectory(file *os.File, sb *models.Superblock, dirInodeNum int32, name string) (int32, error) {
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	var dirInode models.Inode
	err := utils.ReadStruct(file, &dirInode, sb.S_inode_start+(int64(dirInodeNum)*inodeSize))
	if err != nil {
		return -1, err
	}

	blockSize := int64(64)

	// Buscar en todos los bloques directos del directorio
	for i := 0; i < 12; i++ {
		blockNum := dirInode.I_block[i]
		if blockNum == -1 {
			break
		}

		var dirBlock models.FolderBlock
		err = utils.ReadStruct(file, &dirBlock, sb.S_block_start+(int64(blockNum)*blockSize))
		if err != nil {
			return -1, err
		}

		// Buscar en las 4 entradas del bloque
		for j := 0; j < 4; j++ {
			if dirBlock.B_content[j].B_inodo != -1 {
				entryName := utils.ByteToString(dirBlock.B_content[j].B_name[:])
				if entryName == name {
					return dirBlock.B_content[j].B_inodo, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("entrada no encontrada")
}

func findFreeInode(file *os.File, sb *models.Superblock) (int32, error) {
	bitmap := make([]byte, sb.S_inodes_count)
	_, err := file.ReadAt(bitmap, sb.S_bm_inode_start)
	if err != nil {
		return -1, err
	}

	for i := int32(0); i < sb.S_inodes_count; i++ {
		if bitmap[i] == 0 {
			return i, nil
		}
	}

	return -1, fmt.Errorf("no hay inodos libres")
}

func findFreeBlock(file *os.File, sb *models.Superblock) (int32, error) {
	bitmap := make([]byte, sb.S_blocks_count)
	_, err := file.ReadAt(bitmap, sb.S_bm_block_start)
	if err != nil {
		return -1, err
	}

	for i := int32(0); i < sb.S_blocks_count; i++ {
		if bitmap[i] == 0 {
			return i, nil
		}
	}

	return -1, fmt.Errorf("no hay bloques libres")
}

func setBitmapBit(file *os.File, bitmapStart int64, index int32, value bool) error {
	bitValue := byte(0)
	if value {
		bitValue = 1
	}
	_, err := file.WriteAt([]byte{bitValue}, bitmapStart+int64(index))
	return err
}

func addEntryToDirectory(file *os.File, sb *models.Superblock, dirInodeNum int32, name string, inodeNum int32) error {
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	var dirInode models.Inode
	err := utils.ReadStruct(file, &dirInode, sb.S_inode_start+(int64(dirInodeNum)*inodeSize))
	if err != nil {
		return err
	}

	blockSize := int64(64)

	// Buscar en todos los bloques directos del directorio
	for blockIdx := 0; blockIdx < 12; blockIdx++ {
		blockNum := dirInode.I_block[blockIdx]

		if blockNum == -1 {
			// No hay bloque asignado, crear uno nuevo
			freeBlockNum, err := findFreeBlock(file, sb)
			if err != nil {
				return fmt.Errorf("no hay bloques libres para expandir el directorio")
			}

			// Inicializar nuevo bloque
			newDirBlock := models.FolderBlock{}
			for i := 0; i < 4; i++ {
				newDirBlock.B_content[i].B_inodo = -1
			}

			// Agregar la entrada en la primera posición
			copy(newDirBlock.B_content[0].B_name[:], name)
			newDirBlock.B_content[0].B_inodo = inodeNum

			// Escribir el bloque
			err = utils.WriteStruct(file, &newDirBlock, sb.S_block_start+(int64(freeBlockNum)*blockSize))
			if err != nil {
				return err
			}

			// Actualizar el inodo del directorio
			dirInode.I_block[blockIdx] = freeBlockNum
			err = utils.WriteStruct(file, &dirInode, sb.S_inode_start+(int64(dirInodeNum)*inodeSize))
			if err != nil {
				return err
			}

			// Actualizar bitmap de bloques
			err = setBitmapBit(file, sb.S_bm_block_start, freeBlockNum, true)
			if err != nil {
				return err
			}

			// Actualizar superbloque
			sb.S_free_blocks_count--
			var mbr models.MBR
			utils.ReadStruct(file, &mbr, 0)
			var partition *models.Partition
			for i := 0; i < 4; i++ {
				partName := utils.ByteToString(mbr.MbrPartitions[i].PartName[:])
				if partName == activeSession.MountedPart.Name {
					partition = &mbr.MbrPartitions[i]
					break
				}
			}
			utils.WriteStruct(file, sb, partition.PartStart)

			return nil
		}

		// Leer el bloque existente
		var dirBlock models.FolderBlock
		err = utils.ReadStruct(file, &dirBlock, sb.S_block_start+(int64(blockNum)*blockSize))
		if err != nil {
			return err
		}

		// Buscar espacio libre en este bloque
		for i := 0; i < 4; i++ {
			if dirBlock.B_content[i].B_inodo == -1 {
				copy(dirBlock.B_content[i].B_name[:], name)
				dirBlock.B_content[i].B_inodo = inodeNum
				return utils.WriteStruct(file, &dirBlock, sb.S_block_start+(int64(blockNum)*blockSize))
			}
		}
	}

	return fmt.Errorf("directorio lleno (máximo 48 entradas alcanzado)")
}
