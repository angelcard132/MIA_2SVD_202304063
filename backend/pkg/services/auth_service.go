package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

// UserSession - Sesión activa del usuario
type UserSession struct {
	User        string
	UID         int
	GID         int
	PartitionID string
	MountedPart *models.MountedPartition
}

// Variable global para la sesión activa
var activeSession *UserSession = nil

// LOGIN - Inicia sesión en el sistema
func LOGIN(params map[string]string) (string, error) {
	// Verificar que no haya una sesión activa
	if activeSession != nil {
		return "", fmt.Errorf(" Error: Ya existe una sesión activa del usuario '%s'. Use LOGOUT primero", activeSession.User)
	}

	// Validar parámetros obligatorios
	user, ok := params["user"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -user es obligatorio")
	}

	pass, ok := params["pass"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -pass es obligatorio")
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

	// Leer el superbloque
	var sb models.Superblock
	err = utils.ReadStruct(file, &sb, partition.PartStart)
	if err != nil {
		return "", fmt.Errorf(" Error al leer el superbloque: %v", err)
	}

	// Verificar que esté formateada
	if sb.S_magic != 0xEF53 {
		return "", fmt.Errorf(" Error: La partición no está formateada con EXT2")
	}

	// Leer el archivo users.txt
	usersContent, err := readUsersFile(file, &sb)
	if err != nil {
		return "", fmt.Errorf(" Error al leer archivo users.txt: %v", err)
	}

	// Buscar el usuario
	lines := strings.Split(usersContent, "\n")
	var userUID, userGID int
	found := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		// Formato: UID, Tipo, Grupo, Usuario, Contraseña
		if parts[1] == "U" && len(parts) >= 5 {
			if parts[3] == user && parts[4] == pass {
				fmt.Sscanf(parts[0], "%d", &userUID)
				// El GID está en la columna 2 para usuarios
				fmt.Sscanf(parts[2], "%d", &userGID)
				found = true
				break
			}
		}
	}

	if !found {
		return "", fmt.Errorf(" Error: Usuario o contraseña incorrectos")
	}

	// Crear la sesión
	activeSession = &UserSession{
		User:        user,
		UID:         userUID,
		GID:         userGID,
		PartitionID: id,
		MountedPart: mp,
	}

	output := fmt.Sprintf("         SESIÓN INICIADA EXITOSAMENTE      \n")
	output += fmt.Sprintf(" Usuario:         %s\n", user)
	output += fmt.Sprintf(" UID:             %d\n", userUID)
	output += fmt.Sprintf(" GID:             %d\n", userGID)
	output += fmt.Sprintf(" Partición:       %s\n", id)
	output += fmt.Sprintf(" Disco:           %s\n", mp.Path)
	output += fmt.Sprintf("\n Sesión iniciada correctamente\n")

	return output, nil
}

// LOGOUT - Cierra la sesión activa
func LOGOUT(params map[string]string) (string, error) {
	if activeSession == nil {
		return "", fmt.Errorf(" Error: No hay ninguna sesión activa")
	}

	username := activeSession.User
	activeSession = nil

	output := fmt.Sprintf("         SESIÓN CERRADA EXITOSAMENTE       \n")
	output += fmt.Sprintf(" Usuario:         %s\n", username)
	output += fmt.Sprintf("\n Sesión cerrada correctamente\n")
	return output, nil
}

// GetActiveSession - Obtiene la sesión activa
func GetActiveSession() *UserSession {
	return activeSession
}

// readUsersFile - Lee el contenido del archivo users.txt
func readUsersFile(file *os.File, sb *models.Superblock) (string, error) {
	// Leer el inodo del archivo users.txt (inodo 1)
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	var usersInode models.Inode
	err := utils.ReadStruct(file, &usersInode, sb.S_inode_start+inodeSize)
	if err != nil {
		return "", err
	}

	// Leer el contenido del primer bloque
	blockSize := int64(64)
	var usersBlock models.FileBlock
	blockNum := usersInode.I_block[0]
	if blockNum == -1 {
		return "", fmt.Errorf("archivo users.txt vacío")
	}

	err = utils.ReadStruct(file, &usersBlock, sb.S_block_start+(int64(blockNum)*blockSize))
	if err != nil {
		return "", err
	}

	// Convertir a string
	content := string(usersBlock.B_content[:])
	// Limpiar bytes nulos
	content = strings.TrimRight(content, "\x00")

	return content, nil
}

// writeUsersFile - Escribe el contenido del archivo users.txt
func writeUsersFile(file *os.File, sb *models.Superblock, content string) error {
	// Leer el inodo del archivo users.txt (inodo 1)
	inodeSize := int64(unsafe.Sizeof(models.Inode{}))
	var usersInode models.Inode
	err := utils.ReadStruct(file, &usersInode, sb.S_inode_start+inodeSize)
	if err != nil {
		return err
	}

	// Actualizar el tamaño
	usersInode.I_size = int64(len(content))

	// Escribir el inodo actualizado
	err = utils.WriteStruct(file, &usersInode, sb.S_inode_start+inodeSize)
	if err != nil {
		return err
	}

	// Escribir el contenido en el primer bloque
	blockSize := int64(64)
	var usersBlock models.FileBlock
	copy(usersBlock.B_content[:], content)

	blockNum := usersInode.I_block[0]
	err = utils.WriteStruct(file, &usersBlock, sb.S_block_start+(int64(blockNum)*blockSize))
	if err != nil {
		return err
	}

	return nil
}
