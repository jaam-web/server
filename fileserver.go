package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Name    string `json:"name"`
	IsDir   bool   `json:"isDir"`
	Size    int64  `json:"size"`
	ModTime string `json:"modTime"`
	Ext     string `json:"ext"`
}

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	Data    string `json:"data,omitempty"`
}

var rootDir string

func main() {
	var err error
	rootDir, err = os.Getwd()
	if err != nil {
		log.Fatal("Error obteniendo directorio actual:", err)
	}

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api/list", handleList)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/download", handleDownload)
	http.HandleFunc("/api/delete", handleDelete)
	http.HandleFunc("/api/mkdir", handleMkdir)
	http.HandleFunc("/api/preview", handlePreview)
	http.HandleFunc("/api/rename", handleRename)

	port := "8080"
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("       SERVIDOR DE ARCHIVOS PROFESIONAL")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\nDirectorio: %s\n", rootDir)
	fmt.Printf("Puerto: %s\n", port)
	fmt.Println("\nURLs de acceso:")
	fmt.Printf("  Local:    http://localhost:%s\n", port)
	
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		fmt.Println("  Red:")
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					fmt.Printf("            http://%s:%s\n", ipnet.IP.String(), port)
				}
			}
		}
	}
	
	fmt.Println("\nPresiona Ctrl+C para detener el servidor")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Error iniciando servidor:", err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	tmpl.Execute(w, nil)
}

func handleList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "."
	}

	fullPath := filepath.Join(rootDir, path)
	if !strings.HasPrefix(filepath.Clean(fullPath), rootDir) {
		http.Error(w, "Ruta invalida", http.StatusBadRequest)
		return
	}

	files, err := os.ReadDir(fullPath)
	if err != nil {
		http.Error(w, "Error leyendo directorio", http.StatusInternalServerError)
		return
	}

	var fileList []FileInfo
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		ext := filepath.Ext(file.Name())
		fileList = append(fileList, FileInfo{
			Name:    file.Name(),
			IsDir:   file.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			Ext:     ext,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileList)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	path := r.FormValue("path")
	if path == "" {
		path = "."
	}

	fullPath := filepath.Join(rootDir, path)
	if !strings.HasPrefix(filepath.Clean(fullPath), rootDir) {
		sendJSON(w, Response{Success: false, Error: "Ruta invalida"})
		return
	}

	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error parseando formulario"})
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error obteniendo archivo"})
		return
	}
	defer file.Close()

	destPath := filepath.Join(fullPath, handler.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error creando archivo"})
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, file)
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error guardando archivo"})
		return
	}

	sendJSON(w, Response{Success: true, Message: "Archivo subido correctamente"})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "Ruta no especificada", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(rootDir, path)
	if !strings.HasPrefix(filepath.Clean(fullPath), rootDir) {
		http.Error(w, "Ruta invalida", http.StatusBadRequest)
		return
	}

	file, err := os.Open(fullPath)
	if err != nil {
		http.Error(w, "Error abriendo archivo", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "Error obteniendo informacion del archivo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(path))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	io.Copy(w, file)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, Response{Success: false, Error: "Error parseando JSON"})
		return
	}

	fullPath := filepath.Join(rootDir, req.Path)
	if !strings.HasPrefix(filepath.Clean(fullPath), rootDir) {
		sendJSON(w, Response{Success: false, Error: "Ruta invalida"})
		return
	}

	err := os.RemoveAll(fullPath)
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error eliminando"})
		return
	}

	sendJSON(w, Response{Success: true, Message: "Eliminado correctamente"})
}

func handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, Response{Success: false, Error: "Error parseando JSON"})
		return
	}

	fullPath := filepath.Join(rootDir, req.Path, req.Name)
	if !strings.HasPrefix(filepath.Clean(fullPath), rootDir) {
		sendJSON(w, Response{Success: false, Error: "Ruta invalida"})
		return
	}

	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error creando carpeta"})
		return
	}

	sendJSON(w, Response{Success: true, Message: "Carpeta creada correctamente"})
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		sendJSON(w, Response{Success: false, Error: "Ruta no especificada"})
		return
	}

	fullPath := filepath.Join(rootDir, path)
	if !strings.HasPrefix(filepath.Clean(fullPath), rootDir) {
		sendJSON(w, Response{Success: false, Error: "Ruta invalida"})
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	
	textExts := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".xml": true, ".csv": true,
		".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".html": true,
		".css": true, ".scss": true, ".sass": true, ".go": true, ".py": true,
		".java": true, ".c": true, ".cpp": true, ".h": true, ".php": true,
		".rb": true, ".sh": true, ".yaml": true, ".yml": true, ".sql": true,
		".log": true, ".env": true, ".gitignore": true, ".rs": true,
	}

	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".webp": true, ".svg": true, ".ico": true,
	}

	if textExts[ext] {
		content, err := os.ReadFile(fullPath)
		if err != nil {
			sendJSON(w, Response{Success: false, Error: "Error leyendo archivo"})
			return
		}
		
		if len(content) > 1000000 {
			content = content[:1000000]
		}
		
		sendJSON(w, Response{
			Success: true,
			Data:    string(content),
		})
	} else if imageExts[ext] {
		content, err := os.ReadFile(fullPath)
		if err != nil {
			sendJSON(w, Response{Success: false, Error: "Error leyendo archivo"})
			return
		}
		
		encoded := base64.StdEncoding.EncodeToString(content)
		sendJSON(w, Response{
			Success: true,
			Data:    encoded,
		})
	} else {
		sendJSON(w, Response{Success: false, Error: "Tipo de archivo no soportado para preview"})
	}
}

func handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Metodo no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path    string `json:"path"`
		NewName string `json:"newName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, Response{Success: false, Error: "Error parseando JSON"})
		return
	}

	oldPath := filepath.Join(rootDir, req.Path)
	if !strings.HasPrefix(filepath.Clean(oldPath), rootDir) {
		sendJSON(w, Response{Success: false, Error: "Ruta invalida"})
		return
	}

	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, req.NewName)
	
	if !strings.HasPrefix(filepath.Clean(newPath), rootDir) {
		sendJSON(w, Response{Success: false, Error: "Ruta invalida"})
		return
	}

	err := os.Rename(oldPath, newPath)
	if err != nil {
		sendJSON(w, Response{Success: false, Error: "Error renombrando"})
		return
	}

	sendJSON(w, Response{Success: true, Message: "Renombrado correctamente"})
}

func sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

const indexHTML = `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width,initial-scale=1">
    <title>File Manager Pro</title>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Outfit:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <style>
        :root {
            --bg-primary: #0a0e1a;
            --bg-secondary: #111827;
            --bg-tertiary: #1a1f35;
            --accent-primary: #3b82f6;
            --accent-secondary: #8b5cf6;
            --text-primary: #f9fafb;
            --text-secondary: #9ca3af;
            --border-color: #1f2937;
            --success: #10b981;
            --danger: #ef4444;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: Outfit, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            min-height: 100vh;
        }
        .container { display: flex; height: 100vh; }
        .sidebar {
            width: 280px;
            background: var(--bg-secondary);
            border-right: 1px solid var(--border-color);
            display: flex;
            flex-direction: column;
        }
        .logo { padding: 24px; border-bottom: 1px solid var(--border-color); }
        .logo h1 {
            font-size: 24px;
            font-weight: 700;
            background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .stats { padding: 20px 24px; border-bottom: 1px solid var(--border-color); }
        .stat-item {
            display: flex;
            justify-content: space-between;
            padding: 12px 0;
            color: var(--text-secondary);
            font-size: 14px;
        }
        .stat-value {
            color: var(--text-primary);
            font-weight: 600;
            font-family: JetBrains Mono, monospace;
        }
        .actions { padding: 24px; display: flex; flex-direction: column; gap: 12px; }
        .btn {
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 500;
            transition: all 0.2s;
            font-family: Outfit, sans-serif;
            display: flex;
            align-items: center;
            gap: 10px;
            justify-content: center;
        }
        .btn-primary {
            background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary));
            color: white;
        }
        .btn-primary:hover { transform: translateY(-2px); }
        .btn-secondary {
            background: var(--bg-tertiary);
            color: var(--text-primary);
            border: 1px solid var(--border-color);
        }
        .main-content { flex: 1; display: flex; flex-direction: column; overflow: hidden; }
        .topbar {
            background: var(--bg-secondary);
            border-bottom: 1px solid var(--border-color);
            padding: 16px 32px;
            display: flex;
            align-items: center;
            gap: 20px;
        }
        .breadcrumb {
            flex: 1;
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 14px;
            color: var(--text-secondary);
            overflow-x: auto;
        }
        .breadcrumb-item {
            cursor: pointer;
            transition: color 0.2s;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .breadcrumb-item:hover { color: var(--accent-primary); }
        .search-box { position: relative; width: 300px; }
        .search-box input {
            width: 100%;
            padding: 10px 16px 10px 42px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            color: var(--text-primary);
            font-size: 14px;
            font-family: Outfit, sans-serif;
        }
        .search-box input:focus {
            outline: none;
            border-color: var(--accent-primary);
        }
        .search-box i {
            position: absolute;
            left: 16px;
            top: 50%;
            transform: translateY(-50%);
            color: var(--text-secondary);
        }
        .file-list-container { flex: 1; overflow-y: auto; padding: 24px 32px; }
        .drop-zone {
            border: 2px dashed var(--border-color);
            border-radius: 12px;
            padding: 60px 40px;
            text-align: center;
            margin-bottom: 32px;
            background: var(--bg-secondary);
            transition: all 0.3s;
        }
        .drop-zone.dragover {
            border-color: var(--accent-primary);
            background: rgba(59, 130, 246, 0.05);
        }
        .drop-zone i { font-size: 48px; color: var(--accent-primary); margin-bottom: 16px; }
        .file-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
            gap: 16px;
        }
        .file-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 20px;
            transition: all 0.2s;
            cursor: pointer;
        }
        .file-card:hover {
            border-color: var(--accent-primary);
            transform: translateY(-4px);
        }
        .file-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;
        }
        .file-icon {
            font-size: 32px;
            width: 48px;
            height: 48px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--bg-tertiary);
            border-radius: 10px;
            flex-shrink: 0;
        }
        .file-actions {
            display: flex;
            gap: 6px;
            opacity: 0;
            transition: opacity 0.2s;
            flex-shrink: 0;
        }
        .file-card:hover .file-actions { opacity: 1; }
        .icon-btn {
            width: 32px;
            height: 32px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--bg-tertiary);
            border: none;
            border-radius: 6px;
            cursor: pointer;
            color: var(--text-secondary);
            transition: all 0.2s;
        }
        .icon-btn:hover {
            background: var(--bg-primary);
            color: var(--text-primary);
        }
        .file-name {
            font-weight: 500;
            margin-bottom: 8px;
            word-break: break-word;
            overflow: hidden;
            text-overflow: ellipsis;
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
        }
        .file-meta {
            font-size: 12px;
            color: var(--text-secondary);
            font-family: JetBrains Mono, monospace;
        }
        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.85);
            align-items: center;
            justify-content: center;
            z-index: 1000;
            backdrop-filter: blur(10px);
        }
        .modal.show { display: flex; }
        .modal-content {
            background: var(--bg-secondary);
            padding: 32px;
            border-radius: 16px;
            max-width: 450px;
            width: 90%;
            border: 1px solid var(--border-color);
        }
        .modal-content h2 { margin-bottom: 24px; font-size: 24px; }
        .preview-modal {
            max-width: 90vw;
            max-height: 90vh;
            width: auto;
            overflow: hidden;
            display: flex;
            flex-direction: column;
        }
        .preview-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border-color);
        }
        .preview-content {
            overflow: auto;
            max-height: 75vh;
        }
        .preview-content pre {
            font-family: JetBrains Mono, monospace;
            font-size: 14px;
            line-height: 1.6;
            white-space: pre-wrap;
            word-break: break-all;
            background: var(--bg-tertiary);
            padding: 20px;
            border-radius: 8px;
        }
        .preview-content img {
            max-width: 100%;
            max-height: 75vh;
            object-fit: contain;
            border-radius: 8px;
        }
        input[type="text"] {
            width: 100%;
            padding: 12px 16px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            color: var(--text-primary);
            font-size: 14px;
            margin-bottom: 24px;
            font-family: Outfit, sans-serif;
        }
        input[type="text"]:focus {
            outline: none;
            border-color: var(--accent-primary);
        }
        input[type="file"] { display: none; }
        .modal-actions { display: flex; gap: 12px; }
        .empty-state { text-align: center; padding: 80px 40px; color: var(--text-secondary); }
        .empty-state i { font-size: 64px; margin-bottom: 20px; opacity: 0.5; }
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(20px); }
            to { opacity: 1; transform: translateY(0); }
        }
        .file-card { animation: fadeIn 0.3s ease; }
        ::-webkit-scrollbar { width: 8px; height: 8px; }
        ::-webkit-scrollbar-track { background: var(--bg-primary); }
        ::-webkit-scrollbar-thumb { background: var(--bg-tertiary); border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="sidebar">
            <div class="logo">
                <h1><i class="fas fa-hdd"></i> File Manager</h1>
            </div>
            <div class="stats">
                <div class="stat-item">
                    <span>Total Archivos</span>
                    <span class="stat-value" id="totalFiles">0</span>
                </div>
                <div class="stat-item">
                    <span>Total Carpetas</span>
                    <span class="stat-value" id="totalFolders">0</span>
                </div>
            </div>
            <div class="actions">
                <button class="btn btn-primary" onclick="document.getElementById('fileInput').click()">
                    <i class="fas fa-upload"></i> Subir Archivos
                </button>
                <button class="btn btn-secondary" onclick="showCreateFolderModal()">
                    <i class="fas fa-folder-plus"></i> Nueva Carpeta
                </button>
                <button class="btn btn-secondary" onclick="refreshFiles()">
                    <i class="fas fa-sync"></i> Actualizar
                </button>
                <input type="file" id="fileInput" multiple onchange="uploadFiles(this.files)">
            </div>
        </div>
        <div class="main-content">
            <div class="topbar">
                <div class="breadcrumb" id="breadcrumb"></div>
                <div class="search-box">
                    <i class="fas fa-search"></i>
                    <input type="text" id="searchInput" placeholder="Buscar archivos..." onkeyup="filterFiles()">
                </div>
            </div>
            <div class="file-list-container">
                <div class="drop-zone" id="dropZone">
                    <i class="fas fa-cloud-upload-alt"></i>
                    <p>Arrastra archivos aqui para subirlos</p>
                    <small style="color: var(--text-secondary)">o usa el boton de subir archivos</small>
                </div>
                <div class="file-grid" id="fileList"></div>
            </div>
        </div>
    </div>

    <div class="modal" id="createFolderModal">
        <div class="modal-content">
            <h2>Nueva Carpeta</h2>
            <input type="text" id="folderName" placeholder="Nombre de la carpeta">
            <div class="modal-actions">
                <button class="btn btn-primary" style="flex: 1;" onclick="createFolder()">Crear</button>
                <button class="btn btn-secondary" style="flex: 1;" onclick="hideCreateFolderModal()">Cancelar</button>
            </div>
        </div>
    </div>

    <div class="modal" id="renameModal">
        <div class="modal-content">
            <h2>Renombrar</h2>
            <input type="text" id="renameName" placeholder="Nuevo nombre">
            <div class="modal-actions">
                <button class="btn btn-primary" style="flex: 1;" onclick="confirmRename()">Renombrar</button>
                <button class="btn btn-secondary" style="flex: 1;" onclick="hideRenameModal()">Cancelar</button>
            </div>
        </div>
    </div>

    <div class="modal" id="previewModal">
        <div class="modal-content preview-modal">
            <div class="preview-header">
                <h3 id="previewTitle">Vista Previa</h3>
                <button class="icon-btn" onclick="closePreview()">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="preview-content" id="previewContent"></div>
        </div>
    </div>

    <script>
        let currentPath = '';
        let allFiles = [];
        let renameTarget = '';

        loadFiles();

        const dropZone = document.getElementById('dropZone');
        dropZone.addEventListener('dragover', (e) => {
            e.preventDefault();
            dropZone.classList.add('dragover');
        });
        dropZone.addEventListener('dragleave', () => {
            dropZone.classList.remove('dragover');
        });
        dropZone.addEventListener('drop', (e) => {
            e.preventDefault();
            dropZone.classList.remove('dragover');
            uploadFiles(e.dataTransfer.files);
        });

        async function loadFiles() {
            try {
                const response = await fetch('/api/list?path=' + encodeURIComponent(currentPath));
                allFiles = await response.json();
                displayFiles(allFiles);
                updateBreadcrumb();
                updateStats();
            } catch (error) {
                console.error('Error:', error);
            }
        }

        function displayFiles(files) {
            const fileList = document.getElementById('fileList');
            if (files.length === 0) {
                fileList.innerHTML = '<div class="empty-state"><i class="fas fa-folder-open"></i><p>Carpeta vacia</p></div>';
                return;
            }

            fileList.innerHTML = files.map(file => {
                const icon = getFileIcon(file);
                const path = currentPath ? currentPath + '/' + file.name : file.name;
                return '<div class="file-card" onclick="' + (file.isDir ? 'navigateTo(\'' + path + '\')' : 'previewFile(\'' + path + '\', \'' + file.ext + '\', \'' + file.name + '\')') + '">' +
                    '<div class="file-header">' +
                    '<div class="file-icon">' + icon + '</div>' +
                    '<div class="file-actions" onclick="event.stopPropagation()">' +
                    '<button class="icon-btn" onclick="showRenameModal(\'' + path + '\', \'' + file.name + '\')" title="Renombrar"><i class="fas fa-edit"></i></button>' +
                    (!file.isDir ? '<button class="icon-btn" onclick="downloadFile(\'' + path + '\')" title="Descargar"><i class="fas fa-download"></i></button>' : '') +
                    '<button class="icon-btn" onclick="deleteFile(\'' + path + '\', \'' + file.name + '\')" title="Eliminar"><i class="fas fa-trash"></i></button>' +
                    '</div></div>' +
                    '<div class="file-name">' + file.name + '</div>' +
                    '<div class="file-meta">' + (file.isDir ? 'Carpeta' : formatSize(file.size)) + '</div>' +
                    '</div>';
            }).join('');
        }

        function getFileIcon(file) {
            if (file.isDir) return '<i class="fas fa-folder" style="color: #3b82f6;"></i>';
            const ext = file.ext.toLowerCase();
            const icons = {
                '.jpg': ['fa-file-image', '#8b5cf6'], '.jpeg': ['fa-file-image', '#8b5cf6'], 
                '.png': ['fa-file-image', '#8b5cf6'], '.gif': ['fa-file-image', '#8b5cf6'],
                '.pdf': ['fa-file-pdf', '#ef4444'], '.doc': ['fa-file-word', '#3b82f6'], 
                '.docx': ['fa-file-word', '#3b82f6'], '.xls': ['fa-file-excel', '#10b981'], 
                '.xlsx': ['fa-file-excel', '#10b981'], '.zip': ['fa-file-archive', '#f59e0b'], 
                '.rar': ['fa-file-archive', '#f59e0b'], '.mp3': ['fa-file-audio', '#ec4899'], 
                '.mp4': ['fa-file-video', '#f59e0b'], '.js': ['fa-file-code', '#10b981'], 
                '.jsx': ['fa-file-code', '#10b981'], '.ts': ['fa-file-code', '#10b981'],
                '.py': ['fa-file-code', '#10b981'], '.go': ['fa-file-code', '#10b981'],
                '.html': ['fa-file-code', '#10b981'], '.css': ['fa-file-code', '#10b981'],
                '.txt': ['fa-file-alt', '#6b7280'], '.md': ['fa-file-alt', '#6b7280'],
                '.json': ['fa-file-code', '#10b981']
            };
            const [iconClass, color] = icons[ext] || ['fa-file', '#6b7280'];
            return '<i class="fas ' + iconClass + '" style="color: ' + color + ';"></i>';
        }

        function updateBreadcrumb() {
            const breadcrumb = document.getElementById('breadcrumb');
            const parts = currentPath.split('/').filter(p => p);
            let html = '<div class="breadcrumb-item" onclick="navigateTo(\'\')"><i class="fas fa-home"></i> Inicio</div>';
            let path = '';
            parts.forEach(part => {
                path += (path ? '/' : '') + part;
                html += '<i class="fas fa-chevron-right" style="font-size: 10px;"></i>';
                html += '<div class="breadcrumb-item" onclick="navigateTo(\'' + path + '\')">' + part + '</div>';
            });
            breadcrumb.innerHTML = html;
        }

        function updateStats() {
            document.getElementById('totalFolders').textContent = allFiles.filter(f => f.isDir).length;
            document.getElementById('totalFiles').textContent = allFiles.filter(f => !f.isDir).length;
        }

        function navigateTo(path) {
            currentPath = path;
            loadFiles();
        }

        async function uploadFiles(files) {
            for (let file of files) {
                const formData = new FormData();
                formData.append('file', file);
                formData.append('path', currentPath);
                try {
                    await fetch('/api/upload', { method: 'POST', body: formData });
                } catch (error) {
                    console.error('Error:', error);
                }
            }
            loadFiles();
        }

        function downloadFile(path) {
            window.location.href = '/api/download?path=' + encodeURIComponent(path);
        }

        async function deleteFile(path, name) {
            if (!confirm('Eliminar "' + name + '"?')) return;
            try {
                await fetch('/api/delete', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: path })
                });
                loadFiles();
            } catch (error) {
                console.error('Error:', error);
            }
        }

        function showCreateFolderModal() {
            document.getElementById('createFolderModal').classList.add('show');
            document.getElementById('folderName').value = '';
            document.getElementById('folderName').focus();
        }

        function hideCreateFolderModal() {
            document.getElementById('createFolderModal').classList.remove('show');
        }

        async function createFolder() {
            const name = document.getElementById('folderName').value.trim();
            if (!name) return;
            try {
                await fetch('/api/mkdir', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: currentPath, name: name })
                });
                hideCreateFolderModal();
                loadFiles();
            } catch (error) {
                console.error('Error:', error);
            }
        }

        function showRenameModal(path, currentName) {
            renameTarget = path;
            document.getElementById('renameModal').classList.add('show');
            document.getElementById('renameName').value = currentName;
            document.getElementById('renameName').focus();
        }

        function hideRenameModal() {
            document.getElementById('renameModal').classList.remove('show');
            renameTarget = '';
        }

        async function confirmRename() {
            const newName = document.getElementById('renameName').value.trim();
            if (!newName) return;
            try {
                await fetch('/api/rename', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path: renameTarget, newName: newName })
                });
                hideRenameModal();
                loadFiles();
            } catch (error) {
                console.error('Error:', error);
            }
        }

        async function previewFile(path, ext, name) {
            const modal = document.getElementById('previewModal');
            const content = document.getElementById('previewContent');
            const title = document.getElementById('previewTitle');
            
            title.textContent = name;
            modal.classList.add('show');
            content.innerHTML = '<p style="color: var(--text-secondary); text-align: center;">Cargando...</p>';
            
            try {
                const response = await fetch('/api/preview?path=' + encodeURIComponent(path));
                const result = await response.json();
                
                if (!result.success) {
                    content.innerHTML = '<p style="color: var(--text-secondary); text-align: center;">No se puede previsualizar</p>';
                    return;
                }
                
                const imageExts = ['.jpg', '.jpeg', '.png', '.gif', '.bmp', '.webp', '.svg', '.ico'];
                if (imageExts.includes(ext.toLowerCase())) {
                    content.innerHTML = '<img src="data:image/' + ext.substring(1) + ';base64,' + result.data + '" alt="Preview">';
                } else {
                    content.innerHTML = '<pre>' + escapeHtml(result.data) + '</pre>';
                }
            } catch (error) {
                content.innerHTML = '<p style="color: var(--danger); text-align: center;">Error</p>';
            }
        }

        function closePreview() {
            document.getElementById('previewModal').classList.remove('show');
        }

        function filterFiles() {
            const search = document.getElementById('searchInput').value.toLowerCase();
            const filtered = allFiles.filter(f => f.name.toLowerCase().includes(search));
            displayFiles(filtered);
        }

        function refreshFiles() {
            loadFiles();
        }

        function formatSize(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                hideCreateFolderModal();
                hideRenameModal();
                closePreview();
            }
        });
    </script>
</body>
</html>`
