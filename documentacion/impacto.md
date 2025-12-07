Autor: Angel Jose Cardenas Pocon
Curso: Manejo E Implementacion de Archivos, Curso de vacaciones de diciembre 2025
Carnet: 202304063

# Informe de Impacto  

## 1. Resumen 
-Es un simulador educativo que deja crear discos (.mia), manejar particiones (MBR/EBR) y usar un sistema de archivos parecido a EXT2 en una consola web. Su aporte es principalmente para aprender (ver cómo funcionan estas estructuras) y para la parte técnica (ejemplo de arquitectura en Go con frontend en React + Vite).


## 2. Contexto y Problema
-En clase, normalmente se ve MBR/EBR/EXT2 de manera teórica. Hace falta una herramienta práctica que muestre todo el proceso: crear un disco, dividirlo en particiones, montarlo, formatearlo, trabajar con archivos y permisos.


## 3. Objetivos de Impacto
-Académico: Mostrar de forma visual y manipulable las partes de un disco y un FS (MBR, particiones, superbloques, inodos, bloques).

-Técnico: Tener un backend en Go bien organizado (comandos, estructuras, parser, rutas, utilidades) y una interfaz básica en la web.

-Productivo: Ahorrar tiempo al preparar prácticas y permitir correcciones automáticas por medio de scripts.


## 4. Alcance Funcional
-Discos: mkdisk, rmdisk, fdisk (particiones primaria, extendida y lógica).

-Montaje/FS: mount, mounted, mkfs.

-Usuarios/Grupos: login, logout, mkgrp, rmgrp, mkusr, rmusr, chgrp.

-Archivos y carpetas: mkdir, mkfile, cat.

-Soporte de scripts y comandos sugeridos.


## 5. Beneficios Esperados
-Aprender más rápido gracias a resultados inmediatos.

-Ver cambios claros: impresión de estado del FS/MBR.

-Reproducir prácticas con scripts .txt/.smia.

-Portabilidad: solo necesita el binario en Go y la interfaz en Vite, no usa base de datos.


## 6. Indicadores de Éxito (KPIs)

-Instalación rápida: menos de 10 min.

-Ejecuciones correctas: al menos 90% de los comandos en scripts.

-Cobertura de pruebas: al menos 70% de funciones clave.

-Rendimiento: crear/montar/formatear ≤1s en discos de hasta 20MB.

-Satisfacción: 4 de 5 o más en encuestas a estudiantes.


## 7. Impacto Académico
- Favorece la comprensión de estructuras (MBR/EBR/SuperBloque/Inodos) y permisos U/G/O.  
- Reduce ambigüedad entre **paths del host** y **paths internos del FS**.  
- Permite prácticas de corrección automatizada (scripts).


## 8. Impacto Técnico/Organizacional
-Ayuda a entender estructuras (MBR, EBR, superbloque, inodos) y permisos de usuario/grupo/otros.

-Evita confusión entre rutas del sistema host y las del FS simulado.

-Permite prácticas con corrección automática mediante scripts.


## 9. Costos y Ahorros
- **Costo de desarrollo**: horas de estudiante + equipamiento local.  
- **Ahorro**: menos tiempo preparando ambientes; menos errores manuales; mayor reuso de scripts.  
- **Operación**: sin costos de infraestructura (local-first).

---

## 10. Conclusión
Mejora el aprendizaje práctico de sistemas de archivos y es una base técnica fácil de ampliar. 
Al combinar simulación realista + consola web, facilita cursos, prácticas y evaluaciones.