package handlers

import (
	"MIA_VDIC2S2025/backend/pkg/services"
	"MIA_VDIC2S2025/backend/pkg/utils"
	"fmt"
	"strings"
)

// ExecuteCommands - Ejecuta una lista de comandos
func ExecuteCommands(commandsText string) string {
	lines := strings.Split(commandsText, "\n")
	var output strings.Builder

	output.WriteString("       INICIANDO EJECUCIÓN DE COMANDOS     \n")

	executedCount := 0
	errorCount := 0

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)

		// Ignorar líneas vacías
		if line == "" {
			continue
		}

		// Mostrar comentarios
		if strings.HasPrefix(line, "#") {
			output.WriteString(fmt.Sprintf(" %s\n", line))
			continue
		}

		// Parsear el comando
		cmdName, params, err := utils.ParseCommand(line)
		if err != nil {
			output.WriteString(fmt.Sprintf(" Error en línea %d: %v\n", lineNum+1, err))
			errorCount++
			continue
		}

		if cmdName == "" {
			continue
		}

		// Ejecutar el comando
		output.WriteString(fmt.Sprintf("\n Ejecutando: %s\n", cmdName))
		output.WriteString("─────────────────────────────────────────────\n")

		result, err := ExecuteSingleCommand(cmdName, params)
		if err != nil {
			output.WriteString(fmt.Sprintf(" Error: %v\n", err))
			errorCount++
		} else {
			output.WriteString(result)
			executedCount++
		}

		output.WriteString("\n")
	}

	// Resumen final
	output.WriteString("\n            RESUMEN DE EJECUCIÓN           \n")
	output.WriteString(fmt.Sprintf(" Comandos ejecutados: %d\n", executedCount))
	output.WriteString(fmt.Sprintf(" Errores: %d\n", errorCount))
	output.WriteString(fmt.Sprintf(" Total procesado: %d líneas\n", len(lines)))

	return output.String()
}

// ExecuteSingleCommand - Ejecuta un comando individual
func ExecuteSingleCommand(cmdName string, params map[string]string) (string, error) {
	cmdName = strings.ToLower(cmdName)

	switch cmdName {
	case "mkdisk":
		return services.MKDISK(params)

	case "rmdisk":
		return services.RMDISK(params)

	case "fdisk":
		return services.FDISK(params)

	case "mount":
		return services.MOUNT(params)

	case "mounted":
		return services.MOUNTED(params)

	case "mkfs":
		return "Comando MKFS aún no implementado", nil

	case "login":
		return "Comando LOGIN aún no implementado", nil

	case "logout":
		return "Comando LOGOUT aún no implementado", nil

	case "mkgrp":
		return "Comando MKGRP aún no implementado", nil

	case "mkusr":
		return "Comando MKUSR aún no implementado", nil

	case "mkfile":
		return "Comando MKFILE aún no implementado", nil

	case "mkdir":
		return "Comando MKDIR aún no implementado", nil

	case "cat":
		return "Comando CAT aún no implementado", nil

	case "rep":
		return "Comando REP aún no implementado", nil

	default:
		return "", fmt.Errorf("comando no reconocido: %s", cmdName)
	}
}
