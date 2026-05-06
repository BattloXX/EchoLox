# EchoLox

**LoxBerry Plugin** -- emuliert eine Philips Hue Bridge, damit Amazon Alexa den Loxone Miniserver über Virtual Inputs steuern kann.

```
Alexa
  |  Hue-API  (Port 8079)
  v
EchoLox  (LoxBerry Plugin)
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

Amazon Alexa kann nativ keine Loxone Virtual Inputs ansprechen. Die bisherige Lösung -- ha-bridge (Java) -- lief auf dem LoxBerry, verbrauchte aber 150-300 MB RAM und benötigte eine JVM.

### Die Lösung

EchoLox ist ein kompletter Neubau in **Go**:

| Kriterium | ha-bridge (Java) | EchoLox (Go) |
|---|---|---|
| RAM-Verbrauch | ~150-300 MB (JVM) | ~10-20 MB |
| Deployment | JAR + JRE | Einzelnes Binary, keine Deps |
| Startup-Zeit | 3-8 Sekunden | < 100 ms |
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
- **Loxone Miniserver** (beliebige Generation) -- im selben Netzwerk wie LoxBerry
- **Amazon Echo** (beliebiges Modell) -- im selben Netzwerk
- Loxone Miniserver und Echo müssen im gleichen Subnetz liegen wie der LoxBerry (SSDP-Discovery funktioniert nicht über Router-Grenzen)

---

## Installation

### Über den LoxBerry Plugin Manager

1. Öffne die LoxBerry Web-Oberfläche -> **Plugin Manager**
2. Klicke auf **Plugin installieren**
3. Wähle **Von URL installieren** und gib ein:
   ```
   https://github.com/BattloXX/EchoLox/archive/refs/heads/main.zip
   ```
   oder lade die ZIP-Datei herunter und lade sie manuell hoch
4. Nach der Installation erscheint **EchoLox** in der LoxBerry Navigation
5. Der Dienst startet automatisch auf Port **8079**

### Prüfen ob der Dienst läuft

Öffne im Browser:
```
http://<loxberry-ip>:8079/description.xml
```
Du solltest ein XML-Dokument mit `Philips hue bridge 2015` sehen. Wenn das klappt, ist EchoLox erreichbar und Alexa kann ihn finden.

---

## Erste Schritte

### 1. Miniserver-Verbindung prüfen

Öffne **EchoLox -> Einstellungen** (über die LoxBerry Navigation oder direkt `http://<loxberry-ip>:8079/ui/settings.html`).

- Der Miniserver wird automatisch aus der **globalen LoxBerry-Konfiguration** gelesen (`/opt/loxberry/config/system/miniserver.json`). Du musst die IP und Credentials **nicht** erneut eingeben.
- Wähle im Dropdown den gewünschten Miniserver (relevant wenn mehrere konfiguriert sind).
- Klicke **Verbindung testen** -- du solltest "Verbindung OK" sehen.

### 2. Erstes Gerät anlegen

Öffne **EchoLox -> Geräte -> + Neu**.

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

- `echolox_wohnzimmer_licht_on` -- Typ: Virtual Input, Wert 0/1
- `echolox_wohnzimmer_licht_off` -- Typ: Virtual Input, Wert 0/1
- `echolox_wohnzimmer_licht_brightness` -- Typ: Virtual Input, Wert 0-100

Verbinde sie mit deinen Logik-Blöcken und lade die Config auf den Miniserver.

### 4. Alexa: Neue Geräte suchen

Sage: **"Alexa, suche nach neuen Geräten"**  
oder öffne die Alexa-App -> Geräte -> + -> Gerät hinzufügen -> Licht -> Philips Hue.

Alexa findet die Bridge und alle angelegten Geräte. Danach kannst du sagen:  
**"Alexa, schalte Wohnzimmer Licht ein"**

---

## Geräte anlegen

### Gerätetypen

| Typ | Alexa-Befehle | Generierte Virtual Inputs | Wertebereich |
|---|---|---|---|
| `switch` | Ein/Aus | `echolox_{name}_on`, `echolox_{name}_off` | `1` / `1` |
| `dimmer` | Ein/Aus, Helligkeit % | `echolox_{name}_on`, `echolox_{name}_off`, `echolox_{name}_brightness` | `1`/`1`, `0-100` |
| `color` | Ein/Aus, Helligkeit, Farbe | `echolox_{name}_on`, `echolox_{name}_off`, `echolox_{name}_brightness`, `echolox_{name}_hue`, `echolox_{name}_saturation` | diverse |
| `scene` | "Aktiviere ..." | `echolox_{name}_activate` | `1` (Puls) |

### Ein/Aus -- getrennte Virtual Inputs

Jedes Gerät (ausser Szenen) hat separate VIs für Ein und Aus:

| Alexa-Befehl | Gesendeter Virtual Input | Wert |
|---|---|---|
| "Wohnzimmer Licht an" | `echolox_wohnzimmer_licht_on` | `1` |
| "Wohnzimmer Licht aus" | `echolox_wohnzimmer_licht_off` | `1` |
| "Wohnzimmer Licht auf 60%" | `echolox_wohnzimmer_licht_brightness` | `60` |

### Namensnormalisierung

Der Gerätename wird automatisch in einen Loxone-kompatiblen Virtual Input Namen umgewandelt:

| Eingabe | Normalisiert | VI-Prefix |
|---|---|---|
| `Wohnzimmer Licht` | `wohnzimmer_licht` | `echolox_wohnzimmer_licht_` |
| `Küche Decke` | `kueche_decke` | `echolox_kueche_decke_` |
| `Terrasse Süd` | `terrasse_sued` | `echolox_terrasse_sued_` |
| `Jalousie EG` | `jalousie_eg` | `echolox_jalousie_eg_` |

Regeln: Kleinbuchstaben, Umlaute -> ae/oe/ue/ss, Sonderzeichen -> Unterstrich.

### Beispiel: Alexa sagt "Wohnzimmer Licht auf 60 Prozent"

```
Hue API: PUT /api/{user}/lights/1/state
         { "on": true, "bri": 153 }

EchoLox sendet:
  GET .../dev/sps/io/echolox_wohnzimmer_licht_on/1
  GET .../dev/sps/io/echolox_wohnzimmer_licht_brightness/60
```

Die Helligkeit wird von Hue-Skala (0-254) auf Prozent (0-100) umgerechnet.

---

## Virtual Inputs in Loxone einrichten

### HTTP Virtual Input

Im Loxone Config unter **Peripherie -> Virtual Inputs**:

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
| Zuverlässigkeit wichtig | **HTTP** -- Antwort-Bestätigung, Basic Auth |
| Latenz wichtig (< 5 ms) | **UDP** -- keine TCP-Verbindung |
| Bereits MQTT Gateway im Einsatz | **MQTT** |

---

## Alexa-Erkennung

EchoLox emuliert eine Philips Hue Bridge Generation 2 (BSB002). Die Erkennung läuft über **SSDP/UPnP**:

1. Alexa sendet einen `M-SEARCH`-Broadcast ins Netzwerk (UDP Multicast `239.255.255.250:1900`)
2. EchoLox antwortet mit einer `HTTP/1.1 200 OK`-Unicast-Antwort
3. Alexa ruft `http://<loxberry-ip>:8079/description.xml` ab
4. Alexa verbindet sich mit der Hue-API und liest alle Geräte

### Bridge-Identität

Die Bridge-UUID und Bridge-ID werden deterministisch aus der IP-Adresse des LoxBerry abgeleitet. Das bedeutet:
- Die Identität bleibt bei jedem Neustart gleich
- Alexa verliert die Bridge nicht nach einem Reboot
- Kein manuelles Pairing nötig

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
Port einstellbar (Standard: 7777). Kein Handshake, keine Bestätigung -- sehr geringe Latenz.

### MQTT

EchoLox publisht auf `loxone/{name}` mit dem Wert als Payload. Broker-URL in den Einstellungen konfigurierbar.

---

## Status-Übersicht

Die Status-Seite (`/ui/status.html`) zeigt für jeden Virtual Input:

| Status | Bedeutung |
|---|---|
| `ok` | Letzter Send war erfolgreich, VI im Miniserver gefunden |
| `not_found` | VI im Miniserver nicht gefunden -- Name prüfen |
| `access_denied` | Credentials falsch -- Passwort prüfen |
| `not_sent` | Noch kein Befehl gesendet seit Start |

---

## Import alter Konfiguration

Falls du von ha-bridge migrierst, kannst du deine `devices.db` importieren:

1. Öffne **EchoLox -> Import**
2. Lade die `devices.db` hoch (Drag & Drop oder Datei wählen)
3. EchoLox zeigt eine Vorschau der importierten Geräte
4. Klicke **Importieren**

**Hinweis:** Die generierten Virtual Input Namen ändern sich durch den Import (Prefix `echolox_` statt `ha_`). Die Virtual Inputs im Loxone Config müssen entsprechend angepasst werden.

---

## Einstellungen

Öffne **EchoLox -> Einstellungen** (`/ui/settings.html`).

| Einstellung | Standard | Beschreibung |
|---|---|---|
| **Miniserver** | (aus LoxBerry) | Welcher Miniserver verwendet wird |
| **Transport** | HTTP | Übertragungsprotokoll |
| **UDP Port** | 7777 | Port für UDP-Transport |
| **EchoLox Port** | 8079 | Port auf dem EchoLox lauscht |
| **MQTT Broker** | tcp://localhost:1883 | MQTT Broker URL |

Einstellungen werden direkt in `EchoLox.cfg` gespeichert. Portänderungen erfordern einen Neustart von EchoLox.

---

## Konfigurationsdatei

Die Konfiguration liegt unter `/opt/loxberry/config/plugins/EchoLox/EchoLox.cfg`:

```yaml
server:
  port: 8079    # Port auf dem EchoLox lauscht
  ip: ""        # leer = automatisch erkannt

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

data_dir: ""    # leer = Standard-LoxBerry-Datenpfad
```

Die Miniserver-IP und Credentials werden automatisch aus `/opt/loxberry/config/system/miniserver.json` gelesen.

---

## Technische Architektur

```
cmd/EchoLox/
    main.go                    # Entry point, CLI flags

internal/
    bridge/
        bridge.go              # HTTP-Server, Startup-Logik
        config.go              # YAML-Konfiguration
    hue/
        api.go                 # Philips Hue REST API v1.47.0
        state.go               # Brightness/Hue/Sat Konvertierung
    upnp/
        listener.go            # SSDP Multicast-Listener + description.xml
    device/
        model.go               # Device-Struct mit HueID
        manager.go             # CRUD, Persistenz, HueID-Vergabe
        naming.go              # Namensnormalisierung, VI-Generierung
        store.go               # JSON-Datei-Backend
    loxone/
        client.go              # HTTP/UDP/MQTT Send
        lbconfig.go            # LoxBerry miniserver.json lesen
        verify.go              # VI-Status prüfen
        mqttbridge.go          # MQTT-Transport
    api/
        handler.go             # REST API /echolox/api/*
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
        settings.html          # Einstellungen
        import.html            # Import
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
    |     LOCATION: http://<ip>:8079/description.xml
    |
    |-- GET /description.xml ------>|
    |<-- XML (Philips Hue Bridge) --|
    |
    |-- POST /api (pairing) ------->|
    |<-- {"success":{"username":..}}|
    |
    |-- GET /api/{user}/lights ----->|
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

- Go 1.21+

### Lokaler Build

```bash
go build ./cmd/echolox/
./echolox --config ./config/EchoLox.cfg
```

EchoLox startet und ist unter `http://localhost:8079/ui/` erreichbar. Ohne Miniserver-Konfiguration läuft alles im Dry-Run-Modus (Virtual Inputs werden nur geloggt, nicht gesendet).

### Alle Plattformen bauen

```bash
GOOS=linux GOARCH=arm64 go build -o bin/EchoLox-arm64 ./cmd/echolox/
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/EchoLox-armv7 ./cmd/echolox/
GOOS=linux GOARCH=amd64 go build -o bin/EchoLox-amd64 ./cmd/echolox/
```

### Icons generieren

```bash
cd tools/genicons
go run main.go ../../icons
```

---

## Troubleshooting

### Alexa findet die Bridge nicht

**Prüfpunkte:**

1. **description.xml erreichbar?**
   ```
   http://<loxberry-ip>:8079/description.xml
   ```
   Muss ein XML mit `Philips hue bridge 2015` zurückgeben.

2. **Port 8079 offen?**
   ```bash
   ss -tulnp | grep 8079
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

6. **SSDP-Konflikt auf LoxBerry?** LoxBerry und der Loxone Miniserver nutzen selbst SSDP/UPnP (Port 1900 UDP). Wenn ein anderer Dienst Port 1900 exklusiv belegt, kann EchoLox den SSDP-Listener nicht starten:
   ```bash
   ss -ulnp | grep 1900
   ```
   Falls ein Konflikt besteht (z.B. `avahi-daemon`, `miniupnpd`):
   ```bash
   systemctl stop avahi-daemon
   systemctl disable avahi-daemon
   ```

### Alexa erkennt Geräte, aber Befehle kommen nicht an

1. **Testen-Button** in EchoLox -> Geräteliste -> Testen: Sendet einen Testbefehl direkt an den Miniserver.

2. **Status-Seite** prüfen: Zeigt `not_found`?  
   -> Virtual Input im Loxone Config anlegen, Namen exakt prüfen.

3. **Loxone Log** prüfen: Kommen HTTP-Requests beim Miniserver an?

4. **Credentials** prüfen: LoxBerry Miniserver-Konfiguration -> Verbindungstest in EchoLox Einstellungen.

### Virtual Input Name stimmt nicht

Der Name in Loxone Config muss **exakt** dem generierten Namen entsprechen (Gross-/Kleinschreibung beachtet, kein Leerzeichen).

Beispiel:
```
Gerätename in EchoLox:   "Wohnzimmer Licht"
Generierter VI-Name:     "echolox_wohnzimmer_licht_on"
                         "echolox_wohnzimmer_licht_off"
```

### EchoLox startet nicht

```bash
# Status prüfen
systemctl status echolox.service

# Log anzeigen
journalctl -u echolox.service -n 50

# Manuell starten (Debug)
LBHOMEDIR=/opt/loxberry \
  /opt/loxberry/bin/plugins/EchoLox/EchoLox \
  --config /opt/loxberry/config/plugins/EchoLox/EchoLox.cfg
```

### Miniserver wird nicht gefunden

EchoLox liest die Miniserver-Konfiguration aus:
```
/opt/loxberry/config/system/miniserver.json
```

Falls die Datei fehlt oder leer ist, muss zuerst ein Miniserver in der LoxBerry-Konfiguration hinterlegt werden.

---

## FAQ

**Kann ich mehrere Echo-Geräte verwenden?**  
Ja. Alle Echos im gleichen Netzwerk finden die Bridge automatisch.

**Was passiert wenn EchoLox nicht läuft?**  
Alexa meldet "Gerät nicht erreichbar". Loxone selbst läuft unabhängig weiter.

**Kann ich EchoLox ohne LoxBerry verwenden?**  
Ja, als Standalone-Binary. Miniserver-Credentials manuell in `EchoLox.cfg` eintragen.

**Wird HTTPS unterstützt?**  
Nein. Die Hue-API funktioniert nur über HTTP (wie die echte Bridge). Nur über VPN oder wenn Port 8079 im Router weitergeleitet wird.

**Wie viele Geräte werden unterstützt?**  
Theoretisch unbegrenzt. Alexa hat ein Limit von ca. 300 Hue-Lampen pro Bridge.

**Alexa hat die Geräte gefunden, aber nach einem EchoLox-Neustart sind sie weg?**  
Die Bridge-UUID wird aus der IP-Adresse des LoxBerry berechnet und bleibt stabil. Geräte sollten erhalten bleiben. Falls nicht: "Alexa, suche nach neuen Geräten" nochmal ausführen.

**Was bedeutet der Virtual Input Prefix `echolox_`?**  
Alle von EchoLox generierten Virtual Inputs beginnen mit `echolox_` um Kollisionen mit anderen Plugins zu vermeiden. Der Prefix kann nicht geändert werden.

---

## Sicherheit

Nur über VPN oder wenn Port 8079 im Router weitergeleitet wird (nicht empfohlen -- kein HTTPS, keine Authentifizierung). Im lokalen Netzwerk ist EchoLox vollständig offline-fähig -- keine Cloud-Verbindung nötig.

---

*EchoLox -- [github.com/BattloXX/EchoLox](https://github.com/BattloXX/EchoLox)*
