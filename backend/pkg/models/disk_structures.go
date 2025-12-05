package models

// MBR - Master Boot Record
type MBR struct {
	MbrTamano        int64        // Tamaño total del disco en bytes
	MbrFechaCreacion int64        // Fecha y hora de creación (Unix timestamp)
	MbrDskSignature  int32        // Número random que identifica de forma única cada disco
	DskFit           byte         // Tipo de ajuste: B(Best), F(First), W(Worst)
	MbrPartitions    [4]Partition // Array de 4 particiones
}

// Partition - Estructura de partición
type Partition struct {
	PartStatus      byte     // Indica si la partición está montada (0=No montada, 1=Montada)
	PartType        byte     // Tipo de partición: P(Primaria), E(Extendida)
	PartFit         byte     // Tipo de ajuste: B(Best), F(First), W(Worst)
	PartStart       int64    // Byte donde inicia la partición
	PartSize        int64    // Tamaño total de la partición en bytes
	PartName        [16]byte // Nombre de la partición
	PartCorrelative int32    // Correlativo de la partición (inicia en -1, luego 1, 2, 3...)
	PartId          [4]byte  // ID de la partición al montarla
}

// EBR - Extended Boot Record (para particiones lógicas)
type EBR struct {
	PartMount byte     // Indica si la partición está montada
	PartFit   byte     // Tipo de ajuste: B(Best), F(First), W(Worst)
	PartStart int64    // Byte donde inicia la partición
	PartSize  int64    // Tamaño total de la partición en bytes
	PartNext  int64    // Byte donde está el próximo EBR (-1 si no hay siguiente)
	PartName  [16]byte // Nombre de la partición lógica
}

// Superblock - Información del sistema de archivos EXT2
type Superblock struct {
	S_filesystem_type   int32 // Identifica el sistema de archivos (2 = EXT2)
	S_inodes_count      int32 // Número total de inodos
	S_blocks_count      int32 // Número total de bloques
	S_free_blocks_count int32 // Número de bloques libres
	S_free_inodes_count int32 // Número de inodos libres
	S_mtime             int64 // Última fecha de montaje (Unix timestamp)
	S_umtime            int64 // Última fecha de desmontaje (Unix timestamp)
	S_mnt_count         int32 // Contador de montajes
	S_magic             int32 // Valor mágico 0xEF53
	S_inode_s           int32 // Tamaño del inodo
	S_block_s           int32 // Tamaño del bloque
	S_first_ino         int32 // Primer inodo libre
	S_first_blo         int32 // Primer bloque libre
	S_bm_inode_start    int64 // Inicio del bitmap de inodos
	S_bm_block_start    int64 // Inicio del bitmap de bloques
	S_inode_start       int64 // Inicio de la tabla de inodos
	S_block_start       int64 // Inicio de la tabla de bloques
}

// Inode - Estructura de inodo (index node)
type Inode struct {
	I_uid   int32     // UID del usuario propietario
	I_gid   int32     // GID del grupo
	I_size  int64     // Tamaño del archivo en bytes
	I_atime int64     // Última lectura (Unix timestamp)
	I_ctime int64     // Fecha de creación (Unix timestamp)
	I_mtime int64     // Última modificación (Unix timestamp)
	I_block [15]int32 // Bloques: 0-11 directos, 12 simple indirecto, 13 doble indirecto, 14 triple indirecto
	I_type  byte      // 0=Carpeta, 1=Archivo
	I_perm  [3]byte   // Permisos UGO en octal
}

// Content - Contenido de una carpeta
type Content struct {
	B_name  [12]byte // Nombre del archivo/carpeta
	B_inodo int32    // Apuntador al inodo
}

// FolderBlock - Bloque de carpeta
type FolderBlock struct {
	B_content [4]Content // Array de 4 contenidos (64 bytes total)
}

// FileBlock - Bloque de archivo
type FileBlock struct {
	B_content [64]byte // Contenido del archivo (64 bytes)
}

// PointerBlock - Bloque de apuntadores
type PointerBlock struct {
	B_pointers [16]int32 // Array de 16 apuntadores a bloques
}

// MountedPartition - Información de particiones montadas (en memoria)
type MountedPartition struct {
	Id         string // ID de la partición (ej: "221A")
	Path       string // Ruta del disco
	Name       string // Nombre de la partición
	Part       int    // Número de partición en el disco
	DiskLetter byte   // Letra del disco (A, B, C...)
}
