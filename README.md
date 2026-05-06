# EchoLox

**LoxBerry Plugin** â€” emuliert eine Philips Hue Bridge, damit Amazon Alexa den Loxone Miniserver Ã¼ber Virtual Inputs steuern kann.

```
Alexa
  â”‚  Hue-API  (Port 8079)
  â–¼
EchoLox  (LoxBerry Plugin)
  â”‚  HTTP GET  /dev/sps/io/{name}/{value}   (Basic Auth)
  â”‚  oder UDP  {name}={value}\r\n
  â–¼
Loxone Miniserver
  â”‚  Virtual Inputs â†’ Logik-BlÃ¶cke
  â–¼
Echte GerÃ¤te  (Lampen, RollÃ¤den, Szenen â€¦)
```

Loxone bleibt die einzige Automations-Zentrale. EchoLox ist ausschlieÃŸlich die BrÃ¼cke zwischen Alexa und Loxone.

---

## Inhaltsverzeichnis

- [Warum EchoLox?](#warum-echolox)
- [Voraussetzungen](#voraussetzungen)
- [Installation](#installation)
- [Erste Schritte](#erste-schritte)
- [GerÃ¤te anlegen](#gerÃ¤te-anlegen)
- [Virtual Inputs in Loxone einrichten](#virtual-inputs-in-loxone-einrichten)
- [Alexa-Erkennung](#alexa-erkennung)
- [Sprachbefehle](#sprachbefehle)
- [Transports: HTTP, UDP, MQTT](#transports-http-udp-mqtt)
- [Status-Ãœbersicht](#status-Ã¼bersicht)
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

Amazon Alexa kann nativ keine Loxone Virtual Inputs ansprechen. Die bisherige LÃ¶sung â€” ha-bridge (Java) â€” lief auf dem LoxBerry, verbrauchte aber 150â€“300 MB RAM und benÃ¶tigte eine JVM.

### Die LÃ¶sung

EchoLox ist ein kompletter Neubau in **Go**:

| Kriterium | ha-bridge (Java) | EchoLox (Go) |
|---|---|---|
| RAM-Verbrauch | ~150â€“300 MB (JVM) | ~10â€“20 MB |
| Deployment | JAR + JRE | Einzelnes Binary, keine Deps |
| Startup-Zeit | 3â€“8 Sekunden | < 100 ms |
| ARM-Build | JRE muss installiert sein | `GOOS=linux GOARCH=arm64 go build` |
| UnterstÃ¼tzte Ziele | Vera, Fibaro, HASS, LIFX, â€¦ | Nur Loxone (gezielt, schlank) |

### Funktionsprinzip

Alexa erkennt EchoLox als echte Philips Hue Bridge (SSDP/UPnP-Discovery). Jedes GerÃ¤t, das du in EchoLox anlegst, erscheint in der Alexa-App als Hue-Lampe. Wenn du â€žAlexa, schalte Wohnzimmer Licht ein" sagst, sendet EchoLox einen HTTP-GET-Request an den Loxone Miniserver:

```
GET http://192.168.1.7/dev/sps/io/ha_wohnzimmer_licht_on/1
Authorization: Basic base64(user:password)
```

---

## Voraussetzungen

- **LoxBerry** ab Version 2.0 (Raspberry Pi oder x86)
- **Loxone Miniserver** (beliebige Generation) â€” im selben Netzwerk wie LoxBerry
- **Amazon Echo** (beliebiges Modell) â€” im selben Netzwerk
- Loxone Miniserver und Echo mÃ¼ssen im gleichen Subnetz liegen wie der LoxBerry (SSDP-Discovery funktioniert nicht Ã¼ber Router-Grenzen)

---

## Installation

### Ãœber den LoxBerry Plugin Manager

1. Ã–ffne die LoxBerry Web-OberflÃ¤che â†’ **Plugin Manager**
2. Klicke auf **Plugin installieren**
3. WÃ¤hle **Von URL installieren** und gib ein:
   ```
   https://github.com/BattloXX/EchoLox/archive/refs/heads/main.zip
   ```
   oder lade die ZIP-Datei herunter und lade sie manuell hoch
4. Nach der Installation erscheint **EchoLox** in der LoxBerry Navigation
5. Der Dienst startet automatisch auf Port **8079**

### PrÃ¼fen ob der Dienst lÃ¤uft

Ã–ffne im Browser:
```
http://<loxberry-ip>:8079/description.xml
```
Du solltest ein XML-Dokument mit `Philips hue bridge 2015` sehen. Wenn das klappt, ist EchoLox erreichbar und Alexa kann ihn finden.

---

## Erste Schritte

### 1. Miniserver-Verbindung prÃ¼fen

Ã–ffne **EchoLox â†’ Einstellungen** (Ã¼ber die LoxBerry Navigation oder direkt `http://<loxberry-ip>:8079/ui/settings.html`).

- Der Miniserver wird automatisch aus der **globalen LoxBerry-Konfiguration** gelesen (`/opt/loxberry/config/system/miniserver.json`). Du musst die IP und Credentials **nicht** erneut eingeben.
- WÃ¤hle im Dropdown den gewÃ¼nschten Miniserver (relevant wenn mehrere konfiguriert sind).
- Klicke **Verbindung testen** â€” du solltest â€žâœ… Verbindung OK" sehen.

### 2. Erstes GerÃ¤t anlegen

Ã–ffne **EchoLox â†’ GerÃ¤te â†’ + Neu**.

- **Name:** `Wohnzimmer Licht` (genau so, wie du es Alexa nennen wirst)
- **Typ:** `Dimmer`
- **Transport:** `HTTP`
- Die generierten Virtual Input Namen werden sofort angezeigt, z.B.:
  - `ha_wohnzimmer_licht_on`
  - `ha_wohnzimmer_licht_brightness`

Klicke **Speichern**.

### 3. Virtual Inputs im Loxone Config einrichten

Ã–ffne Loxone Config und lege folgende Virtual Inputs an (Namen exakt wie oben):

- `ha_wohnzimmer_licht_on` â€” Typ: Virtual Input, Wert 0/1
- `ha_wohnzimmer_licht_brightness` â€” Typ: Virtual Input, Wert 0â€“100

Verbinde sie mit deinen Logik-BlÃ¶cken und lade die Config auf den Miniserver.

### 4. Alexa: Neue GerÃ¤te suchen

Sage: **â€žAlexa, suche nach neuen GerÃ¤ten"**  
oder Ã¶ffne die Alexa-App â†’ GerÃ¤te â†’ + â†’ GerÃ¤t hinzufÃ¼gen â†’ Licht â†’ Philips Hue.

Alexa findet die Bridge und alle angelegten GerÃ¤te. Danach kannst du sagen:  
**â€žAlexa, schalte Wohnzimmer Licht ein"**

---

## GerÃ¤te anlegen

### GerÃ¤tetypen

| Typ | Alexa-Befehle | Generierte Virtual Inputs | Wertebereich |
|---|---|---|---|
| `switch` | Ein/Aus | `ha_{name}_on` | `1` / `0` |
| `dimmer` | Ein/Aus, Helligkeit % | `ha_{name}_on`, `ha_{name}_brightness` | `1`/`0`, `0â€“100` |
| `color` | Ein/Aus, Helligkeit, Farbe | `ha_{name}_on`, `ha_{name}_brightness`, `ha_{name}_hue`, `ha_{name}_saturation` | diverse |
| `scene` | â€žAktiviere â€¦" | `ha_{name}_activate` | `1` (Puls) |

### Namensnormalisierung

Der GerÃ¤tename wird automatisch in einen Loxone-kompatiblen Virtual Input Namen umgewandelt:

| Eingabe | Normalisiert | VI-Prefix |
|---|---|---|
| `Wohnzimmer Licht` | `wohnzimmer_licht` | `ha_wohnzimmer_licht_` |
| `KÃ¼che Decke` | `kueche_decke` | `ha_kueche_decke_` |
| `Terrasse SÃ¼d` | `terrasse_sued` | `ha_terrasse_sued_` |
| `Jalousie EG` | `jalousie_eg` | `ha_jalousie_eg_` |

Regeln: Kleinbuchstaben, Umlaute â†’ ae/oe/ue/ss, Sonderzeichen â†’ Unterstrich.

### Beispiel: Alexa sagt â€žWohnzimmer Licht auf 60 Prozent"

```
Hue API â†’ PUT /api/{user}/lights/1/state
          { "on": true, "bri": 153 }

EchoLox sendet:
  GET .../dev/sps/io/ha_wohnzimmer_licht_on/1
  GET .../dev/sps/io/ha_wohnzimmer_licht_brightness/60
```

Die Helligkeit wird von Hue-Skala (0â€“254) auf Prozent (0â€“100) umgerechnet.

---

## Virtual Inputs in Loxone einrichten

### HTTP Virtual Input

Im Loxone Config unter **Peripherie â†’ Virtual Inputs**:

1. Neuen **Virtual HTTP Input** anlegen
2. Name: exakt wie von EchoLox generiert (z.B. `ha_wohnzimmer_licht_on`)
3. HTTP-Methode: GET
4. Der Virtual Input empfÃ¤ngt den Wert aus der URL: `/dev/sps/io/ha_wohnzimmer_licht_on/{value}`

### UDP Virtual Input (alternativ)

1. Neuen **Virtual UDP Input** anlegen
2. Port: `7777` (Standard, in EchoLox einstellbar)
3. Format: `{name}={value}` (EchoLox sendet `ha_wohnzimmer_licht_on=1\r\n`)

### Empfehlung

| Szenario | Transport |
|---|---|
| ZuverlÃ¤ssigkeit wichtig | **HTTP** â€” Antwort-BestÃ¤tigung, Basic Auth |
| Latenz wichtig (< 5 ms) | **UDP** â€” keine TCP-Verbindung |
| Bereits MQTT Gateway im Einsatz | **MQTT** |

---

## Alexa-Erkennung

EchoLox emuliert eine Philips Hue Bridge Generation 2 (BSB002). Die Erkennung lÃ¤uft Ã¼ber **SSDP/UPnP**:

1. Alexa sendet einen `M-SEARCH`-Broadcast ins Netzwerk (UDP Multicast `239.255.255.250:1900`)
2. EchoLox antwortet mit einer `HTTP/1.1 200 OK`-Unicast-Antwort
3. Alexa lÃ¤dt `/description.xml` und bestÃ¤tigt die Bridge-IdentitÃ¤t
4. Alexa ruft `/api/{user}/lights` ab und importiert alle GerÃ¤te

### Wichtig fÃ¼r die Erkennung

- LoxBerry und Echo mÃ¼ssen im **gleichen Subnetz** sein
- Port **8079** muss auf dem LoxBerry erreichbar sein (kein Firewall-Block)
- Die Bridge-ID bleibt **dauerhaft stabil** (aus der IP abgeleitet) â€” Alexa verliert die Bridge nicht nach einem Neustart

### Erkennung wiederholen

Wenn neue GerÃ¤te hinzugefÃ¼gt wurden:
- Alexa-App â†’ **GerÃ¤te â†’ Erkennen**
- oder: â€ž**Alexa, suche nach neuen GerÃ¤ten**"

---

## Sprachbefehle

| Befehl | Aktion | Gesendeter Virtual Input |
|---|---|---|
| â€žSchalte X ein" | on=1 | `ha_x_on = 1` |
| â€žSchalte X aus" | on=0 | `ha_x_on = 0` |
| â€žStelle X auf 50 Prozent" | bri=50% | `ha_x_on = 1`, `ha_x_brightness = 50` |
| â€žDimme X auf 20 Prozent" | bri=20% | `ha_x_brightness = 20` |
| â€žStelle X auf Rot" | hue+sat | `ha_x_hue = 0`, `ha_x_saturation = 100` |
| â€žAktiviere Szene Y" | activate=1 | `ha_y_activate = 1` |

### Tipp: Alexa-Gruppen

Fasse mehrere GerÃ¤te in der **Alexa-App zu einer Gruppe** zusammen (z.B. â€žWohnzimmer"). Dann funktioniert â€žAlexa, schalte Wohnzimmer aus" fÃ¼r alle GerÃ¤te der Gruppe gleichzeitig.

---

## Transports: HTTP, UDP, MQTT

### HTTP (Standard)

```
GET http://{miniserver-ip}:{port}/dev/sps/io/{vi-name}/{value}
Authorization: Basic base64(user:password)
```

- BestÃ¤tigung durch HTTP-Statuscode (200 = OK, 401 = falsche Credentials)
- Credentials aus der globalen LoxBerry Miniserver-Konfiguration

### UDP

```
{vi-name}={value}\r\n
```

Ziel: `{miniserver-ip}:{udp-port}` (Standard: 7777)

- Kein Verbindungsaufbau, geringste Latenz
- Kein Fehler-Feedback mÃ¶glich

### MQTT (Ã¼ber LoxBerry MQTT Gateway)

EchoLox published auf:
```
Topic:   ha_bridge/{device_name}/{property}
Payload: {value}
```

Das LoxBerry MQTT Gateway leitet die Nachrichten an den Miniserver weiter. EchoLox registriert beim Speichern eines GerÃ¤ts automatisch die nÃ¶tigen Subscriptions in der MQTT-Gateway-Konfiguration.

---

## Status-Ãœbersicht

Die Status-Seite (`/ui/status.html`) zeigt alle Virtual Inputs mit ihrem aktuellen Zustand â€” Ã¤hnlich der MQTT-Gateway â€žIncoming Overview".

| Icon | Status | Bedeutung |
|---|---|---|
| âœ… | ok | Virtual Input existiert im Miniserver und wurde zuletzt erfolgreich angesprochen |
| ðŸŸ  | not_found | Name im Miniserver nicht gefunden â€” Virtual Input noch nicht angelegt |
| ðŸ”´ | access_denied | Falsche Credentials fÃ¼r den Miniserver |
| â¬œ | not_sent | GerÃ¤t noch nie ausgelÃ¶st |

```
â”Œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¬â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”
â”‚ Loxone Virtual Input Name            â”‚ Letzter Wert â”‚ Zuletzt gesendetâ”‚
â”œâ”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¼â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”¤
â”‚ âœ… ha_wohnzimmer_licht_on            â”‚ 1            â”‚ 05.05. 22:14:03 â”‚
â”‚ âœ… ha_wohnzimmer_licht_brightness    â”‚ 60           â”‚ 05.05. 22:14:03 â”‚
â”‚ ðŸŸ  ha_schlafzimmer_decke_on          â”‚ â€”            â”‚ nie             â”‚
â”‚ â¬œ ha_terrasse_szene_activate        â”‚ â€”            â”‚ nie             â”‚
â””â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”´â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”˜
```

Ãœber **Refresh** wird die Loxone-Struktur (`LoxAPP3.json`) neu abgefragt.

---

## Import alter Konfiguration

Wer bisher ha-bridge verwendet hat, kann die bestehende `devices.db` importieren.

### Web-UI Import

1. Ã–ffne **EchoLox â†’ Import**
2. Lade die `devices.db` hoch (Drag & Drop)
3. EchoLox zeigt eine Vorschau:
   - Erkannte GerÃ¤te mit generierten Virtual Input Namen
   - Ãœbersprungene GerÃ¤te (Vera, Fibaro, Harmony, â€¦ â€” diese Plattformen werden nicht mehr unterstÃ¼tzt)
4. Namen kÃ¶nnen manuell angepasst werden
5. Klicke **Importieren**

### Mapping-Logik

| Alter Typ | Neues Device | Hinweis |
|---|---|---|
| `httpDevice` (on/off URL) | `switch` | Virtual Input: `_on` |
| `httpDevice` (mit dimUrl) | `dimmer` | Virtual Inputs: `_on`, `_brightness` |
| `httpDevice` (mit colorUrl) | `color` | Virtual Inputs: `_on`, `_brightness`, `_hue`, `_saturation` |
| `execDevice` | `switch` | Exec-Logik entfÃ¤llt |
| `veraDevice` | â€” | Ãœbersprungen âš ï¸ |
| `harmonyDevice` | â€” | Ãœbersprungen âš ï¸ |
| alle anderen Plattform-Typen | â€” | Ãœbersprungen âš ï¸ |

### CLI-Import (alternativ)

```bash
EchoLox --import /pfad/zur/devices.db --import-out /opt/loxberry/data/plugins/EchoLox/devices.json
```

---

## Einstellungen

Alle Einstellungen sind Ã¼ber die Web-UI unter `/ui/settings.html` zugÃ¤nglich.

| Einstellung | Standard | Beschreibung |
|---|---|---|
| **Miniserver** | Erster in LoxBerry | Dropdown aus globaler LoxBerry-Konfiguration |
| **Transport** | HTTP | HTTP, UDP oder MQTT |
| **UDP Port** | 7777 | Ziel-Port fÃ¼r UDP-Transport |
| **EchoLox Port** | 8079 | Port auf dem EchoLox lauscht |
| **MQTT Broker** | tcp://localhost:1883 | Nur relevant bei MQTT-Transport |

> **Hinweis:** Miniserver-IP, Benutzername und Passwort werden **ausschlieÃŸlich** aus der globalen LoxBerry-Konfiguration (`/opt/loxberry/config/system/miniserver.json`) gelesen. Du musst sie nicht erneut eingeben.

---

## Konfigurationsdatei

```yaml
# /opt/loxberry/config/plugins/EchoLox/EchoLox.cfg

server:
  port: 8079    # Port auf dem EchoLox lauscht
  ip: ""        # Leer = automatisch erkannt (empfohlen)

upnp:
  name: "EchoLox"
  uuid: ""      # Leer = aus IP abgeleitet (stabil Ã¼ber Neustarts)

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

Die GerÃ¤te werden gespeichert in:
```
/opt/loxberry/data/plugins/EchoLox/devices.json
```

---

## Technische Architektur

### Projektstruktur

```
EchoLox/
â”œâ”€â”€ cmd/EchoLox/
â”‚   â””â”€â”€ main.go                    # Entry Point, Flag-Parsing
â”œâ”€â”€ internal/
â”‚   â”œâ”€â”€ identity/
â”‚   â”‚   â””â”€â”€ identity.go            # Bridge-IdentitÃ¤t (UUID, bridgeid, MAC)
â”‚   â”œâ”€â”€ bridge/
â”‚   â”‚   â”œâ”€â”€ bridge.go              # HTTP-Server Init, Komponentenverbindung
â”‚   â”‚   â””â”€â”€ config.go              # YAML-Config, LoxBerry Env-Vars
â”‚   â”œâ”€â”€ hue/
â”‚   â”‚   â”œâ”€â”€ api.go                 # Hue REST Endpoints (lights, groups, config)
â”‚   â”‚   â””â”€â”€ state.go               # Brightness/Hue/Sat Konvertierung
â”‚   â”œâ”€â”€ upnp/
â”‚   â”‚   â””â”€â”€ listener.go            # SSDP Multicast-Listener + description.xml
â”‚   â”œâ”€â”€ device/
â”‚   â”‚   â”œâ”€â”€ manager.go             # Device-Registry (CRUD, State, HueID)
â”‚   â”‚   â”œâ”€â”€ store.go               # JSON-Persistenz (atomic write)
â”‚   â”‚   â”œâ”€â”€ model.go               # Device/State Typen
â”‚   â”‚   â””â”€â”€ naming.go              # Auto-Namens-Generierung
â”‚   â”œâ”€â”€ loxone/
â”‚   â”‚   â”œâ”€â”€ client.go              # HTTP + UDP Transport
â”‚   â”‚   â”œâ”€â”€ verify.go              # LoxAPP3.json Abfrage, Status-Check
â”‚   â”‚   â”œâ”€â”€ lbconfig.go            # LoxBerry Miniserver-Config lesen
â”‚   â”‚   â””â”€â”€ mqttbridge.go          # MQTT-Gateway Auto-Registrierung
â”‚   â”œâ”€â”€ api/
â”‚   â”‚   â””â”€â”€ handler.go             # Management REST API (/echolox/api/...)
â”‚   â”œâ”€â”€ migrate/
â”‚   â”‚   â””â”€â”€ importer.go            # devices.db â†’ neues Format
â”‚   â””â”€â”€ web/
â”‚       â””â”€â”€ handler.go             # Embedded Web-UI Handler
â””â”€â”€ webembed/
    â””â”€â”€ web/                       # Eingebettetes Frontend (embed.FS)
        â”œâ”€â”€ index.html             # GerÃ¤teÃ¼bersicht
        â”œâ”€â”€ device.html            # Anlegen / Bearbeiten
        â”œâ”€â”€ status.html            # Virtual Input Status
        â”œâ”€â”€ settings.html          # Einstellungen
        â”œâ”€â”€ import.html            # Import
        â””â”€â”€ assets/
            â”œâ”€â”€ app.js
            â””â”€â”€ style.css
```

### Bridge-IdentitÃ¤t

Die Bridge-ID (hue-bridgeid) und UUID werden **deterministisch aus der IP-Adresse** abgeleitet:

```
IP: 192.168.1.100
 â†’ MD5("echolox-bridge:192.168.1.100")
 â†’ suffix = 12 Hex-Zeichen
 â†’ UUID    = 2f402f80-da50-11e1-9b23-{suffix}
 â†’ bridgeid = 001788FFFE{suffix[6:]}  (Philips OUI + FFFE + 6 Hex)
```

Das bedeutet: Nach einem Neustart behÃ¤lt EchoLox dieselbe IdentitÃ¤t â€” Alexa verliert die Bridge nicht.

### SSDP-Flow (Alexa-Erkennung)

```
Echo                          EchoLox
 â”‚                                â”‚
 â”‚â”€â”€ M-SEARCH (UDP Multicast) â”€â”€â”€â–¶â”‚  Port 1900
 â”‚                                â”‚  isMSearch() prÃ¼ft ST-Header
 â”‚â—€â”€â”€ HTTP/1.1 200 OK (Unicast) â”€â”€â”‚  eigener ephemerer UDP-Socket
 â”‚    LOCATION: .../description.xml
 â”‚    USN: uuid:...::urn:...:Basic:1
 â”‚    hue-bridgeid: 001788FFFE...
 â”‚                                â”‚
 â”‚â”€â”€ GET /description.xml â”€â”€â”€â”€â”€â”€â”€â”€â–¶â”‚
 â”‚â—€â”€â”€ XML (serialNumber=bridgeid) â”€â”‚
 â”‚                                â”‚
 â”‚â”€â”€ POST /api (Pairing) â”€â”€â”€â”€â”€â”€â”€â”€â”€â–¶â”‚
 â”‚â—€â”€â”€ [{"success":{"username":...}}]
 â”‚                                â”‚
 â”‚â”€â”€ GET /api/{user}/lights â”€â”€â”€â”€â”€â”€â–¶â”‚
 â”‚â—€â”€â”€ {"1": {...}, "2": {...}}  â”€â”€â”€â”‚
```

### Hue Light IDs

Intern werden GerÃ¤te mit UUIDs gespeichert. FÃ¼r die Hue API wird jedes GerÃ¤t einer stabilen, kurzen numerischen ID (`HueID`) zugewiesen â€” diese wird beim ersten Anlegen vergeben und dann permanent in der `devices.json` gespeichert.

---

## Build & Cross-Compilation

### Voraussetzungen

- Go 1.22 oder neuer
- Internet (fÃ¼r `go mod download`)

### Alle Plattformen bauen

```bash
make all
```

Erzeugt:
- `bin/EchoLox-arm64` â€” Raspberry Pi 4, Orange Pi, Pine64
- `bin/EchoLox-armv7` â€” Raspberry Pi 2/3 (32-bit ARM)
- `bin/EchoLox-amd64` â€” DietPi x86, VirtualBox VM

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

EchoLox startet und ist unter `http://localhost:8079/ui/` erreichbar. Ohne Miniserver-Konfiguration lÃ¤uft alles im Dry-Run-Modus (Virtual Inputs werden nur geloggt, nicht gesendet).

### Plugin-ZIP erstellen

```bash
make zip
```

Erzeugt `EchoLox-1.0.0.zip` â€” direkt in den LoxBerry Plugin Manager hochladbar.

---

## Troubleshooting

### Alexa findet die Bridge nicht

**PrÃ¼fpunkte:**

1. **description.xml erreichbar?**
   ```
   http://<loxberry-ip>:8079/description.xml
   ```
   Muss ein XML mit `Philips hue bridge 2015` zurÃ¼ckgeben.

2. **Port 8079 offen?**
   ```bash
   # Auf dem LoxBerry:
   ss -tulnp | grep 8079
   ```

3. **Gleicher Subnetz?** Echo und LoxBerry mÃ¼ssen im selben Subnetz sein. SSDP-Broadcasts werden nicht Ã¼ber Router-Grenzen weitergeleitet.

4. **SSDP-Listener lÃ¤uft?** Im LoxBerry Log prÃ¼fen:
   ```
   SSDP listener on 239.255.255.250:1900  bridgeid=001788FFFE...
   ```

5. **Firewall?** Auf manchen Systemen blockiert `ufw` oder `iptables` UDP Port 1900:
   ```bash
   iptables -A INPUT -p udp --dport 1900 -j ACCEPT
   ```

6. **SSDP-Konflikt auf LoxBerry?** LoxBerry und der Loxone Miniserver nutzen selbst SSDP/UPnP (Port 1900 UDP). Wenn ein anderer Dienst Port 1900 exklusiv belegt, kann EchoLox den SSDP-Listener nicht starten:
   ```bash
   # Prüfen ob Port 1900 belegt ist:
   ss -ulnp | grep 1900
   ```
   Falls ein Konflikt besteht (z.B. `avahi-daemon`, `miniupnpd` oder ein anderes Plugin), diesen Dienst stoppen:
   ```bash
   systemctl stop avahi-daemon
   systemctl disable avahi-daemon
   ```
   EchoLox benötigt Zugriff auf den UDP-Multicast-Port 1900, um SSDP-M-SEARCH-Anfragen von Alexa empfangen zu können. Ohne funktionierende SSDP-Discovery findet Alexa die Bridge nicht automatisch — die manuelle Geräteerkennung (`description.xml` direkt aufrufen) funktioniert aber weiterhin.


### Alexa erkennt GerÃ¤te, aber Befehle kommen nicht an

1. **Testen-Button** in EchoLox â†’ GerÃ¤teliste â†’ Testen: Sendet einen Testbefehl direkt an den Miniserver.

2. **Status-Seite** prÃ¼fen: Zeigt `not_found`?  
   â†’ Virtual Input im Loxone Config anlegen, Namen exakt prÃ¼fen.

3. **Loxone Log** prÃ¼fen: Kommen HTTP-Requests beim Miniserver an?

4. **Credentials** prÃ¼fen: LoxBerry Miniserver-Konfiguration â†’ Verbindungstest in EchoLox Einstellungen.

### Virtual Input Name stimmt nicht

Der Name in Loxone Config muss **exakt** dem generierten Namen entsprechen (GroÃŸ-/Kleinschreibung beachtet, kein Leerzeichen).

Beispiel: GerÃ¤t heiÃŸt `Wohnzimmer Licht` â†’ Virtual Input muss heiÃŸen `ha_wohnzimmer_licht_on`.

### Nach Neustart verliert Alexa die GerÃ¤te

Das sollte nicht passieren â€” die Bridge-ID ist deterministisch aus der IP abgeleitet. Falls die LoxBerry-IP sich geÃ¤ndert hat, Ã¤ndert sich auch die Bridge-ID. LÃ¶sung: Feste IP fÃ¼r den LoxBerry vergeben (DHCP-Reservation im Router).

### Import schlÃ¤gt fehl

- Nur `devices.db` aus ha-bridge wird unterstÃ¼tzt (JSON-Array-Format)
- Plattform-spezifische GerÃ¤te (Vera, Fibaro, Harmony, â€¦) werden Ã¼bersprungen â€” das ist erwartet
- Bei kaputtem JSON: Datei in einem Editor Ã¶ffnen und auf GÃ¼ltigkeit prÃ¼fen

---

## FAQ

**Kann EchoLox mehrere Loxone Miniserver gleichzeitig ansprechen?**  
Nein â€” aktuell wird pro Transport ein Miniserver unterstÃ¼tzt. Mehrere Miniserver sind geplant.

**Muss ich den Hue-Link-Button drÃ¼cken?**  
Nein â€” EchoLox akzeptiert alle Pairing-Anfragen automatisch.

**Funktioniert EchoLox auch mit Google Home oder Apple HomeKit?**  
Google Home unterstÃ¼tzt die Hue-Emulation ebenfalls, ist aber nicht primÃ¤r getestet. Apple HomeKit nutzt ein anderes Protokoll (HomeKit Accessory Protocol) und wird nicht unterstÃ¼tzt.

**Kann ich bestehende Alexa-Routinen behalten?**  
Ja â€” solange die GerÃ¤tenamen gleich bleiben und die Bridge-ID stabil ist (was sie ist, solange die LoxBerry-IP gleich bleibt), funktionieren alle Routinen weiterhin.

**Wie viele GerÃ¤te sind mÃ¶glich?**  
Die Hue API unterstÃ¼tzt bis zu 50 Lampen. EchoLox hat keine eigene Begrenzung.

**Funktioniert EchoLox ohne Internet?**  
VollstÃ¤ndig â€” SSDP-Discovery und Hue-API laufen rein lokal. Keine Cloud-Verbindung nÃ¶tig.

**Kann ich die Web-UI von auÃŸerhalb des LANs erreichen?**  
Nur Ã¼ber VPN oder wenn Port 8079 im Router weitergeleitet wird (nicht empfohlen â€” kein HTTPS, keine Authentifizierung).

---

## Lizenz

MIT License â€” siehe [LICENSE](LICENSE)

---

## Credits

Basiert auf den Ideen von [ha-bridge](https://github.com/bwssystems/ha-bridge) von bwssystems.  
Neu entwickelt in Go als schlankes LoxBerry-Plugin.

Maintainer: Johannes Battlogg â€” johannes@battlogg.org
