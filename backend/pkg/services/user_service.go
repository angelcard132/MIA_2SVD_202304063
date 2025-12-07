package services

import (
	"MIA_VDIC2S2025/backend/pkg/models"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"os"
	"strings"
)

// MKGRP - Crea un grupo en el sistema
func MKGRP(params map[string]string) (string, error) {
	// Verificar que haya una sesión activa
	if activeSession == nil {
		return "", fmt.Errorf(" Error: No hay una sesión activa. Use LOGIN primero")
	}

	// Verificar que sea el usuario root
	if activeSession.User != "root" {
		return "", fmt.Errorf(" Error: Solo el usuario root puede crear grupos")
	}

	// Validar parámetros
	name, ok := params["name"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -name es obligatorio")
	}

	if len(name) > 60 {
		return "", fmt.Errorf(" Error: El nombre del grupo no puede exceder 60 caracteres")
	}

	// Abrir el disco
	file, err := os.OpenFile(activeSession.MountedPart.Path, os.O_RDWR, 0644)
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
		if partName == activeSession.MountedPart.Name {
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

	// Leer el archivo users.txt
	usersContent, err := readUsersFile(file, &sb)
	if err != nil {
		return "", fmt.Errorf(" Error al leer users.txt: %v", err)
	}

	// Verificar que el grupo no exista
	lines := strings.Split(usersContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 3 && parts[1] == "G" && parts[2] == name {
			return "", fmt.Errorf(" Error: El grupo '%s' ya existe", name)
		}
	}

	// Calcular el siguiente GID
	nextGID := 1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 3 && parts[1] == "G" {
			var gid int
			fmt.Sscanf(parts[0], "%d", &gid)
			if gid >= nextGID {
				nextGID = gid + 1
			}
		}
	}

	// Agregar el nuevo grupo
	newGroup := fmt.Sprintf("%d,G,%s", nextGID, name)
	usersContent += newGroup + "\n"

	// Escribir el archivo actualizado
	err = writeUsersFile(file, &sb, usersContent)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir users.txt: %v", err)
	}

	output := fmt.Sprintf("         GRUPO CREADO EXITOSAMENTE        \n")
	output += fmt.Sprintf(" Grupo:           %s\n", name)
	output += fmt.Sprintf(" ID:             %d\n", nextGID)
	output += fmt.Sprintf("\n Grupo creado correctamente\n")

	return output, nil
}

// MKUSR - Crea un usuario en el sistema
func MKUSR(params map[string]string) (string, error) {
	// Verificar que haya una sesión activa
	if activeSession == nil {
		return "", fmt.Errorf(" Error: No hay una sesión activa. Use LOGIN primero")
	}

	// Verificar que sea el usuario root
	if activeSession.User != "root" {
		return "", fmt.Errorf(" Error: Solo el usuario root puede crear usuarios")
	}

	// Validar parámetros
	user, ok := params["user"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -user es obligatorio")
	}

	pass, ok := params["pass"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -pass es obligatorio")
	}

	grp, ok := params["grp"]
	if !ok {
		return "", fmt.Errorf(" Error: El parámetro -grp es obligatorio")
	}

	if len(user) > 60 {
		return "", fmt.Errorf(" Error: El nombre de usuario no puede exceder 60 caracteres")
	}

	if len(pass) > 60 {
		return "", fmt.Errorf(" Error: La contraseña no puede exceder 60 caracteres")
	}

	if len(grp) > 60 {
		return "", fmt.Errorf(" Error: El nombre del grupo no puede exceder 60 caracteres")
	}

	// Abrir el disco
	file, err := os.OpenFile(activeSession.MountedPart.Path, os.O_RDWR, 0644)
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
		if partName == activeSession.MountedPart.Name {
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

	// Leer el archivo users.txt
	usersContent, err := readUsersFile(file, &sb)
	if err != nil {
		return "", fmt.Errorf(" Error al leer users.txt: %v", err)
	}

	// Verificar que el grupo exista y obtener su GID
	lines := strings.Split(usersContent, "\n")
	groupGID := -1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 3 && parts[1] == "G" && parts[2] == grp {
			fmt.Sscanf(parts[0], "%d", &groupGID)
			break
		}
	}

	if groupGID == -1 {
		return "", fmt.Errorf(" Error: El grupo '%s' no existe", grp)
	}

	// Verificar que el usuario no exista
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 5 && parts[1] == "U" && parts[3] == user {
			return "", fmt.Errorf(" Error: El usuario '%s' ya existe", user)
		}
	}

	// Calcular el siguiente UID
	nextUID := 1
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 5 && parts[1] == "U" {
			var uid int
			fmt.Sscanf(parts[0], "%d", &uid)
			if uid >= nextUID {
				nextUID = uid + 1
			}
		}
	}

	// Agregar el nuevo usuario
	newUser := fmt.Sprintf("%d,U,%d,%s,%s", nextUID, groupGID, user, pass)
	usersContent += newUser + "\n"

	// Escribir el archivo actualizado
	err = writeUsersFile(file, &sb, usersContent)
	if err != nil {
		return "", fmt.Errorf(" Error al escribir users.txt: %v", err)
	}

	output := fmt.Sprintf(" USUARIO CREADO EXITOSAMENTE\n")
	output += fmt.Sprintf(" Usuario:         %s\n", user)
	output += fmt.Sprintf(" UID:             %d\n", nextUID)
	output += fmt.Sprintf(" Grupo:           %s\n", grp)
	output += fmt.Sprintf(" GID:             %d\n", groupGID)
	output += fmt.Sprintf(" Contraseña:      %s\n", pass)
	output += fmt.Sprintf("\n Usuario creado correctamente\n")

	return output, nil
}
