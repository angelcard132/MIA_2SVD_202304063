import { useState } from 'react'
import axios from 'axios'
import './App.css'

const API_URL = 'http://localhost:8080'

function App() {
  const [inputCommands, setInputCommands] = useState('')
  const [outputCommands, setOutputCommands] = useState(
    'Sistema listo para recibir comandos...\n' +
    'Escribe tus comandos o carga un archivo .smia\n\n'
  )
  const [loading, setLoading] = useState(false)
  const [fileName, setFileName] = useState('')

  // Manejar carga de archivo
  const handleFileUpload = (event) => {
    const file = event.target.files[0]
    if (!file) return

    // Validar extensión
    const validExtensions = ['.smia', '.txt', '.mia']
    const fileExtension = file.name.substring(file.name.lastIndexOf('.')).toLowerCase()
    
    if (!validExtensions.includes(fileExtension)) {
      setOutputCommands(prev => 
        prev + '\n Error: El archivo debe tener extensión .smia o .txt\n'
      )
      return
    }

    const reader = new FileReader()
    
    reader.onload = (e) => {
      setInputCommands(e.target.result)
      setFileName(file.name)
      setOutputCommands(prev => 
        prev + `\n Archivo cargado: ${file.name}\n` +
        `Tamaño: ${(file.size / 1024).toFixed(2)} KB\n` +
        ` Listo para ejecutar\n\n`
      )
    }
    
    reader.onerror = () => {
      setOutputCommands(prev => 
        prev + '\n Error al leer el archivo\n'
      )
    }
    
    reader.readAsText(file)
  }

  // Ejecutar comandos
  const handleExecute = async () => {
    if (!inputCommands.trim()) {
      setOutputCommands(prev => 
        prev + '\n Error: No hay comandos para ejecutar\n'
      )
      return
    }

    setLoading(true)
    setOutputCommands(prev => 
      prev + '\n' + '═'.repeat(45) + '\n' +
      'Ejecutando comandos...\n' +
      '═'.repeat(45) + '\n'
    )

    try {
      const response = await axios.post(`${API_URL}/api/execute`, {
        commands: inputCommands
      }, {
        headers: {
          'Content-Type': 'application/json'
        },
        timeout: 30000 // 30 segundos
      })

      if (response.data.success) {
        setOutputCommands(prev => prev + '\n' + response.data.output + '\n')
      } else {
        setOutputCommands(prev => 
          prev + '\n Error en la ejecución\n' +
          (response.data.error || 'Error desconocido') + '\n'
        )
      }
    } catch (error) {
      let errorMessage = '\n Error de conexión:\n'
      
      if (error.code === 'ECONNREFUSED' || error.message.includes('Network Error')) {
        errorMessage += 'No se pudo conectar con el servidor backend\n'
        errorMessage += 'Asegúrate de que el servidor esté corriendo en http://localhost:8080\n'
        errorMessage += 'Ejecuta: cd backend && go run cmd/api/main.go\n'
      } else if (error.response) {
        errorMessage += ` Status: ${error.response.status}\n`
        errorMessage += ` ${error.response.data?.error || error.message}\n`
      } else {
        errorMessage += ` ${error.message}\n`
      }
      
      setOutputCommands(prev => prev + errorMessage)
      console.error('Error detallado:', error)
    } finally {
      setLoading(false)
    }
  }

  // Limpiar área de entrada
  const handleClearInput = () => {
    setInputCommands('')
    setFileName('')
    setOutputCommands(prev => prev + '\n Área de entrada limpiada\n')
  }

  // Limpiar área de salida
  const handleClearOutput = () => {
    setOutputCommands(
      ' Sistema listo para recibir comandos...\n\n'
    )
  }

  // Probar conexión con backend
  const testConnection = async () => {
    setOutputCommands(prev => prev + '\n Probando conexión con el backend...\n')
    try {
      const response = await axios.get(`${API_URL}/ping`)
      setOutputCommands(prev => 
        prev + ' ' + response.data.status + '\n' +
        ' ' + response.data.time + '\n\n'
      )
    } catch (error) {
      setOutputCommands(prev => 
        prev + ' No se pudo conectar con el backend\n' +
        ' Verifica que el servidor esté corriendo\n\n'
      )
    }
  }

  return (
    <div className="app-container">
      <header className="header">
        <div className="header-content">
          <h1>ESCUELA DE VACACIONES SEGUNDO SEMESTRE</h1>
          <p>Proyecto - Manejo e Implementación de Archivos</p>
          <button onClick={testConnection} className="btn btn-test">
            Probar Conexión
          </button>
        </div>
      </header>

      <div className="main-content">
        {/* Panel de Entrada */}
        <div className="panel">
          <div className="panel-header">
            <div className="panel-title">
              <h2>Entrada de Comandos</h2>
              {fileName && <span className="file-badge"> {fileName}</span>}
            </div>
            <div className="button-group">
              <label className="btn btn-secondary">
                Cargar Script
                <input
                  type="file"
                  accept=".smia,.txt,.mia"
                  onChange={handleFileUpload}
                  style={{ display: 'none' }}
                />
              </label>
              <button onClick={handleClearInput} className="btn btn-secondary">
                Limpiar
              </button>
            </div>
          </div>
          <textarea
            className="textarea input-area"
            value={inputCommands}
            onChange={(e) => setInputCommands(e.target.value)}
            placeholder="# Ejemplo de comandos:
mkdisk -size=3000 -unit=K
fdisk -size=300 -diskName=VDIC-A.mia -name=Particion1
mount -path=VDIC-A.mia -name=Particion1

# O arrastra un archivo .smia aquí..."
            spellCheck={false}
          />
          <button
            onClick={handleExecute}
            className={`btn btn-primary btn-execute ${loading ? 'loading' : ''}`}
            disabled={loading}
          >
            {loading ? ' Ejecutando...' : ' Ejecutar Comandos'}
          </button>
        </div>

        {/* Panel de Salida */}
        <div className="panel">
          <div className="panel-header">
            <h2>Salida de Comandos</h2>
            <button onClick={handleClearOutput} className="btn btn-secondary">
              Limpiar
            </button>
          </div>
          <textarea
            className="textarea output-area"
            value={outputCommands}
            readOnly
            spellCheck={false}
          />
        </div>
      </div>

      <footer className="footer">
        <p>Universidad San Carlos de Guatemala - Facultad de Ingeniería</p>
        <p className="footer-status">
          <span className="status-indicator"></span>
          Backend: http://localhost:8080 | Frontend: http://localhost:5173
        </p>
      </footer>
    </div>
  )
}

export default App