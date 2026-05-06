# EchoLox

**LoxBerry Plugin** — emuliert eine Philips Hue Bridge, damit Amazon Alexa den Loxone Miniserver über Virtual Inputs steuern kann.

```
Alexa
  |  Hue-API  (Port 80 via nginx-Proxy oder direkt)
  v
EchoLox  (LoxBerry Plugin, Port 8079)
  |  HTTP GET  /dev/sps/io/{name}/{value}   (Basic Auth)
  |  oder UDP  {name}={value}\r\n
  v
Loxone Miniserver
  |  Virtual Inputs --> Logik-Blöcke
  v
Echte Geräte  (Lampen, Rolläden, Szenen ...)
```

Loxone bleibt die einzige Automations-Zentrale. EchoLox ist ausschliesslich die Brücke zwischen Alexa und Loxone.

---

## Inhaltsverzeichnis

- [Warum EchoLox?](#warum-echolox)
- [Voraussetzungen](#voraussetzungen)
- [Installation](#installation)
- [Erste Schritte](#erste-schritte)
- [Geräte anlegen](#geräte-anlegen)
- [Virtual Inputs in Loxone einrichten](#virtual-inputs-in-loxone-einrichten)
- [Alexa-Erkennung](#alexa-erkennung)
- [Sprachbefehle](#sprachbefehle)
- [Transports: HTTP, UDP, MQTT](#transports-http-udp-mqtt)
- [Status-Übersicht](#status-übersicht)
- [Backup & Restore](#backup--restore)
- [Logs & Diagnose](#logs--diagnose)
- [Import alter Konfiguration](#import-alter-konfiguration)
- [Einstellungen](#einstellungen)
- [Konfigurationsdatei](#konfigurationsdatei)
- [Technische Architektur](#technische-architektur)
- [Build & Cross-Compilation](#build--cross-compilation)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## Warum EchoLox?

### Das Problem

Amazon Alexa kann nativ keine Loxone Virtual Inputs ansprechen. Die bisherige Lösung — ha-bridge (Java) — lief auf dem LoxBerry, verbrauchte aber 150–300 MB RAM und benötigte eine JVM.

### Die Lösung

EchoLox ist ein kompletter Neubau in **Go**:

| Kriterium | ha-bridge (Java) | EchoLox (Go) |
|---|---|---|
| RAM-Verbrauch | ~150–300 MB (JVM) | ~10–20 MB |
| Deployment | JAR + JRE | Einzelnes Binary, keine Deps |
| Startup-Zeit | 3–8 Sekunden | < 100 ms |
| ARM-Build | JRE muss installiert sein | `GOOS=linux GOARCH=arm64 go build` |
| Unterstützte Ziele | Vera, Fibaro, HASS, LIFX, ... | Nur Loxone (gezielt, schlank) |

### Funktionsprinzip

Alexa erkennt EchoLox als echte Philips Hue Bridge (SSDP/UPnP-Discovery). Jedes Gerät, das du in EchoLox anlegst, erscheint in der Alexa-App als Hue-Lampe. Wenn du "Alexa, schalte Wohnzimmer Licht ein" sagst, sendet EchoLox einen HTTP-GET-Request an den Loxone Miniserver:

```
GET http://192.168.1.7/dev/sps/io/echolox_wohnzimmer_licht_on/1
Authorization: Basic base64(user:password)
```

---

## Voraussetzungen

- **LoxBerry** ab Version 2.0 (Raspberry Pi oder x86)
- **Loxone Miniserver** (beliebige Generation) — im selben Netzwerk wie LoxBerry
- **Amazon Echo** (beliebiges Modell) — im selben Netzwerk
- Loxone Miniserver und Echo müssen im gleichen Subnetz liegen wie der LoxBerry (SSDP-Discovery funktioniert nicht über Router-Grenzen)

---

## Installation

### Über den LoxBerry Plugin Manager

1. Öffne die LoxBerry Web-Oberfläche → **Plugin Manager**
2. Klicke auf **Plugin installieren**
3. Wähle **Von URL installieren** und gib die ZIP-URL des aktuellen Releases ein (aus dem [GitHub Releases Tab](https://github.com/BattloXX/EchoLox/releases))
4. Nach der Installation erscheint **EchoLox** in der LoxBerry Navigation
5. Der Dienst startet automatisch auf Port **8079**

Das Installations-Script versucht automatisch einen **nginx-Proxy** für Port 80 einzurichten (Alexa-Kompatibilität — siehe [Alexa-Erkennung](#alexa-erkennung)).

### Prüfen ob der Dienst läuft

Öffne im Browser:
```
http://<loxberry-ip>:8079/description.xml
```
Du solltest ein XML-Dokument mit `Philips hue bridge 2015` sehen.

---

## Erste Schritte

### 1. Miniserver-Verbindung prüfen

Öffne **EchoLox → Einstellungen** (`/ui/settings.html`).

- Der Miniserver wird automatisch aus der globalen LoxBerry-Konfiguration gelesen. IP und Credentials müssen **nicht** erneut eingegeben werden.
- Wähle im Dropdown den gewünschten Miniserver (relevant wenn mehrere konfiguriert sind).
- Klicke **Verbindung testen** — du solltest "Verbindung OK" sehen.

### 2. Erstes Gerät anlegen

Öffne **EchoLox → Geräte → + Neu**.

- **Name:** `Wohnzimmer Licht` (genau so, wie du es Alexa nennen wirst)
- **Typ:** `Dimmer`
- **Transport:** `HTTP`
- Die generierten Virtual Input Namen werden sofort angezeigt, z.B.:
  - `echolox_wohnzimmer_licht_on`
  - `echolox_wohnzimmer_licht_off`
  - `echolox_wohnzimmer_licht_brightness`

Klicke **Speichern**.

### 3. Virtual Inputs im Loxone Config einrichten

Öffne Loxone Config und lege folgende Virtual Inputs an (Namen exakt wie oben):

- `echolox_wohnzimmer_licht_on` — Typ: Virtual Input, Wert 0/1
- `echolox_wohnzimmer_licht_off` — Typ: Virtual Input, Wert 0/1
- `echolox_wohnzimmer_licht_brightness` — Typ: Virtual Input, Wert 0–100

Verbinde sie mit deinen Logik-Blöcken und lade die Config auf den Miniserver.

### 4. Alexa: Neue Geräte suchen

Sage: **"Alexa, suche nach neuen Geräten"**  
oder öffne die Alexa-App → Geräte → + → Gerät hinzufügen → Licht → Philips Hue.

Alexa findet die Bridge und alle angelegten Geräte. Danach kannst du sagen:  
**"Alexa, schalte Wohnzimmer Licht ein"**

---

## Geräte anlegen

### Gerätetypen

| Typ | Alexa-Befehle | Generierte Virtual Inputs | Wertebereich |
|---|---|---|---|
| `switch` | Ein/Aus | `echolox_{name}_on`, `echolox_{name}_off` | `1` / `1` |
| `dimmer` | Ein/Aus, Helligkeit % | `echolox_{name}_on`, `echolox_{name}_off`, `echolox_{name}_brightness` | `1`/`1`, `0–100` |
| `color` | Ein/Aus, Helligkeit, Farbe | `echolox_{name}_on`, `echolox_{name}_off`, `echolox_{name}_brightness`, `echolox_{name}_hue`, `echolox_{name}_saturation` | diverse |
| `scene` | "Aktiviere ..." | `echolox_{name}_activate` | `1` (Puls) |

### Ein/Aus — getrennte Virtual Inputs

| Alexa-Befehl | Gesendeter Virtual Input | Wert |
|---|---|---|
| "Wohnzimmer Licht an" | `echolox_wohnzimmer_licht_on` | `1` |
| "Wohnzimmer Licht aus" | `echolox_wohnzimmer_licht_off` | `1` |
| "Wohnzimmer Licht auf 60%" | `echolox_wohnzimmer_licht_brightness` | `60` |

### Namensnormalisierung

| Eingabe | Normalisiert | VI-Prefix |
|---|---|---|
| `Wohnzimmer Licht` | `wohnzimmer_licht` | `echolox_wohnzimmer_licht_` |
| `Küche Decke` | `kueche_decke` | `echolox_kueche_decke_` |
| `Terrasse Süd` | `terrasse_sued` | `echolox_terrasse_sued_` |
| `Jalousie EG` | `jalousie_eg` | `echolox_jalousie_eg_` |

Regeln: Kleinbuchstaben, Umlaute → ae/oe/ue/ss, Sonderzeichen → Unterstrich.

---

## Virtual Inputs in Loxone einrichten

### HTTP Virtual Input

Im Loxone Config unter **Peripherie → Virtual Inputs**:

1. Neuen **Virtual HTTP Input** anlegen
2. Name: exakt wie von EchoLox generiert (z.B. `echolox_wohnzimmer_licht_on`)
3. HTTP-Methode: GET
4. Der Virtual Input empfängt den Wert aus der URL: `/dev/sps/io/echolox_wohnzimmer_licht_on/{value}`

### UDP Virtual Input (alternativ)

1. Neuen **Virtual UDP Input** anlegen
2. Port: `7777` (Standard, in EchoLox einstellbar)
3. Format: `{name}={value}` (EchoLox sendet `echolox_wohnzimmer_licht_on=1\r\n`)

### Empfehlung

| Szenario | Transport |
|---|---|
| Zuverlässigkeit wichtig | **HTTP** — Antwort-Bestätigung, Basic Auth |
| Latenz wichtig (< 5 ms) | **UDP** — keine TCP-Verbindung |
| Bereits MQTT Gateway im Einsatz | **MQTT** |

---

## Alexa-Erkennung

EchoLox emuliert eine Philips Hue Bridge Generation 2 (BSB002). Die Erkennung läuft über **SSDP/UPnP**:

1. Alexa sendet einen `M-SEARCH`-Broadcast ins Netzwerk (UDP Multicast `239.255.255.250:1900`)
2. EchoLox antwortet mit einer `HTTP/1.1 200 OK`-Unicast-Antwort und kündigt die Bridge zusätzlich via **SSDP NOTIFY** an
3. Alexa ruft `http://<loxberry-ip>:<discovery-port>/description.xml` ab
4. Alexa verbindet sich mit der Hue-API und liest alle Geräte

### Port 80 — wichtig für neuere Alexa-Firmware

Alexa-Geräte mit Firmware ab 2019 erwarten die Hue Bridge zwingend auf **Port 80** in der SSDP-LOCATION. EchoLox läuft selbst auf Port 8079; das Installations-Script richtet daher automatisch einen **nginx-Proxy** ein:

```nginx
location ~ ^/(api/|description\.xml$|hue_logo|favicon\.ico$) {
    proxy_pass http://127.0.0.1:8079;
    proxy_set_header Host $host;
}
```

Wenn die automatische Einrichtung erfolgreich war, setzt `postroot.sh` den `discovery_port` in der Konfiguration auf 80. Du siehst das Ergebnis im LoxBerry-Installationslog (`<OK> EchoLox nginx proxy configured`).

**Manuell einrichten** (falls die Automatik fehlschlägt):

1. Füge obigen Location-Block in den aktiven nginx-Server-Block ein (z.B. `/etc/nginx/sites-enabled/loxberry`)
2. `nginx -t && nginx -s reload`
3. Setze in **EchoLox → Einstellungen → Discovery-Port** den Wert `80`
4. EchoLox neu starten

Ausführliche Anleitung und Diagnose-Hilfe gibt es auf der **[Logs-Seite](#logs--diagnose)** (`/ui/logs.html`).

### Bridge-Identität

Die Bridge-UUID und Bridge-ID werden deterministisch aus der IP-Adresse des LoxBerry abgeleitet:
- Die Identität bleibt bei jedem Neustart gleich
- Alexa verliert die Bridge nicht nach einem Reboot
- Kein manuelles Pairing nötig

### SSDP-Flow

```
Echo Device                    EchoLox (Port 1900 UDP)
    |-- M-SEARCH (Multicast) ------->|
    |<-- HTTP/1.1 200 OK (Unicast) --|
    |     LOCATION: http://<ip>:80/description.xml
    |
    |-- GET :80/description.xml ---->|  (via nginx → 8079)
    |<-- XML (Philips hue bridge) ---|
    |
    |-- POST :80/api (pairing) ----->|
    |<-- {"success":{"username":..}} |
    |
    |-- GET :80/api/{user}/lights --->|
    |<-- { "1": {...}, "2": {...} }  |
```

---

## Sprachbefehle

| Befehl | Aktion |
|---|---|
| "Alexa, schalte [Name] ein" | `{name}_on = 1` |
| "Alexa, schalte [Name] aus" | `{name}_off = 1` |
| "Alexa, dimme [Name] auf 50 Prozent" | `{name}_brightness = 50` |
| "Alexa, stelle [Name] auf rot" | `{name}_hue = 0`, `{name}_saturation = 100` |
| "Alexa, aktiviere [Szenenname]" | `{name}_activate = 1` |

**Wichtig:** Der Gerätename in EchoLox muss exakt so lauten, wie du ihn Alexa sagst.

---

## Transports: HTTP, UDP, MQTT

### HTTP (Standard)

```
GET http://<miniserver-ip>:<port>/dev/sps/io/<name>/<value>
Authorization: Basic base64(<user>:<password>)
```

Credentials und IP werden aus der globalen LoxBerry-Konfiguration gelesen.

### UDP

EchoLox sendet UDP-Pakete an den Miniserver:
```
echolox_wohnzimmer_licht_on=1\r\n
```
Port einstellbar (Standard: 7777). Kein Handshake, keine Bestätigung — sehr geringe Latenz.

### MQTT

EchoLox publisht auf `loxone/{name}` mit dem Wert als Payload. Broker-URL in den Einstellungen konfigurierbar.

---

## Status-Übersicht

Die Status-Seite (`/ui/status.html`) zeigt für jeden Virtual Input:

| Status | Bedeutung |
|---|---|
| `ok` | Letzter Send war erfolgreich, VI im Miniserver gefunden |
| `not_found` | VI im Miniserver nicht gefunden — Namen prüfen |
| `access_denied` | Credentials falsch — Passwort prüfen |
| `not_sent` | Noch kein Befehl gesendet seit Start |

---

## Backup & Restore

EchoLox bietet eine eingebaute Backup-Funktion unter **EchoLox → Backup** (`/ui/backup.html`).

### Was wird gesichert

- `EchoLox.cfg` — alle Einstellungen
- `devices.json` — alle angelegten Geräte

### Optionen

| Funktion | Beschreibung |
|---|---|
| **Lokal speichern** | Erstellt ein ZIP im Datenverzeichnis (`data/backup/`) |
| **Herunterladen** | Erzeugt ein ZIP und lädt es direkt herunter |
| **Hochladen & Wiederherstellen** | ZIP-Datei per Drag & Drop oder Dateiauswahl hochladen |
| **Lokalen Backup wiederherstellen** | Aus der Liste der lokalen Backups wählen |

Nach einer Wiederherstellung EchoLox neu starten (**Einstellungen → Dienst neu starten**).

### Dienst neu starten

Konfigurationsänderungen (besonders Port-Änderungen) werden erst nach einem Neustart wirksam. Der Button **"Dienst neu starten"** auf der Einstellungsseite sendet einen Neustart-Befehl und wartet automatisch bis der Dienst wieder erreichbar ist.

---

## Logs & Diagnose

EchoLox schreibt alle Meldungen in einen **Ring-Buffer** (2000 Einträge) und optional in eine Logdatei (`$LBPLOGDIR/EchoLox.log`).

### Logs-Seite

Öffne **EchoLox → Logs** (`/ui/logs.html`):

- **Level-Anzeige** — aktueller Log-Level (INFO oder DEBUG)
- **Debug aktivieren** — schaltet auf DEBUG-Modus um (zeigt z.B. jeden SSDP-Paket-Empfang)
- **Auto-Refresh** — aktualisiert alle 5 Sekunden automatisch
- **Download** — lädt die aktuellen Einträge als Textdatei herunter

### Log-Level

| Level | Was wird geloggt |
|---|---|
| **INFO** (Standard) | Starts, SSDP M-SEARCH empfangen, SSDP Antworten gesendet, Fehler |
| **DEBUG** | Zusätzlich: jedes SSDP-Paket, `description.xml`-Abrufe, Hue-API-Details |

### Log-API

```
GET  /echolox/api/logs            JSON-Array aller gepufferten Einträge + aktueller Level
GET  /echolox/api/logs/download   Log-Datei als Download
POST /echolox/api/logs/level      {"level": "debug"} oder {"level": "info"}
```

### Alexa nicht gefunden — Diagnose-Workflow

1. Aktiviere **Debug-Modus** auf der Logs-Seite
2. Sage: "Alexa, suche nach neuen Geräten"
3. Prüfe in den Logs ob ein `SSDP M-SEARCH` Eintrag erscheint und EchoLox antwortet
4. Falls kein M-SEARCH: Echo und LoxBerry sind in unterschiedlichen Subnetzen oder Firewall blockiert Port 1900
5. Falls M-SEARCH empfangen aber Alexa findet trotzdem nichts: Port-80-Problem (nginx-Proxy prüfen, `discovery_port` in Einstellungen)

---

## Import alter Konfiguration

Falls du von ha-bridge migrierst, kannst du deine `devices.db` importieren:

1. Öffne **EchoLox → Import**
2. Lade die `devices.db` hoch (Drag & Drop oder Datei wählen)
3. EchoLox zeigt eine Vorschau der importierten Geräte
4. Klicke **Importieren**

**Hinweis:** Der VI-Prefix ändert sich von `ha_` auf `echolox_`. Die Virtual Inputs im Loxone Config müssen entsprechend angepasst werden.

---

## Einstellungen

Öffne **EchoLox → Einstellungen** (`/ui/settings.html`).

| Einstellung | Standard | Beschreibung |
|---|---|---|
| **Miniserver** | (aus LoxBerry) | Welcher Miniserver verwendet wird |
| **Transport** | HTTP | Übertragungsprotokoll |
| **UDP Port** | 7777 | Port für UDP-Transport |
| **EchoLox Port** | 8079 | Port auf dem EchoLox lauscht |
| **Discovery-Port** | 0 | Port in SSDP-LOCATION (0 = gleich wie EchoLox-Port; 80 = Alexa-kompatibel wenn nginx-Proxy läuft) |
| **MQTT Broker** | tcp://localhost:1883 | MQTT Broker URL |

Einstellungen werden direkt in `EchoLox.cfg` gespeichert. Port-Änderungen erfordern einen Neustart (**Dienst neu starten**-Button).

---

## Konfigurationsdatei

Die Konfiguration liegt unter `/opt/loxberry/data/plugins/EchoLox/EchoLox.cfg` (Datenpfad — bleibt bei Updates erhalten):

```yaml
server:
  port: 8079           # Port auf dem EchoLox lauscht
  ip: ""               # leer = automatisch erkannt
  discovery_port: 80   # Port in SSDP-LOCATION (0 = gleich wie port; 80 wenn nginx-Proxy)

upnp:
  name: "EchoLox"

loxone:
  miniserver: "1"      # Miniserver-ID aus LoxBerry-Konfiguration
  transport: "http"    # http, udp oder mqtt
  udp_port: 7777

mqtt:
  broker: "tcp://localhost:1883"
  username: ""
  password: ""

data_dir: ""           # leer = Standard-LoxBerry-Datenpfad
```

Die Miniserver-IP und Credentials werden automatisch aus `/opt/loxberry/config/system/general.json` gelesen.

**Wichtig:** Die Config liegt im `data/`-Verzeichnis, nicht im `config/`-Verzeichnis, da LoxBerry das `config/`-Verzeichnis bei Plugin-Updates überschreibt.

---

## Technische Architektur

```
cmd/EchoLox/
    main.go                    # Entry point, CLI flags

internal/
    bridge/
        bridge.go              # HTTP-Server, Startup-Logik, Port-80-Listener
        config.go              # YAML-Konfiguration (inkl. discovery_port)
    hue/
        api.go                 # Philips Hue REST API v1.47.0
        state.go               # Brightness/Hue/Sat Konvertierung
    upnp/
        listener.go            # SSDP Multicast-Listener, NOTIFY, description.xml
    device/
        model.go               # Device-Struct mit HueID
        manager.go             # CRUD, Persistenz, HueID-Vergabe
        naming.go              # Namensnormalisierung, VI-Generierung
        store.go               # JSON-Datei-Backend
    loxone/
        client.go              # HTTP/UDP/MQTT Send
        lbconfig.go            # LoxBerry general.json lesen
        verify.go              # VI-Status prüfen
        mqttbridge.go          # MQTT-Transport
    logbuf/
        logbuf.go              # Ring-Buffer Logger (INFO/DEBUG, Datei-Output)
    api/
        handler.go             # REST API /echolox/api/* (inkl. Logs, Backup, Restart)
    web/
        handler.go             # Statische Web-UI
    identity/
        identity.go            # Stabile Bridge-UUID aus IP
    migrate/
        importer.go            # ha-bridge devices.db Import

webembed/
    web/                       # Embedded Web-UI (Go embed.FS)
        index.html             # Geräteliste
        device.html            # Gerät anlegen/bearbeiten
        status.html            # VI-Status
        settings.html          # Einstellungen (inkl. Discovery-Port, Neustart)
        backup.html            # Backup & Restore
        logs.html              # Log-Anzeige, Level-Umschaltung, Download
        import.html            # ha-bridge Import
        about.html             # About / GitHub
        assets/
            app.js
            style.css
            logo.png
```

### SSDP-Flow (Alexa-Erkennung)

```
Echo Device                    EchoLox (Port 1900 UDP)
    |-- M-SEARCH (UDP Multicast) -->|
    |<-- HTTP/1.1 200 OK (Unicast) -|
    |     LOCATION: http://<ip>:80/description.xml
    |
    |  [EchoLox sendet auch proaktiv SSDP NOTIFY alle 30 min]
    |
    |-- GET :80/description.xml --->|  (nginx proxy → :8079)
    |<-- XML (Philips Hue Bridge) --|
    |
    |-- POST :80/api (pairing) ---->|
    |<-- {"success":{"username":..}}|
    |
    |-- GET :80/api/{user}/lights -->|
    |<-- { "1": {...}, "2": {...} } -|
```

### Hue API (implementierte Endpunkte)

```
GET  /description.xml
POST /api                          Pairing (immer erfolgreich)
GET  /api/{user}/lights            Alle Geräte
GET  /api/{user}/lights/{id}       Einzelnes Gerät
PUT  /api/{user}/lights/{id}/state Zustand setzen (on, bri, hue, sat)
GET  /api/{user}/groups            Gruppen (Group 0 = alle)
GET  /api/{user}/config            Bridge-Konfiguration
GET  /api/{user}/datastore         Vollständiger Datastore
```

---

## Build & Cross-Compilation

### Voraussetzungen

- Go 1.22+

### Lokaler Build

```bash
go build ./cmd/EchoLox/
./EchoLox --config ./data/EchoLox.cfg
```

EchoLox startet und ist unter `http://localhost:8079/ui/` erreichbar.

### Alle Plattformen bauen

```bash
make all
# oder einzeln:
GOOS=linux GOARCH=arm64       go build -o bin/EchoLox-arm64 ./cmd/EchoLox
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/EchoLox-armv7 ./cmd/EchoLox
GOOS=linux GOARCH=amd64       go build -o bin/EchoLox-amd64  ./cmd/EchoLox
```

### CI/CD

Bei jedem Push auf `main` baut GitHub Actions automatisch alle drei Plattformen, inkrementiert die Patch-Version und erstellt ein GitHub-Prerelease mit der Plugin-ZIP.

---

## Troubleshooting

### Alexa findet die Bridge nicht

**Prüfpunkte in dieser Reihenfolge:**

1. **Debug-Logs aktivieren** — öffne `/ui/logs.html`, schalte auf DEBUG, sage "Alexa, suche Geräte". Siehst du `SSDP M-SEARCH` im Log?

2. **description.xml erreichbar?**
   ```
   http://<loxberry-ip>:80/description.xml   # wenn nginx-Proxy läuft
   http://<loxberry-ip>:8079/description.xml  # direkt
   ```
   Muss ein XML mit `Philips hue bridge 2015` zurückgeben.

3. **Discovery-Port korrekt?** — Einstellungen → Discovery-Port.  
   - `80`: nginx-Proxy muss laufen  
   - `0` / `8079`: Alexa (neue Firmware) findet die Bridge möglicherweise nicht

4. **nginx-Proxy Status** — im LoxBerry-Installationslog:
   ```
   <OK> EchoLox nginx proxy configured (port 80 -> 8079 for Hue API)
   ```
   Falls nicht: Proxy manuell einrichten (Anleitung auf `/ui/logs.html`).

5. **Gleicher Subnetz?** Echo und LoxBerry müssen im selben Subnetz sein.

6. **Firewall?** UDP Port 1900 muss offen sein:
   ```bash
   iptables -A INPUT -p udp --dport 1900 -j ACCEPT
   ```

7. **SSDP-Konflikt?** Prüfen ob Port 1900 bereits belegt ist:
   ```bash
   ss -ulnp | grep 1900
   ```
   Falls `avahi-daemon` oder `miniupnpd` den Port belegen:
   ```bash
   systemctl stop avahi-daemon && systemctl disable avahi-daemon
   ```

### Alexa erkennt Geräte, aber Befehle kommen nicht an

1. **Testen-Button** in EchoLox → Geräteliste → Testen: Sendet direkt an den Miniserver.
2. **Status-Seite** prüfen: `not_found` → Virtual Input Namen im Loxone Config prüfen.
3. **Credentials** prüfen: Verbindungstest in Einstellungen.

### EchoLox startet nicht

```bash
systemctl status echolox.service
journalctl -u echolox.service -n 50

# Manuell starten (Logs direkt sehen):
LBHOMEDIR=/opt/loxberry \
  /opt/loxberry/bin/plugins/EchoLox/EchoLox \
  --config /opt/loxberry/data/plugins/EchoLox/EchoLox.cfg
```

### Geräte nach Update weg

EchoLox speichert `devices.json` im **Datenpfad** (`/opt/loxberry/data/plugins/EchoLox/`), nicht im Config-Pfad. Das Datenpfad-Verzeichnis wird bei Updates nicht gelöscht. Falls Geräte dennoch fehlen, war evtl. eine alte Version installiert, die noch den Config-Pfad verwendete — die Migration erfolgt automatisch beim nächsten Plugin-Install.

---

## FAQ

**Kann ich mehrere Echo-Geräte verwenden?**  
Ja. Alle Echos im gleichen Netzwerk finden die Bridge automatisch.

**Was passiert wenn EchoLox nicht läuft?**  
Alexa meldet "Gerät nicht erreichbar". Loxone selbst läuft unabhängig weiter.

**Kann ich EchoLox ohne LoxBerry verwenden?**  
Ja, als Standalone-Binary. Miniserver-Credentials manuell in `EchoLox.cfg` eintragen. Auf Port 80 gebundene Geräte können EchoLox direkt auf Port 80 betreiben (kein nginx nötig).

**Warum Discovery-Port 80 und nicht direkt Port 8079?**  
Neuere Alexa-Firmware ignoriert SSDP-Antworten mit nicht-Standard-Ports (nicht 80). Der nginx-Proxy leitet die spezifischen Hue-API-Pfade transparent zu EchoLox weiter, ohne andere LoxBerry-Funktionen zu beeinflussen.

**Wird HTTPS unterstützt?**  
Nein. Die Hue-API funktioniert nur über HTTP (wie die echte Bridge).

**Wie viele Geräte werden unterstützt?**  
Theoretisch unbegrenzt. Alexa hat ein Limit von ca. 300 Hue-Lampen pro Bridge.

**Alexa hat die Geräte gefunden, aber nach einem EchoLox-Neustart sind sie weg?**  
Die Bridge-UUID wird aus der IP-Adresse des LoxBerry berechnet und bleibt stabil. Falls nötig: "Alexa, suche nach neuen Geräten" nochmal ausführen.

---

## Sicherheit

EchoLox ist für den Betrieb im lokalen Netzwerk ausgelegt — keine Cloud-Verbindung nötig. Kein HTTPS, keine Authentifizierung an der Hue-API (wie die echte Bridge). Nicht direkt aus dem Internet erreichbar machen.

---

*EchoLox — [github.com/BattloXX/EchoLox](https://github.com/BattloXX/EchoLox)*
