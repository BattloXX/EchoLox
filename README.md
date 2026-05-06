# EchoLox

**LoxBerry Plugin** — emuliert eine Philips Hue Bridge, damit Amazon Alexa den Loxone Miniserver über Virtual Inputs steuern kann.

```
Alexa
  │  Hue-API  (Port 8083)
  ▼
EchoLox  (LoxBerry Plugin)
  │  HTTP GET  /dev/sps/io/{name}/{value}   (Basic Auth)
  │  oder UDP  {name}={value}\r\n
  ▼
Loxone Miniserver
  │  Virtual Inputs → Logik-Blöcke
  ▼
Echte Geräte  (Lampen, Rolläden, Szenen …)
```

Loxone bleibt die einzige Automations-Zentrale. EchoLox ist ausschließlich die Brücke zwischen Alexa und Loxone.

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
| Unterstützte Ziele | Vera, Fibaro, HASS, LIFX, … | Nur Loxone (gezielt, schlank) |

### Funktionsprinzip

Alexa erkennt EchoLox als echte Philips Hue Bridge (SSDP/UPnP-Discovery). Jedes Gerät, das du in EchoLox anlegst, erscheint in der Alexa-App als Hue-Lampe. Wenn du „Alexa, schalte Wohnzimmer Licht ein" sagst, sendet EchoLox einen HTTP-GET-Request an den Loxone Miniserver:

```
GET http://192.168.1.7/dev/sps/io/ha_wohnzimmer_licht_on/1
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
3. Wähle **Von URL installieren** und gib ein:
   ```
   https://github.com/BattloXX/EchoLox/archive/refs/heads/main.zip
   ```
   oder lade die ZIP-Datei herunter und lade sie manuell hoch
4. Nach der Installation erscheint **EchoLox** in der LoxBerry Navigation
5. Der Dienst startet automatisch auf Port **8083**

### Prüfen ob der Dienst läuft

Öffne im Browser:
```
http://<loxberry-ip>:8083/description.xml
```
Du solltest ein XML-Dokument mit `Philips hue bridge 2015` sehen. Wenn das klappt, ist EchoLox erreichbar und Alexa kann ihn finden.

---

## Erste Schritte

### 1. Miniserver-Verbindung prüfen

Öffne **EchoLox → Einstellungen** (über die LoxBerry Navigation oder direkt `http://<loxberry-ip>:8083/ui/settings.html`).

- Der Miniserver wird automatisch aus der **globalen LoxBerry-Konfiguration** gelesen (`/opt/loxberry/config/system/miniserver.json`). Du musst die IP und Credentials **nicht** erneut eingeben.
- Wähle im Dropdown den gewünschten Miniserver (relevant wenn mehrere konfiguriert sind).
- Klicke **Verbindung testen** — du solltest „✅ Verbindung OK" sehen.

### 2. Erstes Gerät anlegen

Öffne **EchoLox → Geräte → + Neu**.

- **Name:** `Wohnzimmer Licht` (genau so, wie du es Alexa nennen wirst)
- **Typ:** `Dimmer`
- **Transport:** `HTTP`
- Die generierten Virtual Input Namen werden sofort angezeigt, z.B.:
  - `ha_wohnzimmer_licht_on`
  - `ha_wohnzimmer_licht_brightness`

Klicke **Speichern**.

### 3. Virtual Inputs im Loxone Config einrichten

Öffne Loxone Config und lege folgende Virtual Inputs an (Namen exakt wie oben):

- `ha_wohnzimmer_licht_on` — Typ: Virtual Input, Wert 0/1
- `ha_wohnzimmer_licht_brightness` — Typ: Virtual Input, Wert 0–100

Verbinde sie mit deinen Logik-Blöcken und lade die Config auf den Miniserver.

### 4. Alexa: Neue Geräte suchen

Sage: **„Alexa, suche nach neuen Geräten"**  
oder öffne die Alexa-App → Geräte → + → Gerät hinzufügen → Licht → Philips Hue.

Alexa findet die Bridge und alle angelegten Geräte. Danach kannst du sagen:  
**„Alexa, schalte Wohnzimmer Licht ein"**

---

## Geräte anlegen

### Gerätetypen

| Typ | Alexa-Befehle | Generierte Virtual Inputs | Wertebereich |
|---|---|---|---|
| `switch` | Ein/Aus | `ha_{name}_on` | `1` / `0` |
| `dimmer` | Ein/Aus, Helligkeit % | `ha_{name}_on`, `ha_{name}_brightness` | `1`/`0`, `0–100` |
| `color` | Ein/Aus, Helligkeit, Farbe | `ha_{name}_on`, `ha_{name}_brightness`, `ha_{name}_hue`, `ha_{name}_saturation` | diverse |
| `scene` | „Aktiviere …" | `ha_{name}_activate` | `1` (Puls) |

### Namensnormalisierung

Der Gerätename wird automatisch in einen Loxone-kompatiblen Virtual Input Namen umgewandelt:

| Eingabe | Normalisiert | VI-Prefix |
|---|---|---|
| `Wohnzimmer Licht` | `wohnzimmer_licht` | `ha_wohnzimmer_licht_` |
| `Küche Decke` | `kueche_decke` | `ha_kueche_decke_` |
| `Terrasse Süd` | `terrasse_sued` | `ha_terrasse_sued_` |
| `Jalousie EG` | `jalousie_eg` | `ha_jalousie_eg_` |

Regeln: Kleinbuchstaben, Umlaute → ae/oe/ue/ss, Sonderzeichen → Unterstrich.

### Beispiel: Alexa sagt „Wohnzimmer Licht auf 60 Prozent"

```
Hue API → PUT /api/{user}/lights/1/state
          { "on": true, "bri": 153 }

EchoLox sendet:
  GET .../dev/sps/io/ha_wohnzimmer_licht_on/1
  GET .../dev/sps/io/ha_wohnzimmer_licht_brightness/60
```

Die Helligkeit wird von Hue-Skala (0–254) auf Prozent (0–100) umgerechnet.

---

## Virtual Inputs in Loxone einrichten

### HTTP Virtual Input

Im Loxone Config unter **Peripherie → Virtual Inputs**:

1. Neuen **Virtual HTTP Input** anlegen
2. Name: exakt wie von EchoLox generiert (z.B. `ha_wohnzimmer_licht_on`)
3. HTTP-Methode: GET
4. Der Virtual Input empfängt den Wert aus der URL: `/dev/sps/io/ha_wohnzimmer_licht_on/{value}`

### UDP Virtual Input (alternativ)

1. Neuen **Virtual UDP Input** anlegen
2. Port: `7777` (Standard, in EchoLox einstellbar)
3. Format: `{name}={value}` (EchoLox sendet `ha_wohnzimmer_licht_on=1\r\n`)

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
2. EchoLox antwortet mit einer `HTTP/1.1 200 OK`-Unicast-Antwort
3. Alexa lädt `/description.xml` und bestätigt die Bridge-Identität
4. Alexa ruft `/api/{user}/lights` ab und importiert alle Geräte

### Wichtig für die Erkennung

- LoxBerry und Echo müssen im **gleichen Subnetz** sein
- Port **8083** muss auf dem LoxBerry erreichbar sein (kein Firewall-Block)
- Die Bridge-ID bleibt **dauerhaft stabil** (aus der IP abgeleitet) — Alexa verliert die Bridge nicht nach einem Neustart

### Erkennung wiederholen

Wenn neue Geräte hinzugefügt wurden:
- Alexa-App → **Geräte → Erkennen**
- oder: „**Alexa, suche nach neuen Geräten**"

---

## Sprachbefehle

| Befehl | Aktion | Gesendeter Virtual Input |
|---|---|---|
| „Schalte X ein" | on=1 | `ha_x_on = 1` |
| „Schalte X aus" | on=0 | `ha_x_on = 0` |
| „Stelle X auf 50 Prozent" | bri=50% | `ha_x_on = 1`, `ha_x_brightness = 50` |
| „Dimme X auf 20 Prozent" | bri=20% | `ha_x_brightness = 20` |
| „Stelle X auf Rot" | hue+sat | `ha_x_hue = 0`, `ha_x_saturation = 100` |
| „Aktiviere Szene Y" | activate=1 | `ha_y_activate = 1` |

### Tipp: Alexa-Gruppen

Fasse mehrere Geräte in der **Alexa-App zu einer Gruppe** zusammen (z.B. „Wohnzimmer"). Dann funktioniert „Alexa, schalte Wohnzimmer aus" für alle Geräte der Gruppe gleichzeitig.

---

## Transports: HTTP, UDP, MQTT

### HTTP (Standard)

```
GET http://{miniserver-ip}:{port}/dev/sps/io/{vi-name}/{value}
Authorization: Basic base64(user:password)
```

- Bestätigung durch HTTP-Statuscode (200 = OK, 401 = falsche Credentials)
- Credentials aus der globalen LoxBerry Miniserver-Konfiguration

### UDP

```
{vi-name}={value}\r\n
```

Ziel: `{miniserver-ip}:{udp-port}` (Standard: 7777)

- Kein Verbindungsaufbau, geringste Latenz
- Kein Fehler-Feedback möglich

### MQTT (über LoxBerry MQTT Gateway)

EchoLox published auf:
```
Topic:   ha_bridge/{device_name}/{property}
Payload: {value}
```

Das LoxBerry MQTT Gateway leitet die Nachrichten an den Miniserver weiter. EchoLox registriert beim Speichern eines Geräts automatisch die nötigen Subscriptions in der MQTT-Gateway-Konfiguration.

---

## Status-Übersicht

Die Status-Seite (`/ui/status.html`) zeigt alle Virtual Inputs mit ihrem aktuellen Zustand — ähnlich der MQTT-Gateway „Incoming Overview".

| Icon | Status | Bedeutung |
|---|---|---|
| ✅ | ok | Virtual Input existiert im Miniserver und wurde zuletzt erfolgreich angesprochen |
| 🟠 | not_found | Name im Miniserver nicht gefunden — Virtual Input noch nicht angelegt |
| 🔴 | access_denied | Falsche Credentials für den Miniserver |
| ⬜ | not_sent | Gerät noch nie ausgelöst |

```
┌──────────────────────────────────────┬──────────────┬─────────────────┐
│ Loxone Virtual Input Name            │ Letzter Wert │ Zuletzt gesendet│
├──────────────────────────────────────┼──────────────┼─────────────────┤
│ ✅ ha_wohnzimmer_licht_on            │ 1            │ 05.05. 22:14:03 │
│ ✅ ha_wohnzimmer_licht_brightness    │ 60           │ 05.05. 22:14:03 │
│ 🟠 ha_schlafzimmer_decke_on          │ —            │ nie             │
│ ⬜ ha_terrasse_szene_activate        │ —            │ nie             │
└──────────────────────────────────────┴──────────────┴─────────────────┘
```

Über **Refresh** wird die Loxone-Struktur (`LoxAPP3.json`) neu abgefragt.

---

## Import alter Konfiguration

Wer bisher ha-bridge verwendet hat, kann die bestehende `devices.db` importieren.

### Web-UI Import

1. Öffne **EchoLox → Import**
2. Lade die `devices.db` hoch (Drag & Drop)
3. EchoLox zeigt eine Vorschau:
   - Erkannte Geräte mit generierten Virtual Input Namen
   - Übersprungene Geräte (Vera, Fibaro, Harmony, … — diese Plattformen werden nicht mehr unterstützt)
4. Namen können manuell angepasst werden
5. Klicke **Importieren**

### Mapping-Logik

| Alter Typ | Neues Device | Hinweis |
|---|---|---|
| `httpDevice` (on/off URL) | `switch` | Virtual Input: `_on` |
| `httpDevice` (mit dimUrl) | `dimmer` | Virtual Inputs: `_on`, `_brightness` |
| `httpDevice` (mit colorUrl) | `color` | Virtual Inputs: `_on`, `_brightness`, `_hue`, `_saturation` |
| `execDevice` | `switch` | Exec-Logik entfällt |
| `veraDevice` | — | Übersprungen ⚠️ |
| `harmonyDevice` | — | Übersprungen ⚠️ |
| alle anderen Plattform-Typen | — | Übersprungen ⚠️ |

### CLI-Import (alternativ)

```bash
EchoLox --import /pfad/zur/devices.db --import-out /opt/loxberry/data/plugins/EchoLox/devices.json
```

---

## Einstellungen

Alle Einstellungen sind über die Web-UI unter `/ui/settings.html` zugänglich.

| Einstellung | Standard | Beschreibung |
|---|---|---|
| **Miniserver** | Erster in LoxBerry | Dropdown aus globaler LoxBerry-Konfiguration |
| **Transport** | HTTP | HTTP, UDP oder MQTT |
| **UDP Port** | 7777 | Ziel-Port für UDP-Transport |
| **EchoLox Port** | 8083 | Port auf dem EchoLox lauscht |
| **MQTT Broker** | tcp://localhost:1883 | Nur relevant bei MQTT-Transport |

> **Hinweis:** Miniserver-IP, Benutzername und Passwort werden **ausschließlich** aus der globalen LoxBerry-Konfiguration (`/opt/loxberry/config/system/miniserver.json`) gelesen. Du musst sie nicht erneut eingeben.

---

## Konfigurationsdatei

```yaml
# /opt/loxberry/config/plugins/EchoLox/EchoLox.cfg

server:
  port: 8083    # Port auf dem EchoLox lauscht
  ip: ""        # Leer = automatisch erkannt (empfohlen)

upnp:
  name: "EchoLox"
  uuid: ""      # Leer = aus IP abgeleitet (stabil über Neustarts)

loxone:
  miniserver: "1"     # ID aus LoxBerry miniserver.json
  transport: "http"   # http | udp | mqtt
  udp_port: 7777

mqtt:
  broker: "tcp://localhost:1883"
  username: ""
  password: ""

data_dir: ""    # Leer = $LBPDATA (automatisch durch LoxBerry gesetzt)
```

Die Geräte werden gespeichert in:
```
/opt/loxberry/data/plugins/EchoLox/devices.json
```

---

## Technische Architektur

### Projektstruktur

```
EchoLox/
├── cmd/EchoLox/
│   └── main.go                    # Entry Point, Flag-Parsing
├── internal/
│   ├── identity/
│   │   └── identity.go            # Bridge-Identität (UUID, bridgeid, MAC)
│   ├── bridge/
│   │   ├── bridge.go              # HTTP-Server Init, Komponentenverbindung
│   │   └── config.go              # YAML-Config, LoxBerry Env-Vars
│   ├── hue/
│   │   ├── api.go                 # Hue REST Endpoints (lights, groups, config)
│   │   └── state.go               # Brightness/Hue/Sat Konvertierung
│   ├── upnp/
│   │   └── listener.go            # SSDP Multicast-Listener + description.xml
│   ├── device/
│   │   ├── manager.go             # Device-Registry (CRUD, State, HueID)
│   │   ├── store.go               # JSON-Persistenz (atomic write)
│   │   ├── model.go               # Device/State Typen
│   │   └── naming.go              # Auto-Namens-Generierung
│   ├── loxone/
│   │   ├── client.go              # HTTP + UDP Transport
│   │   ├── verify.go              # LoxAPP3.json Abfrage, Status-Check
│   │   ├── lbconfig.go            # LoxBerry Miniserver-Config lesen
│   │   └── mqttbridge.go          # MQTT-Gateway Auto-Registrierung
│   ├── api/
│   │   └── handler.go             # Management REST API (/echolox/api/...)
│   ├── migrate/
│   │   └── importer.go            # devices.db → neues Format
│   └── web/
│       └── handler.go             # Embedded Web-UI Handler
└── webembed/
    └── web/                       # Eingebettetes Frontend (embed.FS)
        ├── index.html             # Geräteübersicht
        ├── device.html            # Anlegen / Bearbeiten
        ├── status.html            # Virtual Input Status
        ├── settings.html          # Einstellungen
        ├── import.html            # Import
        └── assets/
            ├── app.js
            └── style.css
```

### Bridge-Identität

Die Bridge-ID (hue-bridgeid) und UUID werden **deterministisch aus der IP-Adresse** abgeleitet:

```
IP: 192.168.1.100
 → MD5("echolox-bridge:192.168.1.100")
 → suffix = 12 Hex-Zeichen
 → UUID    = 2f402f80-da50-11e1-9b23-{suffix}
 → bridgeid = 001788FFFE{suffix[6:]}  (Philips OUI + FFFE + 6 Hex)
```

Das bedeutet: Nach einem Neustart behält EchoLox dieselbe Identität — Alexa verliert die Bridge nicht.

### SSDP-Flow (Alexa-Erkennung)

```
Echo                          EchoLox
 │                                │
 │── M-SEARCH (UDP Multicast) ───▶│  Port 1900
 │                                │  isMSearch() prüft ST-Header
 │◀── HTTP/1.1 200 OK (Unicast) ──│  eigener ephemerer UDP-Socket
 │    LOCATION: .../description.xml
 │    USN: uuid:...::urn:...:Basic:1
 │    hue-bridgeid: 001788FFFE...
 │                                │
 │── GET /description.xml ────────▶│
 │◀── XML (serialNumber=bridgeid) ─│
 │                                │
 │── POST /api (Pairing) ─────────▶│
 │◀── [{"success":{"username":...}}]
 │                                │
 │── GET /api/{user}/lights ──────▶│
 │◀── {"1": {...}, "2": {...}}  ───│
```

### Hue Light IDs

Intern werden Geräte mit UUIDs gespeichert. Für die Hue API wird jedes Gerät einer stabilen, kurzen numerischen ID (`HueID`) zugewiesen — diese wird beim ersten Anlegen vergeben und dann permanent in der `devices.json` gespeichert.

---

## Build & Cross-Compilation

### Voraussetzungen

- Go 1.22 oder neuer
- Internet (für `go mod download`)

### Alle Plattformen bauen

```bash
make all
```

Erzeugt:
- `bin/EchoLox-arm64` — Raspberry Pi 4, Orange Pi, Pine64
- `bin/EchoLox-armv7` — Raspberry Pi 2/3 (32-bit ARM)
- `bin/EchoLox-amd64` — DietPi x86, VirtualBox VM

### Einzelne Plattform

```bash
# arm64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/EchoLox-arm64 ./cmd/EchoLox

# armv7
GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-s -w" -o bin/EchoLox-armv7 ./cmd/EchoLox

# amd64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/EchoLox-amd64 ./cmd/EchoLox
```

### Lokal entwickeln (ohne LoxBerry)

```bash
go run ./cmd/EchoLox --config config/EchoLox.cfg
```

EchoLox startet und ist unter `http://localhost:8083/ui/` erreichbar. Ohne Miniserver-Konfiguration läuft alles im Dry-Run-Modus (Virtual Inputs werden nur geloggt, nicht gesendet).

### Plugin-ZIP erstellen

```bash
make zip
```

Erzeugt `EchoLox-1.0.0.zip` — direkt in den LoxBerry Plugin Manager hochladbar.

---

## Troubleshooting

### Alexa findet die Bridge nicht

**Prüfpunkte:**

1. **description.xml erreichbar?**
   ```
   http://<loxberry-ip>:8083/description.xml
   ```
   Muss ein XML mit `Philips hue bridge 2015` zurückgeben.

2. **Port 8083 offen?**
   ```bash
   # Auf dem LoxBerry:
   ss -tulnp | grep 8083
   ```

3. **Gleicher Subnetz?** Echo und LoxBerry müssen im selben Subnetz sein. SSDP-Broadcasts werden nicht über Router-Grenzen weitergeleitet.

4. **SSDP-Listener läuft?** Im LoxBerry Log prüfen:
   ```
   SSDP listener on 239.255.255.250:1900  bridgeid=001788FFFE...
   ```

5. **Firewall?** Auf manchen Systemen blockiert `ufw` oder `iptables` UDP Port 1900:
   ```bash
   iptables -A INPUT -p udp --dport 1900 -j ACCEPT
   ```

### Alexa erkennt Geräte, aber Befehle kommen nicht an

1. **Testen-Button** in EchoLox → Geräteliste → Testen: Sendet einen Testbefehl direkt an den Miniserver.

2. **Status-Seite** prüfen: Zeigt `not_found`?  
   → Virtual Input im Loxone Config anlegen, Namen exakt prüfen.

3. **Loxone Log** prüfen: Kommen HTTP-Requests beim Miniserver an?

4. **Credentials** prüfen: LoxBerry Miniserver-Konfiguration → Verbindungstest in EchoLox Einstellungen.

### Virtual Input Name stimmt nicht

Der Name in Loxone Config muss **exakt** dem generierten Namen entsprechen (Groß-/Kleinschreibung beachtet, kein Leerzeichen).

Beispiel: Gerät heißt `Wohnzimmer Licht` → Virtual Input muss heißen `ha_wohnzimmer_licht_on`.

### Nach Neustart verliert Alexa die Geräte

Das sollte nicht passieren — die Bridge-ID ist deterministisch aus der IP abgeleitet. Falls die LoxBerry-IP sich geändert hat, ändert sich auch die Bridge-ID. Lösung: Feste IP für den LoxBerry vergeben (DHCP-Reservation im Router).

### Import schlägt fehl

- Nur `devices.db` aus ha-bridge wird unterstützt (JSON-Array-Format)
- Plattform-spezifische Geräte (Vera, Fibaro, Harmony, …) werden übersprungen — das ist erwartet
- Bei kaputtem JSON: Datei in einem Editor öffnen und auf Gültigkeit prüfen

---

## FAQ

**Kann EchoLox mehrere Loxone Miniserver gleichzeitig ansprechen?**  
Nein — aktuell wird pro Transport ein Miniserver unterstützt. Mehrere Miniserver sind geplant.

**Muss ich den Hue-Link-Button drücken?**  
Nein — EchoLox akzeptiert alle Pairing-Anfragen automatisch.

**Funktioniert EchoLox auch mit Google Home oder Apple HomeKit?**  
Google Home unterstützt die Hue-Emulation ebenfalls, ist aber nicht primär getestet. Apple HomeKit nutzt ein anderes Protokoll (HomeKit Accessory Protocol) und wird nicht unterstützt.

**Kann ich bestehende Alexa-Routinen behalten?**  
Ja — solange die Gerätenamen gleich bleiben und die Bridge-ID stabil ist (was sie ist, solange die LoxBerry-IP gleich bleibt), funktionieren alle Routinen weiterhin.

**Wie viele Geräte sind möglich?**  
Die Hue API unterstützt bis zu 50 Lampen. EchoLox hat keine eigene Begrenzung.

**Funktioniert EchoLox ohne Internet?**  
Vollständig — SSDP-Discovery und Hue-API laufen rein lokal. Keine Cloud-Verbindung nötig.

**Kann ich die Web-UI von außerhalb des LANs erreichen?**  
Nur über VPN oder wenn Port 8083 im Router weitergeleitet wird (nicht empfohlen — kein HTTPS, keine Authentifizierung).

---

## Lizenz

MIT License — siehe [LICENSE](LICENSE)

---

## Credits

Basiert auf den Ideen von [ha-bridge](https://github.com/bwssystems/ha-bridge) von bwssystems.  
Neu entwickelt in Go als schlankes LoxBerry-Plugin.

Maintainer: Johannes Battlogg — johannes@battlogg.org
