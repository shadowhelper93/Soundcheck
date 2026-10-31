# Soundcheck (Español)

> **Sonda de Verificación Previa Determinista y Resolutor de Roles para Agentes de Código con IA**

Idiomas: [English](../../README.md) · [Français](README.fr.md) · [فارسی](README.fa.md)

Soundcheck evalúa las herramientas CLI de agentes de programación instaladas en tu sistema (`claude`, `codex`, `opencode`, `agy`, `gemini`, `copilot`) antes de delegarles tareas. Verifica su disponibilidad, intercepta errores engañosos con código de salida cero (falsos éxitos), normaliza anomalías específicas de cada proveedor, asigna roles de ejecución de forma determinista sin consumir tokens de LLM y proporciona un sistema automatizado de tolerancia a fallos.

---

## Características Principales

- **Detección de Códigos de Salida Engañosos**: Muchas herramientas CLI terminan con código `0` incluso tras un fallo crítico (límites de cuota, tokens OAuth expirados). Soundcheck reclasifica estas salidas en errores normalizados (`429`, `401`, `corrupted`).
- **Resolución de Roles con Cero Tokens**: Decide qué agente asume las funciones de `primary`, `subagent`, `reviewer` o `auxiliary` en ≤ 50µs mediante lógica determinista, sin gastar presupuesto en llamadas a LLMs.
- **Ejecución Desatendida con Conmutación por Error**: Ejecuta instrucciones mediante `soundcheck exec "prompt"`. Si el agente principal agota su cuota o no está autenticado, Soundcheck conmuta automáticamente al siguiente agente disponible en la cadena.
- **Sondeo en Cuatro Niveles (Tiers)**:
  - **L0 Presencia**: Comprueba la existencia del binario y su versión en `$PATH` (≤ 150ms).
  - **L1 Autenticación**: Valida credenciales y sesiones activas con caché TTL para proteger los tokens de actualización.
  - **L2 Prueba de Humo**: Ejecución rápida y no destructiva de comprobación.
  - **L3 Verificación de Modelo**: Comprobación directa de respuesta del modelo en vivo.
- **Binario Estático Único**: Servidor backend e interfaz web React interactiva empaquetados en un único archivo ejecutable (~8.3 MB, presupuesto ≤ 20 MB). Sin dependencias externas de ejecución.
- **Protocolo de Contexto de Modelo (MCP)**: Servidor stdio MCP nativo (`soundcheck mcp`) integrado directamente con Claude Desktop, Cursor y Antigravity.
- **Extensión para VS Code**: Disponible en `editors/vscode` con barra de estado, árbol de explorador y panel web incrustado.

---

## Instalación

### macOS y Linux
```bash
curl -fsSL https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.sh | sh
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/soundcheck/soundcheck/main/scripts/install.ps1 | iex
```

### Homebrew
```bash
brew install soundcheck/tap/soundcheck
```

### Scoop (Windows)
```powershell
scoop bucket add soundcheck https://github.com/soundcheck/scoop-bucket
scoop install soundcheck
```

### Actualización Automática
```bash
soundcheck upgrade
```

### Compilación desde Código Fuente
```bash
git clone https://github.com/soundcheck/soundcheck.git
cd soundcheck
make build
./bin/soundcheck doctor
```

---

## Referencia de Comandos CLI

| Comando | Descripción |
|---|---|
| `soundcheck doctor [--deep] [--json]` | Diagnóstico exhaustivo de todos los adaptadores instalados |
| `soundcheck probe <adapter> [--deep]` | Sondea un único adaptador específico (ej. `soundcheck probe claude`) |
| `soundcheck resolve [role]` | Muestra la asignación determinista de roles (ej. `soundcheck resolve primary`) |
| `soundcheck exec <prompt> [--role <role>] [--adapter <adapter>]` | Ejecuta un prompt con conmutación automática ante fallos |
| `soundcheck run <adapter> [-- <args>]` | Inicia una sesión PTY interactiva con flujo en tiempo real |
| `soundcheck manifest [--json]` | Exporta el manifiesto de capacidades en formato JSON |
| `soundcheck serve [--port 3000] [--token <token>]` | Inicia el servidor HTTP y el panel web visual (puerto 3000 por defecto) |
| `soundcheck mcp` | Inicia el servidor stdio bajo el estándar Model Context Protocol (MCP) |
| `soundcheck upgrade [--check]` | Comprueba actualizaciones e instala la versión más reciente |
| `soundcheck version` | Muestra la versión y metadatos de compilación |

---

## Referencia de API REST y Streaming

Al ejecutar `soundcheck serve [--port 3000]`, el servidor local expone:

| Endpoint | Método | Descripción |
|---|---|---|
| `/api/health` | `GET` | Estado del servidor y versión activa |
| `/api/v1/doctor` | `GET` | Informe de diagnóstico completo (admite flujo SSE con `Accept: text/event-stream`) |
| `/api/v1/probe` | `GET` | Sondeo de todos los adaptadores CLI |
| `/api/v1/probe/{id}` | `POST` | Dispara el sondeo de un adaptador individual |
| `/api/v1/resolve` | `GET`, `POST` | Resolución determinista de roles y fallbacks |
| `/api/v1/manifest` | `GET` | Manifiesto de capacidades en formato JSON |
| `/api/v1/config` | `GET` | Lectura de configuración activa (devuelve cabecera `X-Config-Path`) |
| `/api/v1/config` | `PUT` | Validación atómica y guardado de configuración en archivo YAML |
| `/api/v1/config/validate` | `POST` | Valida candidatos frente a los adaptadores reales del sistema |
| `/api/v1/terminal/ws` | `WS` | Conexión WebSocket PTY para terminal interactiva en streaming |

---

## Descubrimiento Dinámico de Modelos (`modelsCommand`)

Los adaptadores pueden declarar opcionalmente un comando de descubrimiento dinámico en su definición YAML (como `modelsCommand: "codex debug models"` para Codex).

- **Con `modelsCommand`**: Soundcheck ejecuta este comando durante el sondeo. Los modelos admitidos por la cuenta activa se incorporan dinámicamente a la lista disponible.
- **Sin `modelsCommand`**: Para herramientas CLI sin comando de inspección, Soundcheck preserva el catálogo estático curado en el archivo YAML.

---

## Arquitectura

Soundcheck implementa una **Arquitectura Hexagonal** estricta:

- **Núcleo Funcional Puro** (`internal/core/*`):
  - Completamente determinista: entrada de datos → salida de datos.
  - Cero operaciones de entrada/salida, sin llamadas a reloj del sistema ni variables de entorno.
  - Exclusivamente biblioteca estándar de Go. Verificado mediante `internal/arch_test.go`.
- **Capa Imperativa (Edge y Transportes)** (`internal/edge/*`, `internal/transport/*`):
  - Ejecución de subprocesos (`internal/edge/exec`), streaming PTY, carga de configuración y actualización atómica.
  - Interfaces: CLI (`cobra`), servidor HTTP (`net/http` con interfaz React incrustada), WebSocket y servidor MCP.

---

## Documentación para Colaboradores

- [Guía de Creación de Adaptadores](../../docs/ADAPTER_AUTHORING.md) — Cómo incorporar nuevos agentes y definir sus niveles de sondeo.
- [Guía de Creación de Quirks](../../docs/QUIRK_AUTHORING.md) — Cómo catalogar comportamientos anómalos y reglas de saneamiento.
- [Guía del Colaborador](../../CONTRIBUTING.md) — Organización por carriles (lanes), estándares de código y puertas de calidad.
- [Manual de Arquitectura Claude/Agent](../../CLAUDE.md) — Reglas y convenciones para agentes de desarrollo y asistentes.

---

## Licencia

Apache-2.0. Copyright (c) 2026 Contribuidores de Soundcheck.
