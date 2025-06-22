import os
import sys
import socket
import urllib.parse
from http.server import HTTPServer, BaseHTTPRequestHandler
from colorama import Fore, Style, init
init(autoreset=True)
import mimetypes

PORT = int(sys.argv[1]) if len(sys.argv) > 1 and not sys.argv[1].startswith('-') else 8080

def get_local_ip():
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    try:
        s.connect(('8.8.8.8', 80))
        ip = s.getsockname()[0]
    except Exception:
        ip = 'localhost'
    finally:
        s.close()
    return ip

def human_size(size):
    if size < 1024:
        return f"{size} B"
    elif size < 1024*1024:
        return f"{size/1024:.2f} KB"
    elif size < 1024*1024*1024:
        return f"{size/1024/1024:.2f} MB"
    else:
        return f"{size/1024/1024/1024:.2f} GB"

class FileServerHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        path = urllib.parse.unquote(self.path)
        if path == "/" or path == "/index.html":
            self.send_response(200)
            self.send_header("Content-type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(self.render_index().encode("utf-8"))
        elif path.startswith("/download/"):
            filename = path[len("/download/"):]
            if not os.path.isfile(filename):
                self.send_error(404, "File not found")
                return
            self.send_file(filename, as_attachment=True)
        elif path.startswith("/view/"):
            filename = path[len("/view/"):]
            if not os.path.isfile(filename):
                self.send_error(404, "File not found")
                return
            self.send_file(filename, as_attachment=False)
        else:
            self.send_error(404, "Not found")

    def render_index(self):
        files = []
        for fname in os.listdir('.'):
            if os.path.isfile(fname):
                size = os.path.getsize(fname)
                mime, _ = mimetypes.guess_type(fname)
                can_view = False
                if mime and (mime.startswith(('text', 'image', 'audio', 'video')) or mime == 'application/pdf'):
                    can_view = True
                files.append({
                    'name': fname,
                    'size': human_size(size),
                    'can_view': can_view
                })
        host = get_local_ip()
        html = f"""<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <title>Server</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        :root {{
            --bg: #0a0a0a;
            --panel: #181818;
            --accent: #00ff41;
            --accent2: #ff003c;
            --text: #e0e0e0;
            --muted: #888;
            --danger: #ff003c;
        }}
        html, body {{
            margin: 0; padding: 0;
            background: var(--bg);
            color: var(--text);
            font-family: 'Fira Mono', 'Consolas', monospace;
            min-height: 100vh;
        }}
        .container {{
            max-width: 900px;
            margin: 24px auto 0 auto;
            background: var(--panel);
            border-radius: 14px;
            box-shadow: 0 0 32px #000d;
            padding: 18px 8px 10px 8px;
        }}
        h1 {{
            color: var(--accent);
            text-shadow: 0 0 12px var(--accent2), 0 0 2px #fff;
            font-size: 2.2em;
            margin-bottom: 10px;
            letter-spacing: 2px;
            font-family: 'Fira Mono', 'Consolas', monospace;
            text-align: center;
            animation: flicker 2.5s infinite alternate;
        }}
        @keyframes flicker {{
            0% {{ opacity: 1; }}
            90% {{ opacity: 0.95; }}
            92% {{ opacity: 0.7; }}
            95% {{ opacity: 1; }}
            100% {{ opacity: 0.92; }}
        }}
        .subtitle {{
            color: var(--accent2);
            font-size: 1.1em;
            margin-bottom: 18px;
            letter-spacing: 1px;
            text-align: center;
        }}
        table {{
            width: 100%;
            border-collapse: collapse;
            background: var(--panel);
            font-size: 1em;
            margin-bottom: 10px;
        }}
        th, td {{
            padding: 10px 6px;
            border-bottom: 1px solid #23232a;
            text-align: left;
        }}
        th {{
            background: #0a0a0a;
            color: var(--accent2);
            font-weight: bold;
            font-size: 1.05em;
            letter-spacing: 1px;
        }}
        tr:hover td {{
            background: #1a1a1a;
            color: var(--accent);
        }}
        .btn {{
            display: inline-block;
            padding: 7px 16px;
            border-radius: 4px;
            font-size: 1em;
            font-family: inherit;
            text-decoration: none;
            margin-right: 5px;
            margin-bottom: 2px;
            border: none;
            cursor: pointer;
            transition: background 0.18s, color 0.18s, box-shadow 0.18s;
            box-shadow: 0 0 6px #000a;
        }}
        .btn-download {{
            background: var(--accent);
            color: #0a0a0a;
            font-weight: bold;
            border: 1px solid #00ff41;
        }}
        .btn-download:hover {{
            background: #0a0a0a;
            color: var(--accent);
            border: 1px solid var(--accent);
            box-shadow: 0 0 8px var(--accent);
        }}
        .btn-view {{
            background: var(--accent2);
            color: #fff;
            font-weight: bold;
            border: 1px solid #ff003c;
        }}
        .btn-view:hover {{
            background: #0a0a0a;
            color: var(--accent2);
            border: 1px solid var(--accent2);
            box-shadow: 0 0 8px var(--accent2);
        }}
        .footer {{
            color: var(--muted);
            margin-top: 24px;
            font-size: 1em;
            text-align: center;
            letter-spacing: 1px;
        }}
        .credit {{
            color: var(--danger);
            font-size: 1em;
            margin-top: 10px;
            text-align: center;
            letter-spacing: 2px;
        }}
        @media (max-width: 600px) {{
            .container {{ padding: 4px 1px 2px 1px; }}
            h1 {{ font-size: 1.1em; }}
            .subtitle {{ font-size: 0.95em; }}
            th, td {{ padding: 7px 2px; font-size: 0.95em; }}
            .btn {{ font-size: 0.95em; padding: 5px 8px; }}
        }}
        ::selection {{
            background: var(--accent2);
            color: #0a0a0a;
        }}
        body::after {{
            content: '';
            position: fixed;
            left: 0; top: 0; width: 100vw; height: 100vh;
            pointer-events: none;
            background: repeating-linear-gradient(0deg, #00ff41 0 1px, transparent 1px 2px);
            opacity: 0.03;
            z-index: 9999;
        }}
    </style>
</head>
<body>
<div class="container">
    <h1>SERVER</h1>
    <div class="subtitle">Archivos disponibles para descargar o visualizar</div>
    <table>
        <thead>
            <tr>
                <th>Nombre</th>
                <th>Tamaño</th>
                <th>Acciones</th>
            </tr>
        </thead>
        <tbody>
"""
        for file in files:
            html += f"<tr><td>{file['name']}</td><td>{file['size']}</td><td>"
            html += f"<a href='/download/{urllib.parse.quote(file['name'])}' class='btn btn-download'>Descargar</a>"
            if file['can_view']:
                html += f"<a href='/view/{urllib.parse.quote(file['name'])}' target='_blank' class='btn btn-view'>Ver</a>"
            html += "</td></tr>"
        html += f"""</tbody>
    </table>
    <div class="footer">
        Acceso desde otro dispositivo: <b>http://{host}:{PORT}/</b>
    </div>
    <div class="credit">
        Programa creado por: JAAM
    </div>
</div>
</body>
</html>"""
        return html
    def send_file(self, filename, as_attachment=True):
        mime, _ = mimetypes.guess_type(filename)
        if not mime:
            mime = "application/octet-stream"
        try:
            with open(filename, "rb") as f:
                fs = os.fstat(f.fileno())
                self.send_response(200)
                self.send_header("Content-Type", mime)
                self.send_header("Content-Length", str(fs.st_size))
                if as_attachment:
                    self.send_header("Content-Disposition", f'attachment; filename="{os.path.basename(filename)}"')
                else:
                    self.send_header("Content-Disposition", f'inline; filename="{os.path.basename(filename)}"')
                self.end_headers()
                chunk_size = 64 * 1024
                while True:
                    chunk = f.read(chunk_size)
                    if not chunk:
                        break
                    self.wfile.write(chunk)
        except Exception as e:
            self.send_error(500, f"Error sending file: {e}")

    def log_message(self, format, *args):
        print(f"[+] {self.client_address[0]} -> {self.requestline}")

def print_help():
    print(Fore.CYAN + "Server - Menú de Ayuda" + Style.RESET_ALL)
    print("Uso:")
    print("  server [PUERTO]")
    print("  server -h | --help")
    print()
    print("Opciones:")
    print("  PUERTO      Puerto en el que escuchar (por defecto: 8080)")
    print("  -h, --help  Muestra este menú de ayuda y sale")
    print()
    print("Ejemplo:")
    print("  server.exe 8080")
    print("  server.exe --help")
    print()
    print("Este programa sirve todos los archivos de la carpeta actual vía una interfaz web.")
    print("Puedes descargar o visualizar archivos desde cualquier dispositivo en tu red.")
    print(Fore.MAGENTA + "Programa creado por: JAAM" + Style.RESET_ALL)
    print(Fore.YELLOW + "Para más información, visita la interfaz web o revisa los comentarios del código." + Style.RESET_ALL)
if __name__ == "__main__":
    if len(sys.argv) > 1 and sys.argv[1] in ("-h", "--help"):
        print_help()
        sys.exit(0)

    banner = r"""
░██████╗███████╗██████╗░██╗░░░██╗███████╗██████╗
██╔════╝██╔════╝██╔══██╗██║░░░██║██╔════╝██╔══██╗
╚█████╗░█████╗░░██████╔╝╚██╗░██╔╝█████╗░░██████╔╝
░╚═══██╗██╔══╝░░██╔══██╗░╚████╔╝░██╔══╝░░██╔══██╗
██████╔╝███████╗██║░░██║░░╚██╔╝░░███████╗██║░░██║
╚═════╝░╚══════╝╚═╝░░╚═╝░░░╚═╝░░░╚══════╝╚═╝░░╚═╝
"""
print(Fore.RED + banner + Style.RESET_ALL)
print(Fore.CYAN + f"Servidor iniciado en: http://{get_local_ip()}:{PORT}/" + Style.RESET_ALL)
print(Fore.YELLOW + "Usa: server -h para ver las instrucciones de uso." + Style.RESET_ALL)
print(Fore.YELLOW + "Presiona Ctrl+C para detener el servidor." + Style.RESET_ALL)
print(Fore.YELLOW + "Usa: server {PORT} ." + Style.RESET_ALL)
print(Fore.RED + "Programa creado por: JAAM" + Style.RESET_ALL)
print("----------------------------------------------------")
server = HTTPServer(('0.0.0.0', PORT), FileServerHandler)
try:
    server.serve_forever()
except KeyboardInterrupt:
    print(Fore.RED  + "\nServidor detenido." + Style.RESET_ALL)
