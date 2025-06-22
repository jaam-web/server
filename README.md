!(server.jpg)
# Servidor de Archivos Web

Este programa es un **servidor web sencillo** que permite compartir y descargar archivos de la carpeta actual a través de una interfaz web moderna y responsiva. Puedes acceder desde cualquier dispositivo en tu red local.

## Características

- Sirve todos los archivos de la carpeta actual.
- Permite **descargar** o **visualizar** archivos compatibles (texto, imágenes, audio, video, PDF).
- Interfaz web atractiva y adaptativa.
- Muestra el tamaño de los archivos en formato legible.
- Indica la URL de acceso desde otros dispositivos de la red.
- Compatible con Windows, Linux y macOS (requiere Python 3).

## Requisitos

- Python 3.x
- [colorama](https://pypi.org/project/colorama/)  
  Instala con:  
  ```sh
  pip install colorama
  ```

## Uso

Desde la terminal, navega a la carpeta donde está `server.py` y ejecuta:

```sh
python server.py [PUERTO]
```

- **PUERTO**: (opcional) Puerto en el que escuchar (por defecto: 8080).

### Ejemplos

## Python
```sh
# Usando el programa de python
python server.py
python server.py 8080
python server.py --help
```
## Windows
```sh
# Usando el programa de Windows
.\server.exe
.\server.exe 8080
.\server.exe --help
```
## Opciones

- `-h`, `--help`: Muestra el menú de ayuda y sale.

## Acceso

Abre un navegador y visita la URL que aparece en la terminal, por ejemplo:  
`http://192.168.1.100:8080/`

Desde ahí podrás ver, descargar o visualizar los archivos disponibles.

## Seguridad

- **No utilices este servidor en entornos públicos o inseguros.**
- Sirve todos los archivos de la carpeta actual, sin autenticación.

## Créditos

Programa creado por: **JAAM**

---
